package studio

import (
	"math"
	"reflect"
	"testing"
)

func TestRigAnimationValidationSamplingAndSceneIRLowering(t *testing.T) {
	document := articulatedTestDocument(t)
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	sample, evaluated, err := SampleAnimation(document, "reach", 0.5)
	if err != nil {
		t.Fatal(err)
	}
	// The clip ends at euler {X:2, Y:2}; the halfway sample must be the
	// half-arc rotation around that end rotation's own axis.
	end := QuaternionFromEuler(Vec3{X: 2, Y: 2})
	angle := 2 * math.Acos(clampFloat(end.W, -1, 1))
	sine := math.Sin(angle / 2)
	axis := Vec3{X: end.X / sine, Y: end.Y / sine, Z: end.Z / sine}
	want := axisAngleQuaternion(axis, angle/2)
	got := sample.Transforms["forearm"].Rotation
	for _, probe := range []Vec3{{X: 1}, {Y: 1}, {Z: 1}} {
		vecNear(t, got.Rotate(probe), want.Rotate(probe), 1e-9, "halfway sample rotation")
	}
	if evaluated.Entities["forearm"].Transform.Rotation != got {
		t.Fatalf("evaluated entity rotation = %+v, want %+v", evaluated.Entities["forearm"].Transform.Rotation, got)
	}
	props, err := Compile(evaluated)
	if err != nil {
		t.Fatal(err)
	}
	if len(props.SceneIR().Objects) == 0 {
		t.Fatal("evaluated articulated document produced empty SceneIR")
	}
}

func TestRigActionsSharePreviewDirectReceiptAndUndo(t *testing.T) {
	document := articulatedTestDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	pose := TransformFromEuler(Vec3{Y: 1}, Vec3{Z: 0.4}, Vec3{X: 1, Y: 1, Z: 1})
	key := TransformKey{Time: 0.25, Transform: pose}
	transaction := Transaction{ID: "agent-rig-pass", Actor: "agent://rig-test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{
		{Kind: OpSetBonePose, ArmatureID: "arm", BoneID: "lower", Transform: &pose},
		{Kind: OpSetAnimationKey, ClipID: "reach", TrackID: "lower-track", Key: &key},
	}}
	previewReceipt, preview, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	transaction.Mode = ModeDirect
	directReceipt, direct, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview, direct) {
		t.Fatal("preview and direct rig transactions diverged")
	}
	if len(previewReceipt.RigChanges) != 1 || len(previewReceipt.AnimationChanges) != 1 || len(directReceipt.RigChanges) != 1 || len(directReceipt.AnimationChanges) != 1 {
		t.Fatalf("missing semantic rig receipt: preview=%+v direct=%+v", previewReceipt, directReceipt)
	}
	if direct.Entities["forearm"].Transform != pose {
		t.Fatal("bone pose did not update bound articulated entity")
	}
	undo, restored, err := workspace.Undo(direct.Revision, "human://tester")
	if err != nil {
		t.Fatal(err)
	}
	if !undo.Applied || restored.Entities["forearm"].Transform == pose {
		t.Fatal("undo did not restore articulated pose")
	}
}

func TestTwoBoneIKIsDeterministicReachableAndCommandDriven(t *testing.T) {
	document := articulatedTestDocument(t)
	first, firstDocument, err := SolveTwoBoneIK(document, "arm", "reach-ik")
	if err != nil {
		t.Fatal(err)
	}
	second, secondDocument, err := SolveTwoBoneIK(document, "arm", "reach-ik")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstDocument, secondDocument) {
		t.Fatal("two-bone IK is not deterministic")
	}
	if first.Error > 1e-8 {
		t.Fatalf("reachable IK target error = %.12f", first.Error)
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	receipt, solved, err := workspace.Execute(Transaction{ID: "solve-reach", Actor: "agent://rig-test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSolveIK, ArmatureID: "arm", ConstraintID: "reach-ik"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.RigChanges) != 2 || solved.Entities["forearm"].Transform.Rotation.IsIdentity() {
		t.Fatalf("IK command evidence = %+v", receipt)
	}
}

func TestSkinRejectsUnnormalizedWeights(t *testing.T) {
	document := articulatedTestDocument(t)
	document.Entities["skinned"].Skin.Weights["v0"] = []VertexInfluence{{Bone: "root", Weight: 0.8}}
	if err := document.Validate(); err == nil {
		t.Fatal("expected unnormalized weight validation failure")
	}
}

func articulatedTestDocument(t *testing.T) Document {
	t.Helper()
	return ArticulatedProofDocument()
}
