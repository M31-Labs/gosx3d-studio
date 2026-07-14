package studio

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testGLB() []byte {
	jsonChunk := []byte(`{"asset":{"version":"2.0"}} `)
	data := make([]byte, 20+len(jsonChunk))
	copy(data[:4], "glTF")
	binary.LittleEndian.PutUint32(data[4:8], 2)
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[12:16], uint32(len(jsonChunk)))
	copy(data[16:20], "JSON")
	copy(data[20:], jsonChunk)
	return data
}

func TestContentAddressedAssetImportAuditAndModelCompile(t *testing.T) {
	project := t.TempDir()
	input := filepath.Join(t.TempDir(), "fixture.glb")
	data := testGLB()
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := OpenWorkspace(project, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	initial, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	previewReceipt, preview, previewAsset, err := workspace.ImportAsset(AssetImportRequest{Path: input, Actor: "test", Mode: ModePropose, ExpectedRevision: initial.Revision})
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if previewReceipt.Applied || len(preview.Assets) != 1 || len(unchanged.Assets) != 0 {
		t.Fatal("proposed import must preview metadata without mutating or storing payload")
	}
	if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(previewAsset.StorePath))); !os.IsNotExist(err) {
		t.Fatalf("proposed payload unexpectedly stored: %v", err)
	}

	receipt, document, asset, err := workspace.ImportAsset(AssetImportRequest{Path: input, Actor: "test", Mode: ModeDirect, ExpectedRevision: initial.Revision})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	wantHash := hex.EncodeToString(sum[:])
	if asset.ContentHash != wantHash || asset.ID != previewAsset.ID || len(receipt.AssetChanges) != 1 || receipt.AssetChanges[0].After == nil {
		t.Fatalf("asset receipt = %#v asset = %#v", receipt.AssetChanges, asset)
	}
	audit := workspace.AuditAssets()
	if !audit.Valid || len(audit.Assets) != 1 || audit.Assets[0].Status != "ok" {
		t.Fatalf("audit = %#v", audit)
	}

	modelID := ID("imported-model")
	model := Entity{ID: modelID, Name: "Imported model", Transform: IdentityTransform(), Model: &ModelComponent{Asset: asset.ID, Bounds: 2, Fit: "contain", Pickable: true}, Visible: true}
	_, document, err = workspace.Execute(Transaction{ID: "create-model", Actor: "test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpCreateEntity, Entity: &model}}})
	if err != nil {
		t.Fatal(err)
	}
	ir, err := CompileIR(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Models) != 1 || ir.Models[0].ID != string(modelID) || ir.Models[0].Src != asset.URI {
		t.Fatalf("compiled models = %#v", ir.Models)
	}
	_, document, err = workspace.Execute(Transaction{ID: "capture-model-prefab", Actor: "test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpCapturePrefab, Target: modelID, PrefabID: "model-prefab", Name: "Model prefab"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, document, err = workspace.Execute(Transaction{ID: "instantiate-model-prefab", Actor: "test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpInstantiatePrefab, PrefabID: "model-prefab", NewID: "model-prefab-instance"}}})
	if err != nil {
		t.Fatal(err)
	}
	dependencies := workspace.AssetDependencies()
	if len(dependencies.Assets) != 1 || dependencies.Assets[0].DirectCount != 2 || len(dependencies.Assets[0].Entities) != 1 || len(dependencies.Assets[0].Prefabs) != 1 || len(dependencies.Assets[0].Instances) != 1 {
		t.Fatalf("dependencies = %#v", dependencies)
	}
	if _, _, err := workspace.Execute(Transaction{ID: "delete-used-asset", Actor: "test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpDeleteAsset, AssetID: asset.ID}}}); err == nil {
		t.Fatal("expected referenced asset deletion to fail")
	}

	reimportPath := filepath.Join(t.TempDir(), "replacement.gltf")
	if err := os.WriteFile(reimportPath, []byte(`{"asset":{"version":"2.0"},"scenes":[{}],"scene":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	request := AssetReimportRequest{AssetID: asset.ID, Path: reimportPath, Actor: "test", Mode: ModePropose, ExpectedRevision: document.Revision}
	previewReceipt, reimportPreview, previewReplacement, err := workspace.ReimportAsset(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Mode = ModeDirect
	directReceipt, reimported, replacement, err := workspace.ReimportAsset(request)
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := reimportPreview.Fingerprint()
	directFingerprint, _ := reimported.Fingerprint()
	if previewReplacement.ID != replacement.ID || previewFingerprint != directFingerprint || len(previewReceipt.AssetChanges) != 2 || len(directReceipt.AssetChanges) != 2 {
		t.Fatalf("reimport parity/receipts preview=%#v direct=%#v", previewReceipt.AssetChanges, directReceipt.AssetChanges)
	}
	if _, exists := reimported.Assets[asset.ID]; exists || reimported.Entities[modelID].Model.Asset != replacement.ID || reimported.Prefabs["model-prefab"].Entities[modelID].Model.Asset != replacement.ID {
		t.Fatalf("reimport did not migrate references: assets=%#v model=%#v prefab=%#v", reimported.Assets, reimported.Entities[modelID].Model, reimported.Prefabs["model-prefab"])
	}
	ir, err = CompileIR(reimported)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Models) != 2 {
		t.Fatalf("reimported models = %#v", ir.Models)
	}
	for _, compiled := range ir.Models {
		if compiled.Src != replacement.URI {
			t.Fatalf("model %q src=%q want=%q", compiled.ID, compiled.Src, replacement.URI)
		}
	}
	dependencies = workspace.AssetDependencies()
	if len(dependencies.Assets) != 1 || dependencies.Assets[0].Asset != replacement.ID || dependencies.Assets[0].DirectCount != 2 {
		t.Fatalf("reimported dependencies = %#v", dependencies)
	}
	_, restored, err := workspace.Undo(reimported.Revision, "test")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Entities[modelID].Model.Asset != asset.ID || restored.Prefabs["model-prefab"].Entities[modelID].Model.Asset != asset.ID {
		t.Fatalf("reimport undo did not restore old references: %#v", restored.Entities[modelID].Model)
	}
	_, replayed, err := workspace.Redo(restored.Revision, "test")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Entities[modelID].Model.Asset != replacement.ID {
		t.Fatalf("reimport redo did not restore replacement: %#v", replayed.Entities[modelID].Model)
	}
	path, _, err := workspace.AssetContentPath(replacement.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if audit := workspace.AuditAssets(); audit.Valid || audit.Assets[0].Status != "mismatch" {
		t.Fatalf("tampered audit = %#v", audit)
	}
}

func TestInspectAssetRejectsInvalidGLB(t *testing.T) {
	data := testGLB()
	binary.LittleEndian.PutUint32(data[8:12], 999)
	if _, err := inspectAsset("bad.glb", data); err == nil {
		t.Fatal("expected invalid declared length to fail")
	}
}

func TestGLTFExternalDependenciesAreRewrittenAndContentAddressed(t *testing.T) {
	source := t.TempDir()
	project := t.TempDir()
	buffer := []byte{1, 2, 3, 4}
	if err := os.WriteFile(filepath.Join(source, "mesh.data"), buffer, 0o600); err != nil {
		t.Fatal(err)
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), []byte("fixture")...)
	if err := os.WriteFile(filepath.Join(source, "albedo.png"), png, 0o600); err != nil {
		t.Fatal(err)
	}
	gltf := []byte(`{"asset":{"version":"2.0"},"buffers":[{"uri":"mesh.data","byteLength":4}],"images":[{"uri":"albedo.png"},{"uri":"albedo.png"}]}`)
	path := filepath.Join(source, "model.gltf")
	if err := os.WriteFile(path, gltf, 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := OpenWorkspace(project, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	document, _ := workspace.Snapshot()
	_, imported, root, err := workspace.ImportAsset(AssetImportRequest{Path: path, Actor: "test", Mode: ModeDirect, ExpectedRevision: document.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Assets) != 3 {
		t.Fatalf("assets=%d, want root plus two unique dependencies", len(imported.Assets))
	}
	if len(root.Dependencies) != 2 {
		t.Fatalf("typed dependencies=%v", root.Dependencies)
	}
	if root.Metadata["sourceContentHash"] == "" || len(strings.Split(root.Metadata["gltfDependencies"], ",")) != 2 {
		t.Fatalf("root metadata=%#v", root.Metadata)
	}
	storedPath, _, err := workspace.AssetContentPath(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), `"mesh.data"`) || strings.Contains(string(stored), `"albedo.png"`) || strings.Count(string(stored), "/api/studio/assets/content/asset-sha256-") != 3 {
		t.Fatalf("root was not deterministically rewritten: %s", stored)
	}
	if audit := workspace.AuditAssets(); !audit.Valid || len(audit.Assets) != 3 {
		t.Fatalf("audit=%#v", audit)
	}
	report := workspace.AssetDependencies()
	usedBy := 0
	for _, entry := range report.Assets {
		if entry.Asset == root.Dependencies[0] || entry.Asset == root.Dependencies[1] {
			if len(entry.UsedByAssets) != 1 || entry.UsedByAssets[0] != root.ID {
				t.Fatalf("dependency report entry=%#v", entry)
			}
			usedBy++
		}
	}
	if usedBy != 2 {
		t.Fatalf("dependency report=%#v", report)
	}
	if _, _, err := workspace.Execute(Transaction{ID: "delete-live-dependency", Actor: "test", Mode: ModeDirect, ExpectedRevision: imported.Revision, Operations: []Operation{{Kind: OpDeleteAsset, AssetID: root.Dependencies[0]}}}); err == nil {
		t.Fatal("referenced glTF dependency deletion succeeded")
	}
}

func TestGLTFDependencyConfinementRejectsRemoteTraversalAndSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.bin")
	if err := os.WriteFile(outside, []byte{1}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "escape.bin")); err != nil {
		t.Fatal(err)
	}
	for name, uri := range map[string]string{"remote": "https://example.com/a.bin", "traversal": "../outside.bin", "symlink": "escape.bin"} {
		payload := []byte(`{"asset":{"version":"2.0"},"buffers":[{"uri":"` + uri + `","byteLength":1}]}`)
		if _, _, err := prepareAssetImport(filepath.Join(base, name+".gltf"), payload); err == nil {
			t.Fatalf("%s dependency %q was accepted", name, uri)
		}
	}
}
