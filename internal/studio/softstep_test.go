package studio

import (
	"math"
	"testing"
)

func softStepDocument(t *testing.T, collider Collider, dropY float64, friction float64, initialVelocity Vec3) Document {
	t.Helper()
	document := SampleDocument()
	root := document.Entities["scene-root"]
	ground := Entity{ID: "ss-ground", Name: "Ground", Parent: root.ID, Transform: IdentityTransform(), Visible: true,
		Mesh:    &MeshComponent{Geometry: Geometry{Kind: "plane", Width: 20, Height: 20}, Material: "board-material", Pickable: true},
		Physics: &PhysicsBody{Kind: "static", Collider: Collider{Kind: "plane", Normal: Vec3{Y: 1}}}}
	falling := Entity{ID: "ss-body", Name: "Falling", Parent: root.ID, Transform: TransformFromEuler(Vec3{Y: dropY}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Mesh:    &MeshComponent{Geometry: Geometry{Kind: "box", Width: 1, Height: 1, Depth: 1}, Material: "player-1-material", Pickable: true},
		Physics: &PhysicsBody{Kind: "dynamic", Mass: 1, GravityScale: 1, Restitution: 0, Friction: friction, Velocity: initialVelocity, Collider: collider}}
	root.Children = append(root.Children, ground.ID, falling.ID)
	document.Entities[root.ID] = root
	document.Entities[ground.ID] = ground
	document.Entities[falling.ID] = falling
	document.Simulations = map[ID]SimulationProfile{"soft": {ID: "soft", Name: "Soft step", TickRate: 60, SubSteps: 4, Gravity: Vec3{Y: -10}, Bodies: []ID{"ss-ground", "ss-body"}}}
	return document
}

func TestSoftStepBoxSettlesOnPlaneDeterministically(t *testing.T) {
	document := softStepDocument(t, Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5, Y: 0.5, Z: 0.5}}, 3, 0.4, Vec3{})
	first, result, err := RunSimulation(document, "soft", 300, nil)
	if err != nil {
		t.Fatal(err)
	}
	restY := result.Entities["ss-body"].Transform.Position.Y
	if math.Abs(restY-0.5) > 0.02 {
		t.Fatalf("box rest height = %v, want ~0.5 (half extent)", restY)
	}
	second, _, err := RunSimulation(document, "soft", 300, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Final.Hash != second.Final.Hash {
		t.Fatal("soft-step solver is not deterministic")
	}
}

func TestSoftStepCapsuleRestHeight(t *testing.T) {
	document := softStepDocument(t, Collider{Kind: "capsule", Radius: 0.25, HalfHeight: 0.5}, 3, 0.4, Vec3{})
	_, result, err := RunSimulation(document, "soft", 300, nil)
	if err != nil {
		t.Fatal(err)
	}
	restY := result.Entities["ss-body"].Transform.Position.Y
	if math.Abs(restY-0.75) > 0.02 {
		t.Fatalf("capsule rest height = %v, want ~0.75 (radius+halfHeight)", restY)
	}
}

func TestSoftStepFrictionDeceleratesSliding(t *testing.T) {
	slippery := softStepDocument(t, Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5, Y: 0.5, Z: 0.5}}, 0.5, 0, Vec3{X: 4})
	_, slickResult, err := RunSimulation(slippery, "soft", 120, nil)
	if err != nil {
		t.Fatal(err)
	}
	grippy := softStepDocument(t, Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5, Y: 0.5, Z: 0.5}}, 0.5, 0.8, Vec3{X: 4})
	_, gripResult, err := RunSimulation(grippy, "soft", 120, nil)
	if err != nil {
		t.Fatal(err)
	}
	slickX := slickResult.Entities["ss-body"].Physics.Velocity.X
	gripX := gripResult.Entities["ss-body"].Physics.Velocity.X
	if gripX >= slickX-0.5 {
		t.Fatalf("friction must decelerate sliding: frictionless vx=%v grippy vx=%v", slickX, gripX)
	}
}

func TestSoftStepSensorEmitsEventsWithoutResponse(t *testing.T) {
	document := softStepDocument(t, Collider{Kind: "sphere", Radius: 0.3}, 4, 0, Vec3{})
	root := document.Entities["scene-root"]
	sensor := Entity{ID: "ss-sensor", Name: "Trigger", Parent: root.ID, Transform: TransformFromEuler(Vec3{Y: 2}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Physics: &PhysicsBody{Kind: "static", Collider: Collider{Kind: "sphere", Radius: 0.6, Sensor: true}}}
	root.Children = append(root.Children, sensor.ID)
	document.Entities[root.ID] = root
	document.Entities[sensor.ID] = sensor
	profile := document.Simulations["soft"]
	profile.Bodies = append(profile.Bodies, sensor.ID)
	document.Simulations["soft"] = profile
	recording, result, err := RunSimulation(document, "soft", 240, nil)
	if err != nil {
		t.Fatal(err)
	}
	begin, end := false, false
	for _, event := range recording.Events {
		if event.Kind == "sensor-begin" && event.Other == "ss-sensor" {
			begin = true
		}
		if event.Kind == "sensor-end" && event.Other == "ss-sensor" {
			end = true
		}
	}
	if !begin || !end {
		t.Fatalf("sensor begin/end events missing: begin=%t end=%t", begin, end)
	}
	if math.Abs(result.Entities["ss-body"].Transform.Position.Y-0.3) > 0.02 {
		t.Fatalf("sensor must not block the fall; rest=%v", result.Entities["ss-body"].Transform.Position.Y)
	}
}

func TestLegacyProfilePathUnchangedWithoutSubSteps(t *testing.T) {
	document := ArticulatedProofDocument()
	first, _, err := RunSimulation(document, "articulated-physics", 60, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := RunSimulation(document, "articulated-physics", 60, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Final.Hash != second.Final.Hash {
		t.Fatal("legacy path lost determinism")
	}
}
