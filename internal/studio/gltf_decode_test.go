package studio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// buildTestGLB assembles a minimal valid GLB: one mesh, one triangles
// primitive, positions + normals + uv0 + uint16 indices for a unit quad.
func buildTestGLB(t *testing.T) []byte {
	t.Helper()
	var bin bytes.Buffer
	write := func(values []float32) (offset, length int) {
		offset = bin.Len()
		for _, value := range values {
			if err := binary.Write(&bin, binary.LittleEndian, value); err != nil {
				t.Fatal(err)
			}
		}
		return offset, bin.Len() - offset
	}
	positions := []float32{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0}
	normals := []float32{0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1}
	uvs := []float32{0, 0, 1, 0, 1, 1, 0, 1}
	positionOffset, positionLength := write(positions)
	normalOffset, normalLength := write(normals)
	uvOffset, uvLength := write(uvs)
	indexOffset := bin.Len()
	for _, index := range []uint16{0, 1, 2, 0, 2, 3} {
		if err := binary.Write(&bin, binary.LittleEndian, index); err != nil {
			t.Fatal(err)
		}
	}
	indexLength := bin.Len() - indexOffset
	for bin.Len()%4 != 0 {
		bin.WriteByte(0)
	}
	root := map[string]any{
		"asset":   map[string]any{"version": "2.0"},
		"buffers": []any{map[string]any{"byteLength": bin.Len()}},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": positionOffset, "byteLength": positionLength},
			map[string]any{"buffer": 0, "byteOffset": normalOffset, "byteLength": normalLength},
			map[string]any{"buffer": 0, "byteOffset": uvOffset, "byteLength": uvLength},
			map[string]any{"buffer": 0, "byteOffset": indexOffset, "byteLength": indexLength},
		},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 4, "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5126, "count": 4, "type": "VEC3"},
			map[string]any{"bufferView": 2, "componentType": 5126, "count": 4, "type": "VEC2"},
			map[string]any{"bufferView": 3, "componentType": 5123, "count": 6, "type": "SCALAR"},
		},
		"meshes": []any{map[string]any{"name": "quad", "primitives": []any{map[string]any{
			"attributes": map[string]any{"POSITION": 0, "NORMAL": 1, "TEXCOORD_0": 2},
			"indices":    3,
		}}}},
	}
	jsonBytes, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	var glb bytes.Buffer
	total := 12 + 8 + len(jsonBytes) + 8 + bin.Len()
	binary.Write(&glb, binary.LittleEndian, uint32(0x46546C67))
	binary.Write(&glb, binary.LittleEndian, uint32(2))
	binary.Write(&glb, binary.LittleEndian, uint32(total))
	binary.Write(&glb, binary.LittleEndian, uint32(len(jsonBytes)))
	binary.Write(&glb, binary.LittleEndian, uint32(0x4E4F534A))
	glb.Write(jsonBytes)
	binary.Write(&glb, binary.LittleEndian, uint32(bin.Len()))
	binary.Write(&glb, binary.LittleEndian, uint32(0x004E4942))
	glb.Write(bin.Bytes())
	return glb.Bytes()
}

func TestDecodeGLBPrimitiveIntoIndexedMesh(t *testing.T) {
	payload := buildTestGLB(t)
	geometry, report, err := DecodeGLTFGeometry(payload, "glb", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if geometry.Kind != "indexed-mesh" || len(geometry.Vertices) != 4 || len(geometry.Faces) != 2 {
		t.Fatalf("geometry v=%d f=%d kind=%s", len(geometry.Vertices), len(geometry.Faces), geometry.Kind)
	}
	if geometry.Vertices[2].ID != "gltf-v0002" || geometry.Faces[1].ID != "gltf-f0001" {
		t.Fatalf("stable ids missing: %+v %+v", geometry.Vertices[2].ID, geometry.Faces[1].ID)
	}
	v2 := geometry.Vertices[2]
	if math.Abs(v2.Position.X-1) > 1e-6 || math.Abs(v2.Position.Y-1) > 1e-6 || v2.UV == nil || math.Abs(v2.UV.X-1) > 1e-6 {
		t.Fatalf("vertex 2 decoded wrong: %+v", v2)
	}
	if report.Mesh != "quad" || report.Triangles != 2 || report.Vertices != 4 || !report.HasNormals || !report.HasUVs {
		t.Fatalf("report=%+v", report)
	}
	if err := validateGeometry(geometry); err != nil {
		t.Fatalf("decoded geometry invalid: %v", err)
	}
	// Determinism.
	second, _, err := DecodeGLTFGeometry(payload, "glb", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(geometry)
	secondJSON, _ := json.Marshal(second)
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("decode is not deterministic")
	}
}

func TestDecodeGLBRejectsUnsupportedShapes(t *testing.T) {
	payload := buildTestGLB(t)
	if _, _, err := DecodeGLTFGeometry(payload, "glb", 5, 0); err == nil {
		t.Fatal("missing mesh index must fail")
	}
	if _, _, err := DecodeGLTFGeometry(payload, "glb", 0, 3); err == nil {
		t.Fatal("missing primitive index must fail")
	}
	if _, _, err := DecodeGLTFGeometry([]byte(`{"asset":{"version":"2.0"},"meshes":[]}`), "gltf", 0, 0); err == nil {
		t.Fatal("gltf without embedded buffers must fail explicitly")
	}
}

func TestDecodeModelGeometryThroughWorkspaceTransaction(t *testing.T) {
	project := t.TempDir()
	input := filepath.Join(t.TempDir(), "fixture-quad.glb")
	if err := os.WriteFile(input, buildTestGLB(t), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := OpenWorkspace(project, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	initial, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	_, imported, asset, err := workspace.ImportAsset(AssetImportRequest{Path: input, Actor: "test", Mode: ModeDirect, ExpectedRevision: initial.Revision})
	if err != nil {
		t.Fatal(err)
	}
	receipt, _, report, err := workspace.DecodeModelGeometry(GeometryDecodeRequest{
		AssetID: asset.ID, NewEntityID: "decoded-quad", Parent: "scene-root", Material: "board-material",
		Actor: "agent://decode-test", Mode: ModeDirect, ExpectedRevision: imported.Revision,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Applied || report.Triangles != 2 {
		t.Fatalf("receipt=%+v report=%+v", receipt, report)
	}
	decoded, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	entity := decoded.Entities["decoded-quad"]
	if entity.Mesh == nil || len(entity.Mesh.Geometry.Vertices) != 4 || len(entity.Mesh.Geometry.Faces) != 2 {
		t.Fatalf("decoded entity=%+v", entity)
	}
	if _, err := Compile(decoded); err != nil {
		t.Fatalf("decoded document does not compile: %v", err)
	}
	if _, restored, err := workspace.Undo(decoded.Revision, "agent://decode-test"); err != nil {
		t.Fatal(err)
	} else if _, exists := restored.Entities["decoded-quad"]; exists {
		t.Fatal("undo did not remove decoded entity")
	}
	if _, _, _, err := workspace.DecodeModelGeometry(GeometryDecodeRequest{AssetID: asset.ID, NewEntityID: "x", Parent: "scene-root", Material: "board-material", Actor: "test", Mode: ModeDirect, ExpectedRevision: 1}); err == nil {
		t.Fatal("stale revision was accepted")
	}
}
