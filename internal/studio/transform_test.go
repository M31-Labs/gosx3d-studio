package studio

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestTransformLegacyJSONMigratesEulerToQuaternion(t *testing.T) {
	legacy := []byte(`{"position":{"x":1,"y":2,"z":3},"rotation":{"x":0,"y":0,"z":0.35},"scale":{"x":1,"y":1,"z":1}}`)
	var migrated Transform
	if err := json.Unmarshal(legacy, &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.Rotation != QuaternionFromEuler(Vec3{Z: 0.35}) {
		t.Fatalf("legacy euler must migrate to its quaternion, got %+v", migrated.Rotation)
	}
	if migrated.Euler != (Vec3{Z: 0.35}) {
		t.Fatalf("legacy euler must remain as display metadata, got %+v", migrated.Euler)
	}
}

func TestTransformMarshalCanonicalizesZeroQuaternion(t *testing.T) {
	literal := Transform{Position: Vec3{X: 1}, Euler: Vec3{Z: 0.2}, Scale: Vec3{X: 1, Y: 1, Z: 1}}
	data, err := json.Marshal(literal)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Transform
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Rotation != QuaternionFromEuler(Vec3{Z: 0.2}) {
		t.Fatalf("zero-quaternion literal must marshal canonically, got %+v", decoded.Rotation)
	}
	again, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(again) {
		t.Fatalf("canonical form must be marshal-stable:\n%s\n%s", data, again)
	}
}

func TestValidateRejectsNonNormalizedQuaternion(t *testing.T) {
	document := SampleDocument()
	entity := document.Entities["board"]
	entity.Transform.Rotation = Quaternion{X: 3, W: 2}
	document.Entities["board"] = entity
	err := document.Validate()
	if err == nil || !strings.Contains(err.Error(), "board") {
		t.Fatalf("non-normalized quaternion must fail validation, got %v", err)
	}
}

func TestValidateRejectsGroupScaleMatchingCompile(t *testing.T) {
	document := SampleDocument()
	entity := document.Entities["scene-root"]
	entity.Transform.Scale = Vec3{X: 2, Y: 1, Z: 1}
	document.Entities["scene-root"] = entity
	err := document.Validate()
	if err == nil || !strings.Contains(err.Error(), "scale-free") {
		t.Fatalf("validation must reject group scale exactly like compilation, got %v", err)
	}
}

func TestEulerAndQuaternionAuthoringCompileEquivalently(t *testing.T) {
	euler := Vec3{X: 0.3, Y: -0.6, Z: 1.1}

	viaEuler := SampleDocument()
	entity := viaEuler.Entities["board"]
	entity.Transform = TransformFromEuler(entity.Transform.Position, euler, entity.Transform.Scale)
	viaEuler.Entities["board"] = entity

	viaQuaternion := SampleDocument()
	entity = viaQuaternion.Entities["board"]
	entity.Transform.Rotation = QuaternionFromEuler(euler)
	entity.Transform.Euler = Vec3{}
	viaQuaternion.Entities["board"] = entity

	first, err := Compile(viaEuler)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Compile(viaQuaternion)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.SceneIR(), second.SceneIR()) {
		t.Fatal("euler-authored and quaternion-authored rotations must lower to identical SceneIR")
	}
}

func TestMeshEntityScaleCompilesThroughSceneIR(t *testing.T) {
	document := SampleDocument()
	entity := document.Entities["board"]
	entity.Transform.Scale = Vec3{X: 2, Y: 1, Z: 1.5}
	document.Entities["board"] = entity
	if err := document.Validate(); err != nil {
		t.Fatalf("mesh scale must validate since gosx v0.31.18: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatalf("mesh scale must compile: %v", err)
	}
	found := false
	for _, object := range props.SceneIR().Objects {
		if object.ID == "board" {
			found = true
			if object.ScaleX != 2 || object.ScaleZ != 1.5 {
				t.Fatalf("board scale = %v,%v,%v", object.ScaleX, object.ScaleY, object.ScaleZ)
			}
		}
	}
	if !found {
		t.Fatal("board missing from SceneIR")
	}
}

func TestGroupEntityScaleStaysAnHonestyGate(t *testing.T) {
	document := SampleDocument()
	entity := document.Entities["scene-root"]
	entity.Transform.Scale = Vec3{X: 2, Y: 2, Z: 2}
	document.Entities["scene-root"] = entity
	if err := document.Validate(); err == nil {
		t.Fatal("group scale must stay rejected: engine groups are scale-free by design")
	}
}
