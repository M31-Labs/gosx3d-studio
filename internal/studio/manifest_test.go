package studio

import "testing"

func TestDefaultManifestIsHonestAndDispatchable(t *testing.T) {
	manifest := DefaultManifest()
	if manifest.Schema != ManifestSchema {
		t.Fatalf("schema = %q, want %q", manifest.Schema, ManifestSchema)
	}
	if len(manifest.Surfaces) < 7 {
		t.Fatalf("surfaces = %d, want at least 7", len(manifest.Surfaces))
	}
	statuses := map[string]string{}
	for _, capability := range manifest.Capabilities {
		statuses[capability.ID] = capability.Status
	}
	if statuses["server-shell"] != "available" {
		t.Fatalf("server shell status = %q", statuses["server-shell"])
	}
	for _, id := range []string{"scene-document", "scene3d-compile", "native-harness"} {
		if statuses[id] != "available" {
			t.Fatalf("%s status = %q, want available", id, statuses[id])
		}
	}
	if statuses["desktop-host"] != "partial" {
		t.Fatalf("desktop-host status = %q, want partial", statuses["desktop-host"])
	}
	if statuses["viewport-exact-selection"] != "available" {
		t.Fatalf("viewport exact selection status = %q", statuses["viewport-exact-selection"])
	}
	for _, id := range []string{"scene3d-mount", "command-journal", "topology-extrude"} {
		if statuses[id] != "available" {
			t.Fatalf("%s status = %q, want available", id, statuses[id])
		}
	}
	if len(manifest.NextSlice) == 0 {
		t.Fatal("next slice must be dispatchable")
	}
}
