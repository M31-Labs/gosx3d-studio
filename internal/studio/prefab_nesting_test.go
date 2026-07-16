package studio

import (
	"strings"
	"testing"
)

func nestedPrefabDocument() Document {
	document := SampleDocument()
	document.Prefabs = map[ID]PrefabDefinition{
		"inner": {ID: "inner", Name: "Inner", Root: "inner-root", Entities: map[ID]Entity{
			"inner-root": {ID: "inner-root", Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{Geometry: Geometry{Kind: "box", Width: 0.2, Height: 0.2, Depth: 0.2}, Material: "board-material", Pickable: true}},
		}},
		"outer": {ID: "outer", Name: "Outer", Root: "outer-root", Entities: map[ID]Entity{
			"outer-root": {ID: "outer-root", Transform: IdentityTransform(), Visible: true, Children: []ID{"outer-nested"}},
			"outer-nested": {ID: "outer-nested", Parent: "outer-root", Transform: TransformFromEuler(Vec3{X: 1}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true, Prefab: &PrefabInstance{Prefab: "inner"}},
		}},
	}
	root := document.Entities["scene-root"]
	instance := Entity{ID: "outer-instance", Name: "Outer instance", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Prefab: &PrefabInstance{Prefab: "outer"}}
	root.Children = append(root.Children, instance.ID)
	document.Entities[root.ID] = root
	document.Entities[instance.ID] = instance
	return document
}

func TestNestedPrefabInstancesValidateAndCompileWithNamespacedIDs(t *testing.T) {
	document := nestedPrefabDocument()
	if err := document.Validate(); err != nil {
		t.Fatalf("nested prefab document must validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, object := range props.SceneIR().Objects {
		ids[object.ID] = true
	}
	want := "outer-instance--outer-nested--inner-root"
	if !ids[want] {
		t.Fatalf("nested runtime id %q missing from SceneIR objects: %v", want, ids)
	}
}

func TestPrefabDefinitionCyclesAreRejectedWithConcretePath(t *testing.T) {
	document := nestedPrefabDocument()
	inner := document.Prefabs["inner"]
	entity := inner.Entities["inner-root"]
	entity.Mesh = nil
	entity.Prefab = &PrefabInstance{Prefab: "outer"}
	inner.Entities["inner-root"] = entity
	document.Prefabs["inner"] = inner
	err := document.Validate()
	if err == nil {
		t.Fatal("prefab cycle must fail validation")
	}
	if !strings.Contains(err.Error(), "outer") || !strings.Contains(err.Error(), "inner") {
		t.Fatalf("cycle error must name the concrete path, got %v", err)
	}
}

func TestPrefabVariantInheritsBaseAndAppliesOverrides(t *testing.T) {
	document := nestedPrefabDocument()
	document.Prefabs["variant"] = PrefabDefinition{
		ID: "variant", Name: "Variant of inner", Base: "inner",
		Overrides: map[ID]PrefabEntityOverride{"inner-root": {Material: "player-1-material"}},
	}
	root := document.Entities["scene-root"]
	instance := Entity{ID: "variant-instance", Name: "Variant instance", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Prefab: &PrefabInstance{Prefab: "variant"}}
	root.Children = append(root.Children, instance.ID)
	document.Entities[root.ID] = root
	document.Entities[instance.ID] = instance
	if err := document.Validate(); err != nil {
		t.Fatalf("variant document must validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, object := range props.SceneIR().Objects {
		if object.ID == "variant-instance--inner-root" {
			found = true
		}
	}
	if !found {
		t.Fatalf("variant instance did not compile base entities: %v", props.SceneIR().Objects)
	}
	// Base cycle through variants must also be rejected.
	broken := document
	inner := broken.Prefabs["inner"]
	inner.Base = "variant"
	broken.Prefabs["inner"] = inner
	if err := broken.Validate(); err == nil {
		t.Fatal("variant base cycle must fail validation")
	}
}

func TestPrefabVariantAddsEntitiesIntoInheritedTree(t *testing.T) {
	document := nestedPrefabDocument()
	document.Prefabs["variant-plus"] = PrefabDefinition{
		ID: "variant-plus", Name: "Variant with addition", Base: "inner",
		Entities: map[ID]Entity{
			"added-marker": {ID: "added-marker", Name: "Added marker", Parent: "inner-root", Transform: TransformFromEuler(Vec3{Y: 0.5}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true, Mesh: &MeshComponent{Geometry: Geometry{Kind: "sphere", Radius: 0.1, Segments: 8}, Material: "player-1-material", Pickable: true}},
		},
	}
	root := document.Entities["scene-root"]
	instance := Entity{ID: "plus-instance", Name: "Plus instance", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Prefab: &PrefabInstance{Prefab: "variant-plus"}}
	root.Children = append(root.Children, instance.ID)
	document.Entities[root.ID] = root
	document.Entities[instance.ID] = instance
	if err := document.Validate(); err != nil {
		t.Fatalf("variant with additions must validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, object := range props.SceneIR().Objects {
		if object.ID == "plus-instance--added-marker" {
			found = true
		}
	}
	if !found {
		t.Fatal("variant-added entity did not compile")
	}
	// Collision with a base entity ID must fail.
	broken := document
	variant := broken.Prefabs["variant-plus"]
	variant.Entities["inner-root"] = Entity{ID: "inner-root", Name: "Collision", Parent: "", Transform: IdentityTransform(), Visible: true}
	broken.Prefabs["variant-plus"] = variant
	if err := broken.Validate(); err == nil {
		t.Fatal("variant addition colliding with a base entity id must fail validation")
	}
}
