package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func newTestDemoStudio(t *testing.T) (http.Handler, *studioDemoProject, *studio.Workspace) {
	t.Helper()
	project, err := newStudioDemoProject()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := project.Close(); err != nil {
			t.Errorf("close public demo project: %v", err)
		}
	})

	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(thisFile)
	workspace := project.Workspace()
	app, err := buildStudioApp(studioConfig{
		root:          root,
		appName:       "GoSX 3D Studio public demo (test)",
		workspace:     workspace,
		demoProject:   project,
		actionToken:   testActionToken,
		desktopHost:   false,
		sessionSecret: "a-private-demo-secret-for-tests-0123456789",
	})
	if err != nil {
		t.Fatal(err)
	}
	return app.Build(), project, workspace
}

func TestStudioDemoModeRequiresExplicitOptIn(t *testing.T) {
	for value, want := range map[string]bool{
		"1":      true,
		" 1\n":   true,
		"":       false,
		"0":      false,
		"true":   false,
		"yes":    false,
		"demo":   false,
		"01":     false,
		"1 true": false,
	} {
		if got := studioDemoModeEnabled(value); got != want {
			t.Errorf("studioDemoModeEnabled(%q) = %t, want %t", value, got, want)
		}
	}
}

func TestStudioDemoResetRestoresARevisionSafeCleanSample(t *testing.T) {
	project, err := newStudioDemoProject()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := project.Close(); err != nil {
			t.Errorf("close public demo project: %v", err)
		}
	})
	workspace := project.Workspace()

	project.mu.Lock()
	baseDir := project.baseDir
	previousGeneration := project.currentGeneration
	project.mu.Unlock()

	initial, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	requireStudioDemoCleanState(t, project, true)
	_, firstEdit, err := workspace.Execute(studio.Transaction{
		ID:               "demo-dirty-1",
		Actor:            "human://demo-test",
		Mode:             studio.ModeDirect,
		ExpectedRevision: initial.Revision,
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Dirty board one"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, secondEdit, err := workspace.Execute(studio.Transaction{
		ID:               "demo-dirty-2",
		Actor:            "agent://demo-test",
		Mode:             studio.ModeDirect,
		ExpectedRevision: firstEdit.Revision,
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Dirty board two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, beforeReset, err := workspace.Undo(secondEdit.Revision, "human://demo-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.SelectAtRevision(beforeReset.Revision, "board"); err != nil {
		t.Fatal(err)
	}
	workspace.RecordViewportConfirmation(studio.SelectionConfirmation{
		Selected:  "board",
		Source:    "gpu",
		Method:    "id-only",
		Confirmed: true,
		Revision:  beforeReset.Revision,
	})
	if len(workspace.RecentReceipts(10)) == 0 || workspace.ProjectStatus().UndoDepth == 0 {
		t.Fatal("test did not populate project-scoped transient history")
	}
	requireStudioDemoCleanState(t, project, false)

	result, err := project.Reset(beforeReset.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if result.PreviousRevision != beforeReset.Revision || result.Revision != beforeReset.Revision+1 {
		t.Fatalf("reset revisions = previous %d current %d, want %d and %d", result.PreviousRevision, result.Revision, beforeReset.Revision, beforeReset.Revision+1)
	}
	if !result.Enabled || !result.SharedScene || !result.Ephemeral || result.Mode != "shared-clean-room" || result.ResetPath != studioDemoResetPath {
		t.Fatalf("reset public state = %+v", result.studioDemoPublicState)
	}
	requireStudioDemoCleanFlag(t, result.Clean, true)
	if result.CleanupWarning != "" {
		t.Fatalf("unexpected cleanup warning: %q", result.CleanupWarning)
	}

	project.mu.Lock()
	currentGeneration := project.currentGeneration
	ownedCount := len(project.ownedGenerations)
	project.mu.Unlock()
	if currentGeneration == previousGeneration || filepath.Dir(currentGeneration) != baseDir || ownedCount != 1 {
		t.Fatalf("generation rotation: prior=%q current=%q base=%q owned=%d", previousGeneration, currentGeneration, baseDir, ownedCount)
	}
	if _, err := os.Stat(previousGeneration); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("prior generation still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(currentGeneration, "scene.scene3d")); err != nil {
		t.Fatalf("fresh generation has no canonical scene: %v", err)
	}

	resetDocument, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	wantSample := studio.SampleDocument()
	resetAtSampleRevision := resetDocument
	resetAtSampleRevision.Revision = wantSample.Revision
	gotFingerprint, err := resetAtSampleRevision.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	wantFingerprint, err := wantSample.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if gotFingerprint != wantFingerprint {
		t.Fatalf("reset document fingerprint = %s, want deterministic sample %s", gotFingerprint, wantFingerprint)
	}

	status := workspace.ProjectStatus()
	if status.Revision != result.Revision || status.SavedRevision != result.Revision || status.Dirty || status.UndoDepth != 0 {
		t.Fatalf("post-reset project status = %+v", status)
	}
	if selection := workspace.Selection(); len(selection) != 0 {
		t.Fatalf("selection leaked across reset: %v", selection)
	}
	selectionState := workspace.SelectionState()
	if selectionState.Revision != result.Revision || len(selectionState.IDs) != 0 || selectionState.Object != "" {
		t.Fatalf("selection state leaked across reset: %+v", selectionState)
	}
	if receipts := workspace.RecentReceipts(10); len(receipts) != 0 {
		t.Fatalf("receipts leaked across reset: %+v", receipts)
	}
	if confirmation := workspace.ViewportConfirmation(); confirmation != nil {
		t.Fatalf("viewport confirmation leaked across reset: %+v", confirmation)
	}
	if play := workspace.PlayState(); play.Active || play.Tick != 0 || play.Simulation != "" {
		t.Fatalf("play state leaked across reset: %+v", play)
	}
	if _, _, err := workspace.Undo(result.Revision, "human://demo-test"); err == nil || !strings.Contains(err.Error(), "nothing to undo") {
		t.Fatalf("undo survived reset: %v", err)
	}
	if _, _, err := workspace.Redo(result.Revision, "human://demo-test"); err == nil || !strings.Contains(err.Error(), "nothing to redo") {
		t.Fatalf("redo survived reset: %v", err)
	}
	if _, _, err := workspace.Execute(studio.Transaction{
		ID:               "stale-after-demo-reset",
		Actor:            "agent://stale",
		Mode:             studio.ModeDirect,
		ExpectedRevision: beforeReset.Revision,
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Stale"}},
	}); !errors.Is(err, studio.ErrRevisionConflict) {
		t.Fatalf("pre-reset revision became valid again: %v", err)
	}

	publicJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(publicJSON), baseDir) || strings.Contains(string(publicJSON), currentGeneration) {
		t.Fatalf("public reset result leaked manager-owned paths: %s", publicJSON)
	}
}

func requireStudioDemoCleanState(t *testing.T, project *studioDemoProject, want bool) {
	t.Helper()
	state, err := project.PublicState()
	if err != nil {
		t.Fatal(err)
	}
	requireStudioDemoCleanFlag(t, state.Clean, want)
}

func requireStudioDemoCleanFlag(t *testing.T, got, want bool) {
	t.Helper()
	if got != want {
		t.Fatalf("demo clean state = %t, want %t", got, want)
	}
}

func TestStudioDemoResetSerializesConcurrentAttempts(t *testing.T) {
	project, err := newStudioDemoProject()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := project.Close(); err != nil {
			t.Errorf("close public demo project: %v", err)
		}
	})

	start := make(chan struct{})
	type outcome struct {
		result studioDemoResetResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			result, err := project.Reset(1)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	ready.Wait()
	close(start)

	successes := 0
	conflicts := 0
	for range 2 {
		outcome := <-outcomes
		switch {
		case outcome.err == nil:
			successes++
			if outcome.result.PreviousRevision != 1 || outcome.result.Revision != 2 {
				t.Fatalf("successful concurrent reset = %+v", outcome.result)
			}
		case errors.Is(outcome.err, studio.ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("concurrent reset error = %v", outcome.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent outcomes: successes=%d conflicts=%d", successes, conflicts)
	}
	state, err := project.PublicState()
	if err != nil {
		t.Fatal(err)
	}
	project.mu.Lock()
	currentGeneration := project.currentGeneration
	baseDir := project.baseDir
	ownedCount := len(project.ownedGenerations)
	project.mu.Unlock()
	if state.Revision != 2 || filepath.Dir(currentGeneration) != baseDir || ownedCount != 1 {
		t.Fatalf("post-concurrency state=%+v generation=%q base=%q owned=%d", state, currentGeneration, baseDir, ownedCount)
	}
	if _, err := os.Stat(filepath.Join(currentGeneration, "scene.scene3d")); err != nil {
		t.Fatalf("winning generation is unavailable: %v", err)
	}
}

func TestStudioDemoCleanupRefusesUntrackedOrOutsidePaths(t *testing.T) {
	project, err := newStudioDemoProject()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := project.Close(); err != nil {
			t.Errorf("close public demo project: %v", err)
		}
	})

	outside := t.TempDir()
	outsideMarker := filepath.Join(outside, "must-survive")
	if err := os.WriteFile(outsideMarker, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	project.mu.Lock()
	untracked, err := os.MkdirTemp(project.baseDir, "untracked-")
	if err != nil {
		project.mu.Unlock()
		t.Fatal(err)
	}
	baseErr := project.removeOwnedGenerationLocked(project.baseDir)
	untrackedErr := project.removeOwnedGenerationLocked(untracked)
	outsideErr := project.removeOwnedGenerationLocked(outside)
	project.mu.Unlock()
	if baseErr == nil || untrackedErr == nil || outsideErr == nil {
		t.Fatalf("unsafe cleanup accepted: base=%v untracked=%v outside=%v", baseErr, untrackedErr, outsideErr)
	}
	if _, err := os.Stat(untracked); err != nil {
		t.Fatalf("untracked direct child was removed: %v", err)
	}
	if _, err := os.Stat(outsideMarker); err != nil {
		t.Fatalf("outside path was altered: %v", err)
	}
	if err := os.RemoveAll(untracked); err != nil {
		t.Fatal(err)
	}
}

func TestStudioDemoHTTPResetRequiresBrowserCSRFAndClearsAllProposals(t *testing.T) {
	handler, project, workspace := newTestDemoStudio(t)
	browserA := newWebMCPTestBrowser(t, handler)
	browserB := newWebMCPTestBrowser(t, handler)
	if browserA.csrf == browserB.csrf {
		t.Fatal("independent demo browsers received the same CSRF token")
	}

	statusResponse := doRequest(t, handler, http.MethodGet, "/api/studio/demo/status", nil, false)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("demo status = %d: %s", statusResponse.Code, statusResponse.Body.String())
	}
	var state studioDemoPublicState
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if !state.Enabled || !state.SharedScene || !state.Ephemeral || !state.Clean || state.ResetPath != studioDemoResetPath || state.Revision != 1 {
		t.Fatalf("demo status = %+v", state)
	}
	project.mu.Lock()
	baseDir := project.baseDir
	currentGeneration := project.currentGeneration
	project.mu.Unlock()
	if strings.Contains(statusResponse.Body.String(), baseDir) || strings.Contains(statusResponse.Body.String(), currentGeneration) {
		t.Fatalf("demo status leaked manager-owned paths: %s", statusResponse.Body.String())
	}

	proposalResponse := browserA.postJSON(t, "/api/studio/webmcp/proposals", map[string]any{
		"expectedRevision": state.Revision,
		"title":            "Proposal invalidated by a shared reset",
		"operations":       []map[string]any{{"kind": "rename-entity", "target": "board", "name": "Proposed board"}},
	})
	if proposalResponse.Code != http.StatusOK {
		t.Fatalf("stage proposal = %d: %s", proposalResponse.Code, proposalResponse.Body.String())
	}
	var proposal struct {
		ProposalID string `json:"proposalId"`
	}
	if err := json.Unmarshal(proposalResponse.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.ProposalID == "" {
		t.Fatal("staged proposal has no id")
	}

	resetBody := map[string]any{"expectedRevision": state.Revision}
	if response := doRequest(t, handler, http.MethodPost, studioDemoResetPath, resetBody, false); response.Code != http.StatusForbidden {
		t.Fatalf("anonymous reset = %d, want 403: %s", response.Code, response.Body.String())
	}
	if response := doRequest(t, handler, http.MethodPost, studioDemoResetPath, resetBody, true); response.Code != http.StatusForbidden {
		t.Fatalf("bearer-only reset = %d, want 403: %s", response.Code, response.Body.String())
	}

	resetResponse := browserB.postJSON(t, studioDemoResetPath, resetBody)
	if resetResponse.Code != http.StatusOK {
		t.Fatalf("browser reset = %d: %s", resetResponse.Code, resetResponse.Body.String())
	}
	var reset studioDemoResetResult
	if err := json.Unmarshal(resetResponse.Body.Bytes(), &reset); err != nil {
		t.Fatal(err)
	}
	if reset.PreviousRevision != state.Revision || reset.Revision != state.Revision+1 || !reset.SharedScene || !reset.Clean {
		t.Fatalf("reset response = %+v", reset)
	}

	commitResponse := browserA.postJSON(t, "/api/studio/webmcp/commits", map[string]any{"proposalId": proposal.ProposalID})
	if commitResponse.Code != http.StatusNotFound {
		t.Fatalf("proposal survived shared reset = %d, want 404: %s", commitResponse.Code, commitResponse.Body.String())
	}
	after, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != state.Revision+1 || after.Entities["board"].Name != studio.SampleDocument().Entities["board"].Name {
		t.Fatalf("shared workspace after reset = revision %d board %q", after.Revision, after.Entities["board"].Name)
	}

	staleResponse := browserB.postJSON(t, studioDemoResetPath, resetBody)
	if staleResponse.Code != http.StatusConflict {
		t.Fatalf("stale reset = %d, want 409: %s", staleResponse.Code, staleResponse.Body.String())
	}
	malformedResponse := browserB.postJSON(t, studioDemoResetPath, map[string]any{"expectedRevision": reset.Revision, "path": "/tmp/not-allowed"})
	if malformedResponse.Code != http.StatusBadRequest {
		t.Fatalf("reset accepted unknown path = %d, want 400: %s", malformedResponse.Code, malformedResponse.Body.String())
	}
}

func TestStudioDemoResetRouteIsUnavailableOutsideDemoMode(t *testing.T) {
	handler, _ := newTestStudio(t)
	browser := newWebMCPTestBrowser(t, handler)
	statusResponse := doRequest(t, handler, http.MethodGet, "/api/studio/demo/status", nil, false)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("non-demo status = %d: %s", statusResponse.Code, statusResponse.Body.String())
	}
	var state studioDemoPublicState
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Enabled || state.SharedScene || state.Ephemeral || state.ResetPath != "" {
		t.Fatalf("non-demo public state = %+v", state)
	}
	response := browser.postJSON(t, studioDemoResetPath, map[string]any{"expectedRevision": 1})
	if response.Code != http.StatusNotFound {
		t.Fatalf("non-demo reset = %d: %s", response.Code, response.Body.String())
	}
}

func TestStudioDemoProjectCloseRemovesOnlyItsOwnedRoot(t *testing.T) {
	project, err := newStudioDemoProject()
	if err != nil {
		t.Fatal(err)
	}
	project.mu.Lock()
	baseDir := project.baseDir
	project.mu.Unlock()
	if err := project.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(baseDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manager-owned root survived close: %v", err)
	}
	if _, err := project.PublicState(); !errors.Is(err, errStudioDemoUnavailable) {
		t.Fatalf("closed project public state error = %v", err)
	}
	if err := project.Close(); err != nil {
		t.Fatalf("second close = %v", err)
	}
}
