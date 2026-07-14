package studio

import (
	"testing"

	"m31labs.dev/gosx/scene"
)

func TestNURBSCurveValidatesEvaluatesAndCompilesDeterministically(t *testing.T) {
	document := curveDocument(t)
	curve := document.Entities["editable"].Mesh.Geometry.Curve
	if start := evaluateNURBS(*curve, 0); start != (Vec3{}) {
		t.Fatalf("curve start = %+v", start)
	}
	if end := evaluateNURBS(*curve, 1); end != (Vec3{X: 2}) {
		t.Fatalf("curve end = %+v", end)
	}
	compiled, err := compileNURBSCurve(curve)
	if err != nil {
		t.Fatal(err)
	}
	buffer := compiled.(scene.BufferGeometry)
	if len(buffer.Positions) != 9*6*3 || len(buffer.Normals) != len(buffer.Positions) || len(buffer.UVs) != 9*6*2 || len(buffer.Indices) != 8*6*6 {
		t.Fatalf("curve buffers positions=%d normals=%d uvs=%d indices=%d", len(buffer.Positions), len(buffer.Normals), len(buffer.UVs), len(buffer.Indices))
	}
	if _, err := Compile(document); err != nil {
		t.Fatalf("compile curve SceneDoc: %v", err)
	}
	analysis, err := AnalyzeEntityCurve(document, "editable")
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Valid || analysis.ControlPoints != 3 || analysis.Degree != 2 || analysis.TessellatedVertices != 54 || analysis.Triangles != 96 || analysis.ApproximateLength <= 2 {
		t.Fatalf("curve analysis = %+v", analysis)
	}
}

func TestCurveControlPointSelectionAndActionShareRevisionSafePath(t *testing.T) {
	document := curveDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.SelectSubobjects(SelectionRequest{ExpectedRevision: document.Revision, Mode: SelectionCurveControlPoint, Object: "editable", IDs: []ID{"p1"}}); err != nil {
		t.Fatal(err)
	}
	state := workspace.SelectionState()
	if state.Mode != SelectionCurveControlPoint || state.IDs[0] != "p1" {
		t.Fatalf("curve selection = %+v", state)
	}
	position := Vec3{X: 1, Y: 2}
	weight := 2.0
	operation := Operation{Kind: OpSetCurveControlPoint, Target: "editable", ControlPoint: "p1", Position: &position, Weight: &weight}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "preview-curve", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "direct-curve", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		t.Fatal(err)
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	if previewFingerprint != directFingerprint {
		t.Fatal("curve preview and direct output differ")
	}
	point := direct.Entities["editable"].Mesh.Geometry.Curve.ControlPoints[1]
	if point.Position != position || point.Weight != weight {
		t.Fatalf("updated point = %+v", point)
	}
	for _, receipt := range []Receipt{previewReceipt, directReceipt} {
		if len(receipt.OperatorRecords) != 1 || receipt.OperatorRecords[0].SelectionMode != SelectionCurveControlPoint || receipt.OperatorRecords[0].UndoPolicy != "geometry-checkpoint" {
			t.Fatalf("curve operator receipt = %+v", receipt.OperatorRecords)
		}
	}
	_, restored, err := workspace.Undo(direct.Revision, "agent://test")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Entities["editable"].Mesh.Geometry.Curve.ControlPoints[1].Position == position {
		t.Fatal("curve checkpoint undo did not restore control point")
	}
}

func TestNURBSValidationRejectsInvalidKnotsAndWeights(t *testing.T) {
	document := curveDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Geometry.Curve.Knots = []float64{0, 0, 1}
	document.Entities[entity.ID] = entity
	if err := document.Validate(); err == nil {
		t.Fatal("invalid NURBS knot vector accepted")
	}
	document = curveDocument(t)
	entity = document.Entities["editable"]
	entity.Mesh.Geometry.Curve.ControlPoints[1].Weight = 0
	document.Entities[entity.ID] = entity
	if err := document.Validate(); err == nil {
		t.Fatal("non-positive rational weight accepted")
	}
}

func curveDocument(t *testing.T) Document {
	t.Helper()
	document := operatorDocument(t)
	entity := document.Entities["editable"]
	entity.Mesh.Geometry = Geometry{Kind: "nurbs-curve", Curve: &CurveGeometry{Degree: 2, ControlPoints: []CurveControlPoint{{ID: "p0", Position: Vec3{}, Weight: 1}, {ID: "p1", Position: Vec3{X: 1, Y: 1}, Weight: 1}, {ID: "p2", Position: Vec3{X: 2}, Weight: 1}}, Knots: []float64{0, 0, 0, 1, 1, 1}, Segments: 8, Radius: 0.1, RadialSegments: 6}}
	document.Entities[entity.ID] = entity
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	return document
}
