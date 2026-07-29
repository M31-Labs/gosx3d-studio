package studio

import "testing"

func TestBlackglassCoastCanonicalSceneDocExportIsPinned(t *testing.T) {
	payload, report, err := ExportSceneDoc(BlackglassCoastDocument())
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 || report.Kind != "scene3d" || len(report.Losses) != 0 {
		t.Fatalf("canonical export = %+v", report)
	}
	const want = "3f219e90d54b3291bfc5c15f8f7aa411c448898414a26517b1d0fd36a90da800"
	if report.Fingerprint != want {
		t.Fatalf("canonical export fingerprint = %s, want %s; regenerate the GoSX handoff artifact", report.Fingerprint, want)
	}
}
