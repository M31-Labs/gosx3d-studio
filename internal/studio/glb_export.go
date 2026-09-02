package studio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// ExportGLB emits a deterministic minimal GLB: the entity tree as nodes with
// TRS transforms, evaluated indexed meshes as triangles primitives with
// float32 position/normal/uv0 and uint32 indices, and referenced materials as
// pbrMetallicRoughness. Everything the format floor cannot carry is named in
// the loss report before bytes are written.
func ExportGLB(document Document) ([]byte, ExportReport, error) {
	if err := document.Validate(); err != nil {
		return nil, ExportReport{}, err
	}
	var bin bytes.Buffer
	writeFloats := func(values []float32) (int, int) {
		offset := bin.Len()
		for _, value := range values {
			binary.Write(&bin, binary.LittleEndian, value)
		}
		return offset, bin.Len() - offset
	}

	bufferViews := make([]map[string]any, 0, 8)
	accessors := make([]map[string]any, 0, 8)
	meshes := make([]map[string]any, 0, 8)
	materials := make([]map[string]any, 0, 8)
	materialIndex := map[ID]int{}
	losses := map[string]*ExportLoss{}
	addLoss := func(domain, reason string) {
		if entry, ok := losses[domain]; ok {
			entry.Count++
			return
		}
		losses[domain] = &ExportLoss{Domain: domain, Reason: reason, Count: 1}
	}

	for _, record := range document.Materials {
		if record.Selena != nil {
			addLoss("selenaShaders", "Selena shader sources are not representable in core glTF")
		}
		if record.Clearcoat != 0 || record.Sheen != 0 || record.Transmission != 0 || record.Iridescence != 0 || record.Anisotropy != 0 || record.Emissive != 0 {
			addLoss("extendedPBR", "clearcoat, sheen, transmission, iridescence, anisotropy, and scalar emissive need glTF extensions not emitted by the floor exporter")
		}
	}
	ensureMaterial := func(id ID) int {
		if index, ok := materialIndex[id]; ok {
			return index
		}
		record := document.Materials[id]
		r, g, b := hexColorComponents(record.Color)
		material := map[string]any{
			"name": record.Name,
			"pbrMetallicRoughness": map[string]any{
				"baseColorFactor": []any{r, g, b, 1.0},
				"metallicFactor":  record.Metalness,
				"roughnessFactor": record.Roughness,
			},
		}
		materials = append(materials, material)
		materialIndex[id] = len(materials) - 1
		return materialIndex[id]
	}

	buildMesh := func(entity Entity) (int, error) {
		geometry, err := evaluateModifiers(entity.Mesh.Geometry, entity.Mesh.Modifiers)
		if err != nil {
			return -1, fmt.Errorf("entity %q modifiers: %w", entity.ID, err)
		}
		if len(entity.Mesh.Modifiers) > 0 {
			addLoss("modifiers", "non-destructive modifier stacks are baked into exported triangles")
		}
		vertexOrder := make([]ID, 0, len(geometry.Vertices))
		vertexAt := map[ID]int{}
		positions := make([]float32, 0, len(geometry.Vertices)*3)
		normals := make([]float32, 0, len(geometry.Vertices)*3)
		uvs := make([]float32, 0, len(geometry.Vertices)*2)
		hasUV := true
		for _, vertex := range geometry.Vertices {
			vertexAt[vertex.ID] = len(vertexOrder)
			vertexOrder = append(vertexOrder, vertex.ID)
			positions = append(positions, float32(vertex.Position.X), float32(vertex.Position.Y), float32(vertex.Position.Z))
			normals = append(normals, float32(vertex.Normal.X), float32(vertex.Normal.Y), float32(vertex.Normal.Z))
			if vertex.UV == nil {
				hasUV = false
			} else {
				uvs = append(uvs, float32(vertex.UV.X), float32(vertex.UV.Y))
			}
		}
		indices := make([]uint32, 0, len(geometry.Faces)*3)
		for _, face := range geometry.Faces {
			if len(face.Vertices) < 3 {
				return -1, fmt.Errorf("entity %q face %q has fewer than three vertices", entity.ID, face.ID)
			}
			for corner := 1; corner+1 < len(face.Vertices); corner++ {
				indices = append(indices,
					uint32(vertexAt[face.Vertices[0]]),
					uint32(vertexAt[face.Vertices[corner]]),
					uint32(vertexAt[face.Vertices[corner+1]]))
			}
			if len(face.Vertices) > 3 {
				addLoss("polygons", "polygon faces are fan-triangulated for GLB")
			}
		}
		attributes := map[string]any{}
		offset, length := writeFloats(positions)
		bufferViews = append(bufferViews, map[string]any{"buffer": 0, "byteOffset": offset, "byteLength": length})
		accessors = append(accessors, map[string]any{"bufferView": len(bufferViews) - 1, "componentType": 5126, "count": len(vertexOrder), "type": "VEC3", "min": floatMin(positions, 3), "max": floatMax(positions, 3)})
		attributes["POSITION"] = len(accessors) - 1
		offset, length = writeFloats(normals)
		bufferViews = append(bufferViews, map[string]any{"buffer": 0, "byteOffset": offset, "byteLength": length})
		accessors = append(accessors, map[string]any{"bufferView": len(bufferViews) - 1, "componentType": 5126, "count": len(vertexOrder), "type": "VEC3"})
		attributes["NORMAL"] = len(accessors) - 1
		if hasUV && len(uvs) == len(vertexOrder)*2 {
			offset, length = writeFloats(uvs)
			bufferViews = append(bufferViews, map[string]any{"buffer": 0, "byteOffset": offset, "byteLength": length})
			accessors = append(accessors, map[string]any{"bufferView": len(bufferViews) - 1, "componentType": 5126, "count": len(vertexOrder), "type": "VEC2"})
			attributes["TEXCOORD_0"] = len(accessors) - 1
		}
		indexOffset := bin.Len()
		for _, index := range indices {
			binary.Write(&bin, binary.LittleEndian, index)
		}
		bufferViews = append(bufferViews, map[string]any{"buffer": 0, "byteOffset": indexOffset, "byteLength": bin.Len() - indexOffset})
		accessors = append(accessors, map[string]any{"bufferView": len(bufferViews) - 1, "componentType": 5125, "count": len(indices), "type": "SCALAR"})
		primitive := map[string]any{"attributes": attributes, "indices": len(accessors) - 1, "material": ensureMaterial(entity.Mesh.Material)}
		meshes = append(meshes, map[string]any{"name": entity.Name, "primitives": []any{primitive}})
		return len(meshes) - 1, nil
	}

	nodes := make([]map[string]any, 0, len(document.Entities))
	var buildNode func(id ID) (int, error)
	buildNode = func(id ID) (int, error) {
		entity := document.Entities[id]
		node := map[string]any{"name": entity.Name}
		transform := entity.Transform.canonical()
		if transform.Position != (Vec3{}) {
			node["translation"] = []any{transform.Position.X, transform.Position.Y, transform.Position.Z}
		}
		if !transform.Rotation.IsIdentity() {
			unit := transform.Rotation.Normalized()
			node["rotation"] = []any{unit.X, unit.Y, unit.Z, unit.W}
		}
		switch {
		case entity.Mesh != nil && entity.Mesh.Geometry.Kind == "indexed-mesh":
			meshIndex, err := buildMesh(entity)
			if err != nil {
				return -1, err
			}
			node["mesh"] = meshIndex
		case entity.Mesh != nil:
			addLoss("primitives", "engine-tessellated primitive geometry (box/sphere/plane/cylinder/curve) is not exported")
		case entity.Light != nil:
			addLoss("lights", "lights need KHR_lights_punctual, which the floor exporter does not emit")
		case entity.Model != nil:
			addLoss("models", "external model asset references are not inlined")
		case entity.Prefab != nil:
			addLoss("prefabs", "prefab instances are not flattened by the floor exporter")
		}
		children := make([]any, 0, len(entity.Children))
		for _, childID := range entity.Children {
			childIndex, err := buildNode(childID)
			if err != nil {
				return -1, err
			}
			children = append(children, childIndex)
		}
		if len(children) > 0 {
			node["children"] = children
		}
		nodes = append(nodes, node)
		return len(nodes) - 1, nil
	}

	rootIndexes := make([]any, 0, len(document.RootIDs))
	for _, id := range document.RootIDs {
		index, err := buildNode(id)
		if err != nil {
			return nil, ExportReport{}, err
		}
		rootIndexes = append(rootIndexes, index)
	}
	for _, domain := range []struct {
		count  int
		name   string
		reason string
	}{
		{len(document.Armatures), "rigs", "armatures and skinning are not emitted by the floor exporter"},
		{len(document.Animations), "animations", "clips are not emitted by the floor exporter"},
		{len(document.Simulations), "simulations", "simulation profiles are authoring state"},
	} {
		if domain.count > 0 {
			losses[domain.name] = &ExportLoss{Domain: domain.name, Reason: domain.reason, Count: domain.count}
		}
	}

	for bin.Len()%4 != 0 {
		bin.WriteByte(0)
	}
	root := map[string]any{
		"asset":       map[string]any{"version": "2.0", "generator": "gosx3d-studio"},
		"scene":       0,
		"scenes":      []any{map[string]any{"nodes": rootIndexes}},
		"nodes":       nodes,
		"buffers":     []any{map[string]any{"byteLength": bin.Len()}},
		"bufferViews": bufferViews,
		"accessors":   accessors,
	}
	if len(meshes) > 0 {
		root["meshes"] = meshes
	}
	if len(materials) > 0 {
		root["materials"] = materials
	}
	jsonBytes, err := json.Marshal(root)
	if err != nil {
		return nil, ExportReport{}, err
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	var glb bytes.Buffer
	binary.Write(&glb, binary.LittleEndian, uint32(0x46546C67))
	binary.Write(&glb, binary.LittleEndian, uint32(2))
	binary.Write(&glb, binary.LittleEndian, uint32(12+8+len(jsonBytes)+8+bin.Len()))
	binary.Write(&glb, binary.LittleEndian, uint32(len(jsonBytes)))
	binary.Write(&glb, binary.LittleEndian, uint32(0x4E4F534A))
	glb.Write(jsonBytes)
	binary.Write(&glb, binary.LittleEndian, uint32(bin.Len()))
	binary.Write(&glb, binary.LittleEndian, uint32(0x004E4942))
	glb.Write(bin.Bytes())

	ordered := make([]ExportLoss, 0, len(losses))
	domains := make([]string, 0, len(losses))
	for domain := range losses {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	for _, domain := range domains {
		ordered = append(ordered, *losses[domain])
	}
	payload := glb.Bytes()
	return payload, exportReport("glb", "glTF 2.0 binary", document, payload, ordered), nil
}

func hexColorComponents(value string) (float64, float64, float64) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(trimmed) != 6 {
		return 0.5, 0.5, 0.5
	}
	parse := func(part string) float64 {
		number, err := strconv.ParseUint(part, 16, 16)
		if err != nil {
			return 0.5
		}
		return float64(number) / 255
	}
	return parse(trimmed[0:2]), parse(trimmed[2:4]), parse(trimmed[4:6])
}

func floatMin(values []float32, stride int) []any {
	out := make([]any, stride)
	for component := 0; component < stride; component++ {
		minimum := math.Inf(1)
		for index := component; index < len(values); index += stride {
			minimum = math.Min(minimum, float64(values[index]))
		}
		out[component] = minimum
	}
	return out
}

func floatMax(values []float32, stride int) []any {
	out := make([]any, stride)
	for component := 0; component < stride; component++ {
		maximum := math.Inf(-1)
		for index := component; index < len(values); index += stride {
			maximum = math.Max(maximum, float64(values[index]))
		}
		out[component] = maximum
	}
	return out
}
