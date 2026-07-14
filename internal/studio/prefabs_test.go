package studio

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLinkedPrefabCaptureInstanceOverrideAndSourceMap(t *testing.T) {
	document := prefabDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	capture := Operation{Kind: OpCapturePrefab, Target: "editable", PrefabID: "piece-prefab", Name: "Piece prefab"}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "preview-prefab", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{capture}})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, captured, err := workspace.Execute(Transaction{ID: "capture-prefab", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{capture}})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	capturedFingerprint, _ := captured.Fingerprint()
	if previewFingerprint != capturedFingerprint {
		t.Fatal("prefab capture preview and direct differ")
	}
	for _, receipt := range []Receipt{previewReceipt, directReceipt} {
		if len(receipt.PrefabChanges) != 1 || receipt.PrefabChanges[0].After == nil || receipt.OperatorRecords[0].Result[0] != "piece-prefab" {
			t.Fatalf("prefab capture receipt = %+v", receipt)
		}
	}
	instantiate := Operation{Kind: OpInstantiatePrefab, PrefabID: "piece-prefab", NewID: "piece-instance", Parent: "scene-root", Transform: &Transform{Position: Vec3{X: 3}, Scale: Vec3{X: 1, Y: 1, Z: 1}}}
	_, instantiated, err := workspace.Execute(Transaction{ID: "instantiate-prefab", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: captured.Revision, Operations: []Operation{instantiate}})
	if err != nil {
		t.Fatal(err)
	}
	visible := false
	override := PrefabEntityOverride{Material: "player-1-material", Visible: &visible}
	_, overridden, err := workspace.Execute(Transaction{ID: "override-prefab", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: instantiated.Revision, Operations: []Operation{{Kind: OpSetPrefabOverride, Target: "piece-instance", PrefabEntity: "editable-child", PrefabOverride: &override}}})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := CompileWithSourceMap(overridden)
	if err != nil {
		t.Fatal(err)
	}
	for _, runtimeID := range []ID{"piece-instance--editable", "piece-instance--editable-child"} {
		location, ok := artifact.SourceMap[runtimeID]
		if !ok || location.Entity != "piece-instance" || !strings.HasPrefix(location.RecordID, "piece-prefab/") {
			t.Fatalf("prefab source map %q = %+v", runtimeID, location)
		}
	}
	encoded, _ := json.Marshal(artifact.SceneIR)
	if !strings.Contains(string(encoded), "piece-instance--editable-child") {
		t.Fatal("compiled SceneIR lacks prefab child runtime id")
	}
	_, _, err = workspace.Execute(Transaction{ID: "delete-referenced-prefab", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: overridden.Revision, Operations: []Operation{{Kind: OpDeletePrefab, PrefabID: "piece-prefab"}}})
	if err == nil {
		t.Fatal("referenced prefab was deleted")
	}
}

func TestPrefabDefinitionChangesInvalidateLinkedInstanceIncrementally(t *testing.T) {
	document := prefabDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, captured, err := workspace.Execute(Transaction{ID: "capture", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpCapturePrefab, Target: "editable", PrefabID: "linked", Name: "Linked"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, instantiated, err := workspace.Execute(Transaction{ID: "instance", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: captured.Revision, Operations: []Operation{{Kind: OpInstantiatePrefab, PrefabID: "linked", NewID: "linked-instance", Parent: "scene-root"}}})
	if err != nil {
		t.Fatal(err)
	}
	compiler := NewIncrementalCompiler()
	if _, _, err := compiler.Compile(instantiated, ""); err != nil {
		t.Fatal(err)
	}
	transform := instantiated.Entities["editable"].Transform
	transform.Position.Y = 2
	_, updated, err := workspace.Execute(Transaction{ID: "update-definition", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: instantiated.Revision, Operations: []Operation{{Kind: OpSetTransform, Target: "editable", Transform: &transform}, {Kind: OpCapturePrefab, Target: "editable", PrefabID: "linked", Name: "Linked"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, stats, err := compiler.Compile(updated, "")
	if err != nil {
		t.Fatal(err)
	}
	if stats.RecompiledEntities == 0 {
		t.Fatal("linked prefab definition change reused stale instance")
	}
	clean, err := Compile(updated)
	if err != nil {
		t.Fatal(err)
	}
	incremental, _, err := compiler.Compile(updated, "")
	if err != nil {
		t.Fatal(err)
	}
	cleanJSON, _ := json.Marshal(clean.SceneIR())
	incrementalJSON, _ := json.Marshal(incremental.SceneIR())
	if string(cleanJSON) != string(incrementalJSON) {
		t.Fatal("incremental prefab compile differs from clean compile")
	}
}

func prefabDocument(t *testing.T) Document {
	t.Helper()
	document := operatorDocument(t)
	source := document.Entities["editable"]
	child := Entity{ID: "editable-child", Name: "Child", Parent: source.ID, Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{Geometry: Geometry{Kind: "box", Width: 0.25, Height: 0.25, Depth: 0.25}, Material: "board-material", Pickable: true}}
	source.Children = append(source.Children, child.ID)
	document.Entities[source.ID] = source
	document.Entities[child.ID] = child
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	return document
}
