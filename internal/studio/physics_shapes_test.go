package studio

import (
	"strings"
	"testing"
)

func TestColliderShapesValidate(t *testing.T) {
	cases := []struct {
		name string
		body PhysicsBody
		ok   bool
	}{
		{"box ok", PhysicsBody{Kind: "dynamic", Mass: 1, Collider: Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5, Y: 0.5, Z: 0.5}}}, true},
		{"box needs extents", PhysicsBody{Kind: "dynamic", Mass: 1, Collider: Collider{Kind: "box"}}, false},
		{"box rejects zero extent", PhysicsBody{Kind: "dynamic", Mass: 1, Collider: Collider{Kind: "box", HalfExtents: &Vec3{X: 0.5}}}, false},
		{"capsule ok", PhysicsBody{Kind: "dynamic", Mass: 1, Collider: Collider{Kind: "capsule", Radius: 0.25, HalfHeight: 0.5}}, true},
		{"capsule needs radius", PhysicsBody{Kind: "dynamic", Mass: 1, Collider: Collider{Kind: "capsule", HalfHeight: 0.5}}, false},
		{"friction range", PhysicsBody{Kind: "dynamic", Mass: 1, Friction: 1.5, Collider: Collider{Kind: "sphere", Radius: 1}}, false},
		{"sensor sphere ok", PhysicsBody{Kind: "static", Collider: Collider{Kind: "sphere", Radius: 1, Sensor: true}}, true},
	}
	for _, test := range cases {
		err := validatePhysicsBody(test.body)
		if test.ok && err != nil {
			t.Fatalf("%s: unexpected error %v", test.name, err)
		}
		if !test.ok && err == nil {
			t.Fatalf("%s: expected validation failure", test.name)
		}
	}
}

func TestSimulationProfileSubStepsValidate(t *testing.T) {
	document := ArticulatedProofDocument()
	profile := document.Simulations["articulated-physics"]
	profile.SubSteps = 4
	if err := validateSimulationProfile(profile, document.Entities); err != nil {
		t.Fatalf("substeps 4 must validate: %v", err)
	}
	profile.SubSteps = 99
	if err := validateSimulationProfile(profile, document.Entities); err == nil || !strings.Contains(err.Error(), "subSteps") {
		t.Fatalf("substeps out of range must fail with a subSteps diagnostic, got %v", err)
	}
}
