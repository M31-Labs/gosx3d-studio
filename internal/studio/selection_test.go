package studio

import "testing"

func TestSubobjectSelectionSupportsStableVertexEdgeAndFaceIDs(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	edges := MeshEdges(document.Entities["editable"].Mesh.Geometry)
	if len(edges) != 4 {
		t.Fatalf("edges = %d, want 4", len(edges))
	}
	if again := MeshEdges(document.Entities["editable"].Mesh.Geometry); again[0] != edges[0] {
		t.Fatalf("edge id is not deterministic: %+v != %+v", again[0], edges[0])
	}
	requests := []SelectionRequest{
		{ExpectedRevision: document.Revision, Mode: SelectionVertex, Object: "editable", IDs: []ID{"a", "c"}},
		{ExpectedRevision: document.Revision, Mode: SelectionEdge, Object: "editable", IDs: []ID{edges[0].ID}},
		{ExpectedRevision: document.Revision, Mode: SelectionFace, Object: "editable", IDs: []ID{"quad"}},
	}
	for _, request := range requests {
		if err := workspace.SelectSubobjects(request); err != nil {
			t.Fatalf("select %s: %v", request.Mode, err)
		}
		state := workspace.SelectionState()
		if state.Mode != request.Mode || state.Object != request.Object || len(state.IDs) != len(request.IDs) {
			t.Fatalf("selection state = %+v, request = %+v", state, request)
		}
		if selected := workspace.Selection(); len(selected) != 1 || selected[0] != "editable" {
			t.Fatalf("legacy object selection = %v", selected)
		}
	}
}

func TestSubobjectSelectionIsRevisionSafeAndReconciled(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.SelectSubobjects(SelectionRequest{ExpectedRevision: document.Revision + 1, Mode: SelectionFace, Object: "editable", IDs: []ID{"quad"}}); err == nil {
		t.Fatal("stale selection revision was accepted")
	}
	if err := workspace.SelectSubobjects(SelectionRequest{ExpectedRevision: document.Revision, Mode: SelectionFace, Object: "editable", IDs: []ID{"missing"}}); err == nil {
		t.Fatal("unknown stable face id was accepted")
	}
	if err := workspace.SelectSubobjects(SelectionRequest{ExpectedRevision: document.Revision, Mode: SelectionFace, Object: "editable", IDs: []ID{"quad"}}); err != nil {
		t.Fatal(err)
	}
	_, after, err := workspace.Execute(Transaction{ID: "extrude-selected", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpExtrudeFaces, Target: "editable", Faces: []ID{"quad"}, Distance: 0.5}}})
	if err != nil {
		t.Fatal(err)
	}
	state := workspace.SelectionState()
	if state.Revision != after.Revision || state.Mode != SelectionObject || len(state.IDs) != 1 || state.IDs[0] != "editable" {
		t.Fatalf("removed face selection did not safely downgrade to object: %+v", state)
	}
}
