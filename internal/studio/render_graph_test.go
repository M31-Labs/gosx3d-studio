package studio

import (
	"errors"
	"reflect"
	"testing"
)

func TestRenderGraphSchedulesValidatesAndAliasesDeterministically(t *testing.T) {
	graph := RenderGraph{
		Resources: map[ID]RenderResource{
			"swap": {ID: "swap", Kind: "texture", Ownership: "imported", Format: "rgba8"},
			"a":    {ID: "a", Kind: "render-target", Ownership: "transient", Format: "rgba16f", Width: 640, Height: 360},
			"b":    {ID: "b", Kind: "render-target", Ownership: "transient", Format: "rgba16f", Width: 640, Height: 360},
		},
		Passes: map[ID]RenderPass{
			"scene":   {ID: "scene", Kind: "scene", Writes: []ID{"a"}},
			"tone":    {ID: "tone", Kind: "fullscreen", Reads: []ID{"a"}, Writes: []ID{"swap"}, Depends: []ID{"scene"}},
			"overlay": {ID: "overlay", Kind: "scene", Writes: []ID{"b"}, Depends: []ID{"tone"}},
		},
	}
	first, err := CompileRenderGraph(graph)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileRenderGraph(graph)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("render graph plan is not deterministic")
	}
	if len(first.Passes) != 3 || first.Passes[0].ID != "scene" || first.Passes[2].ID != "overlay" {
		t.Fatalf("unexpected schedule: %#v", first.Passes)
	}
	if len(first.Allocations) != 2 || first.Allocations[0].Slot != first.Allocations[1].Slot {
		t.Fatalf("non-overlapping transient resources were not aliased: %#v", first.Allocations)
	}

	document := SampleDocument()
	document.RenderGraph = &graph
	ir, err := CompileIR(document)
	if err != nil {
		t.Fatal(err)
	}
	if ir.RenderGraph == nil || !reflect.DeepEqual(*ir.RenderGraph, first) {
		t.Fatal("shared SceneIR did not preserve render graph plan")
	}

	graph.Passes["tone"] = RenderPass{ID: "tone", Kind: "fullscreen", Reads: []ID{"b"}, Depends: []ID{"scene"}}
	if _, err := CompileRenderGraph(graph); err == nil {
		t.Fatal("expected transient read-before-write diagnostic")
	}
}

func TestSetRenderGraphUsesRevisionSafeCommandPath(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	graph := RenderGraph{Resources: map[ID]RenderResource{"color": {ID: "color", Kind: "texture", Ownership: "transient"}}, Passes: map[ID]RenderPass{"draw": {ID: "draw", Kind: "scene", Writes: []ID{"color"}}}}
	receipt, preview, err := workspace.Execute(Transaction{ID: "graph", Actor: "agent", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetRenderGraph, RenderGraph: &graph}}})
	if err != nil {
		t.Fatal(err)
	}
	if preview.RenderGraph == nil || receipt.AfterRevision != document.Revision+1 {
		t.Fatal("render graph command did not commit")
	}
	if _, _, err := workspace.Execute(Transaction{ID: "stale", Actor: "agent", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetRenderGraph, RenderGraph: &graph}}}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale graph update error=%v", err)
	}
}
