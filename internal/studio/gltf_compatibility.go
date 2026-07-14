package studio

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const GLTFMatrixSchema = "gosx3d.studio.gltf-capability-matrix/v1"
const GLTFCorpusSchema = "gosx3d.studio.gltf-corpus/v1"
const GLTFInspectionSchema = "gosx3d.studio.gltf-inspection/v1"

type GLTFCapability struct {
	Extension string            `json:"extension"`
	Domain    string            `json:"domain"`
	Targets   map[string]string `json:"targets"`
	Evidence  string            `json:"evidence"`
}

type GLTFCapabilityMatrix struct {
	Schema  string           `json:"schema"`
	Targets []string         `json:"targets"`
	Entries []GLTFCapability `json:"entries"`
}

type GLTFCorpusCase struct {
	ID         string   `json:"id"`
	Domain     string   `json:"domain"`
	Extensions []string `json:"extensions,omitempty"`
	Expected   string   `json:"expected"`
	Evidence   string   `json:"evidence"`
}

type GLTFCorpusManifest struct {
	Schema string           `json:"schema"`
	Cases  []GLTFCorpusCase `json:"cases"`
}

type GLTFExtensionFinding struct {
	Extension string `json:"extension"`
	Required  bool   `json:"required"`
	Status    string `json:"status"`
	Evidence  string `json:"evidence"`
}

type GLTFTargetVerdict struct {
	Target   string                 `json:"target"`
	Status   string                 `json:"status"`
	Findings []GLTFExtensionFinding `json:"findings,omitempty"`
}

type GLTFInspection struct {
	Schema             string              `json:"schema"`
	Version            string              `json:"version"`
	ExtensionsUsed     []string            `json:"extensionsUsed,omitempty"`
	ExtensionsRequired []string            `json:"extensionsRequired,omitempty"`
	Targets            []GLTFTargetVerdict `json:"targets"`
	Fingerprint        string              `json:"fingerprint"`
}

func DefaultGLTFCapabilityMatrix() GLTFCapabilityMatrix {
	targets := []string{"scene-ir", "native-headless", "webgpu", "webgl"}
	entry := func(extension, domain, evidence string, sceneIR, headless, webgpu, webgl string) GLTFCapability {
		return GLTFCapability{Extension: extension, Domain: domain, Evidence: evidence, Targets: map[string]string{"scene-ir": sceneIR, "native-headless": headless, "webgpu": webgpu, "webgl": webgl}}
	}
	entries := []GLTFCapability{
		entry("KHR_materials_unlit", "materials", "typed material transport and renderer coverage", "available", "available", "available", "available"),
		entry("KHR_materials_pbrSpecularGlossiness", "materials", "migration to metallic-roughness required", "migration", "migration", "migration", "migration"),
		entry("KHR_materials_clearcoat", "materials", "Scene3D physical material clearcoat", "available", "partial", "available", "available"),
		entry("KHR_materials_transmission", "materials", "Scene3D physical transmission", "available", "partial", "available", "available"),
		entry("KHR_materials_ior", "materials", "preserved by import contract; backend parity incomplete", "partial", "partial", "partial", "partial"),
		entry("KHR_materials_volume", "materials", "volume attenuation parity pending", "partial", "partial", "partial", "partial"),
		entry("KHR_materials_specular", "materials", "specular factor parity pending", "partial", "partial", "partial", "partial"),
		entry("KHR_materials_sheen", "materials", "sheen parity pending", "partial", "partial", "partial", "partial"),
		entry("KHR_materials_iridescence", "materials", "iridescence parity pending", "partial", "partial", "partial", "partial"),
		entry("KHR_texture_transform", "textures", "texture transform transport", "available", "partial", "available", "available"),
		entry("KHR_texture_basisu", "compression", "KTX2/Basis transcoder integration pending", "partial", "planned", "partial", "partial"),
		entry("EXT_meshopt_compression", "compression", "Meshopt decoder integration pending", "partial", "planned", "partial", "partial"),
		entry("KHR_draco_mesh_compression", "compression", "Draco decoder integration pending", "partial", "planned", "partial", "partial"),
		entry("KHR_mesh_quantization", "geometry", "quantized accessor transport", "available", "partial", "available", "available"),
		entry("EXT_mesh_gpu_instancing", "geometry", "typed instancing transport", "available", "partial", "available", "available"),
		entry("KHR_lights_punctual", "lighting", "typed punctual light transport", "available", "available", "available", "available"),
		entry("KHR_animation_pointer", "animation", "arbitrary glTF property animation pending", "planned", "planned", "planned", "planned"),
		entry("EXT_texture_webp", "textures", "WebP decode varies by target", "partial", "planned", "available", "available"),
		entry("EXT_texture_avif", "textures", "AVIF decode varies by target", "partial", "planned", "partial", "partial"),
		entry("MSFT_lod", "lod", "author-controlled LOD import pending", "planned", "planned", "planned", "planned"),
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Extension < entries[j].Extension })
	return GLTFCapabilityMatrix{Schema: GLTFMatrixSchema, Targets: targets, Entries: entries}
}

func DefaultGLTFCorpusManifest() GLTFCorpusManifest {
	cases := []GLTFCorpusCase{
		{ID: "gltf.core.triangle", Domain: "geometry", Expected: "equivalent", Evidence: "embedded deterministic fixture"},
		{ID: "gltf.core.skin-animation", Domain: "animation", Expected: "equivalent", Evidence: "articulated SceneDoc and skinning certification"},
		{ID: "gltf.ext.draco-required", Domain: "compression", Extensions: []string{"KHR_draco_mesh_compression"}, Expected: "unsupported-required", Evidence: "required-extension rejection fixture"},
		{ID: "gltf.ext.meshopt-optional", Domain: "compression", Extensions: []string{"EXT_meshopt_compression"}, Expected: "degraded", Evidence: "optional-extension diagnostic fixture"},
		{ID: "gltf.ext.basisu-required", Domain: "compression", Extensions: []string{"KHR_texture_basisu"}, Expected: "unsupported-required", Evidence: "required-extension rejection fixture"},
		{ID: "gltf.ext.unlit", Domain: "materials", Extensions: []string{"KHR_materials_unlit"}, Expected: "equivalent", Evidence: "matrix and material transport tests"},
		{ID: "gltf.ext.spec-gloss", Domain: "materials", Extensions: []string{"KHR_materials_pbrSpecularGlossiness"}, Expected: "migration", Evidence: "explicit metallic-roughness migration verdict"},
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	return GLTFCorpusManifest{Schema: GLTFCorpusSchema, Cases: cases}
}

func InspectGLTF(data []byte, format string) (GLTFInspection, error) {
	rootBytes := data
	if format == "glb" {
		var err error
		rootBytes, err = glbJSONChunk(data)
		if err != nil {
			return GLTFInspection{}, err
		}
	}
	var root struct {
		Asset struct {
			Version string `json:"version"`
		} `json:"asset"`
		ExtensionsUsed     []string `json:"extensionsUsed"`
		ExtensionsRequired []string `json:"extensionsRequired"`
	}
	if err := json.Unmarshal(rootBytes, &root); err != nil {
		return GLTFInspection{}, fmt.Errorf("invalid glTF JSON: %w", err)
	}
	if root.Asset.Version != "2.0" {
		return GLTFInspection{}, fmt.Errorf("glTF 2.0 is required, got %q", root.Asset.Version)
	}
	used := uniqueSortedStrings(root.ExtensionsUsed)
	required := uniqueSortedStrings(root.ExtensionsRequired)
	usedSet := map[string]bool{}
	for _, ext := range used {
		usedSet[ext] = true
	}
	for _, ext := range required {
		if !usedSet[ext] {
			return GLTFInspection{}, fmt.Errorf("required extension %q is absent from extensionsUsed", ext)
		}
	}
	matrix := DefaultGLTFCapabilityMatrix()
	byExtension := map[string]GLTFCapability{}
	for _, capability := range matrix.Entries {
		byExtension[capability.Extension] = capability
	}
	requiredSet := map[string]bool{}
	for _, ext := range required {
		requiredSet[ext] = true
	}
	inspection := GLTFInspection{Schema: GLTFInspectionSchema, Version: root.Asset.Version, ExtensionsUsed: used, ExtensionsRequired: required}
	for _, target := range matrix.Targets {
		verdict := GLTFTargetVerdict{Target: target, Status: "compatible"}
		for _, ext := range used {
			capability, known := byExtension[ext]
			status, evidence := "planned", "extension is not cataloged"
			if known {
				status, evidence = capability.Targets[target], capability.Evidence
			}
			finding := GLTFExtensionFinding{Extension: ext, Required: requiredSet[ext], Status: status, Evidence: evidence}
			verdict.Findings = append(verdict.Findings, finding)
			if requiredSet[ext] && status != "available" {
				verdict.Status = "incompatible"
			} else if verdict.Status != "incompatible" && status != "available" {
				verdict.Status = "degraded"
			}
		}
		inspection.Targets = append(inspection.Targets, verdict)
	}
	canonical, _ := json.Marshal(struct {
		Version        string `json:"version"`
		Used, Required []string
	}{root.Asset.Version, used, required})
	sum := sha256.Sum256(canonical)
	inspection.Fingerprint = hex.EncodeToString(sum[:])
	return inspection, nil
}

func glbJSONChunk(data []byte) ([]byte, error) {
	if len(data) < 20 || string(data[:4]) != "glTF" {
		return nil, fmt.Errorf("invalid GLB header")
	}
	if binary.LittleEndian.Uint32(data[4:8]) != 2 || int(binary.LittleEndian.Uint32(data[8:12])) != len(data) {
		return nil, fmt.Errorf("invalid GLB length or version")
	}
	offset := 12
	for offset+8 <= len(data) {
		length := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length > len(data) {
			return nil, fmt.Errorf("invalid GLB chunk length")
		}
		if kind == 0x4e4f534a {
			return []byte(strings.TrimRight(string(data[offset:offset+length]), " \x00")), nil
		}
		offset += length
	}
	return nil, fmt.Errorf("GLB JSON chunk is required")
}

func uniqueSortedStrings(values []string) []string {
	set := map[string]bool{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			set[v] = true
		}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
