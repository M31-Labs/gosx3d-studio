package studio

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestGLTFCompatibilityRequiredAndOptionalExtensions(t *testing.T) {
	payload := []byte(`{"asset":{"version":"2.0"},"extensionsUsed":["KHR_materials_unlit","KHR_draco_mesh_compression","EXT_meshopt_compression"],"extensionsRequired":["KHR_draco_mesh_compression"]}`)
	inspection, err := InspectGLTF(payload, "gltf")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inspection.ExtensionsUsed, []string{"EXT_meshopt_compression", "KHR_draco_mesh_compression", "KHR_materials_unlit"}) {
		t.Fatalf("extensions=%v", inspection.ExtensionsUsed)
	}
	for _, verdict := range inspection.Targets {
		if verdict.Status != "incompatible" {
			t.Fatalf("%s status=%s", verdict.Target, verdict.Status)
		}
	}
	repeated, _ := InspectGLTF(payload, "gltf")
	if repeated.Fingerprint != inspection.Fingerprint {
		t.Fatal("compatibility fingerprint is not deterministic")
	}

	optional := []byte(`{"asset":{"version":"2.0"},"extensionsUsed":["EXT_meshopt_compression"]}`)
	inspection, err = InspectGLTF(optional, "gltf")
	if err != nil {
		t.Fatal(err)
	}
	for _, verdict := range inspection.Targets {
		if verdict.Status != "degraded" {
			t.Fatalf("optional %s status=%s", verdict.Target, verdict.Status)
		}
	}
}

func TestGLBJSONInspectionAndRequiredSubsetValidation(t *testing.T) {
	payload := []byte(`{"asset":{"version":"2.0"},"extensionsUsed":["KHR_materials_unlit"],"extensionsRequired":["KHR_materials_unlit"]}`)
	for len(payload)%4 != 0 {
		payload = append(payload, ' ')
	}
	glb := make([]byte, 20+len(payload))
	copy(glb[:4], "glTF")
	binary.LittleEndian.PutUint32(glb[4:8], 2)
	binary.LittleEndian.PutUint32(glb[8:12], uint32(len(glb)))
	binary.LittleEndian.PutUint32(glb[12:16], uint32(len(payload)))
	copy(glb[16:20], "JSON")
	copy(glb[20:], payload)
	inspection, err := InspectGLTF(glb, "glb")
	if err != nil {
		t.Fatal(err)
	}
	for _, verdict := range inspection.Targets {
		if verdict.Status != "compatible" {
			t.Fatalf("%s status=%s", verdict.Target, verdict.Status)
		}
	}
	if _, err := InspectGLTF([]byte(`{"asset":{"version":"2.0"},"extensionsRequired":["KHR_materials_unlit"]}`), "gltf"); err == nil {
		t.Fatal("required extension absent from used must fail")
	}
}

func TestGLTFMatrixAndCorpusHaveStableCompleteIdentity(t *testing.T) {
	matrix := DefaultGLTFCapabilityMatrix()
	seen := map[string]bool{}
	for _, entry := range matrix.Entries {
		if seen[entry.Extension] || entry.Extension == "" {
			t.Fatalf("duplicate/empty extension %q", entry.Extension)
		}
		seen[entry.Extension] = true
		for _, target := range matrix.Targets {
			if entry.Targets[target] == "" {
				t.Fatalf("%s lacks %s verdict", entry.Extension, target)
			}
		}
	}
	for _, required := range []string{"KHR_draco_mesh_compression", "EXT_meshopt_compression", "KHR_texture_basisu", "KHR_lights_punctual", "KHR_materials_unlit"} {
		if !seen[required] {
			t.Fatalf("matrix lacks %s", required)
		}
	}
	cases := DefaultGLTFCorpusManifest().Cases
	caseIDs := map[string]bool{}
	for _, item := range cases {
		if caseIDs[item.ID] || item.ID == "" || item.Domain == "" || item.Expected == "" || item.Evidence == "" {
			t.Fatalf("invalid corpus case %#v", item)
		}
		caseIDs[item.ID] = true
	}
}

func TestGLTFImportPersistsCompatibilityVerdictMetadata(t *testing.T) {
	payload := []byte(`{"asset":{"version":"2.0"},"extensionsUsed":["KHR_draco_mesh_compression"],"extensionsRequired":["KHR_draco_mesh_compression"]}`)
	asset, err := inspectAsset("compressed.gltf", payload)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Metadata["gltfMatrixSchema"] != GLTFMatrixSchema || asset.Metadata["gltfInspectionSchema"] != GLTFInspectionSchema || asset.Metadata["gltfTarget.native-headless"] != "incompatible" || asset.Metadata["gltfExtensionsRequired"] != "KHR_draco_mesh_compression" || asset.Metadata["gltfCompatibilityFingerprint"] == "" {
		t.Fatalf("compatibility metadata=%#v", asset.Metadata)
	}
}
