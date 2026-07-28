package studio

import (
	"fmt"
	"testing"
)

// Each undo entry retains the whole document before and after its
// transaction. Unbounded, a long session on a large scene consumed gigabytes,
// so the stack is capped and the depth is observable rather than silent.
func TestUndoHistoryIsBoundedAndKeepsTheNewestSteps(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	revision := document.Revision
	edits := undoHistoryLimit + 12
	for i := 0; i < edits; i++ {
		transform := TransformFromEuler(Vec3{X: float64(i)}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
		receipt, _, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("bound-%d", i), Actor: "human://local-ui", Mode: ModeDirect,
			ExpectedRevision: revision,
			Operations:       []Operation{{Kind: OpSetTransform, Target: target, Transform: &transform}},
		})
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		revision = receipt.AfterRevision
	}

	status := workspace.ProjectStatus()
	if status.UndoDepth != undoHistoryLimit || status.UndoLimit != undoHistoryLimit {
		t.Fatalf("undo depth = %d, limit = %d, want both %d", status.UndoDepth, status.UndoLimit, undoHistoryLimit)
	}

	// The newest steps must survive: undoing the whole retained stack has to
	// walk back through the most recent transactions, not the dropped ones.
	for i := 0; i < undoHistoryLimit; i++ {
		receipt, _, err := workspace.Undo(revision, "human://local-ui")
		if err != nil {
			t.Fatalf("undo %d of %d: %v", i+1, undoHistoryLimit, err)
		}
		revision = receipt.AfterRevision
	}
	if _, _, err := workspace.Undo(revision, "human://local-ui"); err == nil {
		t.Fatal("undo past the retained history was accepted")
	}
	final, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	// Undoing every retained step lands on the state before edit
	// (edits - undoHistoryLimit), which is the oldest edit still retained.
	if got, want := final.Entities[target].Transform.Position.X, float64(edits-undoHistoryLimit-1); got != want {
		t.Fatalf("position after unwinding retained history = %v, want %v", got, want)
	}
}

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
