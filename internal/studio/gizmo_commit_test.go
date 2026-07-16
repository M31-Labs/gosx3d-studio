package studio

import (
	"math"
	"testing"
)

func TestApplyGizmoCommitTranslateIsOneUndoStep(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	before := document.Entities[target].Transform
	receipt, err := ApplyGizmoCommit(workspace, GizmoCommit{Target: target, Mode: "translate", Position: &Vec3{X: 1.5, Y: 0.405, Z: -2}})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Actor != "human://viewport-gizmo" || receipt.Operations != 1 {
		t.Fatalf("receipt=%+v", receipt)
	}
	changed, _ := workspace.Snapshot()
	if changed.Entities[target].Transform.Position.X != 1.5 {
		t.Fatalf("position=%+v", changed.Entities[target].Transform.Position)
	}
	if _, restored, err := workspace.Undo(changed.Revision, "human://tester"); err != nil {
		t.Fatal(err)
	} else if restored.Entities[target].Transform != before {
		t.Fatal("one undo must restore the pre-drag transform")
	}
}

func TestApplyGizmoCommitRotateComposesQuaternionDelta(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	angle := 0.35
	if _, err := ApplyGizmoCommit(workspace, GizmoCommit{Target: target, Mode: "rotate", AngleDelta: &angle}); err != nil {
		t.Fatal(err)
	}
	changed, _ := workspace.Snapshot()
	got := changed.Entities[target].Transform.Rotation
	want := QuaternionFromEuler(Vec3{Z: angle})
	for _, probe := range []Vec3{{X: 1}, {Y: 1}} {
		gotVec, wantVec := got.Rotate(probe), want.Rotate(probe)
		if math.Abs(gotVec.X-wantVec.X) > 1e-9 || math.Abs(gotVec.Y-wantVec.Y) > 1e-9 || math.Abs(gotVec.Z-wantVec.Z) > 1e-9 {
			t.Fatalf("rotation = %+v, want Rz(%v)", got, angle)
		}
	}
}

func TestApplyGizmoCommitScalePerAxisOnMeshes(t *testing.T) {
	document := SampleDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	factor := 2.0
	if _, err := ApplyGizmoCommit(workspace, GizmoCommit{Target: target, Mode: "scale", Axis: "x", ScaleFactor: &factor}); err != nil {
		t.Fatalf("mesh scale commit must succeed since gosx v0.31.18: %v", err)
	}
	changed, _ := workspace.Snapshot()
	scale := changed.Entities[target].Transform.Scale
	if scale.X != 2 || scale.Y != 1 || scale.Z != 1 {
		t.Fatalf("scale = %+v, want x doubled only", scale)
	}
	if _, err := Compile(changed); err != nil {
		t.Fatalf("scaled document must compile: %v", err)
	}
	if _, err := ApplyGizmoCommit(workspace, GizmoCommit{Target: "scene-root", Mode: "scale", Axis: "x", ScaleFactor: &factor}); err == nil {
		t.Fatal("group scale commits must stay rejected")
	}
}
