package studio

import "testing"

func TestMaterialActionsHavePreviewParitySemanticReceiptsAndUndo(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	material := document.Materials["board-material"]
	material.Roughness = 0.42
	material.Metalness = 0.17
	operation := Operation{Kind: OpSetMaterial, MaterialRecord: &material}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "preview-material", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "direct-material", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	if previewFingerprint != directFingerprint {
		t.Fatal("material preview and direct differ")
	}
	for _, receipt := range []Receipt{previewReceipt, directReceipt} {
		if len(receipt.MaterialChanges) != 1 || receipt.MaterialChanges[0].Before == nil || receipt.MaterialChanges[0].After == nil || receipt.OperatorRecords[0].Result[0] != material.ID {
			t.Fatalf("material receipt = %+v", receipt)
		}
	}
	if direct.Materials[material.ID].Roughness != 0.42 {
		t.Fatalf("material update = %+v", direct.Materials[material.ID])
	}
	if _, err := Compile(direct); err != nil {
		t.Fatalf("compile updated material: %v", err)
	}
	_, restored, err := workspace.Undo(direct.Revision, "agent://test")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Materials[material.ID].Roughness == 0.42 {
		t.Fatal("material undo did not restore prior state")
	}
}

func TestInvalidSelenaCannotReplaceLastValidMaterialState(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := workspace.Snapshot()
	beforeFingerprint, _ := before.Fingerprint()
	material := document.Materials["board-material"]
	if material.Selena == nil {
		t.Fatal("sample Selena material missing")
	}
	material.Selena = &SelenaShader{Material: material.Selena.Material, Source: "this is not valid Selena source"}
	_, _, err = workspace.Execute(Transaction{ID: "invalid-selena", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetMaterial, MaterialRecord: &material}}})
	if err == nil {
		t.Fatal("invalid Selena source committed")
	}
	after, _ := workspace.Snapshot()
	afterFingerprint, _ := after.Fingerprint()
	if afterFingerprint != beforeFingerprint {
		t.Fatal("invalid Selena attempt changed last valid material state")
	}
}

func TestDeleteMaterialRejectsReferencesAndDeletesUnusedRecords(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = workspace.Execute(Transaction{ID: "delete-used", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpDeleteMaterial, MaterialID: "board-material"}}})
	if err == nil {
		t.Fatal("referenced material was deleted")
	}
	unused := Material{ID: "unused-material", Name: "Unused", Color: "#ffffff", Roughness: 0.5}
	_, added, err := workspace.Execute(Transaction{ID: "add-unused", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetMaterial, MaterialRecord: &unused}}})
	if err != nil {
		t.Fatal(err)
	}
	receipt, deleted, err := workspace.Execute(Transaction{ID: "delete-unused", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: added.Revision, Operations: []Operation{{Kind: OpDeleteMaterial, MaterialID: unused.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := deleted.Materials[unused.ID]; exists {
		t.Fatal("unused material remained")
	}
	if len(receipt.MaterialChanges) != 1 || receipt.MaterialChanges[0].Before == nil || receipt.MaterialChanges[0].After != nil {
		t.Fatalf("delete material receipt = %+v", receipt.MaterialChanges)
	}
}
