package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

func TestValidateWebMCPOperationsKeepsTheBrowserSurfaceNarrow(t *testing.T) {
	transform := studio.TransformFromEuler(studio.Vec3{X: 1}, studio.Vec3{}, studio.Vec3{X: 1, Y: 1, Z: 1})
	valid := []studio.Operation{
		{Kind: studio.OpRenameEntity, Target: "board", Name: "Hero plinth"},
		{Kind: studio.OpSetTransform, Target: "board", Transform: &transform},
		{Kind: studio.OpAssignMaterial, Target: "board", Material: "board-material"},
	}
	if err := validateWebMCPOperations(valid); err != nil {
		t.Fatalf("valid operations: %v", err)
	}
	for name, operations := range map[string][]studio.Operation{
		"empty":          {},
		"destructive":    {{Kind: studio.OpDeleteEntity, Target: "board"}},
		"missing target": {{Kind: studio.OpRenameEntity, Name: "No target"}},
		"missing name":   {{Kind: studio.OpRenameEntity, Target: "board"}},
		"missing value":  {{Kind: studio.OpSetTransform, Target: "board"}},
		"duplicate":      {{Kind: studio.OpDuplicateEntity, Target: "board", NewID: "board-copy"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateWebMCPOperations(operations); err == nil {
				t.Fatal("invalid operation set was accepted")
			}
		})
	}
	tooMany := make([]studio.Operation, maxWebMCPProposalOperations+1)
	for index := range tooMany {
		tooMany[index] = studio.Operation{Kind: studio.OpRenameEntity, Target: "board", Name: "Bounded"}
	}
	if err := validateWebMCPOperations(tooMany); err == nil {
		t.Fatal("oversized proposal was accepted")
	}
}

func TestWebMCPProposalStagesThenCommitsTheExactRevision(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	before, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Promote the board",
		Rationale:        "Give the human a clearer hero object.",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Hero plinth"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	proposalID, ok := result["proposalId"].(string)
	if !ok || proposalID == "" {
		t.Fatalf("proposal id = %#v", result["proposalId"])
	}
	governance, ok := result["governance"].([]webMCPOperationDecision)
	if !ok || len(governance) != 1 || !governance[0].Allowed || governance[0].Selected != "Allow" || len(governance[0].Arbitrace.Steps) == 0 {
		t.Fatalf("governance = %#v, want one traced Allow decision", result["governance"])
	}
	afterPreview, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if afterPreview.Revision != before.Revision || afterPreview.Entities["board"].Name != before.Entities["board"].Name {
		t.Fatal("proposal mutated canonical scene state")
	}
	commit, err := store.Commit(proposalID, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	receipt, ok := commit["receipt"].(studio.Receipt)
	if !ok {
		t.Fatalf("commit receipt = %#v", commit["receipt"])
	}
	if receipt.Actor != "human://webmcp-review" || !receipt.Applied {
		t.Fatalf("commit receipt = %#v", receipt)
	}
	afterCommit, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if afterCommit.Revision != before.Revision+1 || afterCommit.Entities["board"].Name != "Hero plinth" {
		t.Fatalf("committed document revision=%d name=%q", afterCommit.Revision, afterCommit.Entities["board"].Name)
	}
	if _, err := store.Commit(proposalID, "session-a"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("second commit error = %v", err)
	}
}

func TestWebMCPProposalCarriesGoSXGroupScaleThroughHumanReview(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	before, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	transform := before.Entities["scene-root"].Transform
	transform.Scale = studio.Vec3{X: 1.25, Y: 0.75, Z: 1.5}
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Scale the scene group",
		Operations: []studio.Operation{{
			Kind: studio.OpSetTransform, Target: "scene-root", Transform: &transform,
		}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	preview, ok := result["preview"].(map[string]any)
	if !ok {
		t.Fatalf("preview = %#v", result["preview"])
	}
	if got := preview["revision"]; got != before.Revision+1 {
		t.Fatalf("preview revision = %#v, want %d", got, before.Revision+1)
	}
	canonical, _ := workspace.Snapshot()
	if got := canonical.Entities["scene-root"].Transform.Scale; got == transform.Scale {
		t.Fatal("group-scale preview mutated canonical scene")
	}
	proposalID := result["proposalId"].(string)
	if _, err := store.Commit(proposalID, "session-a"); err != nil {
		t.Fatal(err)
	}
	committed, _ := workspace.Snapshot()
	if got := committed.Entities["scene-root"].Transform.Scale; got != transform.Scale {
		t.Fatalf("committed group scale = %+v, want %+v", got, transform.Scale)
	}
}

func TestWebMCPProposalCarriesReversibleLiveSceneCommandsWithoutCanonicalMutation(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	before, _ := workspace.Snapshot()
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Preview brushed steel",
		Operations: []studio.Operation{{
			Kind: studio.OpAssignMaterial, Target: "board", Material: "board-steel-material",
		}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	commands, ok := result["sceneCommands"].([]scene.Command)
	if !ok || len(commands) == 0 {
		t.Fatalf("forward live scene commands = %#v", result["sceneCommands"])
	}
	reverse, ok := result["reverseSceneCommands"].([]scene.Command)
	if !ok || len(reverse) == 0 {
		t.Fatalf("reverse live scene commands = %#v", result["reverseSceneCommands"])
	}
	preview, err := before.Clone()
	if err != nil {
		t.Fatal(err)
	}
	board := preview.Entities["board"]
	board.Mesh.Material = "board-steel-material"
	preview.Entities[board.ID] = board
	canonicalProps, err := studio.CompileViewport(before)
	if err != nil {
		t.Fatal(err)
	}
	previewProps, err := studio.CompileViewport(preview)
	if err != nil {
		t.Fatal(err)
	}
	forwardDiff := scene.DiffScene(canonicalProps.SceneIR(), previewProps.SceneIR(), scene.DiffOptions{})
	reverseDiff := scene.DiffScene(previewProps.SceneIR(), canonicalProps.SceneIR(), scene.DiffOptions{})
	if len(forwardDiff.RemountFields) != 0 || len(reverseDiff.RemountFields) != 0 {
		t.Fatalf("live board preview requires remount: forward=%v reverse=%v", forwardDiff.RemountFields, reverseDiff.RemountFields)
	}
	if !reflect.DeepEqual(commands, forwardDiff.Commands) || !reflect.DeepEqual(reverse, reverseDiff.Commands) {
		t.Fatalf("stored live commands do not match exact SceneIR diffs:\nforward=%#v\nwant=%#v\nreverse=%#v\nwant=%#v", commands, forwardDiff.Commands, reverse, reverseDiff.Commands)
	}
	for label, commandSet := range map[string][]scene.Command{"forward": commands, "reverse": reverse} {
		if len(commandSet) != 2 || commandSet[0].Kind != scene.CommandRemoveObject || commandSet[1].Kind != scene.CommandCreateObject {
			t.Fatalf("%s board preview commands = %#v, want one remove/create replacement", label, commandSet)
		}
		for _, command := range commandSet {
			if command.ObjectID != "board" {
				t.Fatalf("%s preview command affected %q, want only board", label, command.ObjectID)
			}
		}
	}
	after, _ := workspace.Snapshot()
	if after.Revision != before.Revision || after.Entities["board"].Mesh.Material != before.Entities["board"].Mesh.Material {
		t.Fatal("client-local scene preview mutated canonical state")
	}
}

func TestWebMCPProposalExpiresAndCannotCommit(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	now := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	document, _ := workspace.Snapshot()
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Temporary proposal",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Temporary"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	proposalID := result["proposalId"].(string)
	now = now.Add(webMCPProposalTTL)
	if _, err := store.Commit(proposalID, "session-a"); !errors.Is(err, errWebMCPProposalExpired) {
		t.Fatalf("expired proposal error = %v", err)
	}
}

func TestWebMCPStaleCommitIsRejectedAndRemoved(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	before, _ := workspace.Snapshot()
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Soon stale",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Stale Board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	proposalID := result["proposalId"].(string)
	_, canonical, err := workspace.Execute(studio.Transaction{
		ID: "concurrent-human", Actor: "human://other", Mode: studio.ModeDirect,
		ExpectedRevision: before.Revision,
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Canonical Board"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(proposalID, "session-a"); !errors.Is(err, studio.ErrRevisionConflict) {
		t.Fatalf("stale commit error = %v, want revision conflict", err)
	}
	if current := store.Current("session-a"); current != nil {
		t.Fatalf("stale proposal remained current: %#v", current)
	}
	after, _ := workspace.Snapshot()
	if after.Revision != canonical.Revision || after.Entities["board"].Name != "Canonical Board" {
		t.Fatalf("stale commit changed canonical scene: revision=%d name=%q", after.Revision, after.Entities["board"].Name)
	}
}

func TestWebMCPCurrentInvalidatesEveryProposalWhoseBaseRevisionIsStale(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	before, _ := workspace.Snapshot()
	first, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Session A review",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Session A Board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Session B review",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "scene-root", Name: "Session B Root"}},
	}, "session-b")
	if err != nil {
		t.Fatal(err)
	}
	_, canonical, err := workspace.Execute(studio.Transaction{
		ID: "concurrent-canonical-edit", Actor: "human://other", Mode: studio.ModeDirect,
		ExpectedRevision: before.Revision,
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Canonical Board"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if current := store.Current("session-a"); current != nil {
		t.Fatalf("stale proposal was restored: %#v", current)
	}
	if current := store.Current("session-b"); current != nil {
		t.Fatalf("other owner's stale proposal survived pruning: %#v", current)
	}
	if len(store.proposals) != 0 || len(store.order) != 0 {
		t.Fatalf("stale proposal storage was not compacted: proposals=%d order=%v", len(store.proposals), store.order)
	}
	for proposalID, owner := range map[string]string{
		first["proposalId"].(string):  "session-a",
		second["proposalId"].(string): "session-b",
	} {
		if _, err := store.Commit(proposalID, owner); !errors.Is(err, errWebMCPProposalNotFound) {
			t.Fatalf("commit invalidated proposal %q: %v, want not found", proposalID, err)
		}
	}
	after, _ := workspace.Snapshot()
	if after.Revision != canonical.Revision || after.Entities["board"].Name != "Canonical Board" {
		t.Fatalf("stale-current cleanup changed canonical scene: revision=%d name=%q", after.Revision, after.Entities["board"].Name)
	}
}

func TestWebMCPProposalOwnerAndClearAreNonLeaking(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	document, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Owned proposal",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Session A board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	proposalID := result["proposalId"].(string)
	if _, err := store.Commit(proposalID, "session-b"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("cross-session commit error = %v, want not found", err)
	}
	unchanged, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Revision != document.Revision || unchanged.Entities["board"].Name != document.Entities["board"].Name {
		t.Fatal("owner mismatch changed canonical scene state")
	}
	store.Clear()
	if _, err := store.Commit(proposalID, "session-a"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("commit after clear error = %v, want not found", err)
	}
}

func TestWebMCPProposalCanBeRestoredAndRevokedByItsOwner(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	document, _ := workspace.Snapshot()
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Restorable review",
		Rationale:        "Keep the exact reviewed edit across a reload.",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Launch Board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	proposalID := result["proposalId"].(string)
	current := store.Current("session-a")
	if current == nil || current["proposalId"] != proposalID || current["title"] != "Restorable review" {
		t.Fatalf("current proposal = %#v", current)
	}
	if leaked := store.Current("session-b"); leaked != nil {
		t.Fatalf("cross-session current proposal leaked: %#v", leaked)
	}
	discarded, err := store.Discard(proposalID, "session-a")
	if err != nil || discarded["canonicalSceneChanged"] != false {
		t.Fatalf("discard = %#v, %v", discarded, err)
	}
	if current := store.Current("session-a"); current != nil {
		t.Fatalf("discarded proposal remained current: %#v", current)
	}
	if _, err := store.Commit(proposalID, "session-a"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("commit after discard error = %v, want not found", err)
	}
	unchanged, _ := workspace.Snapshot()
	if unchanged.Revision != document.Revision || unchanged.Entities["board"].Name != document.Entities["board"].Name {
		t.Fatal("discard changed canonical scene")
	}
}

func TestWebMCPCommitClaimMakesCommitAndDiscardMutuallyExclusive(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	document, _ := workspace.Snapshot()
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Atomic review",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Claimed Board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	proposalID := result["proposalId"].(string)
	if _, err := store.claimCommit(proposalID, "session-a"); err != nil {
		t.Fatal(err)
	}
	if current := store.Current("session-a"); current != nil {
		t.Fatalf("claimed proposal remained available to restore: %#v", current)
	}
	if _, err := store.Discard(proposalID, "session-a"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("discard of claimed proposal error = %v, want not found", err)
	}
	if _, err := store.claimCommit(proposalID, "session-a"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("second commit claim error = %v, want not found", err)
	}
	unchanged, _ := workspace.Snapshot()
	if unchanged.Revision != document.Revision || unchanged.Entities["board"].Name != document.Entities["board"].Name {
		t.Fatal("claim or rejected discard changed canonical scene")
	}

	transient := errors.New("transient commit failure")
	store.finishCommit(proposalID, "session-a", transient)
	if current := store.Current("session-a"); current == nil || current["proposalId"] != proposalID {
		t.Fatalf("transient failure did not safely restore proposal: %#v", current)
	}
}

func TestWebMCPStageSupersedesSameOwnerOnlyAfterSuccessfulPreview(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	document, _ := workspace.Snapshot()
	first, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "First review",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "First Board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	firstID := first["proposalId"].(string)
	if _, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Broken replacement",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "missing", Name: "Missing"}},
	}, "session-a"); err == nil {
		t.Fatal("invalid replacement preview succeeded")
	}
	if current := store.Current("session-a"); current == nil || current["proposalId"] != firstID {
		t.Fatalf("failed replacement removed the prior review: %#v", current)
	}

	second, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Second review",
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Second Board"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	secondID := second["proposalId"].(string)
	if current := store.Current("session-a"); current == nil || current["proposalId"] != secondID {
		t.Fatalf("successful replacement is not the sole current review: %#v", current)
	}
	if _, err := store.Commit(firstID, "session-a"); !errors.Is(err, errWebMCPProposalNotFound) {
		t.Fatalf("superseded proposal commit error = %v, want not found", err)
	}
	if _, err := store.Commit(secondID, "session-a"); err != nil {
		t.Fatal(err)
	}
	committed, _ := workspace.Snapshot()
	if committed.Entities["board"].Name != "Second Board" {
		t.Fatalf("committed board name = %q, want replacement", committed.Entities["board"].Name)
	}
}

func TestWebMCPStageRejectsNetNoEffectOperations(t *testing.T) {
	for name, operations := range map[string][]studio.Operation{
		"same trimmed name": {{Kind: studio.OpRenameEntity, Target: "board", Name: "  Board  "}},
		"same material":     {{Kind: studio.OpAssignMaterial, Target: "board", Material: "board-material"}},
		"cancelled rename": {
			{Kind: studio.OpRenameEntity, Target: "board", Name: "Temporary Board"},
			{Kind: studio.OpRenameEntity, Target: "board", Name: "Board"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			workspace, err := studio.NewWorkspace(studio.SampleDocument())
			if err != nil {
				t.Fatal(err)
			}
			store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
			before, _ := workspace.Snapshot()
			if _, err := store.Stage(webMCPProposalRequest{
				ExpectedRevision: before.Revision,
				Title:            "No effect",
				Operations:       operations,
			}, "session-a"); !errors.Is(err, errWebMCPProposalNoChanges) {
				t.Fatalf("stage error = %v, want no-changes", err)
			}
			if current := store.Current("session-a"); current != nil {
				t.Fatalf("no-effect request created a proposal: %#v", current)
			}
			after, _ := workspace.Snapshot()
			if after.Revision != before.Revision {
				t.Fatalf("no-effect request advanced revision to %d", after.Revision)
			}
			if receipts := workspace.RecentReceipts(10); len(receipts) != 0 {
				t.Fatalf("no-effect request recorded preview receipts: %#v", receipts)
			}
		})
	}

	t.Run("same transform", func(t *testing.T) {
		workspace, err := studio.NewWorkspace(studio.SampleDocument())
		if err != nil {
			t.Fatal(err)
		}
		store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
		before, _ := workspace.Snapshot()
		transform := before.Entities["board"].Transform
		_, err = store.Stage(webMCPProposalRequest{
			ExpectedRevision: before.Revision,
			Title:            "No transform effect",
			Operations:       []studio.Operation{{Kind: studio.OpSetTransform, Target: "board", Transform: &transform}},
		}, "session-a")
		if !errors.Is(err, errWebMCPProposalNoChanges) {
			t.Fatalf("stage error = %v, want no-changes", err)
		}
	})
}

func TestWebMCPProposalIncludesMaterialDisplayNames(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	document, _ := workspace.Snapshot()
	result, err := store.Stage(webMCPProposalRequest{
		ExpectedRevision: document.Revision,
		Title:            "Repaint the board",
		Operations:       []studio.Operation{{Kind: studio.OpAssignMaterial, Target: "board", Material: "board-steel-material"}},
	}, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	materials, ok := result["materials"].(map[string]string)
	if !ok {
		t.Fatalf("materials = %#v", result["materials"])
	}
	if materials["board-material"] != "Carved Wood" || materials["board-steel-material"] != "Brushed Steel" {
		t.Fatalf("material display names = %#v", materials)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Carved Wood", "Brushed Steel"} {
		if !bytes.Contains(encoded, []byte(name)) {
			t.Fatalf("proposal JSON omits material name %q: %s", name, encoded)
		}
	}
}

type webMCPTestBrowser struct {
	handler http.Handler
	cookies map[string]*http.Cookie
	csrf    string
}

func newWebMCPTestBrowser(t *testing.T, handler http.Handler) *webMCPTestBrowser {
	t.Helper()
	browser := &webMCPTestBrowser{handler: handler, cookies: map[string]*http.Cookie{}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("establish browser session = %d: %s", response.Code, response.Body.String())
	}
	browser.captureCookies(response.Result().Cookies())
	browser.csrf = webMCPTestCSRFToken(t, response.Body.String())
	if len(browser.cookies) == 0 {
		t.Fatal("browser session response set no cookie")
	}
	return browser
}

func (browser *webMCPTestBrowser) captureCookies(cookies []*http.Cookie) {
	for _, cookie := range cookies {
		copy := *cookie
		browser.cookies[cookie.Name] = &copy
	}
}

func (browser *webMCPTestBrowser) postJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", browser.csrf)
	for _, cookie := range browser.cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	browser.handler.ServeHTTP(response, request)
	browser.captureCookies(response.Result().Cookies())
	return response
}

func (browser *webMCPTestBrowser) getJSON(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Accept", "application/json")
	for _, cookie := range browser.cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	browser.handler.ServeHTTP(response, request)
	browser.captureCookies(response.Result().Cookies())
	return response
}

func webMCPTestCSRFToken(t *testing.T, body string) string {
	t.Helper()
	nameAt := strings.Index(body, `name="csrf_token"`)
	if nameAt < 0 {
		t.Fatal("rendered page has no CSRF field")
	}
	start := strings.LastIndex(body[:nameAt], "<input")
	if start < 0 {
		t.Fatal("CSRF field has no input start tag")
	}
	endAt := strings.Index(body[nameAt:], ">")
	if endAt < 0 {
		t.Fatal("CSRF field has no input end tag")
	}
	tag := body[start : nameAt+endAt]
	valueAt := strings.Index(tag, `value="`)
	if valueAt < 0 {
		t.Fatal("CSRF field has no value")
	}
	value := tag[valueAt+len(`value="`):]
	valueEnd := strings.Index(value, `"`)
	if valueEnd < 0 {
		t.Fatal("CSRF field value is unterminated")
	}
	return html.UnescapeString(value[:valueEnd])
}

func TestWebMCPHTTPProposalRequiresBrowserAuthorityAndHumanCommit(t *testing.T) {
	handler, workspace := newTestStudio(t)
	browserA := newWebMCPTestBrowser(t, handler)
	browserB := newWebMCPTestBrowser(t, handler)
	if browserA.csrf == browserB.csrf {
		t.Fatal("independent browsers received the same session CSRF token")
	}
	document, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"expectedRevision": document.Revision,
		"title":            "Rename the hero",
		"rationale":        "Make the focal object legible.",
		"operations":       []map[string]any{{"kind": "rename-entity", "target": "board", "name": "Hero plinth"}},
	}
	withoutSession := doRequest(t, handler, http.MethodPost, "/api/studio/webmcp/proposals", payload, false)
	if withoutSession.Code < 400 {
		t.Fatalf("anonymous proposal status = %d", withoutSession.Code)
	}
	bearerOnly := doRequest(t, handler, http.MethodPost, "/api/studio/webmcp/proposals", payload, true)
	if bearerOnly.Code != http.StatusForbidden {
		t.Fatalf("bearer-only proposal status = %d, want 403: %s", bearerOnly.Code, bearerOnly.Body.String())
	}
	staged := browserA.postJSON(t, "/api/studio/webmcp/proposals", payload)
	if staged.Code != http.StatusOK {
		t.Fatalf("stage status = %d: %s", staged.Code, staged.Body.String())
	}
	var proposal struct {
		ProposalID string         `json:"proposalId"`
		Receipt    studio.Receipt `json:"receipt"`
	}
	if err := json.Unmarshal(staged.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.ProposalID == "" || proposal.Receipt.Applied || proposal.Receipt.Actor != "agent://webmcp" {
		t.Fatalf("staged proposal = %#v", proposal)
	}
	afterStage, _ := workspace.Snapshot()
	if afterStage.Revision != document.Revision {
		t.Fatal("HTTP proposal mutated canonical state")
	}
	current := browserA.getJSON(t, "/api/studio/webmcp/proposals/current")
	if current.Code != http.StatusOK || !strings.Contains(current.Body.String(), proposal.ProposalID) {
		t.Fatalf("same-session current proposal = %d: %s", current.Code, current.Body.String())
	}
	crossSessionCurrent := browserB.getJSON(t, "/api/studio/webmcp/proposals/current")
	if crossSessionCurrent.Code != http.StatusOK || strings.Contains(crossSessionCurrent.Body.String(), proposal.ProposalID) {
		t.Fatalf("cross-session current proposal = %d: %s", crossSessionCurrent.Code, crossSessionCurrent.Body.String())
	}
	crossSession := browserB.postJSON(t, "/api/studio/webmcp/commits", map[string]any{"proposalId": proposal.ProposalID})
	if crossSession.Code != http.StatusNotFound {
		t.Fatalf("cross-session commit status = %d, want 404: %s", crossSession.Code, crossSession.Body.String())
	}
	if strings.Contains(crossSession.Body.String(), proposal.ProposalID) {
		t.Fatal("cross-session not-found response leaked the proposal id")
	}
	afterCrossSession, _ := workspace.Snapshot()
	if afterCrossSession.Revision != document.Revision {
		t.Fatal("cross-session commit mutated canonical state")
	}
	committed := browserA.postJSON(t, "/api/studio/webmcp/commits", map[string]any{"proposalId": proposal.ProposalID})
	if committed.Code != http.StatusOK {
		t.Fatalf("commit status = %d: %s", committed.Code, committed.Body.String())
	}
	afterCommit, _ := workspace.Snapshot()
	if afterCommit.Revision != document.Revision+1 || afterCommit.Entities["board"].Name != "Hero plinth" {
		t.Fatalf("commit revision=%d name=%q", afterCommit.Revision, afterCommit.Entities["board"].Name)
	}

	discardPayload := map[string]any{
		"expectedRevision": afterCommit.Revision,
		"title":            "Discard this review",
		"operations":       []map[string]any{{"kind": "rename-entity", "target": "board", "name": "Never applied"}},
	}
	discardStage := browserA.postJSON(t, "/api/studio/webmcp/proposals", discardPayload)
	if discardStage.Code != http.StatusOK {
		t.Fatalf("discard stage status = %d: %s", discardStage.Code, discardStage.Body.String())
	}
	var discardProposal struct {
		ProposalID string `json:"proposalId"`
	}
	if err := json.Unmarshal(discardStage.Body.Bytes(), &discardProposal); err != nil {
		t.Fatal(err)
	}
	discarded := browserA.postJSON(t, "/api/studio/webmcp/discards", map[string]any{"proposalId": discardProposal.ProposalID})
	if discarded.Code != http.StatusOK || !strings.Contains(discarded.Body.String(), `"canonicalSceneChanged":false`) {
		t.Fatalf("discard status = %d: %s", discarded.Code, discarded.Body.String())
	}
	commitDiscarded := browserA.postJSON(t, "/api/studio/webmcp/commits", map[string]any{"proposalId": discardProposal.ProposalID})
	if commitDiscarded.Code != http.StatusNotFound {
		t.Fatalf("commit discarded status = %d, want 404: %s", commitDiscarded.Code, commitDiscarded.Body.String())
	}
	afterDiscard, _ := workspace.Snapshot()
	if afterDiscard.Revision != afterCommit.Revision || afterDiscard.Entities["board"].Name != "Hero plinth" {
		t.Fatal("HTTP discard mutated canonical scene")
	}
}

func TestWebMCPHTTPStaleCommitReturnsConflictAndRemovesProposal(t *testing.T) {
	handler, workspace := newTestStudio(t)
	browser := newWebMCPTestBrowser(t, handler)
	before, _ := workspace.Snapshot()
	staged := browser.postJSON(t, "/api/studio/webmcp/proposals", map[string]any{
		"expectedRevision": before.Revision,
		"title":            "Stale HTTP review",
		"operations":       []map[string]any{{"kind": "rename-entity", "target": "board", "name": "Stale Board"}},
	})
	if staged.Code != http.StatusOK {
		t.Fatalf("stage status = %d: %s", staged.Code, staged.Body.String())
	}
	var proposal struct {
		ProposalID string `json:"proposalId"`
	}
	if err := json.Unmarshal(staged.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	_, canonical, err := workspace.Execute(studio.Transaction{
		ID: "concurrent-http-human", Actor: "human://other", Mode: studio.ModeDirect,
		ExpectedRevision: before.Revision,
		Operations:       []studio.Operation{{Kind: studio.OpRenameEntity, Target: "board", Name: "Canonical Board"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	commit := browser.postJSON(t, "/api/studio/webmcp/commits", map[string]any{"proposalId": proposal.ProposalID})
	if commit.Code != http.StatusConflict {
		t.Fatalf("stale commit status = %d, want 409: %s", commit.Code, commit.Body.String())
	}
	current := browser.getJSON(t, "/api/studio/webmcp/proposals/current")
	if current.Code != http.StatusOK || strings.Contains(current.Body.String(), proposal.ProposalID) {
		t.Fatalf("stale proposal remained current: %d: %s", current.Code, current.Body.String())
	}
	after, _ := workspace.Snapshot()
	if after.Revision != canonical.Revision || after.Entities["board"].Name != "Canonical Board" {
		t.Fatal("stale HTTP commit changed canonical scene")
	}
}

func TestProductionSessionCookieAttributes(t *testing.T) {
	t.Setenv("GOSX_ENV", "production")
	handler, _ := newTestStudio(t)
	browser := newWebMCPTestBrowser(t, handler)
	var cookie *http.Cookie
	for _, candidate := range browser.cookies {
		cookie = candidate
		break
	}
	if cookie == nil {
		t.Fatal("production browser session has no cookie")
	}
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("production cookie attributes: Secure=%t HttpOnly=%t SameSite=%v Path=%q", cookie.Secure, cookie.HttpOnly, cookie.SameSite, cookie.Path)
	}
}
