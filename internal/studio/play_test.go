package studio

import "testing"

func TestPlayModeClonesStepsAndDiscardsOnExit(t *testing.T) {
	document := ArticulatedProofDocument()
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	canonicalBefore, _ := document.Fingerprint()
	if err := workspace.EnterPlay("articulated-physics"); err != nil {
		t.Fatal(err)
	}
	if err := workspace.EnterPlay("articulated-physics"); err == nil {
		t.Fatal("double enter must fail")
	}
	state := workspace.PlayState()
	if !state.Active || state.Simulation != "articulated-physics" {
		t.Fatalf("state=%+v", state)
	}
	startY := document.Entities["physics-payload"].Transform.Position.Y
	if err := workspace.StepPlay(60, nil); err != nil {
		t.Fatal(err)
	}
	playDoc, err := workspace.PlaySnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if playDoc.Entities["physics-payload"].Transform.Position.Y >= startY {
		t.Fatalf("gravity did not move the payload: %v -> %v", startY, playDoc.Entities["physics-payload"].Transform.Position.Y)
	}
	if workspace.PlayState().Tick != 60 {
		t.Fatalf("tick=%d", workspace.PlayState().Tick)
	}
	diff := workspace.PlayDiff()
	if len(diff) == 0 {
		t.Fatal("play diff must list runtime-changed entities")
	}
	if err := workspace.ExitPlay(); err != nil {
		t.Fatal(err)
	}
	if workspace.PlayState().Active {
		t.Fatal("exit must deactivate play")
	}
	canonical, _ := workspace.Snapshot()
	canonicalAfter, _ := canonical.Fingerprint()
	if canonicalBefore != canonicalAfter {
		t.Fatal("play mode mutated the canonical document")
	}
	if err := workspace.StepPlay(1, nil); err == nil {
		t.Fatal("stepping without an active session must fail")
	}
}
