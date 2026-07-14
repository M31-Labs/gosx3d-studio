package studio

import (
	"reflect"
	"testing"
)

func TestRetargetAnimationUsesStableMapAndRestRelativeScale(t *testing.T) {
	document := ArticulatedProofDocument()
	clip, err := RetargetAnimationClip(document, "arm-to-tall", "reach", "tall-reach", "Tall Reach")
	if err != nil {
		t.Fatal(err)
	}
	track, ok := clip.Tracks["tall-reach--target-lower"]
	if !ok || track.Armature != "target-arm" || track.Bone != "target-lower" || len(track.Keys) != 2 {
		t.Fatalf("retargeted track = %+v", track)
	}
	if track.Keys[0].Transform.Position.Y != 2 || track.Keys[1].Transform.Rotation != document.Animations["reach"].Tracks["lower-track"].Keys[1].Transform.Rotation {
		t.Fatalf("retargeted keys = %+v", track.Keys)
	}
	if _, err := RetargetAnimationClip(document, "arm-to-tall", "reach", "", "missing id"); err == nil {
		t.Fatal("retarget accepted empty output id")
	}
}

func TestArbiterGovernedStateMachineProducesDeterministicTraceAndSample(t *testing.T) {
	document := ArticulatedProofDocument()
	first, firstDocument, err := StepAnimationMachine(document, "locomotion", 0.25)
	if err != nil {
		t.Fatal(err)
	}
	second, secondDocument, err := StepAnimationMachine(document, "locomotion", 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstDocument, secondDocument) {
		t.Fatal("state-machine step is not deterministic")
	}
	if first.From != "idle" || first.To != "idle" || len(first.Trace) != 1 || first.Trace[0].Eligible {
		t.Fatalf("idle trace = %+v", first)
	}
	machine := document.AnimationMachines["locomotion"]
	machine.Parameters["speed"] = 1
	document.AnimationMachines["locomotion"] = machine
	transitioned, _, err := StepAnimationMachine(document, "locomotion", 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if transitioned.To != "reach" || transitioned.Transition != "idle-to-reach" || len(transitioned.Trace) != 1 || transitioned.Trace[0].Rule != "TransitionGreater" || !transitioned.Trace[0].Eligible {
		t.Fatalf("transition trace = %+v", transitioned)
	}
}

func TestRetargetAndStateMachineActionsSharePreviewDirectReceiptsAndUndo(t *testing.T) {
	document := ArticulatedProofDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	retarget := Transaction{ID: "agent-retarget", Actor: "agent://animation-test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpRetargetAnimation, RetargetMapID: "arm-to-tall", SourceClipID: "reach", NewID: "tall-reach", Name: "Tall Reach"}}}
	previewReceipt, preview, err := workspace.Execute(retarget)
	if err != nil {
		t.Fatal(err)
	}
	retarget.Mode = ModeDirect
	directReceipt, direct, err := workspace.Execute(retarget)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview, direct) || len(previewReceipt.RetargetChanges) != 1 || len(directReceipt.RetargetChanges) != 1 {
		t.Fatalf("retarget receipts = %+v %+v", previewReceipt, directReceipt)
	}
	transaction := Transaction{ID: "agent-machine-step", Actor: "agent://animation-test", Mode: ModePropose, ExpectedRevision: direct.Revision, Operations: []Operation{{Kind: OpSetAnimationParameter, MachineID: "locomotion", Parameter: "speed", Number: 1}, {Kind: OpStepAnimationMachine, MachineID: "locomotion", DeltaTime: 0.25}}}
	machinePreviewReceipt, machinePreview, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	transaction.Mode = ModeDirect
	machineDirectReceipt, machineDirect, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(machinePreview, machineDirect) || machineDirect.AnimationMachines["locomotion"].Current != "reach" || len(machinePreviewReceipt.MachineChanges) != 2 || len(machineDirectReceipt.MachineChanges) != 2 {
		t.Fatalf("machine receipts/state = %+v %+v", machineDirectReceipt, machineDirect.AnimationMachines["locomotion"])
	}
	_, restored, err := workspace.Undo(machineDirect.Revision, "human://tester")
	if err != nil {
		t.Fatal(err)
	}
	if restored.AnimationMachines["locomotion"].Current != "idle" {
		t.Fatal("state-machine undo did not restore state")
	}
}
