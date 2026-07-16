package studio

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestExportSceneDocIsLosslessAndDeterministic(t *testing.T) {
	document := ArticulatedProofDocument()
	payload, report, err := ExportSceneDoc(document)
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "scene3d" || report.Schema != SceneDocSchema || len(report.Losses) != 0 {
		t.Fatalf("scene3d export must be lossless: %+v", report)
	}
	var decoded Document
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	source, _ := document.Fingerprint()
	roundTrip, _ := decoded.Fingerprint()
	if source != roundTrip {
		t.Fatalf("export round trip changed fingerprint: %s vs %s", source, roundTrip)
	}
	second, _, err := ExportSceneDoc(document)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, second) {
		t.Fatal("scene3d export is not byte-deterministic")
	}
	if report.Bytes != len(payload) || len(report.Fingerprint) != 64 {
		t.Fatalf("report accounting wrong: %+v", report)
	}
}

func TestExportSceneIRReportsSemanticLosses(t *testing.T) {
	document := ArticulatedProofDocument()
	payload, report, err := ExportSceneIR(document)
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "scene-ir" || len(payload) == 0 {
		t.Fatalf("report=%+v", report)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("scene-ir payload is not JSON: %v", err)
	}
	domains := map[string]int{}
	for _, loss := range report.Losses {
		if loss.Reason == "" || loss.Count <= 0 {
			t.Fatalf("loss entries must carry reason and count: %+v", loss)
		}
		domains[loss.Domain] = loss.Count
	}
	for _, expected := range []string{"rigs", "animations", "simulations"} {
		if domains[expected] == 0 {
			t.Fatalf("renderer IR export must report %q loss, got %+v", expected, report.Losses)
		}
	}
	second, _, err := ExportSceneIR(document)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, second) {
		t.Fatal("scene-ir export is not byte-deterministic")
	}
}
