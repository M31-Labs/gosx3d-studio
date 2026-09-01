package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

const studioDemoResetPath = "/api/studio/demo/reset"

var errStudioDemoUnavailable = errors.New("public demo reset is unavailable")

type studioDemoResetRequest struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
}

// studioDemoPublicState is deliberately filesystem-free. The visible reset UI
// needs to know whether the clean-room is active, that the scene is shared,
// and where to send the reviewed reset; it must never learn the server's
// manager-owned temporary directory.
type studioDemoPublicState struct {
	Enabled        bool      `json:"enabled"`
	SharedScene    bool      `json:"sharedScene"`
	Ephemeral      bool      `json:"ephemeral"`
	Mode           string    `json:"mode,omitempty"`
	SourceDocument string    `json:"sourceDocument,omitempty"`
	ResetPath      string    `json:"resetPath,omitempty"`
	ProjectID      studio.ID `json:"projectId,omitempty"`
	ProjectName    string    `json:"projectName,omitempty"`
	Revision       uint64    `json:"revision,omitempty"`
}

type studioDemoResetResult struct {
	studioDemoPublicState
	PreviousRevision uint64 `json:"previousRevision"`
	CleanupWarning   string `json:"cleanupWarning,omitempty"`
}

// studioDemoProject owns one temporary base and exactly one active project
// generation. Every reset builds a new generation and asks Workspace to switch
// to it through the existing revision-safe project path. The manager never
// accepts a path from a request.
type studioDemoProject struct {
	mu                sync.Mutex
	baseDir           string
	currentGeneration string
	ownedGenerations  map[string]struct{}
	workspace         *studio.Workspace
	closed            bool
}

func studioDemoModeEnabled(value string) bool {
	return strings.TrimSpace(value) == "1"
}

func newStudioDemoProject() (*studioDemoProject, error) {
	baseDir, err := os.MkdirTemp("", "gosx3d-studio-demo-")
	if err != nil {
		return nil, fmt.Errorf("create public demo root: %w", err)
	}
	resolvedBase, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		_ = os.RemoveAll(baseDir)
		return nil, fmt.Errorf("resolve public demo root: %w", err)
	}
	project := &studioDemoProject{
		baseDir:          filepath.Clean(resolvedBase),
		ownedGenerations: map[string]struct{}{},
	}
	generation, workspace, err := project.createGenerationLocked(studio.SampleDocument())
	if err != nil {
		_ = os.RemoveAll(project.baseDir)
		return nil, err
	}
	project.currentGeneration = generation
	project.workspace = workspace
	return project, nil
}

func (project *studioDemoProject) Workspace() *studio.Workspace {
	if project == nil {
		return nil
	}
	project.mu.Lock()
	defer project.mu.Unlock()
	return project.workspace
}

func (project *studioDemoProject) PublicState() (studioDemoPublicState, error) {
	if project == nil {
		return studioDemoPublicState{}, nil
	}
	project.mu.Lock()
	defer project.mu.Unlock()
	return project.publicStateLocked()
}

func (project *studioDemoProject) publicStateLocked() (studioDemoPublicState, error) {
	if project.closed || project.workspace == nil {
		return studioDemoPublicState{}, errStudioDemoUnavailable
	}
	document, err := project.workspace.Snapshot()
	if err != nil {
		return studioDemoPublicState{}, err
	}
	return studioDemoPublicState{
		Enabled:        true,
		SharedScene:    true,
		Ephemeral:      true,
		Mode:           "shared-clean-room",
		SourceDocument: "studio.SampleDocument",
		ResetPath:      studioDemoResetPath,
		ProjectID:      document.ID,
		ProjectName:    document.Name,
		Revision:       document.Revision,
	}, nil
}

func (project *studioDemoProject) Reset(expectedRevision uint64) (studioDemoResetResult, error) {
	if project == nil {
		return studioDemoResetResult{}, errStudioDemoUnavailable
	}
	project.mu.Lock()
	defer project.mu.Unlock()
	if project.closed || project.workspace == nil {
		return studioDemoResetResult{}, errStudioDemoUnavailable
	}

	current, err := project.workspace.Snapshot()
	if err != nil {
		return studioDemoResetResult{}, err
	}
	if current.Revision != expectedRevision {
		return studioDemoResetResult{}, fmt.Errorf("%w: have %d, expected %d", studio.ErrRevisionConflict, current.Revision, expectedRevision)
	}
	if current.Revision == ^uint64(0) {
		return studioDemoResetResult{}, fmt.Errorf("public demo revision is exhausted")
	}

	seed := studio.SampleDocument()
	// Never return to revision 1. A monotonically increasing revision prevents
	// stale forms and staged proposals from becoming valid again after reset.
	seed.Revision = current.Revision + 1
	nextGeneration, _, err := project.createGenerationLocked(seed)
	if err != nil {
		return studioDemoResetResult{}, err
	}
	nextPath := filepath.Join(nextGeneration, "scene.scene3d")
	if _, err := project.workspace.SwitchProjectFile(nextPath, expectedRevision, true); err != nil {
		_ = project.removeOwnedGenerationLocked(nextGeneration)
		return studioDemoResetResult{}, err
	}

	previousGeneration := project.currentGeneration
	project.currentGeneration = nextGeneration
	cleanupWarning := ""
	if err := project.removeOwnedGenerationLocked(previousGeneration); err != nil {
		// The workspace has already switched atomically. Report cleanup as a
		// warning rather than claiming the reset failed after it took effect.
		// Keep the public result filesystem-free even when the OS error names
		// the manager-owned temporary path.
		cleanupWarning = "prior temporary generation cleanup failed"
	}
	state, err := project.publicStateLocked()
	if err != nil {
		return studioDemoResetResult{}, err
	}
	return studioDemoResetResult{studioDemoPublicState: state, PreviousRevision: current.Revision, CleanupWarning: cleanupWarning}, nil
}

func (project *studioDemoProject) createGenerationLocked(document studio.Document) (string, *studio.Workspace, error) {
	generation, err := os.MkdirTemp(project.baseDir, "generation-")
	if err != nil {
		return "", nil, fmt.Errorf("create public demo generation: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(generation)
	if err != nil {
		_ = os.RemoveAll(generation)
		return "", nil, fmt.Errorf("resolve public demo generation: %w", err)
	}
	generation = filepath.Clean(resolved)
	if filepath.Dir(generation) != project.baseDir {
		_ = os.RemoveAll(generation)
		return "", nil, fmt.Errorf("public demo generation escaped its manager-owned root")
	}
	workspace, err := studio.OpenWorkspace(generation, document)
	if err != nil {
		_ = os.RemoveAll(generation)
		return "", nil, fmt.Errorf("open public demo generation: %w", err)
	}
	// OpenWorkspace writes a missing seed, and Save makes that persistence an
	// explicit generation invariant before SwitchProjectFile is ever given
	// the canonical scene.scene3d path.
	if _, err := workspace.Save(); err != nil {
		_ = os.RemoveAll(generation)
		return "", nil, fmt.Errorf("persist public demo generation: %w", err)
	}
	project.ownedGenerations[generation] = struct{}{}
	return generation, workspace, nil
}

func (project *studioDemoProject) removeOwnedGenerationLocked(generation string) error {
	generation = filepath.Clean(generation)
	if generation == "" || generation == "." || filepath.Dir(generation) != project.baseDir {
		return fmt.Errorf("refuse to remove non-owned public demo generation %q", generation)
	}
	if _, ok := project.ownedGenerations[generation]; !ok {
		return fmt.Errorf("refuse to remove untracked public demo generation %q", generation)
	}
	if err := os.RemoveAll(generation); err != nil {
		return fmt.Errorf("remove prior public demo generation: %w", err)
	}
	delete(project.ownedGenerations, generation)
	return nil
}

func (project *studioDemoProject) Close() error {
	if project == nil {
		return nil
	}
	project.mu.Lock()
	defer project.mu.Unlock()
	if project.closed {
		return nil
	}
	var cleanupErrors []error
	for generation := range project.ownedGenerations {
		if err := project.removeOwnedGenerationLocked(generation); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	if len(project.ownedGenerations) == 0 {
		if err := os.Remove(project.baseDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove public demo root: %w", err))
		}
	}
	project.closed = true
	project.workspace = nil
	project.currentGeneration = ""
	return errors.Join(cleanupErrors...)
}

func decodeStudioDemoReset(reader io.Reader) (studioDemoResetRequest, error) {
	var request studioDemoResetRequest
	decoder := json.NewDecoder(io.LimitReader(reader, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return studioDemoResetRequest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return studioDemoResetRequest{}, fmt.Errorf("request body must contain exactly one JSON value")
		}
		return studioDemoResetRequest{}, err
	}
	if request.ExpectedRevision == 0 {
		return studioDemoResetRequest{}, fmt.Errorf("expectedRevision is required")
	}
	return request, nil
}
