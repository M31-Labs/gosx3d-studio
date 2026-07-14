package studio

import "testing"

func TestVoxelCSGBooleanProducesClosedDeterministicMeshes(t *testing.T) {
	for _, test := range []struct {
		operation     string
		sizeX, volume float64
	}{{"union", 1.5, 1.5}, {"intersection", 0.5, 0.5}, {"subtract", 0.5, 0.5}} {
		t.Run(test.operation, func(t *testing.T) {
			document := csgDocument(t)
			workspace, err := NewWorkspace(document)
			if err != nil {
				t.Fatal(err)
			}
			operation := Operation{Kind: OpCSGBoolean, Target: "editable", Operand: "operand", Boolean: test.operation, VoxelSize: 0.25, NewID: ID("result-" + test.operation)}
			previewReceipt, preview, err := workspace.Execute(Transaction{ID: "preview-" + test.operation, Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
			if err != nil {
				t.Fatal(err)
			}
			directReceipt, direct, err := workspace.Execute(Transaction{ID: "direct-" + test.operation, Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
			if err != nil {
				t.Fatal(err)
			}
			previewFingerprint, _ := preview.Fingerprint()
			directFingerprint, _ := direct.Fingerprint()
			if previewFingerprint != directFingerprint {
				t.Fatal("CSG preview and direct differ")
			}
			analysis, err := AnalyzeEntityGeometry(direct, operation.NewID)
			if err != nil {
				t.Fatal(err)
			}
			if !analysis.Valid || !analysis.Closed || !analysis.Manifold || analysis.Volume == nil || !near(analysis.Bounds.Size.X, test.sizeX) || !near(*analysis.Volume, test.volume) {
				t.Fatalf("CSG analysis = %+v", analysis)
			}
			if len(previewReceipt.OperatorRecords) != 1 || len(directReceipt.OperatorRecords) != 1 {
				t.Fatalf("CSG receipts preview=%+v direct=%+v", previewReceipt, directReceipt)
			}
			if _, err := Compile(direct); err != nil {
				t.Fatalf("compile CSG result: %v", err)
			}
			_, restored, err := workspace.Undo(direct.Revision, "agent://test")
			if err != nil {
				t.Fatal(err)
			}
			if _, exists := restored.Entities[operation.NewID]; exists {
				t.Fatal("CSG checkpoint undo retained result")
			}
		})
	}
}

func TestVoxelCSGRejectsOpenInputsAndExcessiveBudgets(t *testing.T) {
	// SceneDoc permits open modeling meshes; CSG itself is the honesty gate.
	document := csgDocument(t)
	entity := document.Entities["operand"]
	entity.Mesh.Geometry.Faces = entity.Mesh.Geometry.Faces[:1]
	document.Entities[entity.ID] = entity
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = workspace.Execute(Transaction{ID: "open-csg", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpCSGBoolean, Target: "editable", Operand: "operand", Boolean: "union", VoxelSize: 0.25, NewID: "bad"}}})
	if err == nil {
		t.Fatal("CSG accepted open operand")
	}
	document = csgDocument(t)
	workspace, err = NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = workspace.Execute(Transaction{ID: "budget-csg", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpCSGBoolean, Target: "editable", Operand: "operand", Boolean: "union", VoxelSize: 0.001, NewID: "too-large"}}})
	if err == nil {
		t.Fatal("CSG accepted excessive voxel budget")
	}
}

func csgDocument(t *testing.T) Document {
	t.Helper()
	document := operatorDocument(t)
	left := document.Entities["editable"]
	left.Mesh.Geometry = cubeGeometry()
	document.Entities[left.ID] = left
	right := left
	right.ID = "operand"
	right.Name = "Operand"
	right.Transform.Position = Vec3{X: 0.5}
	document.Entities[right.ID] = right
	root := document.Entities["scene-root"]
	root.Children = append(root.Children, right.ID)
	document.Entities[root.ID] = root
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	return document
}

func cubeGeometry() Geometry {
	return Geometry{
		Kind: "indexed-mesh",
		Vertices: []Vertex{
			{ID: "v000", Position: Vec3{}}, {ID: "v100", Position: Vec3{X: 1}},
			{ID: "v110", Position: Vec3{X: 1, Y: 1}}, {ID: "v010", Position: Vec3{Y: 1}},
			{ID: "v001", Position: Vec3{Z: 1}}, {ID: "v101", Position: Vec3{X: 1, Z: 1}},
			{ID: "v111", Position: Vec3{X: 1, Y: 1, Z: 1}}, {ID: "v011", Position: Vec3{Y: 1, Z: 1}},
		},
		Faces: []Face{
			{ID: "nx", Vertices: []ID{"v000", "v001", "v011", "v010"}},
			{ID: "px", Vertices: []ID{"v100", "v110", "v111", "v101"}},
			{ID: "ny", Vertices: []ID{"v000", "v100", "v101", "v001"}},
			{ID: "py", Vertices: []ID{"v010", "v011", "v111", "v110"}},
			{ID: "nz", Vertices: []ID{"v000", "v010", "v110", "v100"}},
			{ID: "pz", Vertices: []ID{"v001", "v101", "v111", "v011"}},
		},
	}
}
