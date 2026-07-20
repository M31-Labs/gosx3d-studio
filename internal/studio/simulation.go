package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

type SimulationProfile struct {
	ID       ID     `json:"id"`
	Name     string `json:"name"`
	TickRate int    `json:"tickRate"`
	Gravity  Vec3   `json:"gravity"`
	Bodies   []ID   `json:"bodies"`
	// SubSteps opts the profile into the soft-step substepping solver
	// (adopted from Box3D's design). Zero keeps the legacy single-step
	// reference loop byte-identical for existing recordings.
	SubSteps int `json:"subSteps,omitempty"`
	// Joints constrain body pairs; the current floor is the distance joint
	// family (limits, motors, springs). Revolute and prismatic joints are
	// gated on body rotation landing.
	Joints []PhysicsJoint `json:"joints,omitempty"`
}

// PhysicsJoint is a deterministic constraint between two profile bodies.
// Kind "distance" supports: a rest Length (rigid rod when Stiffness is 0),
// Min/MaxLength limits (free inside the range), a spring (Stiffness +
// Damping, semi-implicit), and a motor that drives the current length
// toward MotorTarget at MotorSpeed with impulses capped by MotorMaxForce.
// When a motor is active the rest Length is reference-only (the motor owns
// the axis); limits still clamp the range.
type PhysicsJoint struct {
	ID            ID      `json:"id"`
	Kind          string  `json:"kind"`
	BodyA         ID      `json:"bodyA"`
	BodyB         ID      `json:"bodyB"`
	AnchorA       Vec3    `json:"anchorA,omitempty"`
	AnchorB       Vec3    `json:"anchorB,omitempty"`
	Length        float64 `json:"length,omitempty"`
	MinLength     float64 `json:"minLength,omitempty"`
	MaxLength     float64 `json:"maxLength,omitempty"`
	Stiffness     float64 `json:"stiffness,omitempty"`
	Damping       float64 `json:"damping,omitempty"`
	MotorTarget   float64 `json:"motorTarget,omitempty"`
	MotorSpeed    float64 `json:"motorSpeed,omitempty"`
	MotorMaxForce float64 `json:"motorMaxForce,omitempty"`
}

type PhysicsBody struct {
	Kind         string   `json:"kind"`
	Mass         float64  `json:"mass,omitempty"`
	Velocity     Vec3     `json:"velocity,omitempty"`
	GravityScale float64  `json:"gravityScale,omitempty"`
	Restitution  float64  `json:"restitution,omitempty"`
	Friction     float64  `json:"friction,omitempty"`
	Collider     Collider `json:"collider"`
}

type Collider struct {
	Kind   string  `json:"kind"`
	Radius float64 `json:"radius,omitempty"`
	Normal Vec3    `json:"normal,omitempty"`
	Offset float64 `json:"offset,omitempty"`
	// HalfExtents sizes a box collider (axis-aligned floor; collider
	// orientation is a named deferral until OBB SAT lands).
	HalfExtents *Vec3 `json:"halfExtents,omitempty"`
	// HalfHeight is the capsule half-length between hemisphere centers.
	HalfHeight float64 `json:"halfHeight,omitempty"`
	// Sensor colliders report begin/end events without collision response.
	Sensor bool `json:"sensor,omitempty"`
}

type SimulationInput struct {
	Tick    uint64 `json:"tick"`
	Entity  ID     `json:"entity"`
	Impulse Vec3   `json:"impulse"`
}

type SimulationEvent struct {
	Tick     uint64  `json:"tick"`
	Kind     string  `json:"kind"`
	Entity   ID      `json:"entity"`
	Other    ID      `json:"other,omitempty"`
	Position Vec3    `json:"position"`
	Speed    float64 `json:"speed,omitempty"`
}

type SimulationSnapshot struct {
	Tick       uint64           `json:"tick"`
	Transforms map[ID]Transform `json:"transforms"`
	Velocities map[ID]Vec3      `json:"velocities"`
	Hash       string           `json:"hash"`
}

type SimulationRecording struct {
	Schema  string             `json:"schema"`
	Profile ID                 `json:"profile"`
	Ticks   uint64             `json:"ticks"`
	Inputs  []SimulationInput  `json:"inputs,omitempty"`
	Events  []SimulationEvent  `json:"events,omitempty"`
	Initial SimulationSnapshot `json:"initial"`
	Final   SimulationSnapshot `json:"final"`
}

const SimulationRecordingSchema = "gosx3d.studio.simulation-recording/v1"

func validateSimulationProfile(profile SimulationProfile, entities map[ID]Entity) error {
	if profile.ID == "" || strings.TrimSpace(profile.Name) == "" || profile.TickRate < 1 || profile.TickRate > 1000 || !finiteVec(profile.Gravity) || len(profile.Bodies) == 0 {
		return fmt.Errorf("id, name, tickRate 1..1000, finite gravity, and bodies are required")
	}
	if profile.SubSteps < 0 || profile.SubSteps > 8 {
		return fmt.Errorf("subSteps must be 0 (legacy solver) or 1..8")
	}
	if len(profile.Joints) > 0 && profile.SubSteps == 0 {
		return fmt.Errorf("joints require the soft-step solver (subSteps >= 1)")
	}
	seenJoints := map[ID]bool{}
	for _, joint := range profile.Joints {
		if err := validatePhysicsJoint(joint, profile, entities, seenJoints); err != nil {
			return fmt.Errorf("joint %q: %w", joint.ID, err)
		}
	}
	seen := map[ID]bool{}
	for _, id := range profile.Bodies {
		entity, ok := entities[id]
		if !ok || entity.Physics == nil || seen[id] {
			return fmt.Errorf("body %q is missing, duplicate, or has no physics component", id)
		}
		seen[id] = true
	}
	return nil
}

func validatePhysicsBody(body PhysicsBody) error {
	kind := strings.ToLower(strings.TrimSpace(body.Kind))
	if kind != "dynamic" && kind != "static" && kind != "kinematic" {
		return fmt.Errorf("kind must be dynamic, static, or kinematic")
	}
	if !finite(body.Mass) || !finiteVec(body.Velocity) || !finite(body.GravityScale) || !finite(body.Restitution) || body.Restitution < 0 || body.Restitution > 1 {
		return fmt.Errorf("mass, velocity, gravityScale, and restitution must be finite; restitution must be 0..1")
	}
	if !finite(body.Friction) || body.Friction < 0 || body.Friction > 1 {
		return fmt.Errorf("friction must be finite and within 0..1")
	}
	if kind == "dynamic" && body.Mass <= 0 {
		return fmt.Errorf("dynamic body mass must be positive")
	}
	switch strings.ToLower(strings.TrimSpace(body.Collider.Kind)) {
	case "sphere":
		if !finite(body.Collider.Radius) || body.Collider.Radius <= 0 {
			return fmt.Errorf("sphere collider radius must be positive")
		}
	case "plane":
		if vectorLength(body.Collider.Normal) <= 1e-9 || !finiteVec(body.Collider.Normal) || !finite(body.Collider.Offset) {
			return fmt.Errorf("plane collider requires finite non-zero normal and offset")
		}
	case "box":
		if body.Collider.HalfExtents == nil || !finiteVec(*body.Collider.HalfExtents) ||
			body.Collider.HalfExtents.X <= 0 || body.Collider.HalfExtents.Y <= 0 || body.Collider.HalfExtents.Z <= 0 {
			return fmt.Errorf("box collider requires positive finite halfExtents")
		}
	case "capsule":
		if !finite(body.Collider.Radius) || body.Collider.Radius <= 0 || !finite(body.Collider.HalfHeight) || body.Collider.HalfHeight < 0 {
			return fmt.Errorf("capsule collider requires positive radius and non-negative halfHeight")
		}
	default:
		return fmt.Errorf("collider kind must be sphere, plane, box, or capsule")
	}
	return nil
}

// RunSimulation executes a deterministic semi-implicit Euler reference world.
// It is intentionally browser/GPU independent and records exact input/events.
func RunSimulation(document Document, profileID ID, ticks uint64, inputs []SimulationInput) (SimulationRecording, Document, error) {
	profile, ok := document.Simulations[profileID]
	if !ok {
		return SimulationRecording{}, Document{}, fmt.Errorf("simulation %q does not exist", profileID)
	}
	if ticks == 0 || ticks > 1_000_000 {
		return SimulationRecording{}, Document{}, fmt.Errorf("simulation ticks must be 1..1000000")
	}
	result, err := document.Clone()
	if err != nil {
		return SimulationRecording{}, Document{}, err
	}
	orderedInputs := append([]SimulationInput(nil), inputs...)
	sort.SliceStable(orderedInputs, func(i, j int) bool {
		if orderedInputs[i].Tick != orderedInputs[j].Tick {
			return orderedInputs[i].Tick < orderedInputs[j].Tick
		}
		return orderedInputs[i].Entity < orderedInputs[j].Entity
	})
	for _, input := range orderedInputs {
		entity, ok := result.Entities[input.Entity]
		if !ok || entity.Physics == nil || entity.Physics.Kind != "dynamic" || input.Tick >= ticks || !finiteVec(input.Impulse) {
			return SimulationRecording{}, Document{}, fmt.Errorf("invalid simulation input for entity %q at tick %d", input.Entity, input.Tick)
		}
	}
	initial, err := captureSimulationSnapshot(result, profile, 0)
	if err != nil {
		return SimulationRecording{}, Document{}, err
	}
	recording := SimulationRecording{Schema: SimulationRecordingSchema, Profile: profileID, Ticks: ticks, Inputs: orderedInputs, Initial: initial}
	if profile.SubSteps > 0 {
		runSoftStepSimulation(&result, profile, ticks, orderedInputs, &recording)
		final, err := captureSimulationSnapshot(result, profile, ticks)
		if err != nil {
			return SimulationRecording{}, Document{}, err
		}
		recording.Final = final
		return recording, result, nil
	}
	dt := 1 / float64(profile.TickRate)
	inputIndex := 0
	for tick := uint64(0); tick < ticks; tick++ {
		for inputIndex < len(orderedInputs) && orderedInputs[inputIndex].Tick == tick {
			input := orderedInputs[inputIndex]
			entity := result.Entities[input.Entity]
			entity.Physics.Velocity = addVec(entity.Physics.Velocity, scaleVec(input.Impulse, 1/entity.Physics.Mass))
			result.Entities[input.Entity] = entity
			inputIndex++
		}
		for _, id := range profile.Bodies {
			entity := result.Entities[id]
			if entity.Physics.Kind != "dynamic" {
				continue
			}
			gravityScale := entity.Physics.GravityScale
			entity.Physics.Velocity = addVec(entity.Physics.Velocity, scaleVec(profile.Gravity, gravityScale*dt))
			entity.Transform.Position = addVec(entity.Transform.Position, scaleVec(entity.Physics.Velocity, dt))
			resolveSpherePlanes(&entity, id, result, profile, tick+1, &recording.Events)
			result.Entities[id] = entity
		}
	}
	final, err := captureSimulationSnapshot(result, profile, ticks)
	if err != nil {
		return SimulationRecording{}, Document{}, err
	}
	recording.Final = final
	return recording, result, nil
}

func resolveSpherePlanes(entity *Entity, id ID, document Document, profile SimulationProfile, tick uint64, events *[]SimulationEvent) {
	if entity.Physics.Collider.Kind != "sphere" {
		return
	}
	radius := entity.Physics.Collider.Radius
	for _, otherID := range profile.Bodies {
		if otherID == id {
			continue
		}
		other := document.Entities[otherID]
		if other.Physics == nil || other.Physics.Kind != "static" || other.Physics.Collider.Kind != "plane" {
			continue
		}
		normal := normalizeVec(other.Physics.Collider.Normal)
		planeOffset := other.Physics.Collider.Offset + dotVec(normal, other.Transform.Position)
		distance := dotVec(normal, entity.Transform.Position) - planeOffset
		if distance >= radius {
			continue
		}
		entity.Transform.Position = addVec(entity.Transform.Position, scaleVec(normal, radius-distance))
		normalSpeed := dotVec(entity.Physics.Velocity, normal)
		impactSpeed := math.Abs(normalSpeed)
		if normalSpeed < 0 {
			entity.Physics.Velocity = addVec(entity.Physics.Velocity, scaleVec(normal, -(1+entity.Physics.Restitution)*normalSpeed))
		}
		*events = append(*events, SimulationEvent{Tick: tick, Kind: "contact", Entity: id, Other: otherID, Position: entity.Transform.Position, Speed: impactSpeed})
	}
}

func captureSimulationSnapshot(document Document, profile SimulationProfile, tick uint64) (SimulationSnapshot, error) {
	snapshot := SimulationSnapshot{Tick: tick, Transforms: map[ID]Transform{}, Velocities: map[ID]Vec3{}}
	for _, id := range profile.Bodies {
		entity := document.Entities[id]
		snapshot.Transforms[id] = entity.Transform
		snapshot.Velocities[id] = entity.Physics.Velocity
	}
	payload, err := json.Marshal(struct {
		Tick       uint64
		Transforms map[ID]Transform
		Velocities map[ID]Vec3
	}{tick, snapshot.Transforms, snapshot.Velocities})
	if err != nil {
		return SimulationSnapshot{}, err
	}
	sum := sha256.Sum256(payload)
	snapshot.Hash = hex.EncodeToString(sum[:])
	return snapshot, nil
}

func ReplaySimulation(document Document, recording SimulationRecording) (SimulationRecording, Document, error) {
	if recording.Schema != SimulationRecordingSchema {
		return SimulationRecording{}, Document{}, fmt.Errorf("unsupported simulation recording schema %q", recording.Schema)
	}
	replayed, result, err := RunSimulation(document, recording.Profile, recording.Ticks, recording.Inputs)
	if err != nil {
		return SimulationRecording{}, Document{}, err
	}
	if replayed.Initial.Hash != recording.Initial.Hash || replayed.Final.Hash != recording.Final.Hash {
		return SimulationRecording{}, Document{}, fmt.Errorf("simulation replay diverged: initial %s/%s final %s/%s", replayed.Initial.Hash, recording.Initial.Hash, replayed.Final.Hash, recording.Final.Hash)
	}
	return replayed, result, nil
}

func applySetPhysicsBody(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok {
		return nil, fmt.Errorf("entity %q does not exist", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.Physics == nil {
		return nil, fmt.Errorf("set-physics-body requires physics")
	}
	if err := validatePhysicsBody(*operation.Physics); err != nil {
		return nil, err
	}
	copy := *operation.Physics
	entity.Physics = &copy
	document.Entities[operation.Target] = entity
	return []ID{operation.Target}, nil
}

func applySimulateTicks(document *Document, operation Operation) ([]ID, error) {
	_, result, err := RunSimulation(*document, operation.SimulationID, operation.Ticks, operation.Inputs)
	if err != nil {
		return nil, err
	}
	*document = result
	profile := result.Simulations[operation.SimulationID]
	affected := []ID{operation.SimulationID}
	affected = append(affected, profile.Bodies...)
	return affected, nil
}

func simulationChanges(transaction Transaction, before, after Document) []SimulationChange {
	out := []SimulationChange{}
	for _, operation := range transaction.Operations {
		if operation.Kind != OpSimulateTicks {
			continue
		}
		profile, ok := before.Simulations[operation.SimulationID]
		if !ok {
			continue
		}
		beforeSnapshot, beforeErr := captureSimulationSnapshot(before, profile, 0)
		afterSnapshot, afterErr := captureSimulationSnapshot(after, profile, operation.Ticks)
		if beforeErr != nil || afterErr != nil {
			continue
		}
		out = append(out, SimulationChange{Profile: operation.SimulationID, Ticks: operation.Ticks, BeforeHash: beforeSnapshot.Hash, AfterHash: afterSnapshot.Hash})
	}
	return out
}

func validatePhysicsJoint(joint PhysicsJoint, profile SimulationProfile, entities map[ID]Entity, seen map[ID]bool) error {
	if joint.ID == "" || seen[joint.ID] {
		return fmt.Errorf("id must be non-empty and unique")
	}
	seen[joint.ID] = true
	if strings.ToLower(strings.TrimSpace(joint.Kind)) != "distance" {
		return fmt.Errorf("kind %q is not supported; the current floor is \"distance\" (revolute/prismatic are gated on body rotation)", joint.Kind)
	}
	if joint.BodyA == joint.BodyB {
		return fmt.Errorf("bodyA and bodyB must differ")
	}
	inProfile := map[ID]bool{}
	for _, id := range profile.Bodies {
		inProfile[id] = true
	}
	for _, id := range []ID{joint.BodyA, joint.BodyB} {
		entity, ok := entities[id]
		if !ok || entity.Physics == nil || !inProfile[id] {
			return fmt.Errorf("body %q must be a profile body with a physics component", id)
		}
	}
	values := []float64{joint.Length, joint.MinLength, joint.MaxLength, joint.Stiffness, joint.Damping, joint.MotorTarget, joint.MotorSpeed, joint.MotorMaxForce}
	for _, value := range values {
		if !finite(value) || value < 0 {
			return fmt.Errorf("joint parameters must be finite and non-negative")
		}
	}
	if joint.MaxLength > 0 && joint.MinLength > joint.MaxLength {
		return fmt.Errorf("minLength must not exceed maxLength")
	}
	if !finiteVec(joint.AnchorA) || !finiteVec(joint.AnchorB) {
		return fmt.Errorf("anchors must be finite")
	}
	return nil
}

// applySetSimulationJoint adds/replaces (Joint set) or removes (JointID set)
// a distance joint on a simulation profile. Post-apply document validation
// enforces the full joint contract.
func applySetSimulationJoint(document *Document, operation Operation) ([]ID, error) {
	profile, ok := document.Simulations[operation.SimulationID]
	if !ok {
		return nil, fmt.Errorf("simulation %q does not exist", operation.SimulationID)
	}
	switch {
	case operation.Joint != nil:
		replaced := false
		for index := range profile.Joints {
			if profile.Joints[index].ID == operation.Joint.ID {
				profile.Joints[index] = *operation.Joint
				replaced = true
			}
		}
		if !replaced {
			profile.Joints = append(profile.Joints, *operation.Joint)
		}
	case operation.JointID != "":
		kept := profile.Joints[:0]
		found := false
		for _, joint := range profile.Joints {
			if joint.ID == operation.JointID {
				found = true
				continue
			}
			kept = append(kept, joint)
		}
		if !found {
			return nil, fmt.Errorf("joint %q does not exist on %q", operation.JointID, profile.ID)
		}
		profile.Joints = kept
	default:
		return nil, fmt.Errorf("set-simulation-joint requires joint (set) or jointId (remove)")
	}
	document.Simulations[profile.ID] = profile
	return []ID{profile.ID}, nil
}
