package studio

import (
	"strings"
	"testing"
)

func TestWorldContractValidatesRuntimeHandoff(t *testing.T) {
	document := SampleDocument()
	document.World = &WorldContract{
		WaterZones: map[ID]WaterZone{
			"blackglass-cove": {ID: "blackglass-cove", Name: "Blackglass Cove", Center: Vec3{Y: -2}, Size: Vec3{X: 28, Y: 8, Z: 20}, SurfaceY: 0, Current: Vec3{X: 0.18, Z: -0.06}, BuoyancyScale: 1.15, LinearDrag: 0.35, RuntimeProfile: "blackglass-coast"},
		},
		Markers: map[ID]WorldMarker{
			"arrival": {ID: "arrival", Name: "Arrival", Kind: "player-spawn", Entity: "board", Position: Vec3{X: 2, Y: 1, Z: 6}},
		},
	}
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	if depth := document.World.WaterZones["blackglass-cove"].WaterZoneDepth(Vec3{Y: -1}); depth != 1 {
		t.Fatalf("water depth = %v, want 1", depth)
	}
	if depth := document.World.WaterZones["blackglass-cove"].WaterZoneDepth(Vec3{X: 99, Y: -1}); depth != 0 {
		t.Fatalf("outside water depth = %v, want 0", depth)
	}
}

func TestWorldContractRejectsInvalidAuthoring(t *testing.T) {
	document := SampleDocument()
	document.World = &WorldContract{WaterZones: map[ID]WaterZone{
		"cove": {ID: "cove", Name: "Cove", Size: Vec3{X: 4, Y: 2, Z: 4}, SurfaceY: 3, RuntimeProfile: "coast"},
	}}
	if err := document.Validate(); err == nil || !strings.Contains(err.Error(), "surfaceY") {
		t.Fatalf("expected surface bounds validation, got %v", err)
	}
	document.World = &WorldContract{Markers: map[ID]WorldMarker{
		"spawn": {ID: "spawn", Name: "Spawn", Kind: "player-spawn", Entity: "missing"},
	}}
	if err := document.Validate(); err == nil || !strings.Contains(err.Error(), "missing entity") {
		t.Fatalf("expected marker entity validation, got %v", err)
	}
}

func TestWorldContractIsLosslessInSourceExportsAndNamedInRendererLosses(t *testing.T) {
	document := SampleDocument()
	document.World = &WorldContract{Markers: map[ID]WorldMarker{
		"camera": {ID: "camera", Name: "Opening camera", Kind: "camera-start", Position: Vec3{Y: 4}},
	}}
	if _, report, err := ExportSceneDoc(document); err != nil || len(report.Losses) != 0 {
		t.Fatalf("SceneDoc export = %+v, %v", report, err)
	}
	if _, report, err := ExportEmbeddedGo(document, "coast"); err != nil || len(report.Losses) != 0 {
		t.Fatalf("Go export = %+v, %v", report, err)
	}
	_, report, err := ExportSceneIR(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, loss := range report.Losses {
		if loss.Domain == "world" && loss.Count == 1 {
			return
		}
	}
	t.Fatalf("renderer export did not disclose world semantic loss: %+v", report.Losses)
}
