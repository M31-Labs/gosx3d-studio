package studio

import (
	"math"
	"testing"

	"m31labs.dev/gosx/scene"
)

func TestIndexedMeshOperatorsAreDeterministicAndCheckpointSafe(t *testing.T) {
	base := operatorDocument(t)
	before, err := base.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	operations := []Operation{
		{Kind: OpInsetFaces, Target: "editable", Faces: []ID{"quad"}, Amount: 0.25},
		{Kind: OpTriangulateFaces, Target: "editable", Faces: []ID{"quad"}},
		{Kind: OpRecalculateNormals, Target: "editable"},
	}
	previewWorkspace, err := NewWorkspace(base)
	if err != nil {
		t.Fatal(err)
	}
	receipt, preview, err := previewWorkspace.Execute(Transaction{ID: "preview-topology", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: base.Revision, Operations: operations})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Applied || preview.Revision != base.Revision+1 {
		t.Fatalf("preview receipt applied=%v revision=%d", receipt.Applied, preview.Revision)
	}
	if len(receipt.OperatorRecords) != len(operations) || receipt.OperatorRecords[0].SelectionMode != SelectionFace || receipt.OperatorRecords[0].CoordinateSpace != "object" || receipt.OperatorRecords[0].UndoPolicy != "geometry-checkpoint" {
		t.Fatalf("normalized operator records = %+v", receipt.OperatorRecords)
	}
	unchanged, _ := previewWorkspace.Snapshot()
	unchangedFingerprint, _ := unchanged.Fingerprint()
	if unchangedFingerprint != before {
		t.Fatal("preview mutated workspace")
	}

	directWorkspace, err := NewWorkspace(base)
	if err != nil {
		t.Fatal(err)
	}
	_, applied, err := directWorkspace.Execute(Transaction{ID: "direct-topology", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: base.Revision, Operations: operations})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(applied.Entities["editable"].Mesh.Geometry.Faces); got != 6 {
		t.Fatalf("inset plus triangulate faces = %d, want 6", got)
	}
	for _, vertex := range applied.Entities["editable"].Mesh.Geometry.Vertices {
		if vertex.Normal == (Vec3{}) {
			t.Fatalf("vertex %q has no recalculated normal", vertex.ID)
		}
	}
	appliedFingerprint, _ := applied.Fingerprint()
	if fingerprint, _ := preview.Fingerprint(); fingerprint != appliedFingerprint {
		t.Fatal("preview and direct results differ")
	}
	_, restored, err := directWorkspace.Undo(applied.Revision, "agent://test")
	if err != nil {
		t.Fatal(err)
	}
	restored.Revision = base.Revision
	restoredFingerprint, _ := restored.Fingerprint()
	if restoredFingerprint != before {
		t.Fatal("undo did not restore exact topology checkpoint")
	}
}

func TestPlanarUVProjectionIsTypedDeterministicAndUndoable(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	receipt, projected, err := workspace.Execute(Transaction{ID: "project-uv", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpProjectPlanarUV, Target: "editable", Projection: "xz"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.OperatorRecords) != 1 || receipt.OperatorRecords[0].Projection != "xz" || receipt.OperatorRecords[0].UndoPolicy != "geometry-checkpoint" {
		t.Fatalf("uv operator record = %+v", receipt.OperatorRecords)
	}
	geometry := projected.Entities["editable"].Mesh.Geometry
	for _, vertex := range geometry.Vertices {
		if vertex.UV == nil || vertex.UV.X < 0 || vertex.UV.X > 1 || vertex.UV.Y < 0 || vertex.UV.Y > 1 {
			t.Fatalf("vertex %q uv = %+v", vertex.ID, vertex.UV)
		}
	}
	compiled, err := compileIndexedGeometry(geometry)
	if err != nil {
		t.Fatal(err)
	}
	buffer := compiled.(scene.BufferGeometry)
	if len(buffer.UVs) != len(geometry.Vertices)*2 {
		t.Fatalf("compiled uv floats = %d", len(buffer.UVs))
	}
	analysis := analyzeGeometry(geometry)
	if !analysis.UV.Complete || analysis.UV.MappedVertices != 4 || analysis.UV.DegenerateFaces != 0 || analysis.UV.Bounds == nil || analysis.UV.Bounds.Min != (Vec2{}) || analysis.UV.Bounds.Max != (Vec2{X: 1, Y: 1}) {
		t.Fatalf("uv analysis = %+v", analysis.UV)
	}
	_, restored, err := workspace.Undo(projected.Revision, "agent://test")
	if err != nil {
		t.Fatal(err)
	}
	for _, vertex := range restored.Entities["editable"].Mesh.Geometry.Vertices {
		if vertex.UV != nil {
			t.Fatalf("undo retained uv on %q", vertex.ID)
		}
	}
}

func TestDissolveSharedEdgeMergesAdjacentFaces(t *testing.T) {
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Geometry.Faces = []Face{{ID: "left", Vertices: []ID{"a", "b", "c"}}, {ID: "right", Vertices: []ID{"a", "c", "d"}}}
	document.Entities[entity.ID] = entity
	var diagonal ID
	for _, edge := range MeshEdges(entity.Mesh.Geometry) {
		if edge.A == "a" && edge.B == "c" {
			diagonal = edge.ID
		}
	}
	if diagonal == "" {
		t.Fatal("shared diagonal edge was not indexed")
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	receipt, dissolved, err := workspace.Execute(Transaction{ID: "dissolve", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpDissolveEdges, Target: "editable", Edges: []ID{diagonal}}}})
	if err != nil {
		t.Fatal(err)
	}
	faces := dissolved.Entities["editable"].Mesh.Geometry.Faces
	if len(faces) != 1 || faces[0].ID != "left" || len(faces[0].Vertices) != 4 {
		t.Fatalf("dissolved faces = %+v", faces)
	}
	if receipt.OperatorRecords[0].SelectionMode != SelectionEdge || receipt.OperatorRecords[0].Selection[0] != diagonal {
		t.Fatalf("dissolve record = %+v", receipt.OperatorRecords[0])
	}
}

func TestBridgeClosedLoopsCreatesStableQuadStrip(t *testing.T) {
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Geometry = Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
		{ID: "a0", Position: Vec3{}}, {ID: "a1", Position: Vec3{X: 1}}, {ID: "a2", Position: Vec3{Z: 1}},
		{ID: "b0", Position: Vec3{Y: 1}}, {ID: "b1", Position: Vec3{X: 1, Y: 1}}, {ID: "b2", Position: Vec3{Y: 1, Z: 1}},
	}, Faces: []Face{{ID: "cap-a", Vertices: []ID{"a0", "a2", "a1"}}, {ID: "cap-b", Vertices: []ID{"b0", "b1", "b2"}}}}
	document.Entities[entity.ID] = entity
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{Kind: OpBridgeLoops, Target: "editable", Loops: [][]ID{{"a0", "a1", "a2"}, {"b0", "b1", "b2"}}, NewID: "bridge", Closed: true}
	_, bridged, err := workspace.Execute(Transaction{ID: "bridge", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	faces := bridged.Entities["editable"].Mesh.Geometry.Faces
	if len(faces) != 5 || faces[2].ID != "bridge--0" || faces[4].ID != "bridge--2" {
		t.Fatalf("bridge faces = %+v", faces)
	}
	if _, err := Compile(bridged); err != nil {
		t.Fatalf("compile bridged SceneDoc: %v", err)
	}
}

func TestLoopCutTraversesQuadStripAndReusesSharedCutVertex(t *testing.T) {
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Geometry = Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
		{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}}, {ID: "c", Position: Vec3{X: 2}},
		{ID: "d", Position: Vec3{Z: 1}}, {ID: "e", Position: Vec3{X: 1, Z: 1}}, {ID: "f", Position: Vec3{X: 2, Z: 1}},
	}, Faces: []Face{{ID: "left", Vertices: []ID{"a", "b", "e", "d"}}, {ID: "right", Vertices: []ID{"b", "c", "f", "e"}}}}
	document.Entities[entity.ID] = entity
	var seed ID
	for _, edge := range MeshEdges(entity.Mesh.Geometry) {
		if edge.A == "b" && edge.B == "e" {
			seed = edge.ID
		}
	}
	if seed == "" {
		t.Fatal("quad-strip seed edge missing")
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, cut, err := workspace.Execute(Transaction{ID: "loop-cut", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpLoopCut, Target: "editable", Edges: []ID{seed}, Amount: 0.5, NewID: "center"}}})
	if err != nil {
		t.Fatal(err)
	}
	geometry := cut.Entities["editable"].Mesh.Geometry
	if len(geometry.Vertices) != 9 || len(geometry.Faces) != 4 {
		t.Fatalf("loop-cut topology vertices=%d faces=%d", len(geometry.Vertices), len(geometry.Faces))
	}
	shared := 0
	for _, vertex := range geometry.Vertices {
		if math.Abs(vertex.Position.X-1) < 1e-9 && math.Abs(vertex.Position.Z-0.5) < 1e-9 {
			shared++
		}
	}
	if shared != 1 {
		t.Fatalf("shared crossed edge generated %d midpoint vertices", shared)
	}
	if _, err := Compile(cut); err != nil {
		t.Fatalf("compile loop-cut SceneDoc: %v", err)
	}
}

func TestFillAndWeldUseStableIDs(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, filled, err := workspace.Execute(Transaction{ID: "fill", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpFillFace, Target: "editable", Vertices: []ID{"a", "d", "c"}, NewID: "diagonal-fill"}}})
	if err != nil {
		t.Fatal(err)
	}
	if filled.Entities["editable"].Mesh.Geometry.Faces[1].ID != "diagonal-fill" {
		t.Fatal("fill did not preserve requested stable face id")
	}
	_, welded, err := workspace.Execute(Transaction{ID: "weld", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: filled.Revision, Operations: []Operation{{Kind: OpWeldVertices, Target: "editable", Vertices: []ID{"b", "a"}, Tolerance: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	geometry := welded.Entities["editable"].Mesh.Geometry
	if len(geometry.Vertices) != 3 || geometry.Vertices[0].ID != "a" || math.Abs(geometry.Vertices[0].Position.X-0.25) > 1e-9 {
		t.Fatalf("unexpected deterministic weld result: %+v", geometry.Vertices)
	}
}

func TestMeshOperatorsRejectUnknownSubobjects(t *testing.T) {
	document := operatorDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = workspace.Execute(Transaction{ID: "bad-face", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpExtrudeFaces, Target: "editable", Faces: []ID{"missing"}, Distance: 1}}})
	if err == nil {
		t.Fatal("extrude accepted an unknown stable face id")
	}
}

func operatorDocument(t *testing.T) Document {
	t.Helper()
	document := SampleDocument()
	entity := Entity{
		ID: "editable", Name: "Editable", Parent: "scene-root", Transform: IdentityTransform(), Visible: true,
		Mesh: &MeshComponent{
			Material: "board-material", Pickable: true,
			Geometry: Geometry{
				Kind: "indexed-mesh",
				Vertices: []Vertex{
					{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 0.5}},
					{ID: "c", Position: Vec3{X: 0.5, Z: 0.5}}, {ID: "d", Position: Vec3{Z: 0.5}},
				},
				Faces: []Face{{ID: "quad", Vertices: []ID{"a", "b", "c", "d"}}},
			},
		},
	}
	root := document.Entities["scene-root"]
	root.Children = append(root.Children, entity.ID)
	document.Entities[root.ID] = root
	document.Entities[entity.ID] = entity
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	return document
}
