package main

import (
	"os"
	"strings"
	"testing"
)

func readWebMCPFixture(t *testing.T, path string) string {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(value)
}

func TestWebMCPAdapterRegistersThePublicToolContract(t *testing.T) {
	adapter := readWebMCPFixture(t, "public/studio-webmcp.js")
	if count := strings.Count(adapter, "document.modelContext.registerTool({"); count != 4 {
		t.Fatalf("registerTool count = %d, want 4", count)
	}
	for _, tool := range []string{
		`name: "scene_get_state"`,
		`name: "scene_find_objects"`,
		`name: "scene_focus_object"`,
		`name: "scene_preview_actions"`,
	} {
		if !strings.Contains(adapter, tool) {
			t.Errorf("adapter does not register %s", tool)
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
