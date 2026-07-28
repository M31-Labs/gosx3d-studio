package app

import (
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func TestHierarchyAndInspectorReflectCanonicalDocument(t *testing.T) {
	document := studio.SampleDocument()
	selected, _ := studio.FirstPickTarget(document)
	hierarchy := hierarchyView(document, selected)
	if len(hierarchy) != len(document.Entities) {
		t.Fatalf("hierarchy = %d, entities = %d", len(hierarchy), len(document.Entities))
	}
	found := false
	for _, item := range hierarchy {
		if item["id"] == string(selected) {
			found = true
			if !strings.Contains(item["class"].(string), "selected") {
				t.Fatalf("selected class = %q", item["class"])
			}
		}
	}
	if !found {
		t.Fatalf("selected entity %q missing", selected)
	}
	inspector := inspectorView(document, selected)
	if inspector["id"] != string(selected) || inspector["material"] != "Coral Pieces" {
		t.Fatalf("inspector = %#v", inspector)
	}
}

// The evidence suite renders frames and builds workspaces. Measured on the
// sample document it costs about 2.3 seconds, and every edit changes the
// revision, so running it inline stalled the editor once per edit. The render
// path must return without waiting for it, and the card must say which
// document the evidence it shows describes.
func TestCertificationViewDoesNotBlockTheRenderPath(t *testing.T) {
	document := studio.SampleDocument()

	liveCertCache.Lock()
	liveCertCache.view, liveCertCache.running, liveCertCache.fingerprint, liveCertCache.revision = nil, "", "", 0
	liveCertCache.Unlock()

	start := time.Now()
	view := liveCertificationView(document)
	elapsed := time.Since(start)

	// The suite itself takes seconds. A render that waited for it would not
	// come back inside this budget.
	if elapsed > 500*time.Millisecond {
		t.Fatalf("render path took %v; the evidence suite is still running inline", elapsed)
	}
	if view["certState"] != "pending" {
		t.Fatalf("first render certState = %v, want pending", view["certState"])
	}

	deadline := time.Now().Add(60 * time.Second)
	for {
		view = liveCertificationView(document)
		if view["certState"] == "current" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background evidence never became current; last state = %v", view["certState"])
		}
		time.Sleep(20 * time.Millisecond)
	}
	if view["certRevision"] == "" || view["liveChecksTotal"] == "0" {
		t.Fatalf("current evidence view = %#v", view)
	}
}

func TestTimelineViewReflectsCanonicalRigClipAndSimulation(t *testing.T) {
	view := timelineView(studio.ArticulatedProofDocument())
	if view["armatureId"] != "arm" || view["clipId"] != "idle" || view["trackId"] != "idle-lower" || view["simulationId"] != "articulated-physics" || view["tickRate"] != "60" || view["retargetMapId"] != "arm-to-tall" || view["machineId"] != "locomotion" || view["machineParameter"] != "speed" {
		t.Fatalf("timeline view = %#v", view)
	}
}
