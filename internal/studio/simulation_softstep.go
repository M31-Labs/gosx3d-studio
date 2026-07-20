package studio

import (
	"math"
	"sort"
)

// The soft-step solver adopts Box3D's core ideas — substepping with soft
// contact constraints, deterministic ordering, sensors as response-free
// contacts — in a pure-Go reference implementation that keeps Studio's
// hash-verified recording/replay contract. It is opted into per profile via
// SubSteps; the legacy single-step loop stays byte-identical otherwise.
//
// Floors are named, not hidden: colliders are world-axis-aligned (entity
// rotation is not applied to collision shapes until OBB SAT lands),
// box-vs-box contacts are not generated yet, and bodies do not rotate.

const (
	softStepRelaxIterations   = 2
	softStepBiasFactor        = 0.2
	softStepSlop              = 0.005
	softStepMaxBiasSpeed      = 4.0
	softStepRestitutionCutoff = 0.5
)

type softContact struct {
	body    ID // the dynamic body being resolved
	other   ID
	normal  Vec3 // points from other toward body (push direction)
	depth   float64
	point   Vec3
	sensor  bool
	normalJ float64 // accumulated normal impulse for friction clamping
}

func runSoftStepSimulation(result *Document, profile SimulationProfile, ticks uint64, orderedInputs []SimulationInput, recording *SimulationRecording) {
	dt := 1 / float64(profile.TickRate)
	substeps := profile.SubSteps
	h := dt / float64(substeps)
	bodies := append([]ID(nil), profile.Bodies...)
	sort.Slice(bodies, func(i, j int) bool { return bodies[i] < bodies[j] })
	activeSensorPairs := map[string]bool{}
	activeContactPairs := map[string]bool{}
	inputIndex := 0
	for tick := uint64(0); tick < ticks; tick++ {
		for inputIndex < len(orderedInputs) && orderedInputs[inputIndex].Tick == tick {
			input := orderedInputs[inputIndex]
			entity := result.Entities[input.Entity]
			entity.Physics.Velocity = addVec(entity.Physics.Velocity, scaleVec(input.Impulse, 1/entity.Physics.Mass))
			result.Entities[input.Entity] = entity
			inputIndex++
		}
		tickSensorPairs := map[string]bool{}
		tickContactPairs := map[string]softContact{}
		for step := 0; step < substeps; step++ {
			for _, id := range bodies {
				entity := result.Entities[id]
				if entity.Physics.Kind != "dynamic" {
					continue
				}
				entity.Physics.Velocity = addVec(entity.Physics.Velocity, scaleVec(profile.Gravity, entity.Physics.GravityScale*h))
				result.Entities[id] = entity
			}
			contacts := collectSoftContacts(result, bodies)
			for iteration := 0; iteration < softStepRelaxIterations; iteration++ {
				for index := range contacts {
					contact := &contacts[index]
					if contact.sensor {
						continue
					}
					solveSoftContact(result, contact, h, iteration == 0)
				}
				for _, joint := range profile.Joints {
					solveDistanceJoint(result, joint, h)
				}
			}
			for _, id := range bodies {
				entity := result.Entities[id]
				if entity.Physics.Kind != "dynamic" {
					continue
				}
				entity.Transform.Position = addVec(entity.Transform.Position, scaleVec(entity.Physics.Velocity, h))
				result.Entities[id] = entity
			}
			for index := range contacts {
				contact := contacts[index]
				key := string(contact.body) + "|" + string(contact.other)
				if contact.sensor {
					tickSensorPairs[key] = true
				} else if existing, ok := tickContactPairs[key]; !ok || contact.depth > existing.depth {
					tickContactPairs[key] = contact
				}
			}
		}
		emitContactBegins(activeContactPairs, tickContactPairs, tick+1, recording)
		emitSensorTransitions(activeSensorPairs, tickSensorPairs, tick+1, result, recording)
	}
}

// emitContactBegins records one "contact" event per solid pair the first
// tick it touches, with the accumulated normal impulse as the impact speed
// proxy. Ongoing resting contact does not spam events; separation re-arms.
func emitContactBegins(active map[string]bool, current map[string]softContact, tick uint64, recording *SimulationRecording) {
	keys := make([]string, 0, len(current))
	for key := range current {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !active[key] {
			contact := current[key]
			recording.Events = append(recording.Events, SimulationEvent{Tick: tick, Kind: "contact", Entity: contact.body, Other: contact.other, Position: contact.point, Speed: contact.normalJ})
			active[key] = true
		}
	}
	for key := range active {
		if _, ok := current[key]; !ok {
			delete(active, key)
		}
	}
}

// emitSensorTransitions diffs the overlapping sensor pairs against the
// previous tick and records begin/end events in deterministic order.
func emitSensorTransitions(active, current map[string]bool, tick uint64, document *Document, recording *SimulationRecording) {
	keys := make([]string, 0, len(active)+len(current))
	for key := range active {
		keys = append(keys, key)
	}
	for key := range current {
		if !active[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		body, other := splitSensorKey(key)
		position := document.Entities[body].Transform.Position
		if current[key] && !active[key] {
			recording.Events = append(recording.Events, SimulationEvent{Tick: tick, Kind: "sensor-begin", Entity: body, Other: other, Position: position})
			active[key] = true
		} else if !current[key] && active[key] {
			recording.Events = append(recording.Events, SimulationEvent{Tick: tick, Kind: "sensor-end", Entity: body, Other: other, Position: position})
			delete(active, key)
		}
	}
}

func splitSensorKey(key string) (ID, ID) {
	for index := 0; index < len(key); index++ {
		if key[index] == '|' {
			return ID(key[:index]), ID(key[index+1:])
		}
	}
	return ID(key), ""
}

// collectSoftContacts runs a deterministic sorted-pair sweep: planes pair
// with every dynamic body; finite shapes prune by AABB overlap first.
func collectSoftContacts(document *Document, bodies []ID) []softContact {
	contacts := []softContact{}
	for _, id := range bodies {
		entity := document.Entities[id]
		if entity.Physics.Kind != "dynamic" {
			continue
		}
		for _, otherID := range bodies {
			if otherID == id {
				continue
			}
			other := document.Entities[otherID]
			if other.Physics == nil {
				continue
			}
			if other.Physics.Kind == "dynamic" && otherID < id {
				continue // dynamic pairs handled once, resolved symmetrically
			}
			if !softAABBOverlap(entity, other) {
				continue
			}
			if contact, ok := softNarrowphase(entity, other); ok {
				contact.body = id
				contact.other = otherID
				contact.sensor = entity.Physics.Collider.Sensor || other.Physics.Collider.Sensor
				contacts = append(contacts, contact)
			}
		}
	}
	sort.Slice(contacts, func(i, j int) bool {
		if contacts[i].body != contacts[j].body {
			return contacts[i].body < contacts[j].body
		}
		return contacts[i].other < contacts[j].other
	})
	return contacts
}

func softAABB(entity Entity) (Vec3, Vec3, bool) {
	position := entity.Transform.Position
	collider := entity.Physics.Collider
	switch collider.Kind {
	case "sphere":
		r := collider.Radius
		return Vec3{X: position.X - r, Y: position.Y - r, Z: position.Z - r}, Vec3{X: position.X + r, Y: position.Y + r, Z: position.Z + r}, true
	case "box":
		he := *collider.HalfExtents
		return Vec3{X: position.X - he.X, Y: position.Y - he.Y, Z: position.Z - he.Z}, Vec3{X: position.X + he.X, Y: position.Y + he.Y, Z: position.Z + he.Z}, true
	case "capsule":
		r, hh := collider.Radius, collider.HalfHeight
		return Vec3{X: position.X - r, Y: position.Y - hh - r, Z: position.Z - r}, Vec3{X: position.X + r, Y: position.Y + hh + r, Z: position.Z + r}, true
	}
	return Vec3{}, Vec3{}, false // planes and unknowns: no finite AABB
}

func softAABBOverlap(a, b Entity) bool {
	minA, maxA, finiteA := softAABB(a)
	minB, maxB, finiteB := softAABB(b)
	if !finiteA || !finiteB {
		return true // plane participates with everything
	}
	return minA.X <= maxB.X && minB.X <= maxA.X && minA.Y <= maxB.Y && minB.Y <= maxA.Y && minA.Z <= maxB.Z && minB.Z <= maxA.Z
}

// softNarrowphase produces one contact between a dynamic body and another
// collider. The normal points from other toward body.
func softNarrowphase(body, other Entity) (softContact, bool) {
	collider := body.Physics.Collider
	position := body.Transform.Position
	switch other.Physics.Collider.Kind {
	case "plane":
		normal := normalizeVec(other.Physics.Collider.Normal)
		planeOffset := other.Physics.Collider.Offset + dotVec(normal, other.Transform.Position)
		support := softSupportRadius(collider, normal)
		distance := dotVec(normal, position) - planeOffset
		depth := support - distance
		if collider.Kind == "capsule" {
			// Two endpoint spheres; deepest one wins.
			depth = -math.MaxFloat64
			for _, sign := range []float64{-1, 1} {
				end := addVec(position, Vec3{Y: sign * collider.HalfHeight})
				endDepth := collider.Radius - (dotVec(normal, end) - planeOffset)
				if endDepth > depth {
					depth = endDepth
				}
			}
		}
		if depth <= 0 {
			return softContact{}, false
		}
		return softContact{normal: normal, depth: depth, point: addVec(position, scaleVec(normal, -(support-depth)))}, true
	case "sphere", "box", "capsule":
		closest := softClosestPoint(other, position, collider)
		delta := subSoftVec(position, closest)
		distance := vectorLength(delta)
		reach := softSupportRadius(collider, Vec3{}) // radius-like reach of the dynamic body
		if collider.Kind == "box" {
			// Approximate the box by its inscribed sphere against finite shapes;
			// exact box-vs-finite manifolds are a named deferral.
			reach = math.Min(math.Min(collider.HalfExtents.X, collider.HalfExtents.Y), collider.HalfExtents.Z)
		}
		if distance >= reach || distance <= 1e-12 {
			return softContact{}, false
		}
		normal := scaleVec(delta, 1/distance)
		return softContact{normal: normal, depth: reach - distance, point: closest}, true
	}
	return softContact{}, false
}

// softSupportRadius projects the collider's support extent along a normal
// (or returns the spherical reach when the normal is zero).
func softSupportRadius(collider Collider, normal Vec3) float64 {
	switch collider.Kind {
	case "sphere":
		return collider.Radius
	case "capsule":
		if normal == (Vec3{}) {
			return collider.Radius
		}
		return collider.Radius + math.Abs(normal.Y)*collider.HalfHeight
	case "box":
		if normal == (Vec3{}) {
			he := *collider.HalfExtents
			return math.Min(math.Min(he.X, he.Y), he.Z)
		}
		he := *collider.HalfExtents
		return math.Abs(normal.X)*he.X + math.Abs(normal.Y)*he.Y + math.Abs(normal.Z)*he.Z
	}
	return 0
}

// softClosestPoint returns the closest point on the other collider's surface
// region toward the dynamic body's center.
func softClosestPoint(other Entity, target Vec3, _ Collider) Vec3 {
	position := other.Transform.Position
	collider := other.Physics.Collider
	switch collider.Kind {
	case "sphere":
		delta := subSoftVec(target, position)
		length := vectorLength(delta)
		if length <= 1e-12 {
			return position
		}
		return addVec(position, scaleVec(delta, collider.Radius/length))
	case "box":
		he := *collider.HalfExtents
		return Vec3{
			X: clampFloat(target.X, position.X-he.X, position.X+he.X),
			Y: clampFloat(target.Y, position.Y-he.Y, position.Y+he.Y),
			Z: clampFloat(target.Z, position.Z-he.Z, position.Z+he.Z),
		}
	case "capsule":
		segmentY := clampFloat(target.Y, position.Y-collider.HalfHeight, position.Y+collider.HalfHeight)
		center := Vec3{X: position.X, Y: segmentY, Z: position.Z}
		delta := subSoftVec(target, center)
		length := vectorLength(delta)
		if length <= 1e-12 {
			return center
		}
		return addVec(center, scaleVec(delta, collider.Radius/length))
	}
	return position
}

// solveSoftContact applies a soft normal impulse (bias from penetration
// depth) plus a Coulomb-clamped friction impulse. Restitution applies on the
// first relaxation pass only, above the approach-speed cutoff.
func solveSoftContact(document *Document, contact *softContact, h float64, firstPass bool) {
	body := document.Entities[contact.body]
	other := document.Entities[contact.other]
	invMassBody := 0.0
	if body.Physics.Kind == "dynamic" && body.Physics.Mass > 0 {
		invMassBody = 1 / body.Physics.Mass
	}
	invMassOther := 0.0
	if other.Physics.Kind == "dynamic" && other.Physics.Mass > 0 {
		invMassOther = 1 / other.Physics.Mass
	}
	invMassSum := invMassBody + invMassOther
	if invMassSum <= 0 {
		return
	}
	relative := subSoftVec(body.Physics.Velocity, other.Physics.Velocity)
	normalSpeed := dotVec(relative, contact.normal)
	bias := math.Min(softStepBiasFactor*math.Max(contact.depth-softStepSlop, 0)/h, softStepMaxBiasSpeed)
	restitution := 0.0
	if firstPass && normalSpeed < -softStepRestitutionCutoff {
		restitution = math.Max(body.Physics.Restitution, other.Physics.Restitution) * -normalSpeed
	}
	impulse := -(normalSpeed - bias - restitution) / invMassSum
	if impulse < 0 {
		return
	}
	contact.normalJ += impulse
	body.Physics.Velocity = addVec(body.Physics.Velocity, scaleVec(contact.normal, impulse*invMassBody))
	other.Physics.Velocity = addVec(other.Physics.Velocity, scaleVec(contact.normal, -impulse*invMassOther))

	relative = subSoftVec(body.Physics.Velocity, other.Physics.Velocity)
	tangent := subSoftVec(relative, scaleVec(contact.normal, dotVec(relative, contact.normal)))
	tangentSpeed := vectorLength(tangent)
	friction := math.Max(body.Physics.Friction, other.Physics.Friction)
	if tangentSpeed > 1e-9 && friction > 0 {
		direction := scaleVec(tangent, 1/tangentSpeed)
		frictionImpulse := math.Min(tangentSpeed/invMassSum, friction*contact.normalJ)
		body.Physics.Velocity = addVec(body.Physics.Velocity, scaleVec(direction, -frictionImpulse*invMassBody))
		other.Physics.Velocity = addVec(other.Physics.Velocity, scaleVec(direction, frictionImpulse*invMassOther))
	}
	document.Entities[contact.body] = body
	document.Entities[contact.other] = other
}

func subSoftVec(a, b Vec3) Vec3 { return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z} }

// solveDistanceJoint applies impulses along the anchor axis for the distance
// joint family: rigid rod (Length, Stiffness 0), one-sided Min/MaxLength
// limits, a semi-implicit spring (Stiffness/Damping), and a force-capped
// motor toward MotorTarget. Deterministic: joints are stored in profile
// order and solved inside the sorted relaxation loop.
func solveDistanceJoint(document *Document, joint PhysicsJoint, h float64) {
	a := document.Entities[joint.BodyA]
	b := document.Entities[joint.BodyB]
	if a.Physics == nil || b.Physics == nil {
		return
	}
	invA, invB := 0.0, 0.0
	if a.Physics.Kind == "dynamic" && a.Physics.Mass > 0 {
		invA = 1 / a.Physics.Mass
	}
	if b.Physics.Kind == "dynamic" && b.Physics.Mass > 0 {
		invB = 1 / b.Physics.Mass
	}
	invSum := invA + invB
	if invSum <= 0 {
		return
	}
	pa := addVec(a.Transform.Position, joint.AnchorA)
	pb := addVec(b.Transform.Position, joint.AnchorB)
	delta := subSoftVec(pb, pa)
	distance := vectorLength(delta)
	if distance <= 1e-9 {
		return
	}
	axis := scaleVec(delta, 1/distance)
	relative := dotVec(subSoftVec(b.Physics.Velocity, a.Physics.Velocity), axis)
	apply := func(impulse float64) {
		a.Physics.Velocity = addVec(a.Physics.Velocity, scaleVec(axis, -impulse*invA))
		b.Physics.Velocity = addVec(b.Physics.Velocity, scaleVec(axis, impulse*invB))
		relative = dotVec(subSoftVec(b.Physics.Velocity, a.Physics.Velocity), axis)
	}
	if joint.Stiffness > 0 {
		stretch := distance - joint.Length
		apply((-joint.Stiffness*stretch - joint.Damping*relative) * h)
	} else if joint.Length > 0 && !(joint.MotorMaxForce > 0 && joint.MotorSpeed > 0) {
		violation := distance - joint.Length
		bias := math.Min(math.Abs(softStepBiasFactor*violation/h), softStepMaxBiasSpeed)
		if violation < 0 {
			bias = -bias
		}
		apply(-(relative + bias) / invSum)
	}
	if joint.MaxLength > 0 && distance > joint.MaxLength {
		bias := math.Min(softStepBiasFactor*(distance-joint.MaxLength)/h, softStepMaxBiasSpeed)
		impulse := -(relative + bias) / invSum
		if impulse < 0 {
			apply(impulse)
		}
	}
	if joint.MinLength > 0 && distance < joint.MinLength {
		bias := math.Min(softStepBiasFactor*(joint.MinLength-distance)/h, softStepMaxBiasSpeed)
		impulse := (bias - relative) / invSum
		if impulse > 0 {
			apply(impulse)
		}
	}
	if joint.MotorMaxForce > 0 && joint.MotorSpeed > 0 {
		targetVelocity := clampFloat((joint.MotorTarget-distance)/h, -joint.MotorSpeed, joint.MotorSpeed)
		impulse := clampFloat((targetVelocity-relative)/invSum, -joint.MotorMaxForce*h, joint.MotorMaxForce*h)
		apply(impulse)
	}
	document.Entities[joint.BodyA] = a
	document.Entities[joint.BodyB] = b
}
