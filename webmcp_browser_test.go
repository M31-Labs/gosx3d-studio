package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var registeredWebMCPToolPattern = regexp.MustCompile(`document\.modelContext\.registerTool\(\s*\{\s*name\s*:\s*"([^"]+)"`)

func readWebMCPFixture(t *testing.T, path string) string {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(value)
}

func javascriptFunctionSource(t *testing.T, source, name string) string {
	t.Helper()
	marker := "function " + name + "("
	start := strings.Index(source, marker)
	if start < 0 {
		t.Fatalf("JavaScript function %s was not found", name)
	}
	openingOffset := strings.IndexByte(source[start:], '{')
	if openingOffset < 0 {
		t.Fatalf("JavaScript function %s has no body", name)
	}
	opening := start + openingOffset
	depth := 0
	var quote byte
	escaped := false
	lineComment := false
	blockComment := false
	for index := opening; index < len(source); index++ {
		character := source[index]
		var next byte
		if index+1 < len(source) {
			next = source[index+1]
		}
		if lineComment {
			if character == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if character == '*' && next == '/' {
				blockComment = false
				index++
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '/' && next == '/' {
			lineComment = true
			index++
			continue
		}
		if character == '/' && next == '*' {
			blockComment = true
			index++
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		switch character {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start : index+1]
			}
		}
	}
	t.Fatalf("JavaScript function %s has an unterminated body", name)
	return ""
}

func requireSourceFragments(t *testing.T, source, contract string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(source, fragment) {
			t.Errorf("%s is missing %q", contract, fragment)
		}
	}
}

func TestWebMCPAdapterRegistersThePublicToolContract(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	matches := registeredWebMCPToolPattern.FindAllStringSubmatch(adapter, -1)
	if len(matches) != 4 {
		t.Fatalf("registered tool count = %d, want 4", len(matches))
	}
	want := map[string]bool{
		"scene_get_state":       true,
		"scene_find_objects":    true,
		"scene_focus_object":    true,
		"scene_preview_actions": true,
	}
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		name := match[1]
		if seen[name] {
			t.Errorf("adapter registers tool %q more than once", name)
		}
		seen[name] = true
		if !want[name] {
			t.Errorf("adapter registers unexpected tool %q", name)
		}
		if strings.Contains(strings.ToLower(name), "commit") {
			t.Errorf("agent adapter registers commit-like tool %q", name)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("adapter does not register tool %q", name)
		}
	}
	if !strings.Contains(adapter, `"/api/studio/webmcp/proposals"`) {
		t.Fatal("preview tool does not call the session proposal endpoint")
	}
	if strings.Contains(adapter, "/api/studio/webmcp/commits") {
		t.Fatal("agent adapter must not expose or call the human commit endpoint")
	}
	if strings.Contains(strings.ToLower(adapter), "authorization") {
		t.Fatal("browser adapter must not contain bearer authorization logic")
	}
	if strings.Contains(adapter, "cannot scale a group") || !strings.Contains(adapter, "entity.light &&") {
		t.Fatal("adapter must expose GoSX v0.54 group scale while rejecting meaningless light scale")
	}
}

func TestWebMCPReviewHydratesTheCurrentSessionProposal(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	hydration := javascriptFunctionSource(t, review, "discoverPendingProposal")
	requireSourceFragments(t, hydration, "pending-proposal hydration",
		`"/api/studio/webmcp/proposals/current"`,
		`method: "GET"`,
		`cache: "no-store"`,
		`credentials: "same-origin"`,
		`generation !== proposalHydration`,
		`payload && payload.proposal`,
		`pendingProposal = proposal`,
		`renderProposal()`,
	)
	if !strings.Contains(review, "discoverPendingProposal();") {
		t.Fatal("review UI defines pending-proposal hydration but never invokes it")
	}
}

func TestWebMCPReviewRevokesDiscardedProposalsOnTheServer(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	discard := javascriptFunctionSource(t, review, "discardProposal")
	requireSourceFragments(t, discard, "proposal discard",
		`"/api/studio/webmcp/discards"`,
		`method: "POST"`,
		`JSON.stringify({ proposalId: proposalId })`,
		`proposalHydration++`,
		`clearProposal(`,
	)
	if !strings.Contains(review, "discardProposal(discard)") {
		t.Fatal("discard control does not invoke the server-backed discard flow")
	}
}

func TestWebMCPReviewPropagatesHTTPStatusAndClearsTerminalCommitFailures(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	responseError := javascriptFunctionSource(t, review, "responseError")
	requireSourceFragments(t, responseError, "HTTP error propagation",
		`error.status`,
		`response.status`,
	)

	commit := javascriptFunctionSource(t, review, "commitProposal")
	for _, status := range []string{"404", "409", "410"} {
		if !strings.Contains(commit, "error.status === "+status) {
			t.Errorf("terminal commit handling does not recognize HTTP %s", status)
		}
	}
	terminalFailures := strings.Index(commit, "error.status === 404")
	if terminalFailures < 0 || !strings.Contains(commit[terminalFailures:], "clearProposal(") {
		t.Error("terminal commit failures do not clear the stale proposal from review")
	}
	if !strings.Contains(commit, "Revision conflict") || !strings.Contains(commit, "restage") {
		t.Error("revision-conflict handling does not direct the user to inspect and restage")
	}
}

func TestWebMCPHumanReviewRemainsASeparateUIStep(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	page := readWebMCPFixture(t, "app/page.gsx")

	for _, event := range []string{"studio:webmcp:status", "studio:webmcp:focus", "studio:webmcp:proposal"} {
		if !strings.Contains(adapter, event) || !strings.Contains(review, event) {
			t.Errorf("adapter and review UI do not share event %q", event)
		}
	}
	if !strings.Contains(review, `"/api/studio/webmcp/commits"`) {
		t.Fatal("review UI does not call the human commit endpoint")
	}
	if !strings.Contains(review, `headers["X-CSRF-Token"]`) {
		t.Fatal("review UI does not explicitly forward the rendered CSRF token")
	}
	if strings.Contains(strings.ToLower(review), "authorization") {
		t.Fatal("review UI must use the browser session, not a bearer credential")
	}
	if !strings.Contains(page, "data-webmcp-commit") || !strings.Contains(page, "Apply staged changes") {
		t.Fatal("page has no explicit human proposal approval control")
	}
	if !strings.Contains(page, "data-webmcp-proposal-policy") || !strings.Contains(review, "Arbiter · Allow") {
		t.Fatal("human review does not surface the governed operation decision")
	}
	if !strings.Contains(review, "navigation.revalidate") || !strings.Contains(review, "force: true, revalidate: true") {
		t.Fatal("human review does not force a same-URL GoSX refresh after apply or reset")
	}
	if !strings.Contains(review, `["position", "rotation", "scale"]`) {
		t.Fatal("human review does not identify the transform field that actually changed")
	}
	uiScript := strings.Index(page, `/studio-webmcp-ui.js`)
	adapterScript := strings.Index(page, `/studio-webmcp.js`)
	if uiScript < 0 || adapterScript < 0 || uiScript > adapterScript {
		t.Fatal("review UI must subscribe before the adapter registers and emits status")
	}
}

func TestCoreStudioInteractionPolishContracts(t *testing.T) {
	page := readWebMCPFixture(t, "app/page.gsx")
	interactions := readWebMCPFixture(t, "public/studio-interactions.js")
	styles := readWebMCPFixture(t, "public/styles.css")
	if !strings.Contains(page, "data-scene-runtime-status") || !strings.Contains(interactions, "data-gosx-scene3d-ready") {
		t.Fatal("footer does not report the observed Scene3D runtime state")
	}
	if !strings.Contains(styles, ".tree li[hidden]") || !strings.Contains(styles, `button[aria-pressed="true"]`) {
		t.Fatal("filtered hierarchy rows or active camera controls lack visible CSS state")
	}
	for _, required := range []string{`role="tree"`, `role="treeitem"`, `href="#inspector-panel"`} {
		if !strings.Contains(page, required) {
			t.Fatalf("hierarchy accessibility contract missing %q", required)
		}
	}
	if !strings.Contains(interactions, "syncHierarchyRoving") || !strings.Contains(interactions, `event.key === "ArrowDown"`) {
		t.Fatal("hierarchy does not implement a bounded keyboard focus model")
	}
	for _, required := range []string{
		`data-selection-id={data.inspector.id}`,
		`selected={material.id == data.inspector.materialId}`,
		`disabled={!data.timeline.boneAvailable}`,
		`disabled={!data.timeline.clipAvailable}`,
		`disabled={!data.timeline.simulationAvailable}`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("core editor polish contract missing %q", required)
		}
	}
}

func TestPublicDemoResetIsVisibleAndNeverAgentCallable(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	page := readWebMCPFixture(t, "app/page.gsx")
	if !strings.Contains(page, "Reset shared scene") || !strings.Contains(page, "Shared public demo") {
		t.Fatal("page has no visible shared-demo disclosure and reset control")
	}
	if !strings.Contains(review, `"/api/studio/demo/reset"`) || !strings.Contains(review, "window.confirm") {
		t.Fatal("human UI does not explicitly confirm and call the demo reset endpoint")
	}
	if strings.Contains(adapter, "/api/studio/demo/reset") || strings.Contains(adapter, "data-studio-demo-reset") {
		t.Fatal("WebMCP adapter must not expose the human-only demo reset")
	}
}
