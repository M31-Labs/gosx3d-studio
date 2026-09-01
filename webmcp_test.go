package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
