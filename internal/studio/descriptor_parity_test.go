package studio

import (
	"reflect"
	"strings"
	"testing"
)

// ADR 0002: built-in components stay typed Go structs, and the descriptor
// catalog is the published schema contract — enforced here. Every serialized
// field of a core component struct must be described by a catalog field,
// either by its exact JSON name or through the declared grouping alias.
func TestComponentCatalogCoversEverydSerializedStructField(t *testing.T) {
	aliases := map[string]map[string]string{
		"material": {
			"id": "pbr", "name": "pbr", "color": "pbr", "roughness": "pbr", "metalness": "pbr",
			"clearcoat": "pbr", "transmission": "pbr", "emissive": "pbr", "selena": "selena",
		},
		"armature": {"id": "bones", "name": "bones", "rootBones": "bones"},
		"animation": {"id": "tracks", "name": "tracks", "loop": "tracks"},
		"simulation": {"id": "tickRate", "name": "tickRate"},
		"retarget-map": {"id": "bones", "name": "bones", "scale": "bones"},
		"animation-state-machine": {"id": "states", "name": "states", "initial": "states", "current": "states", "stateTime": "states"},
		"render-graph": {"id": "resources", "name": "resources"},
		"physics": {"gravityScale": "kind", "restitution": "collider"},
	}
	structs := map[string]reflect.Type{
		"transform":               reflect.TypeOf(Transform{}),
		"mesh":                    reflect.TypeOf(MeshComponent{}),
		"material":                reflect.TypeOf(Material{}),
		"armature":                reflect.TypeOf(Armature{}),
		"skin":                    reflect.TypeOf(SkinComponent{}),
		"animation":               reflect.TypeOf(AnimationClip{}),
		"physics":                 reflect.TypeOf(PhysicsBody{}),
		"simulation":              reflect.TypeOf(SimulationProfile{}),
		"retarget-map":            reflect.TypeOf(RetargetMap{}),
		"animation-state-machine": reflect.TypeOf(AnimationStateMachine{}),
		"render-graph":            reflect.TypeOf(RenderGraph{}),
	}
	catalog := map[string]map[string]bool{}
	for _, component := range ComponentCatalog() {
		fields := map[string]bool{}
		for _, field := range component.Fields {
			fields[field.Name] = true
		}
		catalog[component.Type] = fields
	}
	for componentType, structType := range structs {
		described, ok := catalog[componentType]
		if !ok {
			t.Errorf("component %q has no catalog entry", componentType)
			continue
		}
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if tag == "" || tag == "-" {
				continue
			}
			if described[tag] {
				continue
			}
			if alias, ok := aliases[componentType][tag]; ok && described[alias] {
				continue
			}
			t.Errorf("component %q field %q (struct %s.%s) is not described by the catalog", componentType, tag, structType.Name(), field.Name)
		}
	}
}
