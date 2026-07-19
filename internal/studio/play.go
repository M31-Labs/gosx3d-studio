package studio

import "fmt"

// PlayStatus reports the current play-mode session, spec §7.8: entering play
// clones the editable document into a runtime world; runtime mutations are
// discarded on exit; pause and single-step work at the fixed simulation tick.
type PlayStatus struct {
	Active     bool   `json:"active"`
	Simulation ID     `json:"simulation,omitempty"`
	Tick       uint64 `json:"tick"`
}

// PlayEntityDiff names one entity whose runtime transform diverged from the
// authored document during the play session.
type PlayEntityDiff struct {
	Entity   ID   `json:"entity"`
	Authored Vec3 `json:"authored"`
	Runtime  Vec3 `json:"runtime"`
}

// EnterPlay clones the canonical document into an isolated runtime world for
// the named simulation profile. Editor transactions keep operating on the
// canonical document; the play clone never writes back implicitly.
func (w *Workspace) EnterPlay(simulation ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.play != nil {
		return fmt.Errorf("play mode is already active for %q", w.play.simulation)
	}
	if _, ok := w.doc.Simulations[simulation]; !ok {
		return fmt.Errorf("simulation %q does not exist", simulation)
	}
	clone, err := w.doc.Clone()
	if err != nil {
		return err
	}
	w.play = &playSession{document: clone, simulation: simulation}
	return nil
}

type playSession struct {
	document   Document
	simulation ID
	tick       uint64
}

// StepPlay advances the play world by the given number of fixed simulation
// ticks using the same deterministic reference loop as recordings.
func (w *Workspace) StepPlay(ticks uint64, inputs []SimulationInput) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.play == nil {
		return fmt.Errorf("play mode is not active")
	}
	_, stepped, err := RunSimulation(w.play.document, w.play.simulation, ticks, inputs)
	if err != nil {
		return err
	}
	w.play.document = stepped
	w.play.tick += ticks
	return nil
}

// ExitPlay discards the runtime world (spec default). Apply-back of selected
// runtime values remains an explicit authoring operation via PlayDiff.
func (w *Workspace) ExitPlay() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.play == nil {
		return fmt.Errorf("play mode is not active")
	}
	w.play = nil
	return nil
}

// PlayState reports whether a session is active and its tick position.
func (w *Workspace) PlayState() PlayStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.play == nil {
		return PlayStatus{}
	}
	return PlayStatus{Active: true, Simulation: w.play.simulation, Tick: w.play.tick}
}

// PlaySnapshot clones the runtime world for rendering and inspection.
func (w *Workspace) PlaySnapshot() (Document, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.play == nil {
		return Document{}, fmt.Errorf("play mode is not active")
	}
	return w.play.document.Clone()
}

// PlayDiff lists entities whose runtime position diverged from the authored
// document, so apply-back stays an explicit reviewed choice.
func (w *Workspace) PlayDiff() []PlayEntityDiff {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.play == nil {
		return nil
	}
	out := []PlayEntityDiff{}
	for id, entity := range w.play.document.Entities {
		authored, ok := w.doc.Entities[id]
		if !ok {
			continue
		}
		if authored.Transform.Position != entity.Transform.Position {
			out = append(out, PlayEntityDiff{Entity: id, Authored: authored.Transform.Position, Runtime: entity.Transform.Position})
		}
	}
	return out
}
