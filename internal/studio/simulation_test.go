package studio

import (
	"reflect"
	"testing"
)

func TestFixedStepPhysicsRecordsContactsAndReplaysExactly(t *testing.T) {
	document := ArticulatedProofDocument()
	inputs := []SimulationInput{{Tick: 30, Entity: "physics-payload", Impulse: Vec3{X: 1.5}}}
	first, firstDocument, err := RunSimulation(document, "articulated-physics", 120, inputs)
	if err != nil {
		t.Fatal(err)
	}
	second, secondDocument, err := RunSimulation(document, "articulated-physics", 120, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstDocument, secondDocument) {
		t.Fatal("fixed-step simulation is not deterministic")
	}
	if len(first.Events) == 0 || first.Events[0].Kind != "contact" {
		t.Fatalf("simulation events = %+v", first.Events)
	}
	if first.Final.Hash == first.Initial.Hash {
		t.Fatal("simulation did not change state")
	}
	if firstDocument.Entities["physics-payload"].Transform.Position.Y < 0.199999 {
		t.Fatalf("payload penetrated ground: %+v", firstDocument.Entities["physics-payload"].Transform.Position)
	}
	replayed, replayDocument, err := ReplaySimulation(document, first)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Final.Hash != first.Final.Hash || !reflect.DeepEqual(replayDocument, firstDocument) {
		t.Fatal("recording replay diverged")
	}
}

func TestSimulationCommandHasPreviewDirectParityReceiptAndUndo(t *testing.T) {
	document := ArticulatedProofDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	transaction := Transaction{ID: "agent-physics-step", Actor: "agent://simulation-test", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSimulateTicks, SimulationID: "articulated-physics", Ticks: 60, Inputs: []SimulationInput{{Tick: 0, Entity: "physics-payload", Impulse: Vec3{Z: 0.5}}}}}}
	previewReceipt, preview, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	transaction.Mode = ModeDirect
	directReceipt, direct, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview, direct) || len(previewReceipt.SimulationChanges) != 1 || len(directReceipt.SimulationChanges) != 1 {
		t.Fatalf("simulation command receipts preview=%+v direct=%+v", previewReceipt, directReceipt)
	}
	if directReceipt.SimulationChanges[0].BeforeHash == directReceipt.SimulationChanges[0].AfterHash {
		t.Fatal("simulation receipt hashes did not change")
	}
	_, restored, err := workspace.Undo(direct.Revision, "human://tester")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Entities["physics-payload"].Transform != document.Entities["physics-payload"].Transform {
		t.Fatal("simulation undo did not restore authored state")
	}
}

func TestSimulationRejectsInvalidInputAndDivergentReplay(t *testing.T) {
	document := ArticulatedProofDocument()
	if _, _, err := RunSimulation(document, "articulated-physics", 10, []SimulationInput{{Tick: 10, Entity: "physics-payload"}}); err == nil {
		t.Fatal("accepted input outside recording tick range")
	}
	recording, _, err := RunSimulation(document, "articulated-physics", 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	recording.Final.Hash = "tampered"
	if _, _, err := ReplaySimulation(document, recording); err == nil {
		t.Fatal("accepted divergent replay")
	}
}
