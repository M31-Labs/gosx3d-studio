package studio

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetGarbageCollectionPlansPreviewsAndCheckpointsExactly(t *testing.T) {
	source := t.TempDir()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "mesh.bin"), []byte{1, 2, 3, 4}, 0o600); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(source, "unused.gltf")
	if err := os.WriteFile(input, []byte(`{"asset":{"version":"2.0"},"buffers":[{"uri":"mesh.bin","byteLength":4}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := OpenWorkspace(project, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	initial, _ := workspace.Snapshot()
	_, imported, root, err := workspace.ImportAsset(AssetImportRequest{Path: input, Actor: "test", Mode: ModeDirect, ExpectedRevision: initial.Revision})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanAssetGarbage(imported)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Assets) != 2 || plan.Assets[0] != root.ID || plan.Fingerprint == "" {
		t.Fatalf("plan=%#v", plan)
	}
	preview, err := workspace.CollectAssetGarbage(AssetGCRequest{Actor: "agent://test", Mode: ModePropose, ExpectedRevision: imported.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Receipt.Applied || len(preview.Document.Assets) != 0 {
		t.Fatalf("preview=%#v", preview)
	}
	snapshot, _ := workspace.Snapshot()
	if len(snapshot.Assets) != 2 {
		t.Fatal("preview mutated workspace")
	}
	if _, err := workspace.CollectAssetGarbage(AssetGCRequest{Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: imported.Revision, ConfirmPlan: "wrong"}); err == nil {
		t.Fatal("wrong plan fingerprint accepted")
	}
	paths := []string{}
	for _, asset := range imported.Assets {
		paths = append(paths, filepath.Join(project, filepath.FromSlash(asset.StorePath)))
	}
	result, err := workspace.CollectAssetGarbage(AssetGCRequest{Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: imported.Revision, ConfirmPlan: plan.Fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Checkpointed || result.DeletedPayloads != 2 || len(result.Document.Assets) != 0 {
		t.Fatalf("result=%#v", result)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("payload remains at %s", path)
		}
	}
	if _, _, err := workspace.Undo(result.Document.Revision, "test"); err == nil {
		t.Fatal("checkpointed garbage collection remained undoable")
	}
}

func TestAssetGarbageKeepsTransitiveDependenciesAndRejectsCycles(t *testing.T) {
	dep := inspectBinaryBuffer("mesh.bin", []byte{1})
	root, err := inspectAsset("root.gltf", []byte(`{"asset":{"version":"2.0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	root.Dependencies = []ID{dep.ID}
	document := SampleDocument()
	document.Assets = map[ID]AssetRecord{root.ID: root, dep.ID: dep}
	document.Entities["live-model"] = Entity{ID: "live-model", Name: "Live", Transform: IdentityTransform(), Model: &ModelComponent{Asset: root.ID, Pickable: true}, Visible: true}
	document.RootIDs = append(document.RootIDs, "live-model")
	plan, err := PlanAssetGarbage(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Assets) != 0 {
		t.Fatalf("live transitive assets planned=%v", plan.Assets)
	}
	dep.Dependencies = []ID{root.ID}
	document.Assets[dep.ID] = dep
	if err := document.Validate(); err == nil {
		t.Fatal("asset dependency cycle accepted")
	}
}

func TestAssetGarbageOrdersSharedDependentsBeforeDependency(t *testing.T) {
	dep := inspectBinaryBuffer("shared.bin", []byte{9})
	first, _ := inspectAsset("a.gltf", []byte(`{"asset":{"version":"2.0"},"extras":{"id":"a"}}`))
	second, _ := inspectAsset("b.gltf", []byte(`{"asset":{"version":"2.0"},"extras":{"id":"b"}}`))
	first.Dependencies = []ID{dep.ID}
	second.Dependencies = []ID{dep.ID}
	document := SampleDocument()
	document.Assets = map[ID]AssetRecord{first.ID: first, second.ID: second, dep.ID: dep}
	plan, err := PlanAssetGarbage(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Assets) != 3 || plan.Assets[2] != dep.ID {
		t.Fatalf("shared dependency order=%v", plan.Assets)
	}
}

func TestAssetGarbageActionDiscoveryDeclaresCheckpointPolicy(t *testing.T) {
	for _, descriptor := range ActionDescriptors() {
		if descriptor.Name == string(OpCollectUnusedAssets) {
			if descriptor.Endpoint != "/api/studio/assets/garbage-collect" || descriptor.UndoPolicy != "explicit-checkpoint" || descriptor.SupportsBatch {
				t.Fatalf("descriptor=%#v", descriptor)
			}
			return
		}
	}
	t.Fatal("collect-unused-assets action is not discoverable")
}

// Garbage collection plans outward from the document, so a payload nothing
// references is invisible to it and stays forever. Undoing an import is enough
// to create one: undo restores the document, and the document is not what owns
// the bytes. The audit walks the store instead, which is the only way to see
// them.
func TestAuditReportsPayloadsNothingReferences(t *testing.T) {
	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	if audit := workspace.AuditAssets(); len(audit.Orphans) != 0 {
		t.Fatalf("a fresh project reported orphans: %+v", audit.Orphans)
	}

	// A file in the store that no record names is exactly what a failed
	// import or an undone one leaves behind.
	storeDir := filepath.Join(dir, "assets", "sha256")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(storeDir, "0000000000000000000000000000000000000000000000000000000000000000.bin")
	if err := os.WriteFile(stray, []byte("unreferenced payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	audit := workspace.AuditAssets()
	if len(audit.Orphans) != 1 {
		t.Fatalf("orphans = %+v, want exactly one", audit.Orphans)
	}
	if audit.Orphans[0].Path != "assets/sha256/0000000000000000000000000000000000000000000000000000000000000000.bin" {
		t.Fatalf("orphan path = %q", audit.Orphans[0].Path)
	}
	if audit.Orphans[0].Bytes != int64(len("unreferenced payload")) {
		t.Fatalf("orphan bytes = %d", audit.Orphans[0].Bytes)
	}
	// An unreferenced file is housekeeping, not an integrity failure: every
	// payload the document does reference is still present and correct.
	if !audit.Valid {
		t.Fatal("an orphan made the integrity audit report invalid")
	}
}
