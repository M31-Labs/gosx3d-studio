package studio

import (
	"math"
	"testing"
)

func TestGeometryAnalysisReportsOpenMeshMeasurementsAndStableFindings(t *testing.T) {
	document := operatorDocument(t)
	analysis, err := AnalyzeEntityGeometry(document, "editable")
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Schema != "gosx3d.studio.geometry-analysis/v1" || analysis.Revision != document.Revision {
		t.Fatalf("analysis identity = %+v", analysis)
	}
	if analysis.Vertices != 4 || analysis.Edges != 4 || analysis.Faces != 1 || analysis.Triangles != 2 {
		t.Fatalf("topology counts = %+v", analysis)
	}
	if math.Abs(analysis.SurfaceArea-0.25) > 1e-9 || analysis.Bounds.Size != (Vec3{X: 0.5, Z: 0.5}) {
		t.Fatalf("measurements = area %f bounds %+v", analysis.SurfaceArea, analysis.Bounds)
	}
	if analysis.Closed || !analysis.Manifold || analysis.Volume != nil || !analysis.Valid {
		t.Fatalf("open mesh classification = %+v", analysis)
	}
	if analysis.UV.Complete || analysis.UV.MissingVertices != 4 || analysis.UV.MappedVertices != 0 {
		t.Fatalf("uv inspection = %+v", analysis.UV)
	}
	if len(analysis.Findings) != 4 || analysis.Findings[0].Code != "boundary-edge" || analysis.Findings[0].ID == "" {
		t.Fatalf("findings = %+v", analysis.Findings)
	}
}

func TestGeometryAnalysisComputesClosedTetrahedronVolume(t *testing.T) {
	geometry := Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
		{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}},
		{ID: "c", Position: Vec3{Y: 1}}, {ID: "d", Position: Vec3{Z: 1}},
	}, Faces: []Face{
		{ID: "abc", Vertices: []ID{"a", "c", "b"}},
		{ID: "abd", Vertices: []ID{"a", "b", "d"}},
		{ID: "acd", Vertices: []ID{"a", "d", "c"}},
		{ID: "bcd", Vertices: []ID{"b", "c", "d"}},
	}}
	analysis := analyzeGeometry(geometry)
	if !analysis.Closed || !analysis.Manifold || analysis.Volume == nil || math.Abs(*analysis.Volume-1.0/6.0) > 1e-9 {
		t.Fatalf("closed tetrahedron analysis = %+v", analysis)
	}
}

func TestGeometryAnalysisDetectsDegenerateAndIsolatedTopology(t *testing.T) {
	geometry := Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
		{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}},
		{ID: "c", Position: Vec3{X: 2}}, {ID: "isolated", Position: Vec3{Y: 1}},
	}, Faces: []Face{{ID: "line-face", Vertices: []ID{"a", "b", "c"}}}}
	analysis := analyzeGeometry(geometry)
	if analysis.Valid {
		t.Fatal("degenerate geometry was reported valid")
	}
	codes := map[string]bool{}
	for _, finding := range analysis.Findings {
		codes[finding.Code] = true
	}
	if !codes["degenerate-face"] || !codes["isolated-vertex"] {
		t.Fatalf("findings = %+v", analysis.Findings)
	}
}
