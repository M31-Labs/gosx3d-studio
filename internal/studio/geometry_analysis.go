package studio

import (
	"fmt"
	"math"
	"sort"
)

type GeometryAnalysis struct {
	Schema      string            `json:"schema"`
	Entity      ID                `json:"entity"`
	Revision    uint64            `json:"revision"`
	Vertices    int               `json:"vertices"`
	Edges       int               `json:"edges"`
	Faces       int               `json:"faces"`
	Triangles   int               `json:"triangles"`
	Bounds      GeometryBounds    `json:"bounds"`
	SurfaceArea float64           `json:"surfaceArea"`
	Volume      *float64          `json:"volume,omitempty"`
	Closed      bool              `json:"closed"`
	Manifold    bool              `json:"manifold"`
	Findings    []GeometryFinding `json:"findings"`
	UV          UVAnalysis        `json:"uv"`
	Valid       bool              `json:"valid"`
}

type UVAnalysis struct {
	MappedVertices  int       `json:"mappedVertices"`
	MissingVertices int       `json:"missingVertices"`
	OutsideUnit     int       `json:"outsideUnit"`
	DegenerateFaces int       `json:"degenerateFaces"`
	Complete        bool      `json:"complete"`
	Bounds          *UVBounds `json:"bounds,omitempty"`
}

type UVBounds struct {
	Min Vec2 `json:"min"`
	Max Vec2 `json:"max"`
}

type GeometryBounds struct {
	Min  Vec3 `json:"min"`
	Max  Vec3 `json:"max"`
	Size Vec3 `json:"size"`
}

type GeometryFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	ID       ID     `json:"id,omitempty"`
	Message  string `json:"message"`
}

func AnalyzeEntityGeometry(document Document, entityID ID) (GeometryAnalysis, error) {
	entity, ok := document.Entities[entityID]
	if !ok || entity.Mesh == nil || entity.Mesh.Geometry.Kind != "indexed-mesh" {
		return GeometryAnalysis{}, fmt.Errorf("geometry analysis target %q is not an indexed mesh", entityID)
	}
	analysis := analyzeGeometry(entity.Mesh.Geometry)
	analysis.Schema = "gosx3d.studio.geometry-analysis/v1"
	analysis.Entity = entityID
	analysis.Revision = document.Revision
	return analysis, nil
}

func analyzeGeometry(geometry Geometry) GeometryAnalysis {
	analysis := GeometryAnalysis{Vertices: len(geometry.Vertices), Faces: len(geometry.Faces), Findings: []GeometryFinding{}, Manifold: true}
	vertices := make(map[ID]Vertex, len(geometry.Vertices))
	used := map[ID]bool{}
	if len(geometry.Vertices) > 0 {
		analysis.Bounds.Min = geometry.Vertices[0].Position
		analysis.Bounds.Max = geometry.Vertices[0].Position
	}
	positionOwners := map[string]ID{}
	uvBounds := UVBounds{Min: Vec2{X: math.Inf(1), Y: math.Inf(1)}, Max: Vec2{X: math.Inf(-1), Y: math.Inf(-1)}}
	for _, vertex := range geometry.Vertices {
		vertices[vertex.ID] = vertex
		analysis.Bounds.Min = minVec(analysis.Bounds.Min, vertex.Position)
		analysis.Bounds.Max = maxVec(analysis.Bounds.Max, vertex.Position)
		key := fmt.Sprintf("%.12g/%.12g/%.12g", vertex.Position.X, vertex.Position.Y, vertex.Position.Z)
		if owner, exists := positionOwners[key]; exists {
			analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "duplicate-position", Severity: "warning", ID: vertex.ID, Message: fmt.Sprintf("shares its position with vertex %q", owner)})
		} else {
			positionOwners[key] = vertex.ID
		}
		if vertex.UV == nil {
			analysis.UV.MissingVertices++
		} else {
			analysis.UV.MappedVertices++
			uvBounds.Min.X, uvBounds.Min.Y = math.Min(uvBounds.Min.X, vertex.UV.X), math.Min(uvBounds.Min.Y, vertex.UV.Y)
			uvBounds.Max.X, uvBounds.Max.Y = math.Max(uvBounds.Max.X, vertex.UV.X), math.Max(uvBounds.Max.Y, vertex.UV.Y)
			if vertex.UV.X < 0 || vertex.UV.X > 1 || vertex.UV.Y < 0 || vertex.UV.Y > 1 {
				analysis.UV.OutsideUnit++
				analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "uv-outside-unit", Severity: "info", ID: vertex.ID, Message: "uv lies outside the zero-to-one tile"})
			}
		}
	}
	analysis.UV.Complete = analysis.UV.MissingVertices == 0 && len(geometry.Vertices) > 0
	if analysis.UV.MappedVertices > 0 {
		analysis.UV.Bounds = &uvBounds
	}
	analysis.Bounds.Size = addVec(analysis.Bounds.Max, scaleVec(analysis.Bounds.Min, -1))
	edgeUse := map[string]int{}
	edgeID := map[string]ID{}
	signedVolume := 0.0
	for _, face := range geometry.Faces {
		for index, a := range face.Vertices {
			used[a] = true
			b := face.Vertices[(index+1)%len(face.Vertices)]
			key := canonicalEdgeKey(a, b)
			edgeUse[key]++
			edgeID[key] = stableEdgeID(a, b)
		}
		origin := vertices[face.Vertices[0]].Position
		faceArea := 0.0
		for index := 1; index+1 < len(face.Vertices); index++ {
			b := vertices[face.Vertices[index]].Position
			c := vertices[face.Vertices[index+1]].Position
			cross := crossVec(addVec(b, scaleVec(origin, -1)), addVec(c, scaleVec(origin, -1)))
			triangleArea := 0.5 * vectorLength(cross)
			faceArea += triangleArea
			signedVolume += dotVec(origin, crossVec(b, c)) / 6
			analysis.Triangles++
		}
		analysis.SurfaceArea += faceArea
		if faceArea <= 1e-12 {
			analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "degenerate-face", Severity: "error", ID: face.ID, Message: "face has zero surface area"})
		}
		uvArea := 0.0
		uvComplete := true
		for index, vertexID := range face.Vertices {
			current := vertices[vertexID].UV
			next := vertices[face.Vertices[(index+1)%len(face.Vertices)]].UV
			if current == nil || next == nil {
				uvComplete = false
				break
			}
			uvArea += current.X*next.Y - next.X*current.Y
		}
		if uvComplete && math.Abs(uvArea)*0.5 <= 1e-12 {
			analysis.UV.DegenerateFaces++
			analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "degenerate-uv-face", Severity: "warning", ID: face.ID, Message: "face has zero uv area"})
		}
	}
	analysis.Edges = len(edgeUse)
	keys := make([]string, 0, len(edgeUse))
	for key := range edgeUse {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	boundary := 0
	for _, key := range keys {
		switch count := edgeUse[key]; {
		case count == 1:
			boundary++
			analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "boundary-edge", Severity: "info", ID: edgeID[key], Message: "edge belongs to one face"})
		case count > 2:
			analysis.Manifold = false
			analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "non-manifold-edge", Severity: "error", ID: edgeID[key], Message: fmt.Sprintf("edge belongs to %d faces", count)})
		}
	}
	for _, vertex := range geometry.Vertices {
		if !used[vertex.ID] {
			analysis.Findings = append(analysis.Findings, GeometryFinding{Code: "isolated-vertex", Severity: "warning", ID: vertex.ID, Message: "vertex is not referenced by a face"})
		}
	}
	analysis.Closed = boundary == 0 && analysis.Manifold && len(geometry.Faces) > 0
	if analysis.Closed {
		volume := math.Abs(signedVolume)
		analysis.Volume = &volume
	}
	analysis.Valid = true
	for _, finding := range analysis.Findings {
		if finding.Severity == "error" {
			analysis.Valid = false
			break
		}
	}
	return analysis
}

func canonicalEdgeKey(a, b ID) string {
	if b < a {
		a, b = b, a
	}
	return fmt.Sprintf("%d:%s%d:%s", len(a), a, len(b), b)
}

func stableEdgeID(a, b ID) ID {
	for _, edge := range MeshEdges(Geometry{Faces: []Face{{ID: "edge", Vertices: []ID{a, b}}}}) {
		if (edge.A == a && edge.B == b) || (edge.A == b && edge.B == a) {
			return edge.ID
		}
	}
	return ""
}

func minVec(a, b Vec3) Vec3 {
	return Vec3{X: math.Min(a.X, b.X), Y: math.Min(a.Y, b.Y), Z: math.Min(a.Z, b.Z)}
}
func maxVec(a, b Vec3) Vec3 {
	return Vec3{X: math.Max(a.X, b.X), Y: math.Max(a.Y, b.Y), Z: math.Max(a.Z, b.Z)}
}
func crossVec(a, b Vec3) Vec3 {
	return Vec3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
func dotVec(a, b Vec3) float64        { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func vectorLength(value Vec3) float64 { return math.Sqrt(dotVec(value, value)) }
