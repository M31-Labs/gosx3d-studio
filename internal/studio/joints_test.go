package studio

import (
	"math"
	"testing"
)

func jointDocument(t *testing.T, joint PhysicsJoint) Document {
	t.Helper()
	document := SampleDocument()
	root := document.Entities["scene-root"]
	anchor := Entity{ID: "j-anchor", Name: "Anchor", Parent: root.ID, Transform: TransformFromEuler(Vec3{Y: 5}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Physics: &PhysicsBody{Kind: "static", Collider: Collider{Kind: "sphere", Radius: 0.1}}}
	bob := Entity{ID: "j-bob", Name: "Bob", Parent: root.ID, Transform: TransformFromEuler(Vec3{X: 0.5, Y: 4.5}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Mesh:    &MeshComponent{Geometry: Geometry{Kind: "sphere", Radius: 0.2, Segments: 8}, Material: "player-1-material", Pickable: true},
		Physics: &PhysicsBody{Kind: "dynamic", Mass: 1, GravityScale: 1, Restitution: 0, Friction: 0.2, Collider: Collider{Kind: "sphere", Radius: 0.2}}}
	root.Children = append(root.Children, anchor.ID, bob.ID)
	document.Entities[root.ID] = root
	document.Entities[anchor.ID] = anchor
	document.Entities[bob.ID] = bob
	document.Simulations = map[ID]SimulationProfile{"jointed": {
		ID: "jointed", Name: "Jointed", TickRate: 60, SubSteps: 4, Gravity: Vec3{Y: -10},
		Bodies: []ID{"j-anchor", "j-bob"},
		Joints: []PhysicsJoint{joint},
	}}
	return document
}

func TestDistanceJointHoldsRestLengthUnderGravity(t *testing.T) {
	document := jointDocument(t, PhysicsJoint{ID: "pendulum", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", Length: 1})
	first, result, err := RunSimulation(document, "jointed", 600, nil)
	if err != nil {
		t.Fatal(err)
	}
	anchor := result.Entities["j-anchor"].Transform.Position
	bob := result.Entities["j-bob"].Transform.Position
	distance := math.Sqrt((bob.X-anchor.X)*(bob.X-anchor.X) + (bob.Y-anchor.Y)*(bob.Y-anchor.Y) + (bob.Z-anchor.Z)*(bob.Z-anchor.Z))
	if math.Abs(distance-1) > 0.05 {
		t.Fatalf("joint length at rest = %v, want ~1", distance)
	}
	if bob.Y > anchor.Y-0.8 {
		t.Fatalf("bob must hang below the anchor: bob=%v anchor=%v", bob.Y, anchor.Y)
	}
	second, _, err := RunSimulation(document, "jointed", 600, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Final.Hash != second.Final.Hash {
		t.Fatal("jointed simulation lost determinism")
	}
}

func TestDistanceJointLimitsClampRange(t *testing.T) {
	document := jointDocument(t, PhysicsJoint{ID: "leash", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", MinLength: 0.5, MaxLength: 1.5})
	_, result, err := RunSimulation(document, "jointed", 600, nil)
	if err != nil {
		t.Fatal(err)
	}
	anchor := result.Entities["j-anchor"].Transform.Position
	bob := result.Entities["j-bob"].Transform.Position
	distance := math.Sqrt((bob.X-anchor.X)*(bob.X-anchor.X) + (bob.Y-anchor.Y)*(bob.Y-anchor.Y) + (bob.Z-anchor.Z)*(bob.Z-anchor.Z))
	if distance > 1.55 || distance < 0.45 {
		t.Fatalf("limited joint distance = %v, want within [0.5, 1.5]", distance)
	}
}

func TestDistanceJointSpringSettlesTowardRest(t *testing.T) {
	document := jointDocument(t, PhysicsJoint{ID: "spring", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", Length: 1, Stiffness: 40, Damping: 8})
	_, result, err := RunSimulation(document, "jointed", 900, nil)
	if err != nil {
		t.Fatal(err)
	}
	anchor := result.Entities["j-anchor"].Transform.Position
	bob := result.Entities["j-bob"].Transform.Position
	distance := math.Sqrt((bob.X-anchor.X)*(bob.X-anchor.X) + (bob.Y-anchor.Y)*(bob.Y-anchor.Y) + (bob.Z-anchor.Z)*(bob.Z-anchor.Z))
	// A soft spring under gravity settles stretched past rest but bounded.
	if distance < 1 || distance > 1.6 {
		t.Fatalf("spring settled at %v, want stretched within (1, 1.6]", distance)
	}
}

func TestDistanceJointMotorDrivesTowardTarget(t *testing.T) {
	document := jointDocument(t, PhysicsJoint{ID: "winch", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", Length: 1.2, MotorTarget: 0.4, MotorSpeed: 0.5, MotorMaxForce: 100})
	_, result, err := RunSimulation(document, "jointed", 600, nil)
	if err != nil {
		t.Fatal(err)
	}
	anchor := result.Entities["j-anchor"].Transform.Position
	bob := result.Entities["j-bob"].Transform.Position
	distance := math.Sqrt((bob.X-anchor.X)*(bob.X-anchor.X) + (bob.Y-anchor.Y)*(bob.Y-anchor.Y) + (bob.Z-anchor.Z)*(bob.Z-anchor.Z))
	if math.Abs(distance-0.4) > 0.08 {
		t.Fatalf("motor should winch to 0.4, got %v", distance)
	}
}

func TestJointValidation(t *testing.T) {
	document := jointDocument(t, PhysicsJoint{ID: "pendulum", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", Length: 1})
	profile := document.Simulations["jointed"]
	bad := []PhysicsJoint{
		{ID: "", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob"},
		{ID: "x", Kind: "revolute", BodyA: "j-anchor", BodyB: "j-bob"},
		{ID: "x", Kind: "distance", BodyA: "j-anchor", BodyB: "missing"},
		{ID: "x", Kind: "distance", BodyA: "j-bob", BodyB: "j-bob"},
		{ID: "x", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", MinLength: 2, MaxLength: 1},
	}
	for index, joint := range bad {
		profile.Joints = []PhysicsJoint{joint}
		if err := validateSimulationProfile(profile, document.Entities); err == nil {
			t.Fatalf("bad joint %d accepted", index)
		}
	}
}

func TestSetSimulationJointOperationIsRevisionSafeAndUndoable(t *testing.T) {
	document := jointDocument(t, PhysicsJoint{ID: "pendulum", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", Length: 1})
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	winch := PhysicsJoint{ID: "winch", Kind: "distance", BodyA: "j-anchor", BodyB: "j-bob", MotorTarget: 0.5, MotorSpeed: 1, MotorMaxForce: 50}
	receipt, changed, err := workspace.Execute(Transaction{ID: "add-joint", Actor: "agent://joints", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetSimulationJoint, SimulationID: "jointed", Joint: &winch}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(changed.Simulations["jointed"].Joints) != 2 || !receipt.Applied {
		t.Fatalf("joint not added: %+v", changed.Simulations["jointed"].Joints)
	}
	invalid := PhysicsJoint{ID: "bad", Kind: "revolute", BodyA: "j-anchor", BodyB: "j-bob"}
	if _, _, err := workspace.Execute(Transaction{ID: "bad-joint", Actor: "agent://joints", Mode: ModeDirect, ExpectedRevision: changed.Revision, Operations: []Operation{{Kind: OpSetSimulationJoint, SimulationID: "jointed", Joint: &invalid}}}); err == nil {
		t.Fatal("invalid joint kind must be rejected by post-apply validation")
	}
	if _, restored, err := workspace.Undo(changed.Revision, "agent://joints"); err != nil {
		t.Fatal(err)
	} else if len(restored.Simulations["jointed"].Joints) != 1 {
		t.Fatal("undo did not remove the joint")
	}
}
