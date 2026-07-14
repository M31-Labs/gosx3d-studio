package studio

import (
	"encoding/json"
	"testing"
)

func TestModifierStackEvaluatesInOrderWithStableGeneratedIDs(t *testing.T) {
	document := operatorDocument(t)
	geometry := document.Entities["editable"].Mesh.Geometry
	modifiers := []Modifier{{ID: "mirror-x", Kind: "mirror", Enabled: true, Axis: "x"}, {ID: "array-y", Kind: "array", Enabled: true, Count: 3, Offset: Vec3{Y: 2}}}
	evaluated, err := evaluateModifiers(geometry, modifiers)
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluated.Vertices) != 24 || len(evaluated.Faces) != 6 {
		t.Fatalf("evaluated topology vertices=%d faces=%d", len(evaluated.Vertices), len(evaluated.Faces))
	}
	vertexIDs := map[ID]bool{}
	for _, vertex := range evaluated.Vertices {
		if vertexIDs[vertex.ID] {
			t.Fatalf("duplicate evaluated vertex %q", vertex.ID)
		}
		vertexIDs[vertex.ID] = true
	}
	for _, required := range []ID{"a", "mirror-x--mirror--a", "array-y--array-1--a", "array-y--array-2--mirror-x--mirror--a"} {
		if !vertexIDs[required] {
			t.Fatalf("missing generated vertex %q", required)
		}
	}
	if len(geometry.Vertices) != 4 || len(geometry.Faces) != 1 {
		t.Fatal("modifier evaluation mutated authored geometry")
	}
}

func TestModifierActionsHavePreviewDirectParityAndCheckpointUndo(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	mirror := Modifier{ID: "mirror-z", Kind: "mirror", Enabled: true, Axis: "z"}
	array := Modifier{ID: "array-x", Kind: "array", Enabled: true, Count: 2, Offset: Vec3{X: 2}}
	operations := []Operation{{Kind: OpSetModifier, Target: "editable", Modifier: &mirror}, {Kind: OpSetModifier, Target: "editable", Modifier: &array}}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "preview-modifiers", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: operations})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "direct-modifiers", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: operations})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	if previewFingerprint != directFingerprint {
		t.Fatal("modifier preview and direct differ")
	}
	if len(direct.Entities["editable"].Mesh.Modifiers) != 2 || len(previewReceipt.OperatorRecords) != 2 || len(directReceipt.OperatorRecords) != 2 {
		t.Fatalf("modifier state/receipts preview=%+v direct=%+v", previewReceipt, directReceipt)
	}
	if _, err := Compile(direct); err != nil {
		t.Fatalf("compile evaluated modifier stack: %v", err)
	}
	_, removed, err := workspace.Execute(Transaction{ID: "remove-modifier", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: direct.Revision, Operations: []Operation{{Kind: OpRemoveModifier, Target: "editable", ModifierID: "mirror-z"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Entities["editable"].Mesh.Modifiers) != 1 || removed.Entities["editable"].Mesh.Modifiers[0].ID != "array-x" {
		t.Fatalf("remaining modifiers = %+v", removed.Entities["editable"].Mesh.Modifiers)
	}
	_, restored, err := workspace.Undo(removed.Revision, "agent://test")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Entities["editable"].Mesh.Modifiers) != 2 {
		t.Fatal("modifier removal undo did not restore stack")
	}
}

func TestModifierValidationRejectsUnsupportedOrUnsafeParameters(t *testing.T) {
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	for _, modifier := range []Modifier{{ID: "bad-axis", Kind: "mirror", Enabled: true, Axis: "q"}, {ID: "bad-count", Kind: "array", Enabled: true, Count: 1, Offset: Vec3{X: 1}}, {ID: "unknown", Kind: "shrinkwrap", Enabled: true}} {
		entity.Mesh.Modifiers = []Modifier{modifier}
		document.Entities[entity.ID] = entity
		if err := document.Validate(); err == nil {
			t.Fatalf("invalid modifier accepted: %+v", modifier)
		}
	}
}

func TestSolidifyModifierProducesClosedDeterministicVolume(t *testing.T) {
	document := operatorDocument(t)
	geometry := document.Entities["editable"].Mesh.Geometry
	modifier := Modifier{ID: "shell", Kind: "solidify", Enabled: true, Thickness: 0.2}
	first, err := evaluateModifiers(geometry, []Modifier{modifier})
	if err != nil {
		t.Fatal(err)
	}
	second, err := evaluateModifiers(geometry, []Modifier{modifier})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Vertices) != 8 || len(first.Faces) != 6 {
		t.Fatalf("solidified topology vertices=%d faces=%d", len(first.Vertices), len(first.Faces))
	}
	firstDocument, _ := document.Clone()
	entity := firstDocument.Entities["editable"]
	entity.Mesh.Geometry = first
	entity.Mesh.Modifiers = nil
	firstDocument.Entities[entity.ID] = entity
	analysis, err := AnalyzeEntityGeometry(firstDocument, entity.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Valid || !analysis.Closed || !analysis.Manifold || analysis.Volume == nil || !near(*analysis.Volume, 0.05) {
		t.Fatalf("solidified analysis = %#v", analysis)
	}
	firstBytes, _ := json.Marshal(first)
	secondBytes, _ := json.Marshal(second)
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("solidify output is not deterministic")
	}
}

func TestSolidifyActionPreviewDirectUndoAndSceneIR(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	modifier := Modifier{ID: "human-shell", Kind: "solidify", Enabled: true, Thickness: 0.125}
	operation := Operation{Kind: OpSetModifier, Target: "editable", Modifier: &modifier}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "solidify-preview", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "solidify-direct", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	if previewFingerprint != directFingerprint || len(previewReceipt.OperatorRecords) != 1 || len(directReceipt.OperatorRecords) != 1 {
		t.Fatal("solidify action lost preview/direct or receipt parity")
	}
	ir, err := CompileIR(direct)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Objects) == 0 {
		t.Fatal("solidified mesh did not lower to SceneIR")
	}
	_, restored, err := workspace.Undo(direct.Revision, "human://test")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Entities["editable"].Mesh.Modifiers) != 0 {
		t.Fatal("solidify undo did not restore authored stack")
	}
}

func TestModifierReorderChangesEvaluationAndApplyBakesThroughCheckpoint(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	mirror := Modifier{ID: "mirror", Kind: "mirror", Enabled: true, Axis: "x"}
	array := Modifier{ID: "array", Kind: "array", Enabled: true, Count: 2, Offset: Vec3{X: 2}}
	solidify := Modifier{ID: "shell", Kind: "solidify", Enabled: true, Thickness: 0.1}
	_, stacked, err := workspace.Execute(Transaction{ID: "stack", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{
		{Kind: OpSetModifier, Target: "editable", Modifier: &mirror},
		{Kind: OpSetModifier, Target: "editable", Modifier: &array},
		{Kind: OpSetModifier, Target: "editable", Modifier: &solidify},
	}})
	if err != nil {
		t.Fatal(err)
	}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "reorder-preview", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: stacked.Revision, Operations: []Operation{{Kind: OpReorderModifier, Target: "editable", ModifierID: "shell", ModifierIndex: 0}}})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, reordered, err := workspace.Execute(Transaction{ID: "reorder-direct", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: stacked.Revision, Operations: []Operation{{Kind: OpReorderModifier, Target: "editable", ModifierID: "shell", ModifierIndex: 0}}})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := reordered.Fingerprint()
	if previewFingerprint != directFingerprint || reordered.Entities["editable"].Mesh.Modifiers[0].ID != "shell" || previewReceipt.OperatorRecords[0].ModifierIndex != 0 || directReceipt.OperatorRecords[0].ModifierID != "shell" {
		t.Fatal("modifier reorder lost parity, identity, or receipt parameters")
	}
	applyPreviewReceipt, applyPreview, err := workspace.Execute(Transaction{ID: "apply-preview", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: reordered.Revision, Operations: []Operation{{Kind: OpApplyModifier, Target: "editable", ModifierID: "mirror"}}})
	if err != nil {
		t.Fatal(err)
	}
	applyReceipt, applied, err := workspace.Execute(Transaction{ID: "apply-direct", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: reordered.Revision, Operations: []Operation{{Kind: OpApplyModifier, Target: "editable", ModifierID: "mirror"}}})
	if err != nil {
		t.Fatal(err)
	}
	applyPreviewFingerprint, _ := applyPreview.Fingerprint()
	appliedFingerprint, _ := applied.Fingerprint()
	mesh := applied.Entities["editable"].Mesh
	if applyPreviewFingerprint != appliedFingerprint || len(mesh.Modifiers) != 1 || mesh.Modifiers[0].ID != "array" || len(mesh.Geometry.Vertices) != 16 || len(mesh.Geometry.Faces) != 12 {
		t.Fatalf("applied geometry/modifiers = v%d f%d %#v", len(mesh.Geometry.Vertices), len(mesh.Geometry.Faces), mesh.Modifiers)
	}
	if applyPreviewReceipt.OperatorRecords[0].UndoPolicy != "geometry-checkpoint" || applyReceipt.OperatorRecords[0].ModifierID != "mirror" {
		t.Fatalf("apply receipts = %#v %#v", applyPreviewReceipt.OperatorRecords, applyReceipt.OperatorRecords)
	}
	if _, err := Compile(applied); err != nil {
		t.Fatal(err)
	}
	_, restored, err := workspace.Undo(applied.Revision, "human://test")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Entities["editable"].Mesh.Modifiers) != 3 {
		t.Fatal("apply checkpoint undo did not restore stack")
	}
}

func TestModifierReorderAndApplyRejectInvalidIdentityOrIndex(t *testing.T) {
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Modifiers = []Modifier{{ID: "mirror", Kind: "mirror", Enabled: true, Axis: "x"}}
	document.Entities[entity.ID] = entity
	for _, operation := range []Operation{
		{Kind: OpReorderModifier, Target: "editable", ModifierID: "mirror", ModifierIndex: 1},
		{Kind: OpReorderModifier, Target: "editable", ModifierID: "missing", ModifierIndex: 0},
		{Kind: OpApplyModifier, Target: "editable", ModifierID: "missing"},
	} {
		clone, _ := document.Clone()
		if _, err := applyOperation(&clone, operation); err == nil {
			t.Fatalf("invalid modifier operation accepted: %#v", operation)
		}
	}
}

func TestSubdivisionModifierSharesEdgesAndPreservesClosedManifold(t *testing.T) {
	geometry := cubeGeometry()
	modifier := Modifier{ID: "catmull", Kind: "subdivision", Enabled: true, Levels: 1}
	first, err := evaluateModifiers(geometry, []Modifier{modifier})
	if err != nil {
		t.Fatal(err)
	}
	second, err := evaluateModifiers(geometry, []Modifier{modifier})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Vertices) != 26 || len(first.Faces) != 24 || len(MeshEdges(first)) != 48 {
		t.Fatalf("subdivision topology vertices=%d edges=%d faces=%d", len(first.Vertices), len(MeshEdges(first)), len(first.Faces))
	}
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Geometry = first
	entity.Mesh.Modifiers = nil
	document.Entities[entity.ID] = entity
	analysis, err := AnalyzeEntityGeometry(document, entity.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Valid || !analysis.Closed || !analysis.Manifold || analysis.Volume == nil || *analysis.Volume <= 0 {
		t.Fatalf("subdivision analysis = %#v", analysis)
	}
	firstBytes, _ := json.Marshal(first)
	secondBytes, _ := json.Marshal(second)
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("subdivision output is not deterministic")
	}
}

func TestSubdivisionLevelsAndActionParity(t *testing.T) {
	document := operatorDocument(t)
	modifier := Modifier{ID: "subdivide", Kind: "subdivision", Enabled: true, Levels: 2}
	evaluated, err := evaluateModifiers(document.Entities["editable"].Mesh.Geometry, []Modifier{modifier})
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluated.Vertices) != 25 || len(evaluated.Faces) != 16 {
		t.Fatalf("two-level subdivision topology v=%d f=%d", len(evaluated.Vertices), len(evaluated.Faces))
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{Kind: OpSetModifier, Target: "editable", Modifier: &modifier}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "subdivision-preview", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "subdivision-direct", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	if previewFingerprint != directFingerprint || previewReceipt.OperatorRecords[0].Modifier.Levels != 2 || directReceipt.OperatorRecords[0].Modifier.ID != "subdivide" {
		t.Fatal("subdivision action lost parity or normalized parameters")
	}
	if _, err := Compile(direct); err != nil {
		t.Fatal(err)
	}
	_, restored, err := workspace.Undo(direct.Revision, "human://test")
	if err != nil || len(restored.Entities["editable"].Mesh.Modifiers) != 0 {
		t.Fatalf("subdivision undo: %v modifiers=%#v", err, restored.Entities["editable"].Mesh.Modifiers)
	}
}

func TestSubdivisionValidationRejectsUnsafeLevels(t *testing.T) {
	document := operatorDocument(t)
	for _, levels := range []int{0, 5} {
		entity := document.Entities["editable"]
		entity.Mesh.Modifiers = []Modifier{{ID: "bad-subdivision", Kind: "subdivision", Enabled: true, Levels: levels}}
		document.Entities[entity.ID] = entity
		if err := document.Validate(); err == nil {
			t.Fatalf("subdivision levels %d accepted", levels)
		}
	}
}
