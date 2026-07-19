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

func TestExactPickCacheTracksDocumentChanges(t *testing.T) {
	document := SampleDocument()
	target, origin := FirstPickTarget(document)
	request := PickRequest{Origin: origin, Direction: Vec3{Y: -1}}
	first, err := ExactPick(document, request)
	if err != nil || first.Selected != target {
		t.Fatalf("first pick: %v %v", first.Selected, err)
	}
	second, err := ExactPick(document, request)
	if err != nil || second.Selected != first.Selected {
		t.Fatal("cached pick diverged")
	}
	// Mutate the document (delete the picked entity): the cache must miss.
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := workspace.Execute(Transaction{ID: "del", Actor: "t", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpDeleteEntity, Target: target}}}); err != nil {
		t.Fatal(err)
	}
	changed, _ := workspace.Snapshot()
	third, err := ExactPick(changed, request)
	if err != nil {
		t.Fatal(err)
	}
	if third.Selected == target {
		t.Fatal("stale cached graph served after mutation")
	}
}
