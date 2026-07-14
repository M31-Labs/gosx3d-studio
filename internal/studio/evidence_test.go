package studio

import (
	"bytes"
	"testing"
)

func TestM0CertificationIsDeterministicAndHonest(t *testing.T) {
	first, err := CertifyM0(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	second, err := CertifyM0(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	left, _ := first.MarshalDeterministic()
	right, _ := second.MarshalDeterministic()
	if !bytes.Equal(left, right) {
		t.Fatal("M0 certification is not deterministic")
	}
	if !first.Valid || first.ReleaseStatus != "partial" || len(first.Checks) < 6 {
		t.Fatalf("certification validity/status/checks = %t/%q/%d", first.Valid, first.ReleaseStatus, len(first.Checks))
	}
	for _, check := range first.Checks {
		if check.Status != "pass" {
			t.Fatalf("check failed: %+v", check)
		}
	}
}

func TestCurrentCertificationAddsDeterministicM1M2FoundationsWithoutClaimingRelease(t *testing.T) {
	first, err := CertifyCurrent(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	second, err := CertifyCurrent(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	left, _ := first.MarshalDeterministic()
	right, _ := second.MarshalDeterministic()
	if !bytes.Equal(left, right) {
		t.Fatal("current certification is not byte deterministic")
	}
	if !first.Valid || first.Slice != "M0+M1+M2-foundation" || first.ReleaseStatus != "partial" {
		t.Fatalf("current certification identity = valid %t slice %q release %q", first.Valid, first.Slice, first.ReleaseStatus)
	}
	checks := map[string]string{}
	for _, check := range first.Checks {
		checks[check.ID] = check.Status
	}
	for _, id := range []string{"m1-subobject-selection", "m1-topology-actions", "m1-geometry-analysis", "m1-uv-authoring", "m1-structural-operators", "m1-loop-cut", "m1-nurbs-curve", "m1-modifier-stack", "m1-voxel-csg", "m1-material-authoring", "m1-linked-prefab", "m2-rig-animation-foundation", "m2-fixed-step-simulation", "m2-retarget-state-machine"} {
		if checks[id] != "pass" {
			t.Fatalf("current certification check %q = %q", id, checks[id])
		}
	}
}
