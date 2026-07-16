package studio

import (
	"encoding/json"
	"fmt"
	"strings"
)

func applyCapturePrefab(document *Document, operation Operation) ([]ID, error) {
	if operation.PrefabID == "" || strings.TrimSpace(operation.Name) == "" || operation.Target == "" {
		return nil, fmt.Errorf("capture-prefab requires target, prefabId, and name")
	}
	if _, ok := document.Entities[operation.Target]; !ok {
		return nil, fmt.Errorf("prefab source entity %q does not exist", operation.Target)
	}
	entities := map[ID]Entity{}
	var capture func(ID) error
	capture = func(id ID) error {
		entity := document.Entities[id]
		if entity.Prefab != nil {
			return fmt.Errorf("nested prefab source %q is not supported", id)
		}
		copy, err := cloneEntity(entity)
		if err != nil {
			return err
		}
		entities[id] = copy
		for _, child := range entity.Children {
			if err := capture(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := capture(operation.Target); err != nil {
		return nil, err
	}
	root := entities[operation.Target]
	root.Parent = ""
	entities[root.ID] = root
	definition := PrefabDefinition{ID: operation.PrefabID, Name: strings.TrimSpace(operation.Name), Root: operation.Target, Entities: entities}
	if err := validatePrefabDefinition(definition, document.Materials, document.Assets, document.Prefabs); err != nil {
		return nil, err
	}
	if document.Prefabs == nil {
		document.Prefabs = map[ID]PrefabDefinition{}
	}
	document.Prefabs[definition.ID] = definition
	return []ID{definition.ID}, nil
}

func applyInstantiatePrefab(document *Document, operation Operation) ([]ID, error) {
	if operation.PrefabID == "" || operation.NewID == "" {
		return nil, fmt.Errorf("instantiate-prefab requires prefabId and newId")
	}
	definition, ok := document.Prefabs[operation.PrefabID]
	if !ok {
		return nil, fmt.Errorf("prefab %q does not exist", operation.PrefabID)
	}
	if _, exists := document.Entities[operation.NewID]; exists {
		return nil, fmt.Errorf("entity %q already exists", operation.NewID)
	}
	transform := IdentityTransform()
	if operation.Transform != nil {
		transform = *operation.Transform
	}
	if !finiteTransform(transform) {
		return nil, fmt.Errorf("prefab instance transform must be finite")
	}
	entity := Entity{ID: operation.NewID, Name: definition.Name, Parent: operation.Parent, Transform: transform, Visible: true, Prefab: &PrefabInstance{Prefab: definition.ID, Overrides: map[ID]PrefabEntityOverride{}}}
	if entity.Parent == "" {
		document.RootIDs = append(document.RootIDs, entity.ID)
	} else {
		parent, ok := document.Entities[entity.Parent]
		if !ok {
			return nil, fmt.Errorf("parent %q does not exist", entity.Parent)
		}
		parent.Children = append(parent.Children, entity.ID)
		document.Entities[parent.ID] = parent
	}
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applySetPrefabOverride(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Prefab == nil {
		return nil, fmt.Errorf("prefab override target %q is not an instance", operation.Target)
	}
	definition := document.Prefabs[entity.Prefab.Prefab]
	if _, ok := definition.Entities[operation.PrefabEntity]; !ok {
		return nil, fmt.Errorf("prefab entity %q does not exist", operation.PrefabEntity)
	}
	if entity.Prefab.Overrides == nil {
		entity.Prefab.Overrides = map[ID]PrefabEntityOverride{}
	}
	if operation.PrefabOverride == nil {
		delete(entity.Prefab.Overrides, operation.PrefabEntity)
	} else {
		override := *operation.PrefabOverride
		if override.Transform != nil && !finiteTransform(*override.Transform) {
			return nil, fmt.Errorf("prefab override transform must be finite")
		}
		if override.Material != "" {
			if _, ok := document.Materials[override.Material]; !ok {
				return nil, fmt.Errorf("material %q does not exist", override.Material)
			}
			if definition.Entities[operation.PrefabEntity].Mesh == nil {
				return nil, fmt.Errorf("prefab entity %q has no material", operation.PrefabEntity)
			}
		}
		entity.Prefab.Overrides[operation.PrefabEntity] = override
	}
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyDeletePrefab(document *Document, operation Operation) ([]ID, error) {
	if operation.PrefabID == "" {
		return nil, fmt.Errorf("delete-prefab requires prefabId")
	}
	if _, ok := document.Prefabs[operation.PrefabID]; !ok {
		return nil, fmt.Errorf("prefab %q does not exist", operation.PrefabID)
	}
	for _, entity := range document.Entities {
		if entity.Prefab != nil && entity.Prefab.Prefab == operation.PrefabID {
			return nil, fmt.Errorf("prefab %q is referenced by instance %q", operation.PrefabID, entity.ID)
		}
	}
	delete(document.Prefabs, operation.PrefabID)
	return []ID{operation.PrefabID}, nil
}

func cloneEntity(entity Entity) (Entity, error) {
	data, err := json.Marshal(entity)
	if err != nil {
		return Entity{}, err
	}
	var clone Entity
	if err := json.Unmarshal(data, &clone); err != nil {
		return Entity{}, err
	}
	return clone, nil
}
