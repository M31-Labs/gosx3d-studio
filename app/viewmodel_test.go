package app

import (
	"strings"
	"testing"

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

func TestTimelineViewReflectsCanonicalRigClipAndSimulation(t *testing.T) {
	view := timelineView(studio.ArticulatedProofDocument())
	if view["armatureId"] != "arm" || view["clipId"] != "reach" || view["trackId"] != "lower-track" || view["simulationId"] != "articulated-physics" || view["tickRate"] != "60" {
		t.Fatalf("timeline view = %#v", view)
	}
}
