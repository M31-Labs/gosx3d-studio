package studio

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// GLTFDecodeReport summarizes what a geometry decode produced and what the
// floor implementation skipped.
type GLTFDecodeReport struct {
	Mesh       string   `json:"mesh"`
	Primitive  int      `json:"primitive"`
	Vertices   int      `json:"vertices"`
	Triangles  int      `json:"triangles"`
	HasNormals bool     `json:"hasNormals"`
	HasUVs     bool     `json:"hasUVs"`
	Skipped    []string `json:"skipped,omitempty"`
}

type gltfRoot struct {
	Buffers []struct {
		URI        string `json:"uri"`
		ByteLength int    `json:"byteLength"`
	} `json:"buffers"`
	BufferViews []struct {
		Buffer     int `json:"buffer"`
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
		ByteStride int `json:"byteStride"`
	} `json:"bufferViews"`
	Accessors []struct {
		BufferView    *int   `json:"bufferView"`
		ByteOffset    int    `json:"byteOffset"`
		ComponentType int    `json:"componentType"`
		Count         int    `json:"count"`
		Type          string `json:"type"`
		Sparse        any    `json:"sparse"`
	} `json:"accessors"`
	Meshes []struct {
		Name       string `json:"name"`
		Primitives []struct {
			Attributes map[string]int `json:"attributes"`
			Indices    *int           `json:"indices"`
			Mode       *int           `json:"mode"`
		} `json:"primitives"`
	} `json:"meshes"`
}

const gltfDecodeTriangleBudget = 100_000

// DecodeGLTFGeometry decodes one triangles primitive of one mesh from a GLB
// or embedded-buffer glTF payload into stable-ID indexed-mesh geometry. The
// floor supports float32 POSITION/NORMAL/TEXCOORD_0 and 8/16/32-bit indices;
// Draco, Meshopt, sparse accessors, interleaved strides beyond tight packing,
// skinning, and morph targets fail explicitly or are reported as skipped.
func DecodeGLTFGeometry(payload []byte, format string, meshIndex, primitiveIndex int) (Geometry, GLTFDecodeReport, error) {
	jsonBytes := payload
	var binChunk []byte
	if strings.EqualFold(format, "glb") {
		chunk, err := glbJSONChunk(payload)
		if err != nil {
			return Geometry{}, GLTFDecodeReport{}, err
		}
		jsonBytes = chunk
		bin, err := glbBinaryChunk(payload)
		if err != nil {
			return Geometry{}, GLTFDecodeReport{}, err
		}
		binChunk = bin
	}
	var root gltfRoot
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("glTF JSON: %w", err)
	}
	if meshIndex < 0 || meshIndex >= len(root.Meshes) {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("mesh index %d does not exist (%d meshes)", meshIndex, len(root.Meshes))
	}
	mesh := root.Meshes[meshIndex]
	if primitiveIndex < 0 || primitiveIndex >= len(mesh.Primitives) {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("primitive index %d does not exist (%d primitives)", primitiveIndex, len(mesh.Primitives))
	}
	primitive := mesh.Primitives[primitiveIndex]
	if primitive.Mode != nil && *primitive.Mode != 4 {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("primitive mode %d is not TRIANGLES; only mode 4 is decoded", *primitive.Mode)
	}
	buffers := make([][]byte, len(root.Buffers))
	for index, buffer := range root.Buffers {
		switch {
		case buffer.URI == "" && binChunk != nil:
			buffers[index] = binChunk
		case strings.HasPrefix(buffer.URI, "data:"):
			comma := strings.IndexByte(buffer.URI, ',')
			if comma < 0 || !strings.Contains(buffer.URI[:comma], "base64") {
				return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("buffer %d data URI is not base64", index)
			}
			decoded, err := base64.StdEncoding.DecodeString(buffer.URI[comma+1:])
			if err != nil {
				return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("buffer %d data URI: %w", index, err)
			}
			buffers[index] = decoded
		default:
			return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("buffer %d requires external URI %q; geometry decode currently needs GLB or embedded data URIs", index, buffer.URI)
		}
	}
	readAccessor := func(accessorIndex int, wantType string, components int) ([]float64, error) {
		if accessorIndex < 0 || accessorIndex >= len(root.Accessors) {
			return nil, fmt.Errorf("accessor %d does not exist", accessorIndex)
		}
		accessor := root.Accessors[accessorIndex]
		if accessor.Sparse != nil {
			return nil, fmt.Errorf("accessor %d uses sparse storage; not decoded", accessorIndex)
		}
		if accessor.Type != wantType {
			return nil, fmt.Errorf("accessor %d type %q, want %q", accessorIndex, accessor.Type, wantType)
		}
		if accessor.BufferView == nil {
			return nil, fmt.Errorf("accessor %d has no buffer view", accessorIndex)
		}
		view := root.BufferViews[*accessor.BufferView]
		if view.Buffer < 0 || view.Buffer >= len(buffers) {
			return nil, fmt.Errorf("accessor %d buffer view is out of range", accessorIndex)
		}
		data := buffers[view.Buffer]
		componentSize := map[int]int{5120: 1, 5121: 1, 5122: 2, 5123: 2, 5125: 4, 5126: 4}[accessor.ComponentType]
		if componentSize == 0 {
			return nil, fmt.Errorf("accessor %d component type %d unsupported", accessorIndex, accessor.ComponentType)
		}
		stride := view.ByteStride
		if stride == 0 {
			stride = componentSize * components
		}
		out := make([]float64, 0, accessor.Count*components)
		base := view.ByteOffset + accessor.ByteOffset
		for element := 0; element < accessor.Count; element++ {
			offset := base + element*stride
			if offset+componentSize*components > len(data) {
				return nil, fmt.Errorf("accessor %d overflows its buffer", accessorIndex)
			}
			for component := 0; component < components; component++ {
				at := offset + component*componentSize
				switch accessor.ComponentType {
				case 5126:
					out = append(out, float64(math.Float32frombits(binary.LittleEndian.Uint32(data[at:]))))
				case 5125:
					out = append(out, float64(binary.LittleEndian.Uint32(data[at:])))
				case 5123:
					out = append(out, float64(binary.LittleEndian.Uint16(data[at:])))
				case 5121:
					out = append(out, float64(data[at]))
				default:
					return nil, fmt.Errorf("accessor %d component type %d unsupported for decode", accessorIndex, accessor.ComponentType)
				}
			}
		}
		return out, nil
	}

	positionAccessor, ok := primitive.Attributes["POSITION"]
	if !ok {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("primitive has no POSITION attribute")
	}
	positions, err := readAccessor(positionAccessor, "VEC3", 3)
	if err != nil {
		return Geometry{}, GLTFDecodeReport{}, err
	}
	vertexCount := len(positions) / 3
	report := GLTFDecodeReport{Mesh: mesh.Name, Primitive: primitiveIndex, Vertices: vertexCount}
	var normals, uvs []float64
	if accessor, ok := primitive.Attributes["NORMAL"]; ok {
		if normals, err = readAccessor(accessor, "VEC3", 3); err != nil {
			return Geometry{}, GLTFDecodeReport{}, err
		}
		report.HasNormals = true
	}
	if accessor, ok := primitive.Attributes["TEXCOORD_0"]; ok {
		if uvs, err = readAccessor(accessor, "VEC2", 2); err != nil {
			return Geometry{}, GLTFDecodeReport{}, err
		}
		report.HasUVs = true
	}
	for attribute := range primitive.Attributes {
		if attribute != "POSITION" && attribute != "NORMAL" && attribute != "TEXCOORD_0" {
			report.Skipped = append(report.Skipped, attribute)
		}
	}
	uniqueSortedInPlace(&report.Skipped)
	var indices []float64
	if primitive.Indices != nil {
		if indices, err = readAccessor(*primitive.Indices, "SCALAR", 1); err != nil {
			return Geometry{}, GLTFDecodeReport{}, err
		}
	} else {
		indices = make([]float64, vertexCount)
		for i := range indices {
			indices[i] = float64(i)
		}
	}
	if len(indices)%3 != 0 {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("index count %d is not a triangle multiple", len(indices))
	}
	triangles := len(indices) / 3
	if triangles > gltfDecodeTriangleBudget {
		return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("primitive has %d triangles; decode budget is %d", triangles, gltfDecodeTriangleBudget)
	}
	report.Triangles = triangles
	geometry := Geometry{Kind: "indexed-mesh", Vertices: make([]Vertex, 0, vertexCount), Faces: make([]Face, 0, triangles)}
	for i := 0; i < vertexCount; i++ {
		vertex := Vertex{ID: ID(fmt.Sprintf("gltf-v%04d", i)), Position: Vec3{X: positions[i*3], Y: positions[i*3+1], Z: positions[i*3+2]}}
		if normals != nil {
			vertex.Normal = Vec3{X: normals[i*3], Y: normals[i*3+1], Z: normals[i*3+2]}
		}
		if uvs != nil {
			vertex.UV = &Vec2{X: uvs[i*2], Y: uvs[i*2+1]}
		}
		geometry.Vertices = append(geometry.Vertices, vertex)
	}
	for i := 0; i < triangles; i++ {
		a, b, c := int(indices[i*3]), int(indices[i*3+1]), int(indices[i*3+2])
		if a >= vertexCount || b >= vertexCount || c >= vertexCount {
			return Geometry{}, GLTFDecodeReport{}, fmt.Errorf("triangle %d indexes past vertex count %d", i, vertexCount)
		}
		geometry.Faces = append(geometry.Faces, Face{ID: ID(fmt.Sprintf("gltf-f%04d", i)), Vertices: []ID{
			ID(fmt.Sprintf("gltf-v%04d", a)), ID(fmt.Sprintf("gltf-v%04d", b)), ID(fmt.Sprintf("gltf-v%04d", c)),
		}})
	}
	return geometry, report, nil
}

// glbBinaryChunk extracts the BIN chunk following the JSON chunk.
func glbBinaryChunk(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("GLB shorter than its header")
	}
	offset := 12
	for offset+8 <= len(data) {
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		kind := binary.LittleEndian.Uint32(data[offset+4:])
		if offset+8+length > len(data) {
			return nil, fmt.Errorf("GLB chunk overflows the file")
		}
		if kind == 0x004E4942 {
			return data[offset+8 : offset+8+length], nil
		}
		offset += 8 + length
	}
	return nil, nil
}

func uniqueSortedInPlace(values *[]string) {
	*values = uniqueSortedStrings(*values)
}

// GeometryDecodeRequest asks the workspace to decode a registered glTF/GLB
// asset primitive into an authored indexed-mesh entity through the shared
// transaction path.
type GeometryDecodeRequest struct {
	AssetID          ID              `json:"assetId"`
	MeshIndex        int             `json:"meshIndex"`
	PrimitiveIndex   int             `json:"primitiveIndex"`
	NewEntityID      ID              `json:"newEntityId"`
	Name             string          `json:"name"`
	Parent           ID              `json:"parent"`
	Material         ID              `json:"material"`
	Actor            string          `json:"actor"`
	Mode             TransactionMode `json:"mode"`
	ExpectedRevision uint64          `json:"expectedRevision"`
}

// DecodeModelGeometry reads a registered asset's immutable payload, verifies
// its content hash, decodes one triangles primitive into stable-ID geometry,
// and creates the entity through the ordinary revision-safe command path.
func (w *Workspace) DecodeModelGeometry(request GeometryDecodeRequest) (Receipt, Document, GLTFDecodeReport, error) {
	document, err := w.Snapshot()
	if err != nil {
		return Receipt{}, Document{}, GLTFDecodeReport{}, err
	}
	asset, ok := document.Assets[request.AssetID]
	if !ok {
		return Receipt{}, Document{}, GLTFDecodeReport{}, fmt.Errorf("asset %q does not exist", request.AssetID)
	}
	if asset.Format != "glb" && asset.Format != "gltf" {
		return Receipt{}, Document{}, GLTFDecodeReport{}, fmt.Errorf("asset %q format %q is not glb/gltf", asset.ID, asset.Format)
	}
	payload, err := w.readVerifiedAssetPayload(asset)
	if err != nil {
		return Receipt{}, Document{}, GLTFDecodeReport{}, err
	}
	geometry, report, err := DecodeGLTFGeometry(payload, asset.Format, request.MeshIndex, request.PrimitiveIndex)
	if err != nil {
		return Receipt{}, Document{}, GLTFDecodeReport{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = fmt.Sprintf("%s mesh %d/%d", asset.SourceName, request.MeshIndex, request.PrimitiveIndex)
	}
	entity := Entity{
		ID: request.NewEntityID, Name: name, Parent: request.Parent, Transform: IdentityTransform(), Visible: true,
		Mesh: &MeshComponent{Geometry: geometry, Material: request.Material, Pickable: true, CastShadow: true, ReceiveShadow: true},
	}
	receipt, preview, err := w.Execute(Transaction{
		ID:    fmt.Sprintf("decode-geometry-%s-%d-%d", asset.ID, request.MeshIndex, request.PrimitiveIndex),
		Actor: request.Actor, Mode: request.Mode, ExpectedRevision: request.ExpectedRevision,
		Operations: []Operation{{Kind: OpCreateEntity, Entity: &entity, Parent: request.Parent}},
	})
	if err != nil {
		return Receipt{}, Document{}, GLTFDecodeReport{}, err
	}
	return receipt, preview, report, nil
}

// readVerifiedAssetPayload loads an immutable payload from the project store
// and re-hashes it before trusting the bytes, matching the content handler
// and integrity-audit behavior.
func (w *Workspace) readVerifiedAssetPayload(asset AssetRecord) ([]byte, error) {
	w.mu.RLock()
	projectDir := w.projectDir
	w.mu.RUnlock()
	if projectDir == "" {
		return nil, fmt.Errorf("workspace has no project directory for asset payloads")
	}
	data, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(asset.StorePath)))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != asset.ContentHash {
		return nil, fmt.Errorf("asset %q payload does not match its content hash; refusing to decode", asset.ID)
	}
	return data, nil
}
