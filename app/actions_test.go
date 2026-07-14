package app

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func TestHumanTransformUsesRevisionSafeUndoableTransaction(t *testing.T) {
	document := studio.SampleDocument()
	workspace, err := studio.NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target := studio.ID("piece-player-1-01")
	entity := document.Entities[target]
	values := map[string]string{
		"target": string(target), "expectedRevision": strconv.FormatUint(document.Revision, 10),
		"x": "1.25", "y": "0.5", "z": "-2", "rx": "0", "ry": "0.75", "rz": "0",
	}
	receipt, err := executeHumanTransform(workspace, values)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Actor != "human://local-ui" || receipt.Operations != 1 || receipt.Changes[0].Kind != studio.OpSetTransform {
		t.Fatalf("receipt=%+v", receipt)
	}
	changed, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if changed.Entities[target].Transform.Position.X != 1.25 {
		t.Fatalf("position=%+v", changed.Entities[target].Transform.Position)
	}
	_, restored, err := workspace.Undo(changed.Revision, "human://local-ui")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Entities[target].Transform != entity.Transform {
		t.Fatalf("undo transform=%+v want=%+v", restored.Entities[target].Transform, entity.Transform)
	}
	if _, err := executeHumanTransform(workspace, values); err == nil {
		t.Fatal("stale human revision was accepted")
	}
}

func TestHumanAssetReimportUsesAtomicSharedCommand(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.gltf")
	replacementPath := filepath.Join(t.TempDir(), "replacement.gltf")
	if err := os.WriteFile(source, []byte(`{"asset":{"version":"2.0"},"scene":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacementPath, []byte(`{"asset":{"version":"2.0"},"scenes":[{}],"scene":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := studio.OpenWorkspace(project, studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	document, _ := workspace.Snapshot()
	importReceipt, asset, err := executeHumanAssetImport(workspace, map[string]string{"path": source, "expectedRevision": strconv.FormatUint(document.Revision, 10)})
	if err != nil {
		t.Fatal(err)
	}
	if importReceipt.Actor != "human://local-ui" || importReceipt.OperatorRecords[0].Kind != studio.OpRegisterAsset || importReceipt.OperatorRecords[0].Result[0] != asset.ID {
		t.Fatalf("human import receipt = %#v", importReceipt)
	}
	if _, _, err := executeHumanAssetImport(workspace, map[string]string{"path": replacementPath, "expectedRevision": strconv.FormatUint(document.Revision, 10)}); err == nil {
		t.Fatal("stale human initial import was accepted")
	}
	stored, err := os.ReadDir(filepath.Join(project, "assets", "sha256"))
	if err != nil || len(stored) != 1 {
		t.Fatalf("stale import stored an orphan payload: entries=%d err=%v", len(stored), err)
	}
	document, _ = workspace.Snapshot()
	model := studio.Entity{ID: "human-model", Name: "Human model", Transform: studio.IdentityTransform(), Model: &studio.ModelComponent{Asset: asset.ID, Pickable: true}, Visible: true}
	_, document, err = workspace.Execute(studio.Transaction{ID: "create-human-model", Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: document.Revision, Operations: []studio.Operation{{Kind: studio.OpCreateEntity, Entity: &model}}})
	if err != nil {
		t.Fatal(err)
	}
	receipt, replacement, err := executeHumanAssetReimport(workspace, map[string]string{
		"assetId": string(asset.ID), "path": replacementPath, "expectedRevision": strconv.FormatUint(document.Revision, 10),
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := workspace.Snapshot()
	if receipt.Actor != "human://local-ui" || receipt.OperatorRecords[0].Kind != studio.OpReimportAsset || receipt.OperatorRecords[0].Result[0] != replacement.ID || updated.Entities["human-model"].Model.Asset != replacement.ID {
		t.Fatalf("human reimport receipt/state = %#v %#v", receipt, updated.Entities["human-model"].Model)
	}
	if _, _, err := executeHumanAssetReimport(workspace, map[string]string{"assetId": string(asset.ID), "path": replacementPath, "expectedRevision": strconv.FormatUint(document.Revision, 10)}); err == nil {
		t.Fatal("stale human reimport revision was accepted")
	}
}

func TestHumanSolidifyUsesSharedModifierCommandAndRejectsStaleRevision(t *testing.T) {
	document := studio.SampleDocument()
	entity := document.Entities["board"]
	entity.Mesh.Geometry = studio.Geometry{Kind: "indexed-mesh", Vertices: []studio.Vertex{
		{ID: "a", Position: studio.Vec3{}}, {ID: "b", Position: studio.Vec3{X: 1}},
		{ID: "c", Position: studio.Vec3{X: 1, Z: 1}}, {ID: "d", Position: studio.Vec3{Z: 1}},
	}, Faces: []studio.Face{{ID: "surface", Vertices: []studio.ID{"a", "b", "c", "d"}}}}
	document.Entities[entity.ID] = entity
	workspace, err := studio.NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		"target": "board", "expectedRevision": strconv.FormatUint(document.Revision, 10),
		"modifierId": "shell", "thickness": "0.2",
	}
	receipt, err := executeHumanSolidify(workspace, values)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Actor != "human://local-ui" || receipt.OperatorRecords[0].Kind != studio.OpSetModifier || receipt.OperatorRecords[0].Modifier.Thickness != 0.2 {
		t.Fatalf("receipt = %#v", receipt)
	}
	changed, _ := workspace.Snapshot()
	if len(changed.Entities["board"].Mesh.Modifiers) != 1 || changed.Entities["board"].Mesh.Modifiers[0].Kind != "solidify" {
		t.Fatalf("modifiers = %#v", changed.Entities["board"].Mesh.Modifiers)
	}
	if _, err := executeHumanSolidify(workspace, values); err == nil {
		t.Fatal("stale human solidify revision was accepted")
	}
	array := studio.Modifier{ID: "array", Kind: "array", Enabled: true, Count: 2, Offset: studio.Vec3{X: 2}}
	_, stacked, err := workspace.Execute(studio.Transaction{ID: "add-array", Actor: "agent://test", Mode: studio.ModeDirect, ExpectedRevision: changed.Revision, Operations: []studio.Operation{{Kind: studio.OpSetModifier, Target: "board", Modifier: &array}}})
	if err != nil {
		t.Fatal(err)
	}
	reorderValues := map[string]string{"target": "board", "expectedRevision": strconv.FormatUint(stacked.Revision, 10), "modifierId": "array", "modifierIndex": "0"}
	reorderReceipt, err := executeHumanModifierOperation(workspace, reorderValues, studio.OpReorderModifier)
	if err != nil {
		t.Fatal(err)
	}
	reordered, _ := workspace.Snapshot()
	if reorderReceipt.Actor != "human://local-ui" || reordered.Entities["board"].Mesh.Modifiers[0].ID != "array" {
		t.Fatalf("human reorder receipt/state = %#v %#v", reorderReceipt, reordered.Entities["board"].Mesh.Modifiers)
	}
	applyValues := map[string]string{"target": "board", "expectedRevision": strconv.FormatUint(reordered.Revision, 10), "modifierId": "shell"}
	applyReceipt, err := executeHumanModifierOperation(workspace, applyValues, studio.OpApplyModifier)
	if err != nil {
		t.Fatal(err)
	}
	applied, _ := workspace.Snapshot()
	if applyReceipt.OperatorRecords[0].UndoPolicy != "geometry-checkpoint" || len(applied.Entities["board"].Mesh.Modifiers) != 0 || len(applied.Entities["board"].Mesh.Geometry.Vertices) != 16 {
		t.Fatalf("human apply receipt/state = %#v %#v", applyReceipt.OperatorRecords, applied.Entities["board"].Mesh)
	}
	subdivisionValues := map[string]string{"target": "board", "expectedRevision": strconv.FormatUint(applied.Revision, 10), "modifierId": "smooth", "levels": "1"}
	subdivisionReceipt, err := executeHumanSubdivision(workspace, subdivisionValues)
	if err != nil {
		t.Fatal(err)
	}
	subdivided, _ := workspace.Snapshot()
	modifier := subdivided.Entities["board"].Mesh.Modifiers[0]
	if subdivisionReceipt.Actor != "human://local-ui" || modifier.ID != "smooth" || modifier.Kind != "subdivision" || modifier.Levels != 1 {
		t.Fatalf("human subdivision receipt/modifier = %#v %#v", subdivisionReceipt, modifier)
	}
	if _, err := executeHumanSubdivision(workspace, subdivisionValues); err == nil {
		t.Fatal("stale human subdivision revision was accepted")
	}
}
