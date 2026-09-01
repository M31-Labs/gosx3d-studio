package studio

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx/scene/preview"
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

func TestValidateAcceptsGroupScaleAndRejectsLightScale(t *testing.T) {
	document := SampleDocument()
	entity := document.Entities["scene-root"]
	entity.Transform.Scale = Vec3{X: 2, Y: 1, Z: 1}
	document.Entities["scene-root"] = entity
	if err := document.Validate(); err != nil {
		t.Fatalf("v0.54 group scale must validate: %v", err)
	}

	light := document.Entities["key-light"]
	light.Transform.Scale = Vec3{X: 2, Y: 1, Z: 1}
	document.Entities[light.ID] = light
	if err := document.Validate(); err == nil || !strings.Contains(err.Error(), "light scale has no render meaning") {
		t.Fatalf("meaningless light scale must remain rejected, got %v", err)
	}
	if _, err := Compile(document); err == nil || !strings.Contains(err.Error(), "light scale has no render meaning") {
		t.Fatalf("compiler must reject the same meaningless light scale, got %v", err)
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

func TestGroupEntityScaleMatchesSceneIRRaycastAndPreview(t *testing.T) {
	document := SampleDocument()
	root := document.Entities["scene-root"]
	root.Transform = TransformFromEuler(Vec3{X: 3, Y: -2, Z: 5}, Vec3{}, Vec3{X: 2, Y: 3, Z: 4})
	root.Children = []ID{"scaled-child"}
	child := Entity{
		ID: "scaled-child", Name: "Scaled child", Parent: root.ID, Visible: true,
		Transform: TransformFromEuler(Vec3{X: 1, Y: 2, Z: 3}, Vec3{}, Vec3{X: 5, Y: 6, Z: 7}),
		Mesh: &MeshComponent{
			Geometry: Geometry{Kind: "box", Width: 1, Height: 1, Depth: 1},
			Material: "board-material", Pickable: true,
		},
	}
	document.Entities = map[ID]Entity{root.ID: root, child.ID: child}

	if err := document.Validate(); err != nil {
		t.Fatalf("scaled hierarchy must validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatalf("scaled hierarchy must compile: %v", err)
	}
	ir := props.SceneIR()
	if len(ir.Objects) != 1 || ir.Objects[0].ID != string(child.ID) {
		t.Fatalf("scaled SceneIR objects = %+v", ir.Objects)
	}
	object := ir.Objects[0]
	assertFloatSliceClose(t, object.ParentMatrix, []float64{
		2, 0, 0, 0,
		0, 3, 0, 0,
		0, 0, 4, 0,
		3, -2, 5, 1,
	})
	if object.X != 1 || object.Y != 2 || object.Z != 3 || object.ScaleX != 5 || object.ScaleY != 6 || object.ScaleZ != 7 {
		t.Fatalf("SceneIR lost authored local transform: %+v", object)
	}

	frame := preview.Bundle(props, preview.Options{})
	if len(frame.InstancedMeshes) != 1 {
		t.Fatalf("preview meshes = %d, want one", len(frame.InstancedMeshes))
	}
	assertFloatSliceClose(t, frame.InstancedMeshes[0].Transforms, []float64{
		10, 0, 0, 0,
		0, 18, 0, 0,
		0, 0, 28, 0,
		5, 4, 17, 1,
	})

	trace := scene.TraceGraph(props.Graph, scene.Ray{
		Origin: scene.Vec3(5, 4, 40), Direction: scene.Vec3(0, 0, -1),
	}, scene.PickableOnly())
	if trace.Closest == nil || trace.Closest.ID != string(child.ID) {
		t.Fatalf("scaled ray trace missed child: %+v", trace)
	}
	if math.Abs(trace.Closest.Point.X-5) > 1e-9 || math.Abs(trace.Closest.Point.Y-4) > 1e-9 || math.Abs(trace.Closest.Point.Z-31) > 1e-9 {
		t.Fatalf("scaled ray hit = %+v, want point (5,4,31)", trace.Closest)
	}
}

func assertFloatSliceClose(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("float slice length = %d, want %d (%v)", len(got), len(want), got)
	}
	for index := range want {
		if math.Abs(got[index]-want[index]) > 1e-9 {
			t.Fatalf("float slice[%d] = %.15g, want %.15g (all=%v)", index, got[index], want[index], got)
		}
	}
}
