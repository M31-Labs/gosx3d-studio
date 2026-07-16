package studio

import "testing"

func TestWorkspaceRecentReceiptsNewestFirstAcrossExecuteAndUndo(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	transform := TransformFromEuler(Vec3{X: 1}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
	target, _ := FirstPickTarget(document)
	receipt, _, err := workspace.Execute(Transaction{ID: "history-1", Actor: "human://local-ui", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetTransform, Target: target, Transform: &transform}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := workspace.Undo(receipt.AfterRevision, "human://local-ui"); err != nil {
		t.Fatal(err)
	}
	receipts := workspace.RecentReceipts(10)
	if len(receipts) != 2 {
		t.Fatalf("want 2 receipts, got %d", len(receipts))
	}
	if receipts[0].TransactionID == "history-1" {
		t.Fatalf("receipts must be newest-first, got %q first", receipts[0].TransactionID)
	}
	if receipts[1].TransactionID != "history-1" || receipts[1].Actor != "human://local-ui" {
		t.Fatalf("history lost the original transaction: %+v", receipts[1])
	}
	if limited := workspace.RecentReceipts(1); len(limited) != 1 {
		t.Fatalf("limit must apply, got %d", len(limited))
	}
}
