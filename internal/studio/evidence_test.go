package studio

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// Certification checks compare fingerprints to prove a proposal and a direct
// commit produce the same document. Capture used to discard the error, so an
// unhashable document produced "" on both sides, "" equalled "", and the check
// passed because it had failed. A failed capture must equal nothing.
func TestEvidenceFingerprintTreatsAFailedCaptureAsUnequal(t *testing.T) {
	good := fingerprintOf(SampleDocument())
	if !good.Equal(fingerprintOf(SampleDocument())) {
		t.Fatal("two captures of the same document are not equal")
	}

	// NaN in the camera is rejected by Validate now, but the evidence path
	// must stay safe for any document that reaches it.
	broken := SampleDocument()
	broken.Camera.Position.X = math.NaN()
	first := fingerprintOf(broken)
	second := fingerprintOf(broken)
	if first.err == nil {
		t.Fatal("expected an unhashable document; the marshaller no longer rejects NaN")
	}
	if first.Equal(second) {
		t.Fatal("two failed captures compared equal, so a broken document would read as a match")
	}
	if first.Equal(good) || good.Equal(first) {
		t.Fatal("a failed capture compared equal to a real fingerprint")
	}

	// Evidence strings must name the failure rather than print an empty value
	// that reads like a real fingerprint.
	if rendered := first.String(); !strings.HasPrefix(rendered, "unhashable: ") {
		t.Fatalf("failed capture rendered as %q", rendered)
	}
	if rendered := good.String(); len(rendered) != 64 {
		t.Fatalf("captured fingerprint rendered as %q, want 64 hex characters", rendered)
	}
}

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
