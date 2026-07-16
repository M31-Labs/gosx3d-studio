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
