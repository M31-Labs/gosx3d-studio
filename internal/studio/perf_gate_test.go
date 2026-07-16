package studio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The spec's editor budgets (§19) target selection feedback under 16ms,
// inspector-edit-to-viewport under 50ms, and undo under 100ms at production
// scene sizes. This gate tracks the document-operation costs on a synthetic
// 1,000-entity scene so O(document) regressions surface as numbers instead of
// anecdotes. Budgets live in perf-budget.json under documentOperations and are
// tripwires against regression, not proof the spec targets are met.

type documentOperationBudgets struct {
	EntityCount        int     `json:"entityCount"`
	SetTransformMs     float64 `json:"setTransformDirectMs"`
	UndoMs             float64 `json:"undoMs"`
	CompileFullMs      float64 `json:"compileFullMs"`
	ExactPickMs        float64 `json:"exactPickMs"`
	FingerprintMs      float64 `json:"fingerprintMs"`
	ValidateMs         float64 `json:"validateMs"`
	IterationsForGate  int     `json:"iterationsForGate"`
	SpecSelectionMs    float64 `json:"specSelectionFeedbackMs"`
	SpecInspectorMs    float64 `json:"specInspectorEditMs"`
	SpecUndoMs         float64 `json:"specUndoMs"`
	NotesOnAspirations string  `json:"notes"`
}

func loadDocumentOperationBudgets(t *testing.T) documentOperationBudgets {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "perf-budget.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		DocumentOperations *documentOperationBudgets `json:"documentOperations"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.DocumentOperations == nil {
		t.Fatal("perf-budget.json must declare documentOperations budgets")
	}
	return *manifest.DocumentOperations
}

func syntheticPerfDocument(t *testing.T, entityCount int) Document {
	t.Helper()
	document := SampleDocument()
	root := document.Entities["scene-root"]
	for i := 0; i < entityCount; i++ {
		id := ID(fmt.Sprintf("perf-box-%04d", i))
		entity := meshEntity(id, fmt.Sprintf("Perf box %d", i), Vec3{X: float64(i % 32), Y: float64(i / 32), Z: -2}, Geometry{Kind: "box", Width: 0.5, Height: 0.5, Depth: 0.5}, "board-material", true)
		entity.Parent = root.ID
		root.Children = append(root.Children, id)
		document.Entities[id] = entity
	}
	document.Entities[root.ID] = root
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	return document
}

func bestDuration(iterations int, run func() error) (time.Duration, error) {
	best := time.Duration(0)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		if err := run(); err != nil {
			return 0, err
		}
		elapsed := time.Since(start)
		if best == 0 || elapsed < best {
			best = elapsed
		}
	}
	return best, nil
}

func gateDuration(t *testing.T, name string, got time.Duration, budgetMs float64) {
	t.Helper()
	t.Logf("%s: %.2fms (budget %.0fms)", name, float64(got.Microseconds())/1000, budgetMs)
	if float64(got.Milliseconds()) > budgetMs {
		t.Errorf("%s took %v, budget is %.0fms", name, got, budgetMs)
	}
}

func TestDocumentOperationBudgetsAtScale(t *testing.T) {
	if testing.Short() {
		t.Skip("perf gate skipped in short mode")
	}
	budgets := loadDocumentOperationBudgets(t)
	if budgets.EntityCount <= 0 || budgets.IterationsForGate <= 0 {
		t.Fatal("documentOperations budgets must declare entityCount and iterationsForGate")
	}
	document := syntheticPerfDocument(t, budgets.EntityCount)

	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	revision := snapshot.Revision

	transform := TransformFromEuler(Vec3{X: 1, Y: 2, Z: 3}, Vec3{Z: 0.3}, Vec3{X: 1, Y: 1, Z: 1})
	executeDuration, err := bestDuration(budgets.IterationsForGate, func() error {
		receipt, _, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("perf-set-transform-%d", revision), Actor: "agent://perf-gate", Mode: ModeDirect,
			ExpectedRevision: revision,
			Operations:       []Operation{{Kind: OpSetTransform, Target: "perf-box-0000", Transform: &transform}},
		})
		if err != nil {
			return err
		}
		revision = receipt.AfterRevision
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gateDuration(t, "set-transform direct transaction", executeDuration, budgets.SetTransformMs)

	undoDuration, err := bestDuration(1, func() error {
		receipt, _, err := workspace.Undo(revision, "agent://perf-gate")
		if err != nil {
			return err
		}
		if !receipt.Applied {
			return fmt.Errorf("undo was not applied")
		}
		revision = receipt.AfterRevision
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gateDuration(t, "undo", undoDuration, budgets.UndoMs)

	compileDuration, err := bestDuration(budgets.IterationsForGate, func() error {
		_, err := Compile(document)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	gateDuration(t, "full compile", compileDuration, budgets.CompileFullMs)

	pickDuration, err := bestDuration(budgets.IterationsForGate, func() error {
		_, err := ExactPick(document, PickRequest{Origin: Vec3{Y: 6, Z: 6}, Direction: Vec3{Y: -0.707, Z: -0.707}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	gateDuration(t, "exact pick (includes full recompile today)", pickDuration, budgets.ExactPickMs)

	fingerprintDuration, err := bestDuration(budgets.IterationsForGate, func() error {
		_, err := document.Fingerprint()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	gateDuration(t, "document fingerprint", fingerprintDuration, budgets.FingerprintMs)

	validateDuration, err := bestDuration(budgets.IterationsForGate, func() error {
		return document.Validate()
	})
	if err != nil {
		t.Fatal(err)
	}
	gateDuration(t, "document validate", validateDuration, budgets.ValidateMs)
}
