package studio

import (
	"fmt"
	"testing"
)

// Execute retains the outgoing document in history rather than a copy of it,
// which is only sound while nothing mutates a document in place. This walks a
// run of edits all the way back and checks every intermediate state, so an
// in-place mutation introduced later shows up as corrupted history instead of
// as a silent wrong answer.
func TestHistoryIsNotCorruptedByLaterEdits(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	revision := document.Revision

	const edits = 12
	for i := 0; i < edits; i++ {
		transform := TransformFromEuler(Vec3{X: float64(i + 1)}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
		receipt, _, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("alias-%d", i), Actor: "human://local-ui", Mode: ModeDirect,
			ExpectedRevision: revision,
			Operations:       []Operation{{Kind: OpSetTransform, Target: target, Transform: &transform}},
		})
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		revision = receipt.AfterRevision
	}

	// Undo back through every step. After undoing the edit that set X to
	// (i+1), the position must read i.
	for i := edits - 1; i >= 0; i-- {
		receipt, _, err := workspace.Undo(revision, "human://local-ui")
		if err != nil {
			t.Fatalf("undo of edit %d: %v", i, err)
		}
		revision = receipt.AfterRevision
		snapshot, err := workspace.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := snapshot.Entities[target].Transform.Position.X, float64(i); got != want {
			t.Fatalf("after undoing edit %d, position.x = %v, want %v", i, got, want)
		}
	}
}

// Receipts carry the fingerprint of the document a transaction started from
// and the one it produced. Those fingerprints are served from a cache on the
// workspace, so a stale cache would make every receipt misidentify the
// document it describes.
func TestReceiptFingerprintsMatchTheDocumentsTheyName(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	revision := document.Revision

	previous, err := document.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		transform := TransformFromEuler(Vec3{X: float64(i + 1)}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
		receipt, committed, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("fingerprint-%d", i), Actor: "agent://test", Mode: ModeDirect,
			ExpectedRevision: revision,
			Operations:       []Operation{{Kind: OpSetTransform, Target: target, Transform: &transform}},
		})
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		revision = receipt.AfterRevision

		if receipt.BeforeFingerprint != previous {
			t.Fatalf("edit %d: receipt names before %q, want %q", i, receipt.BeforeFingerprint, previous)
		}
		// The returned document is what the receipt claims to have produced.
		actual, err := committed.Fingerprint()
		if err != nil {
			t.Fatal(err)
		}
		if receipt.AfterFingerprint != actual {
			t.Fatalf("edit %d: receipt names after %q, committed document is %q", i, receipt.AfterFingerprint, actual)
		}
		previous = actual
	}

	// A proposal must not disturb the cache: it never becomes canonical.
	transform := TransformFromEuler(Vec3{X: 99}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
	proposal, preview, err := workspace.Execute(Transaction{
		ID: "fingerprint-proposal", Actor: "agent://test", Mode: ModePropose,
		ExpectedRevision: revision,
		Operations:       []Operation{{Kind: OpSetTransform, Target: target, Transform: &transform}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.BeforeFingerprint != previous {
		t.Fatalf("proposal names before %q, want %q", proposal.BeforeFingerprint, previous)
	}
	// The preview is the caller's to keep, and mutating it must not reach the
	// workspace: a proposal returns the working document without a copy.
	preview.Entities[target] = Entity{ID: target, Name: "mutated by the caller"}
	live, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if live.Entities[target].Name == "mutated by the caller" {
		t.Fatal("mutating a returned preview reached the canonical document")
	}
	current, err := live.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if current != previous {
		t.Fatalf("a proposal changed the canonical document: %q != %q", current, previous)
	}
}

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
