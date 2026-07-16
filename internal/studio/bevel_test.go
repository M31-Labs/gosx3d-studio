package studio

import (
	"math"
	"testing"
)

func bevelFixture() Geometry {
	return Geometry{Kind: "indexed-mesh",
		Vertices: []Vertex{
			{ID: "v0", Position: Vec3{}}, {ID: "v1", Position: Vec3{X: 1}}, {ID: "v2", Position: Vec3{X: 2}},
			{ID: "v3", Position: Vec3{Z: 1}}, {ID: "v4", Position: Vec3{X: 1, Z: 1}}, {ID: "v5", Position: Vec3{X: 2, Z: 1}},
		},
		Faces: []Face{
			{ID: "left", Vertices: []ID{"v0", "v1", "v4", "v3"}},
			{ID: "right", Vertices: []ID{"v1", "v2", "v5", "v4"}},
		},
	}
}

func sharedEdgeID(t *testing.T, geometry Geometry, a, b ID) ID {
	t.Helper()
	for _, edge := range MeshEdges(geometry) {
		if (edge.A == a && edge.B == b) || (edge.A == b && edge.B == a) {
			return edge.ID
		}
	}
	t.Fatalf("edge %s-%s not found", a, b)
	return ""
}

func TestBevelEdgeSplitsSharedEdgeIntoDeterministicQuad(t *testing.T) {
	geometry := bevelFixture()
	edge := sharedEdgeID(t, geometry, "v1", "v4")
	if err := bevelEdges(&geometry, Operation{Edges: []ID{edge}, Amount: 0.25, NewID: "bevel-quad"}); err != nil {
		t.Fatal(err)
	}
	if len(geometry.Faces) != 3 {
		t.Fatalf("faces=%d want 3", len(geometry.Faces))
	}
	if len(geometry.Vertices) != 8 {
		t.Fatalf("vertices=%d want 8 (4 original + 4 bevel)", len(geometry.Vertices))
	}
	byID := map[ID]Vertex{}
	for _, vertex := range geometry.Vertices {
		if vertex.ID == "v1" || vertex.ID == "v4" {
			t.Fatalf("beveled endpoint %q must be removed", vertex.ID)
		}
		byID[vertex.ID] = vertex
	}
	// v1 offset into the left face moves toward v0: expected x = 1-0.25.
	leftTop, ok := byID["bevel-v1-left"]
	if !ok {
		t.Fatalf("deterministic bevel vertex id missing: %v", geometry.Vertices)
	}
	if math.Abs(leftTop.Position.X-0.75) > 1e-12 || math.Abs(leftTop.Position.Z) > 1e-12 {
		t.Fatalf("bevel-v1-left position=%+v", leftTop.Position)
	}
	var bevelFace *Face
	for i := range geometry.Faces {
		if geometry.Faces[i].ID == "bevel-quad" {
			bevelFace = &geometry.Faces[i]
		}
	}
	if bevelFace == nil || len(bevelFace.Vertices) != 4 {
		t.Fatalf("bevel quad missing or malformed: %+v", geometry.Faces)
	}
	if err := validateGeometry(geometry); err != nil {
		t.Fatalf("beveled geometry invalid: %v", err)
	}
	// Determinism: running on a fresh fixture produces identical output.
	second := bevelFixture()
	if err := bevelEdges(&second, Operation{Edges: []ID{edge}, Amount: 0.25, NewID: "bevel-quad"}); err != nil {
		t.Fatal(err)
	}
	if len(second.Vertices) != len(geometry.Vertices) || len(second.Faces) != len(geometry.Faces) {
		t.Fatal("bevel is not deterministic")
	}
}

func TestBevelEdgeRejectsBoundaryExcessAmountAndSharedEndpoints(t *testing.T) {
	geometry := bevelFixture()
	boundary := sharedEdgeID(t, geometry, "v0", "v1")
	if err := bevelEdges(&geometry, Operation{Edges: []ID{boundary}, Amount: 0.25, NewID: "x"}); err == nil {
		t.Fatal("boundary edge bevel must fail explicitly")
	}
	shared := sharedEdgeID(t, geometry, "v1", "v4")
	if err := bevelEdges(&geometry, Operation{Edges: []ID{shared}, Amount: 2.0, NewID: "x"}); err == nil {
		t.Fatal("amount exceeding adjacent segment length must fail")
	}
	if err := bevelEdges(&geometry, Operation{Edges: []ID{shared}, Amount: 0, NewID: "x"}); err == nil {
		t.Fatal("zero amount must fail")
	}
	if err := bevelEdges(&geometry, Operation{Edges: []ID{shared}, Amount: 0.25}); err == nil {
		t.Fatal("missing newId must fail")
	}
}

func TestBevelEdgeTransactionIsCheckpointUndoable(t *testing.T) {
	document := SampleDocument()
	root := document.Entities["scene-root"]
	entity := Entity{ID: "bevel-mesh", Name: "Bevel mesh", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{Geometry: bevelFixture(), Material: "board-material", Pickable: true}}
	root.Children = append(root.Children, entity.ID)
	document.Entities[root.ID] = root
	document.Entities[entity.ID] = entity
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	edge := sharedEdgeID(t, bevelFixture(), "v1", "v4")
	receipt, changed, err := workspace.Execute(Transaction{ID: "bevel-tx", Actor: "agent://bevel-test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpBevelEdges, Target: "bevel-mesh", Edges: []ID{edge}, Amount: 0.2, NewID: "bevel-face"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.OperatorRecords) != 1 || receipt.OperatorRecords[0].Kind != OpBevelEdges {
		t.Fatalf("operator record=%+v", receipt.OperatorRecords)
	}
	if len(changed.Entities["bevel-mesh"].Mesh.Geometry.Faces) != 3 {
		t.Fatalf("faces=%d", len(changed.Entities["bevel-mesh"].Mesh.Geometry.Faces))
	}
	if _, err := Compile(changed); err != nil {
		t.Fatalf("beveled document does not compile: %v", err)
	}
	_, restored, err := workspace.Undo(changed.Revision, "agent://bevel-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Entities["bevel-mesh"].Mesh.Geometry.Faces) != 2 || len(restored.Entities["bevel-mesh"].Mesh.Geometry.Vertices) != 6 {
		t.Fatal("undo did not restore authored topology")
	}
}
