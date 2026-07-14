package studio

import (
	"errors"
	"testing"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx/scene/harness"
	"m31labs.dev/gosx/scene/preview"
)

func TestSampleDocumentCompilesToSharedSceneIR(t *testing.T) {
	document := SampleDocument()
	if err := document.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	ir := props.SceneIR()
	if len(ir.Objects) != 7 {
		t.Fatalf("objects = %d, want 7", len(ir.Objects))
	}
	if len(ir.Lights) != 2 {
		t.Fatalf("lights = %d, want 2", len(ir.Lights))
	}
	portableIR, err := CompileIR(document)
	if err != nil {
		t.Fatalf("compile portable IR: %v", err)
	}
	if portableIR.Schema != scene.SceneIRSchema {
		t.Fatalf("portable schema = %q, want %q", portableIR.Schema, scene.SceneIRSchema)
	}
	trace := scene.TraceGraph(props.Graph, scene.Ray{Origin: scene.Vec3(0, 4, 0), Direction: scene.Vec3(0, -1, 0)}, scene.PickableOnly())
	if trace.Closest == nil {
		t.Fatal("center ray missed sample document")
	}
	if trace.Closest.ID != "piece-jade-01" {
		t.Fatalf("closest id = %q, want piece-jade-01", trace.Closest.ID)
	}
	if trace.PrimitivesTested == 0 || trace.NodesVisited == 0 {
		t.Fatalf("trace telemetry = %+v", trace)
	}
}

func TestSampleDocumentProducesVisibleNativeEvidence(t *testing.T) {
	document := SampleDocument()
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	session := harness.New(props, preview.Options{Width: 320, Height: 200, DisableShadows: true, DisablePostFX: true, MaxSegments: 16})
	if _, err := session.Render(0); err != nil {
		t.Fatal(err)
	}
	report := session.Report()
	if len(report.Events) != 1 || report.Events[0].Frame == nil {
		t.Fatalf("frame evidence = %+v", report.Events)
	}
	if report.Events[0].Frame.Coverage <= 0.001 || report.Events[0].Frame.UniqueColors < 4 {
		t.Fatalf("blank native evidence: %+v", *report.Events[0].Frame)
	}
	if err := session.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceProposalDirectAndUndo(t *testing.T) {
	workspace, err := NewWorkspace(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	transaction := Transaction{ID: "tx-material", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: 1, Operations: []Operation{{Kind: OpAssignMaterial, Target: "piece-jade-01", Material: "gold-material"}}}
	receipt, preview, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Applied || preview.Revision != 2 || preview.Entities["piece-jade-01"].Mesh.Material != "gold-material" {
		t.Fatalf("proposal receipt=%+v preview=%+v", receipt, preview.Entities["piece-jade-01"])
	}
	snapshot, _ := workspace.Snapshot()
	if snapshot.Revision != 1 || snapshot.Entities["piece-jade-01"].Mesh.Material != "jade-material" {
		t.Fatal("proposal mutated workspace")
	}
	transaction.Mode = ModeDirect
	receipt, _, err = workspace.Execute(transaction)
	if err != nil || !receipt.Applied {
		t.Fatalf("direct receipt=%+v err=%v", receipt, err)
	}
	if _, _, err := workspace.Execute(transaction); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale transaction error = %v", err)
	}
	_, restored, err := workspace.Undo(2, "identity://test")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Revision != 3 || restored.Entities["piece-jade-01"].Mesh.Material != "jade-material" {
		t.Fatalf("restored revision=%d material=%q", restored.Revision, restored.Entities["piece-jade-01"].Mesh.Material)
	}
}

func TestValidateRejectsCycles(t *testing.T) {
	document := SampleDocument()
	root := document.Entities["scene-root"]
	piece := document.Entities["piece-jade-01"]
	root.Parent = piece.ID
	piece.Children = append(piece.Children, root.ID)
	document.Entities[root.ID] = root
	document.Entities[piece.ID] = piece
	if err := document.Validate(); err == nil {
		t.Fatal("expected invalid root/cycle")
	}
}
