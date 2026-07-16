package studio

import (
	"math"
	"testing"
)

func exactSurfacePoint(t *testing.T, document Document) (ID, Vec3) {
	t.Helper()
	_, origin := FirstPickTarget(document)
	result, err := ExactPick(document, PickRequest{Origin: origin, Direction: Vec3{Y: -1}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected == "" || result.Trace.Closest == nil {
		t.Fatal("sample document must produce an exact pick")
	}
	hit := result.Trace.Closest.Point
	return result.Selected, Vec3{X: hit.X, Y: hit.Y, Z: hit.Z}
}

func otherPickableEntity(t *testing.T, document Document, exclude ID) ID {
	t.Helper()
	for id, entity := range document.Entities {
		if id != exclude && entity.Visible && entity.Mesh != nil && entity.Mesh.Pickable {
			return id
		}
	}
	t.Fatal("no second pickable entity in document")
	return ""
}

func TestConfirmViewportSelectionAgreesOnTrueSurfacePoint(t *testing.T) {
	document := SampleDocument()
	target, world := exactSurfacePoint(t, document)
	confirmation, err := ConfirmViewportSelection(document, ViewportPick{Selected: target, Kind: "mesh", World: &world})
	if err != nil {
		t.Fatal(err)
	}
	if !confirmation.Confirmed || confirmation.Method != "exact-cpu" {
		t.Fatalf("true surface point must confirm, got %+v", confirmation)
	}
	if confirmation.Selected != target || confirmation.Disagreement != nil {
		t.Fatalf("agreement must keep the selection and carry no disagreement, got %+v", confirmation)
	}
	if confirmation.Revision != document.Revision {
		t.Fatalf("confirmation revision = %d, want %d", confirmation.Revision, document.Revision)
	}
}

func TestConfirmViewportSelectionSurfacesIDDisagreementAndCanonicalWins(t *testing.T) {
	document := SampleDocument()
	cpuTarget, world := exactSurfacePoint(t, document)
	claimed := otherPickableEntity(t, document, cpuTarget)
	confirmation, err := ConfirmViewportSelection(document, ViewportPick{Selected: claimed, Kind: "mesh", World: &world})
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Selected != cpuTarget {
		t.Fatalf("canonical CPU hit must win the selection, got %q want %q", confirmation.Selected, cpuTarget)
	}
	if confirmation.Disagreement == nil || confirmation.Disagreement.Reason != "id-mismatch" {
		t.Fatalf("id mismatch must surface a disagreement, got %+v", confirmation.Disagreement)
	}
	if confirmation.Disagreement.GPUSelected != claimed || confirmation.Disagreement.CPUSelected != cpuTarget {
		t.Fatalf("disagreement must carry both ids, got %+v", confirmation.Disagreement)
	}
}

func TestConfirmViewportSelectionSurfacesEmptySpaceMiss(t *testing.T) {
	document := SampleDocument()
	target, _ := exactSurfacePoint(t, document)
	world := Vec3{X: 50, Y: 50, Z: 50}
	confirmation, err := ConfirmViewportSelection(document, ViewportPick{Selected: target, Kind: "mesh", World: &world})
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Confirmed {
		t.Fatalf("empty-space report must not confirm, got %+v", confirmation)
	}
	if confirmation.Disagreement == nil || confirmation.Disagreement.Reason != "no-cpu-hit" {
		t.Fatalf("empty-space report must surface no-cpu-hit, got %+v", confirmation.Disagreement)
	}
}

func TestConfirmViewportSelectionDegradesHonestlyWithoutWorldPoint(t *testing.T) {
	document := SampleDocument()
	target, _ := exactSurfacePoint(t, document)
	confirmation, err := ConfirmViewportSelection(document, ViewportPick{Selected: target, Kind: "mesh"})
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Confirmed || confirmation.Method != "id-only" || confirmation.Disagreement != nil {
		t.Fatalf("missing world point must degrade to id-only, got %+v", confirmation)
	}
	if confirmation.Selected != target {
		t.Fatalf("id-only path must keep validated selection, got %+v", confirmation)
	}
}

func TestConfirmViewportSelectionStillRejectsInvalidIDs(t *testing.T) {
	document := SampleDocument()
	world := Vec3{}
	for _, id := range []ID{"missing", "board-pedestal"} {
		if _, err := ConfirmViewportSelection(document, ViewportPick{Selected: id, World: &world}); err == nil {
			t.Fatalf("selection %q was accepted", id)
		}
	}
}

func TestConfirmViewportSelectionPositionGapKeepsAgreementVisible(t *testing.T) {
	document := SampleDocument()
	target, world := exactSurfacePoint(t, document)
	// Same entity, but the reported point is nudged along the surface more
	// than the agreement tolerance: the ID still matches, so selection is
	// confirmed, but the numeric gap must stay visible.
	nudged := Vec3{X: world.X + 0.02, Y: world.Y, Z: world.Z}
	confirmation, err := ConfirmViewportSelection(document, ViewportPick{Selected: target, Kind: "mesh", World: &nudged})
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Selected != target {
		t.Fatalf("selection must stay on the agreed entity, got %+v", confirmation)
	}
	if confirmation.Disagreement != nil {
		if confirmation.Disagreement.Reason != "position-gap" && confirmation.Disagreement.Reason != "no-cpu-hit" {
			t.Fatalf("unexpected disagreement reason %q", confirmation.Disagreement.Reason)
		}
		if math.IsNaN(confirmation.Disagreement.DistanceGap) {
			t.Fatal("distance gap must be a number")
		}
	}
}
