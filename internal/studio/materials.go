package studio

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx/scene"
)

func applySetMaterial(document *Document, operation Operation) ([]ID, error) {
	if operation.MaterialRecord == nil || operation.MaterialRecord.ID == "" {
		return nil, fmt.Errorf("set-material requires materialRecord with id")
	}
	material := *operation.MaterialRecord
	if err := validateMaterialRecord(material, document.Assets); err != nil {
		return nil, err
	}
	if material.Selena != nil {
		standard := scene.StandardMaterial{Color: material.Color, Roughness: material.Roughness, Metalness: material.Metalness, Clearcoat: material.Clearcoat, Transmission: material.Transmission, Emissive: material.Emissive}
		if _, _, err := scene.CompileSelenaMaterial([]byte(material.Selena.Source), scene.SelenaMaterialOptions{Material: material.Selena.Material, Standard: standard}); err != nil {
			return nil, fmt.Errorf("material %q Selena validation: %w", material.ID, err)
		}
	}
	document.Materials[material.ID] = material
	return []ID{material.ID}, nil
}

func applyDeleteMaterial(document *Document, operation Operation) ([]ID, error) {
	if operation.MaterialID == "" {
		return nil, fmt.Errorf("delete-material requires materialId")
	}
	if _, ok := document.Materials[operation.MaterialID]; !ok {
		return nil, fmt.Errorf("material %q does not exist", operation.MaterialID)
	}
	for _, entity := range document.Entities {
		if entity.Mesh != nil && entity.Mesh.Material == operation.MaterialID {
			return nil, fmt.Errorf("material %q is referenced by entity %q", operation.MaterialID, entity.ID)
		}
	}
	delete(document.Materials, operation.MaterialID)
	return []ID{operation.MaterialID}, nil
}

var materialTextureChannels = map[string]bool{"color": true, "normal": true, "roughness": true, "metalness": true, "emissive": true}

func validateMaterialRecord(material Material, assets map[ID]AssetRecord) error {
	if material.ID == "" || strings.TrimSpace(material.Name) == "" || strings.TrimSpace(material.Color) == "" {
		return fmt.Errorf("material id, name, and color are required")
	}
	values := []struct {
		name  string
		value float64
	}{{"roughness", material.Roughness}, {"metalness", material.Metalness}, {"clearcoat", material.Clearcoat}, {"transmission", material.Transmission}}
	for _, field := range values {
		if !finite(field.value) || field.value < 0 || field.value > 1 {
			return fmt.Errorf("material %q %s must be within [0,1]", material.ID, field.name)
		}
	}
	if !finite(material.Emissive) || material.Emissive < 0 {
		return fmt.Errorf("material %q emissive must be finite and non-negative", material.ID)
	}
	if material.Selena != nil && (strings.TrimSpace(material.Selena.Material) == "" || strings.TrimSpace(material.Selena.Source) == "") {
		return fmt.Errorf("material %q has incomplete Selena source", material.ID)
	}
	for channel, slot := range material.Textures {
		if !materialTextureChannels[channel] {
			return fmt.Errorf("material %q texture channel %q is not supported (color, normal, roughness, metalness, emissive)", material.ID, channel)
		}
		asset, ok := assets[slot.Asset]
		if !ok {
			return fmt.Errorf("material %q texture %q references missing asset %q", material.ID, channel, slot.Asset)
		}
		if asset.Kind != "image" {
			return fmt.Errorf("material %q texture %q references non-image asset %q (kind %q)", material.ID, channel, slot.Asset, asset.Kind)
		}
		if slot.ColorSpace != "" && slot.ColorSpace != "srgb" && slot.ColorSpace != "linear" {
			return fmt.Errorf("material %q texture %q color space %q must be srgb or linear", material.ID, channel, slot.ColorSpace)
		}
	}
	return nil
}
