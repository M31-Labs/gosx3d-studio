package studio

import "testing"

func TestBlackglassCoastDocumentIsAnExportableRuntimeHandoff(t *testing.T) {
	document := BlackglassCoastDocument()
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	if document.World == nil || len(document.World.WaterZones) != 1 || len(document.World.Markers) != 5 {
		t.Fatalf("world contract = %#v", document.World)
	}
	if len(document.Entities) < 14 {
		t.Fatalf("expected a meaningful cove blockout, got %d entities", len(document.Entities))
	}
	if _, err := CompileIR(document); err != nil {
		t.Fatalf("compile coast SceneIR: %v", err)
	}
	if _, report, err := ExportEmbeddedGo(document, "blackglasscoast"); err != nil || len(report.Losses) != 0 {
		t.Fatalf("Go export = %+v, %v", report, err)
	}
	_, report, err := ExportSceneIR(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, loss := range report.Losses {
		if loss.Domain == "world" && loss.Count == 6 {
			return
		}
	}
	t.Fatalf("world handoff was not disclosed: %+v", report.Losses)
}
func TestBlackglassCoastDocumentProducesBrowserFreeEvidence(t *testing.T) {
	report, err := CertifyM0(BlackglassCoastDocument())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid || len(report.Harness.Events) == 0 || report.Harness.Events[0].Frame == nil {
		t.Fatalf("Blackglass Coast evidence = %+v", report)
	}
}
