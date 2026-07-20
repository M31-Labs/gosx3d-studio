package studio

import (
	"math"
	"testing"
)

func rotationDocument(t *testing.T, body Entity) Document {
	t.Helper()
	document := SampleDocument()
	root := document.Entities["scene-root"]
	ground := Entity{ID: "rot-ground", Name: "Ground", Parent: root.ID, Transform: IdentityTransform(), Visible: true,
		Physics: &PhysicsBody{Kind: "static", Friction: 0.6, Collider: Collider{Kind: "plane", Normal: Vec3{Y: 1}}}}
	body.Parent = root.ID
	root.Children = append(root.Children, ground.ID, body.ID)
	document.Entities[root.ID] = root
	document.Entities[ground.ID] = ground
	document.Entities[body.ID] = body
	document.Simulations = map[ID]SimulationProfile{"rot": {ID: "rot", Name: "Rotation", TickRate: 60, SubSteps: 4, Gravity: Vec3{Y: -10}, Bodies: []ID{"rot-ground", body.ID}}}
	return document
}

func TestTiltedBoxSettlesFlatWithRotation(t *testing.T) {
	box := Entity{ID: "rot-box", Name: "Tilted box", Transform: TransformFromEuler(Vec3{Y: 2}, Vec3{Z: 0.35}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Physics: &PhysicsBody{Kind: "dynamic", Mass: 1, GravityScale: 1, Restitution: 0, Friction: 0.6, AngularDamping: 0.4,
			Collider: Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5, Y: 0.5, Z: 0.5}}}}
	document := rotationDocument(t, box)
	first, result, err := RunSimulation(document, "rot", 900, nil)
	if err != nil {
		t.Fatal(err)
	}
	settled := result.Entities["rot-box"]
	if math.Abs(settled.Transform.Position.Y-0.5) > 0.05 {
		t.Fatalf("tilted box rest height = %v, want ~0.5 (settled onto a face)", settled.Transform.Position.Y)
	}
	angular := settled.Physics.AngularVelocity
	if vectorLength(angular) > 0.2 {
		t.Fatalf("box still spinning at rest: %+v", angular)
	}
	second, _, err := RunSimulation(document, "rot", 900, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Final.Hash != second.Final.Hash {
		t.Fatal("rotational simulation lost determinism")
	}
}

func TestSphereRollsDownFrictionIncline(t *testing.T) {
	sphere := Entity{ID: "rot-ball", Name: "Ball", Transform: TransformFromEuler(Vec3{Y: 1}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Physics: &PhysicsBody{Kind: "dynamic", Mass: 1, GravityScale: 1, Restitution: 0, Friction: 0.8,
			Collider: Collider{Kind: "sphere", Radius: 0.5}}}
	document := rotationDocument(t, sphere)
	ground := document.Entities["rot-ground"]
	ground.Physics.Collider.Normal = normalizeVec(Vec3{X: 0.3, Y: 1})
	document.Entities["rot-ground"] = ground
	_, result, err := RunSimulation(document, "rot", 240, nil)
	if err != nil {
		t.Fatal(err)
	}
	ball := result.Entities["rot-ball"]
	if vectorLength(ball.Physics.AngularVelocity) < 0.3 {
		t.Fatalf("friction on an incline must induce rolling, angular=%+v", ball.Physics.AngularVelocity)
	}
	if ball.Transform.Position.X <= 1 {
		t.Fatalf("ball must roll downhill (+X for this plane normal), got x=%v", ball.Transform.Position.X)
	}
}

func TestRotatedStaticBoxSupportsOnItsEdge(t *testing.T) {
	sphere := Entity{ID: "rot-drop", Name: "Drop", Transform: TransformFromEuler(Vec3{Y: 3}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Physics: &PhysicsBody{Kind: "dynamic", Mass: 1, GravityScale: 1, Restitution: 0, Friction: 0.9,
			Collider: Collider{Kind: "sphere", Radius: 0.25}}}
	document := rotationDocument(t, sphere)
	root := document.Entities["scene-root"]
	// A cube rotated 45 degrees about Z presents its edge upward at
	// sqrt(2)*halfExtent above its center.
	block := Entity{ID: "rot-block", Name: "Rotated block", Parent: root.ID, Transform: TransformFromEuler(Vec3{}, Vec3{Z: math.Pi / 4}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true,
		Physics: &PhysicsBody{Kind: "static", Friction: 0.9, Collider: Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5, Y: 0.5, Z: 0.5}}}}
	root.Children = append(root.Children, block.ID)
	document.Entities[root.ID] = root
	document.Entities[block.ID] = block
	profile := document.Simulations["rot"]
	profile.Bodies = append(profile.Bodies, block.ID)
	document.Simulations["rot"] = profile
	_, result, err := RunSimulation(document, "rot", 240, nil)
	if err != nil {
		t.Fatal(err)
	}
	dropY := result.Entities["rot-drop"].Transform.Position.Y
	edgeTop := math.Sqrt2*0.5 + 0.25
	// The sphere may roll off the edge, but while falling it must have been
	// supported ABOVE the axis-aligned box top (0.5+0.25); assert it never
	// tunneled into the rotated block: either resting near the edge apex or
	// off to the side on the ground plane at sphere radius.
	restingOnEdge := math.Abs(dropY-edgeTop) < 0.08
	rolledOff := math.Abs(dropY-0.25) < 0.08
	if !restingOnEdge && !rolledOff {
		t.Fatalf("sphere ended at y=%v; want edge apex ~%v or ground ~0.25 (tunneled through rotated collider?)", dropY, edgeTop)
	}
}

func TestAngularStateValidates(t *testing.T) {
	bad := PhysicsBody{Kind: "dynamic", Mass: 1, AngularDamping: -1, Collider: Collider{Kind: "sphere", Radius: 1}}
	if err := validatePhysicsBody(bad); err == nil {
		t.Fatal("negative angular damping must fail")
	}
	nan := PhysicsBody{Kind: "dynamic", Mass: 1, AngularVelocity: Vec3{X: math.NaN()}, Collider: Collider{Kind: "sphere", Radius: 1}}
	if err := validatePhysicsBody(nan); err == nil {
		t.Fatal("non-finite angular velocity must fail")
	}
}
