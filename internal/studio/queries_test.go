package studio

import "testing"

func TestViewportSelectionAgreesWithExactHarnessPick(t *testing.T) {
	document := SampleDocument()
	target, origin := FirstPickTarget(document)
	result, err := ExactPick(document, PickRequest{Origin: origin, Direction: Vec3{Y: -1}, Select: true})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := ValidateViewportSelection(document, result.Selected, "sphere")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Selected != target || selection.Selected != result.Selected {
		t.Fatalf("viewport=%q harness=%q target=%q", selection.Selected, result.Selected, target)
	}
	if selection.Source != "scene3d-mount-input" {
		t.Fatalf("source=%q", selection.Source)
	}
}

func TestViewportSelectionRejectsStaleAndNonPickableIDs(t *testing.T) {
	document := SampleDocument()
	for _, id := range []ID{"missing", "board-pedestal"} {
		if _, err := ValidateViewportSelection(document, id, "mesh"); err == nil {
			t.Fatalf("selection %q was accepted", id)
		}
	}
}
