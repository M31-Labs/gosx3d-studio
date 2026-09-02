package studio

import (
	"testing"

	"m31labs.dev/gosx/scene"
)

func findTransformControls(nodes []scene.Node) *scene.TransformControls {
	for _, node := range nodes {
		if controls, ok := node.(scene.TransformControls); ok {
			return &controls
		}
	}
	return nil
}

func TestCompileSelectedEmitsTransformControlsForSelection(t *testing.T) {
	document := SampleDocument()
	target, _ := FirstPickTarget(document)
	props, err := CompileSelected(document, target)
	if err != nil {
		t.Fatal(err)
	}
	if props.GizmoInputSignal != "studio.viewport.gizmoMode" {
		t.Fatalf("gizmo input signal = %q", props.GizmoInputSignal)
	}
	controls := findTransformControls(props.Graph.Nodes)
	if controls == nil {
		t.Fatal("selection must emit a TransformControls helper")
	}
	if controls.Target != string(target) || controls.Mode != "translate" {
		t.Fatalf("gizmo target=%q mode=%q", controls.Target, controls.Mode)
	}
	entity := document.Entities[target]
	if controls.Position != toSceneVec(entity.Transform.Position) {
		t.Fatalf("gizmo position %+v does not track the selected entity %+v", controls.Position, entity.Transform.Position)
	}
}

func TestCompileWithoutSelectionEmitsNoGizmo(t *testing.T) {
	document := SampleDocument()
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	if controls := findTransformControls(props.Graph.Nodes); controls != nil {
		t.Fatalf("unselected compile must not emit a gizmo, got %+v", controls)
	}
}

func TestCompileViewportUsesStableSignalDrivenGizmo(t *testing.T) {
	document := SampleDocument()
	props, err := CompileViewport(document)
	if err != nil {
		t.Fatal(err)
	}
	controls := findTransformControls(props.Graph.Nodes)
	if controls == nil {
		t.Fatal("interactive viewport must always lower stable TransformControls helpers")
	}
	if controls.ID != "studio-gizmo" || controls.Target != "" || controls.Mode != "" || controls.Size != 1.2 {
		t.Fatalf("signal-driven viewport gizmo = %+v", controls)
	}
	if props.SelectionInputSignal != "studio.viewport.selectedID" {
		t.Fatalf("selection input signal = %q", props.SelectionInputSignal)
	}
	for _, object := range props.SceneIR().Objects {
		if object.Selected {
			t.Fatalf("viewport object %q bakes selection into the persistent scene", object.ID)
		}
	}
}

func TestCompilePropsExposeCameraRigSignals(t *testing.T) {
	props, err := Compile(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	if props.CameraInputSignal != "studio.viewport.cameraIn" || props.CameraOutputSignal != "studio.viewport.cameraOut" {
		t.Fatalf("camera rig signals missing: in=%q out=%q", props.CameraInputSignal, props.CameraOutputSignal)
	}
}
