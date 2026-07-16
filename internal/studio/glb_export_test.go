package studio

import (
	"bytes"
	"testing"
)

func glbExportDocument() Document {
	document := SampleDocument()
	root := document.Entities["scene-root"]
	quad := Geometry{Kind: "indexed-mesh",
		Vertices: []Vertex{
			{ID: "v0", Position: Vec3{}, Normal: Vec3{Z: 1}, UV: &Vec2{}},
			{ID: "v1", Position: Vec3{X: 1}, Normal: Vec3{Z: 1}, UV: &Vec2{X: 1}},
			{ID: "v2", Position: Vec3{X: 1, Y: 1}, Normal: Vec3{Z: 1}, UV: &Vec2{X: 1, Y: 1}},
			{ID: "v3", Position: Vec3{Y: 1}, Normal: Vec3{Z: 1}, UV: &Vec2{Y: 1}},
		},
		Faces: []Face{{ID: "quad", Vertices: []ID{"v0", "v1", "v2", "v3"}}},
	}
	entity := Entity{ID: "export-quad", Name: "Export quad", Parent: root.ID, Transform: TransformFromEuler(Vec3{X: 2}, Vec3{Z: 0.5}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true, Mesh: &MeshComponent{Geometry: quad, Material: "board-material", Pickable: true}}
	root.Children = append(root.Children, entity.ID)
	document.Entities[root.ID] = root
	document.Entities[entity.ID] = entity
	return document
}

func TestExportGLBRoundTripsThroughOwnDecoder(t *testing.T) {
	document := glbExportDocument()
	payload, report, err := ExportGLB(document)
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "glb" || report.Bytes != len(payload) {
		t.Fatalf("report=%+v", report)
	}
	lossDomains := map[string]bool{}
	for _, loss := range report.Losses {
		lossDomains[loss.Domain] = true
	}
	if !lossDomains["primitives"] || !lossDomains["selenaShaders"] {
		t.Fatalf("expected primitive and selena losses, got %+v", report.Losses)
	}
	inspection, err := InspectGLTF(payload, "glb")
	if err != nil {
		t.Fatalf("exported GLB fails our own inspection: %v", err)
	}
	_ = inspection
	geometry, decodeReport, err := DecodeGLTFGeometry(payload, "glb", 0, 0)
	if err != nil {
		t.Fatalf("exported GLB fails our own decoder: %v", err)
	}
	if decodeReport.Triangles != 2 || len(geometry.Vertices) != 4 {
		t.Fatalf("round trip geometry v=%d t=%d", len(geometry.Vertices), decodeReport.Triangles)
	}
	if geometry.Vertices[2].Position.X != 1 || geometry.Vertices[2].Position.Y != 1 {
		t.Fatalf("round trip vertex 2 = %+v", geometry.Vertices[2])
	}
	second, _, err := ExportGLB(document)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, second) {
		t.Fatal("GLB export is not byte-deterministic")
	}
}
