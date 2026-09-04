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
		`activateScenePreview(proposal)`,
	)
	if !strings.Contains(review, "discoverPendingProposal();") {
		t.Fatal("review UI defines pending-proposal hydration but never invokes it")
	}
}

func TestWebMCPProposalUsesReversibleLiveSceneCommands(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	page := readWebMCPFixture(t, "app/page.gsx")
	requireSourceFragments(t, adapter, "proposal command transport",
		"sceneCommands", "reverseSceneCommands",
	)
	activate := javascriptFunctionSource(t, review, "activateScenePreview")
	requireSourceFragments(t, activate, "live proposal preview",
		"dispatchSceneCommands", "reverseSceneCommands", "sceneCommands", "markScenePreview",
	)
	revert := javascriptFunctionSource(t, review, "revertScenePreview")
	requireSourceFragments(t, revert, "live proposal rollback",
		"reverseSceneCommands", "dispatchSceneCommands", "markScenePreview(restored ? false : true)",
	)
	requireSourceFragments(t, page, "live proposal preview disclosure",
		"data-webmcp-preview-badge", "Agent preview · not committed",
	)
}

func TestWebMCPPreviewLocksCanonicalEditsAndSelfCleansOnExpiry(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	lock := javascriptFunctionSource(t, review, "lockSceneMutationControls")
	requireSourceFragments(t, lock, "review mutation lock",
		"form[data-gosx-form]", "[data-gizmo-mode]", "data-webmcp-review-locked",
		"data-webmcp-review-enabled",
	)
	render := javascriptFunctionSource(t, review, "renderProposal")
	requireSourceFragments(t, render, "review lock activation", "lockSceneMutationControls(true)")
	clear := javascriptFunctionSource(t, review, "clearProposal")
	requireSourceFragments(t, clear, "review lock release", "lockSceneMutationControls(false)")
	expiry := javascriptFunctionSource(t, review, "renderProposalExpiry")
	requireSourceFragments(t, expiry, "expired preview cleanup",
		"remaining <= 0", "discardProposal(null)", "restoring canonical scene",
	)
	revert := javascriptFunctionSource(t, review, "revertScenePreview")
	requireSourceFragments(t, revert, "rollback failure disclosure",
		"activeScenePreview = matchedPreview", "markScenePreview(restored ? false : true)",
	)
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
	if !strings.Contains(commit[terminalFailures:], "revertScenePreview(proposal.proposalId)") ||
		!strings.Contains(commit[terminalFailures:], "refreshPage()") {
		t.Error("terminal commit failures do not restore the current canonical viewport")
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
	if !strings.Contains(page, "data-webmcp-proposal-policy") || !strings.Contains(review, "checks passed") {
		t.Fatal("human review does not surface the governed operation decision")
	}
	if !strings.Contains(review, "navigation.revalidate") || !strings.Contains(review, "force: true, revalidate: true") {
		t.Fatal("human review does not force a same-URL GoSX refresh after apply or reset")
	}
	if !strings.Contains(review, `["position", "rotation", "scale"]`) {
		t.Fatal("human review does not identify the transform field that actually changed")
	}
	uiScript := strings.Index(page, `script src="/studio-webmcp-ui.js"`)
	adapterScript := strings.Index(page, `script src="/studio-webmcp.js"`)
	if uiScript < 0 || adapterScript < 0 || uiScript > adapterScript {
		t.Fatal("review UI must subscribe before the adapter registers and emits status")
	}
}

func TestWebMCPFocusNavigationRetriesOnlyWhileItIsTheLatestIntent(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	focus := javascriptFunctionSource(t, review, "focusEntity")
	requireSourceFragments(t, focus, "focus navigation intent",
		`pendingFocusNavigation = intent`,
		`scheduleFocusNavigation(intent)`,
		`window.__gosxStudioSelection.apply(id)`,
	)
	retry := javascriptFunctionSource(t, review, "scheduleFocusNavigation")
	requireSourceFragments(t, retry, "bounded focus navigation retry",
		`MAX_FOCUS_NAVIGATION_ATTEMPTS`,
		`navigation.navigate(target, { preserveScroll: true })`,
		`applied !== false`,
		`focusNavigationLanded(intent, window.location.href)`,
	)
	for _, required := range []string{
		`document.addEventListener("gosx:navigate"`,
		`document.addEventListener("gosx:scene3d:input"`,
		`event.target.closest("a[data-entity-id]")`,
		`clearAgentFocus()`,
	} {
		if !strings.Contains(review, required) {
			t.Errorf("latest-focus reconciliation missing %q", required)
		}
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
	reset := javascriptFunctionSource(t, review, "resetDemo")
	if !strings.Contains(reset, `clearAgentFocus()`) || !strings.Contains(reset, "refreshPage(window.location.pathname)") {
		t.Fatal("demo reset does not clear the prior agent focus and query-string selection")
	}
	if strings.Contains(adapter, "/api/studio/demo/reset") || strings.Contains(adapter, "data-studio-demo-reset") {
		t.Fatal("WebMCP adapter must not expose the human-only demo reset")
	}
}

func TestWebMCPAdapterEmitsPersistentOutcomeReceipts(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	page := readWebMCPFixture(t, "app/page.gsx")

	for _, required := range []string{
		`trace: "studio:webmcp:trace"`,
		`"Inspect · revision "`,
		`"Find · "`,
		`"Focus · "`,
		`"Stage · "`,
		`emitTrace(callId, tool, "complete"`,
	} {
		if !strings.Contains(adapter, required) {
			t.Errorf("adapter trace contract missing %q", required)
		}
	}
	for _, required := range []string{
		`traceStorageKey`,
		`window.sessionStorage.setItem(traceStorageKey`,
		`document.addEventListener("studio:webmcp:trace"`,
		`traceEntries.slice(-8)`,
		`renderTrace()`,
	} {
		if !strings.Contains(review, required) {
			t.Errorf("persistent trace UI contract missing %q", required)
		}
	}
	if !strings.Contains(page, "data-webmcp-trace") {
		t.Error("page does not expose the visible WebMCP outcome trace")
	}
}

func TestWebMCPAdapterRejectsNoOpProposals(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	normalize := javascriptFunctionSource(t, adapter, "normalizeOperations")
	for _, required := range []string{
		`"ALREADY_SATISFIED"`,
		`entity.name === name`,
		`entity.mesh.material === material`,
		`vec3Equal(position, current.position)`,
		`operations cancel out or already match`,
	} {
		if !strings.Contains(normalize, required) {
			t.Errorf("no-op rejection contract missing %q", required)
		}
	}
}

func TestWebMCPReviewLocksBothTerminalActionsAndShowsTrustEvidence(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	page := readWebMCPFixture(t, "app/page.gsx")
	buttons := javascriptFunctionSource(t, review, "reviewButtons")
	requireSourceFragments(t, buttons, "mutually exclusive review actions",
		`[data-webmcp-commit], [data-webmcp-discard]`,
		`action.disabled = disabled === true`,
	)
	commit := javascriptFunctionSource(t, review, "commitProposal")
	discard := javascriptFunctionSource(t, review, "discardProposal")
	if !strings.Contains(commit, "reviewButtons(true)") || !strings.Contains(discard, "reviewButtons(true)") {
		t.Fatal("Apply and Discard do not both lock the review controls before their requests")
	}
	for _, required := range []string{
		"data-webmcp-proposal-policy-reasons",
		"data-webmcp-proposal-expiry",
		"data-webmcp-proposal-fingerprint",
	} {
		if !strings.Contains(page, required) || !strings.Contains(review, required) {
			t.Errorf("proposal trust evidence is missing shared hook %q", required)
		}
	}
}

func TestWebMCPReviewKeepsTheHumanDecisionVisuallyPrimary(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	page := readWebMCPFixture(t, "app/page.gsx")
	styles := readWebMCPFixture(t, "public/styles.css")

	scroll := javascriptFunctionSource(t, review, "scrollAgentPanelTop")
	requireSourceFragments(t, scroll, "terminal review scroll",
		`one(".agent-panel")`,
		`scrollTo({ top: 0, behavior: "auto" })`,
		`agentPanel.scrollTop = 0`,
	)
	for _, functionName := range []string{"resetDemo", "commitProposal", "discardProposal"} {
		terminal := javascriptFunctionSource(t, review, functionName)
		if !strings.Contains(terminal, "scrollAgentPanelTop()") {
			t.Errorf("%s does not reveal the WebMCP status header before its successful refresh", functionName)
		}
	}
	for _, required := range []string{
		`.agent-panel.has-pending-proposal .agent-actions`,
		`.agent-panel.has-pending-proposal .webmcp-flow`,
		`grid-template-columns: auto minmax(0, 1fr)`,
		`.agent-panel.has-pending-proposal::after`,
		`flex: 0 0 var(--space-xl)`,
		`.studio-shell[data-studio-demo="true"] .application-menu > button:disabled`,
		`display: none`,
	} {
		if !strings.Contains(styles, required) {
			t.Errorf("pending review visual hierarchy is missing %q", required)
		}
	}
	for _, required := range []string{
		"Canonical material",
		"One ephemeral scene shared across visitors.",
		"Find 1 object in 150. Stage 2 exact edits. Keep the only Apply.",
		"Native WebMCP",
		"scene_get_state",
		"scene_find_objects",
		"scene_focus_object",
		"scene_preview_actions",
		"0 commit tools",
		"no auto-commit",
		"WebMCP tool receipts",
		"Try it in 30 seconds",
		"data-webmcp-preview-changes",
		"data-webmcp-approval-outcome",
		"data-webmcp-review-gate",
	} {
		if !strings.Contains(page, required) {
			t.Errorf("review truth copy is missing %q", required)
		}
	}
	for _, required := range []string{
		"Browser agents can find and preview exact scene edits. Only you can apply them.",
	} {
		if !strings.Contains(adapter, required) {
			t.Errorf("judge-facing WebMCP value copy is missing %q", required)
		}
	}
	for _, required := range []string{
		"awaiting your review",
		"Human approved ",
		"same reviewed transaction",
		"showApprovalOutcome",
		"Human-only approval · creates revision ",
	} {
		if !strings.Contains(review, required) {
			t.Errorf("judge-facing proposal outcome copy is missing %q", required)
		}
	}
	for _, required := range []string{
		`.studio-shell[data-studio-demo="true"] .judge-value-card`,
		`.scene-stage[data-webmcp-preview="true"] .judge-value-card`,
		`--panel-agent: 23rem`,
		`.viewport-preview-card`,
		`.viewport-approval-outcome`,
		`position: sticky`,
		`pointer-events: none`,
	} {
		if !strings.Contains(styles, required) {
			t.Errorf("judge-facing demo presentation is missing %q", required)
		}
	}
	missionIndex := strings.Index(page, `class="webmcp-demo-mission"`)
	receiptsIndex := strings.Index(page, `class="webmcp-trace-shell"`)
	if missionIndex < 0 || receiptsIndex < 0 || missionIndex > receiptsIndex {
		t.Fatal("the 30-second judge task must appear before lower-priority WebMCP receipts")
	}
	if !strings.Contains(adapter, "· visible UI") {
		t.Error("focus receipt does not distinguish visible UI synchronization")
	}
}

func TestPublicDemoStatusPollingCannotRaceAnActiveReset(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	discover := javascriptFunctionSource(t, review, "discoverDemoState")
	requireSourceFragments(t, discover, "serialized demo status polling",
		`demoResetInFlight || demoStatusInFlight`,
		`generation = ++demoStatusGeneration`,
		`generation !== demoStatusGeneration || demoResetInFlight`,
		`demoStatusInFlight = false`,
	)
	reset := javascriptFunctionSource(t, review, "resetDemo")
	requireSourceFragments(t, reset, "exclusive public demo reset",
		`if (demoResetInFlight) return`,
		`demoResetInFlight = true`,
		`demoStatusGeneration++`,
		`demoResetInFlight = false`,
	)
	if !strings.Contains(review, "Could not verify the shared baseline · retrying automatically.") {
		t.Fatal("a transient demo-status failure still disappears instead of preserving a visible retry state")
	}
}

func TestStudioCameraHotkeysDoNotCollideWithHierarchyNavigation(t *testing.T) {
	camera := readWebMCPFixture(t, "public/studio-camera.js")
	interactions := readWebMCPFixture(t, "public/studio-interactions.js")
	page := readWebMCPFixture(t, "app/page.gsx")
	requireSourceFragments(t, camera, "interactive camera shortcut guard",
		`isInteractiveTarget(event.target)`,
		`[role='treeitem']`,
		`event.defaultPrevented`,
	)
	requireSourceFragments(t, interactions, "keyboard-operable hierarchy",
		`event.key === " "`,
		`current.click()`,
		`syncHierarchyRoving(next)`,
	)
	if !strings.Contains(page, `data-hierarchy-row data-entity-name={item.name} data-hierarchy-id={item.id} data-entity-type={item.kind} role="none"`) {
		t.Fatal("hierarchy row wrappers must be role=none so links own the treeitem semantics")
	}
}

func TestPublicDemoPromptRequiresCleanBaseline(t *testing.T) {
	review := readWebMCPFixture(t, "public/studio-webmcp-ui.js")
	discover := javascriptFunctionSource(t, review, "discoverDemoState")
	requireSourceFragments(t, discover, "clean baseline gate",
		`state.clean === true`,
		`copy.disabled = !demoClean`,
		`"Prepare clean demo"`,
		`"Clean baseline ready`,
	)
	copyPrompt := javascriptFunctionSource(t, review, "copyDemoPrompt")
	if !strings.Contains(copyPrompt, "if (!demoClean)") {
		t.Fatal("copy handler does not enforce the clean baseline when invoked directly")
	}
}
