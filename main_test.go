package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"m31labs.dev/gosx/desktop"
	"m31labs.dev/gosx/server"
	"m31labs.dev/gosx/session"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

func TestViewportSelectionBridgeConsumesSceneMountInput(t *testing.T) {
	asset, err := os.ReadFile("public/studio-selection.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"gosx:scene3d:input", `input.type !== "select"`, "input.selectedID", "/api/studio/viewport-selection", "input.worldX", "payload.world", "confirmation.selected", "input.rayOriginX", "payload.ray", `headers["X-CSRF-Token"]`, "__gosx_page_nav"} {
		if !strings.Contains(string(asset), required) {
			t.Fatalf("selection bridge missing %q", required)
		}
	}
}

func TestGizmoBridgeDrivesSharedModeSignal(t *testing.T) {
	asset, err := os.ReadFile("public/studio-gizmo.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"studio.viewport.gizmoMode", "studio.viewport.selectedID", "data-selection-id", "gosx:ready", "__gosx_runtime_api", "setSharedSignalValue", "__gosx_notify_shared_signal", "data-gizmo-mode", "gizmo-commit", "/api/studio/gizmo-commit", `headers["X-CSRF-Token"]`, "input.phase", "__gosx_page_nav"} {
		if !strings.Contains(string(asset), required) {
			t.Fatalf("gizmo bridge missing %q", required)
		}
	}
	page, err := os.ReadFile("app/page.gsx")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`data-gizmo-mode="translate"`, `data-gizmo-mode="rotate"`, `data-gizmo-mode="scale"`, "/studio-gizmo.js"} {
		if !strings.Contains(string(page), required) {
			t.Fatalf("page toolbar missing %q", required)
		}
	}
}

func TestAssetContentHandlerVerifiesHashAndServesImmutableBytes(t *testing.T) {
	project := t.TempDir()
	input := filepath.Join(t.TempDir(), "model.gltf")
	payload := []byte(`{"asset":{"version":"2.0"}}`)
	if err := os.WriteFile(input, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := studio.OpenWorkspace(project, studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	document, _ := workspace.Snapshot()
	_, _, asset, err := workspace.ImportAsset(studio.AssetImportRequest{Path: input, Actor: "test", Mode: studio.ModeDirect, ExpectedRevision: document.Revision})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, asset.URI, nil)
	result := httptest.NewRecorder()
	assetContentHandler(workspace).ServeHTTP(result, request)
	if result.Code != http.StatusOK || result.Body.String() != string(payload) || !strings.Contains(result.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset response status=%d headers=%v body=%q", result.Code, result.Header(), result.Body.String())
	}
	path, _, _ := workspace.AssetContentPath(asset.ID)
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	result = httptest.NewRecorder()
	assetContentHandler(workspace).ServeHTTP(result, request)
	if result.Code != http.StatusConflict {
		t.Fatalf("tampered asset status = %d", result.Code)
	}
}

func TestDesktopProjectBridgeUsesNativeDialogAndRevisionSafeEndpoint(t *testing.T) {
	asset, err := os.ReadFile("public/studio-project.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"gosxDesktop.dialog.openFile", "/api/studio/project/open-from-desktop", "expectedRevision", "discardUnsaved", "X-GoSX-Desktop-Intent"} {
		if !strings.Contains(string(asset), required) {
			t.Fatalf("project bridge missing %q", required)
		}
	}
}

func TestDesktopAssetBridgeUsesNativeDialogAndSharedHumanForm(t *testing.T) {
	asset, err := os.ReadFile("public/studio-project.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"#studio-choose-asset", "studio-asset-path", "gosxDesktop.dialog.openFile",
		"*.glb;*.gltf;*.png;*.jpg;*.jpeg;*.wav;*.ogg;*.mp4;*.obj",
		"Native chooser unavailable; enter a trusted local path.", "confirm import",
	} {
		if !strings.Contains(string(asset), required) {
			t.Fatalf("asset dialog bridge missing %q", required)
		}
	}
	page, err := os.ReadFile("app/page.gsx")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`id="studio-asset-import"`, `actionPath("importAsset")`, `id="studio-choose-asset"`, `aria-live="polite"`} {
		if !strings.Contains(string(page), required) {
			t.Fatalf("asset import form missing %q", required)
		}
	}
	menu := studioDesktopMenu(nil)
	if len(menu.Items) == 0 || menu.Items[0].Submenu == nil {
		t.Fatal("desktop File menu missing")
	}
	found := false
	for _, item := range menu.Items[0].Submenu.Items {
		found = found || item.ID == "file.import-asset"
	}
	if !found {
		t.Fatal("desktop File menu does not expose Import Asset")
	}
}

func TestAssetGarbageCollectionHasHumanAndAgentSurfaces(t *testing.T) {
	page, err := os.ReadFile("app/page.gsx")
	if err != nil {
		t.Fatal(err)
	}
	serverSource, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"collectAssets", "confirmPlan", "Checkpoint and collect"} {
		if !strings.Contains(string(page), required) {
			t.Fatalf("human asset collection missing %q", required)
		}
	}
	if !strings.Contains(string(serverSource), "/api/studio/assets/garbage-collect") {
		t.Fatal("agent asset collection endpoint missing")
	}
}

func TestStudioActionAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/studio/transactions/call", nil)
	if err := authorizeAction(request, "token"); err == nil {
		t.Fatal("missing bearer token was accepted")
	}
	request.Header.Set("Authorization", "Bearer token")
	if err := authorizeAction(request, "token"); err != nil {
		t.Fatalf("valid bearer token: %v", err)
	}
}

func TestWindowsDefaultsToDesktopHostWithExplicitServerOverride(t *testing.T) {
	if !desktopMode("windows", "", "") {
		t.Fatal("Windows did not default to the desktop host")
	}
	if desktopMode("windows", "", "1") {
		t.Fatal("server-only override did not disable the desktop host")
	}
	if desktopMode("linux", "", "") {
		t.Fatal("unsupported Linux host defaulted to desktop")
	}
	if !desktopMode("linux", "1", "") {
		t.Fatal("explicit desktop request was ignored")
	}
}

func TestDesktopMenuAndPlatformDiagnosticsAreMachineReadable(t *testing.T) {
	menu := studioDesktopMenu(nil)
	plan, err := desktop.BuildMenuPlan(menu)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 2 || plan.NextCommandID < 6 {
		t.Fatalf("menu plan = %+v", plan)
	}
	report := currentPlatformReport()
	if report.Schema != "gosx3d.studio.platform/v1" || report.OS == "" || report.Architecture == "" || report.ProjectSemantics != "portable" {
		t.Fatalf("platform report = %+v", report)
	}
}

func TestStudioBearerBypassesCSRFButOtherPostsDoNot(t *testing.T) {
	sessions, err := session.New("01234567890123456789012345678901", session.Options{})
	if err != nil {
		t.Fatal(err)
	}
	hit := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(http.StatusNoContent) })
	handler := sessions.Middleware(studioCSRF(sessions, "token")(hit))
	authorized := httptest.NewRequest(http.MethodPost, "/api/studio/transactions/call", nil)
	authorized.Header.Set("Authorization", "Bearer token")
	authorizedResult := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResult, authorized)
	if authorizedResult.Code != http.StatusNoContent {
		t.Fatalf("authorized status = %d", authorizedResult.Code)
	}
	sessionRoute := httptest.NewRequest(http.MethodPost, "/api/studio/webmcp/proposals", nil)
	sessionRoute.Header.Set("Accept", "application/json")
	sessionRoute.Header.Set("Authorization", "Bearer token")
	sessionResult := httptest.NewRecorder()
	handler.ServeHTTP(sessionResult, sessionRoute)
	if sessionResult.Code != http.StatusForbidden {
		t.Fatalf("bearer-only session route status = %d, want 403", sessionResult.Code)
	}
	unauthorized := httptest.NewRequest(http.MethodPost, "/api/studio/transactions/call", nil)
	unauthorized.Header.Set("Accept", "application/json")
	unauthorizedResult := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResult, unauthorized)
	if unauthorizedResult.Code != http.StatusForbidden {
		t.Fatalf("unauthorized status = %d", unauthorizedResult.Code)
	}
	telemetry := httptest.NewRequest(http.MethodPost, server.ClientEventsRoute, strings.NewReader(`{"events":[]}`))
	telemetryResult := httptest.NewRecorder()
	handler.ServeHTTP(telemetryResult, telemetry)
	if telemetryResult.Code != http.StatusNoContent {
		t.Fatalf("telemetry status = %d, want CSRF-exempt append-only route", telemetryResult.Code)
	}
}

func TestCameraRigBridgeDrivesCameraSignals(t *testing.T) {
	asset, err := os.ReadFile("public/studio-camera.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"studio.viewport.cameraIn", "studio.viewport.cameraOut", "__gosx_runtime_api", "setSharedSignalValue", "__gosx_subscribe_shared_signal", "orthographic", "rotationX", "data-camera-view", "data-camera-focus-x"} {
		if !strings.Contains(string(asset), required) {
			t.Fatalf("camera rig missing %q", required)
		}
	}
	page, err := os.ReadFile("app/page.gsx")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`data-camera-view="perspective"`, `data-camera-view="top"`, "data-camera-home", "/studio-camera.js"} {
		if !strings.Contains(string(page), required) {
			t.Fatalf("page camera controls missing %q", required)
		}
	}
}

func TestStudioRuntimeRootUsesBundledManifestWhenPresent(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "dist")
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "build.json"), []byte(`{"runtime":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := studioRuntimeRoot(root); got != dist {
		t.Fatalf("runtime root = %q, want bundled dist %q", got, dist)
	}
	if got := studioRuntimeRoot(dist); got != dist {
		t.Fatalf("production runtime root = %q, want %q", got, dist)
	}
}

func TestRenderBlueprintBuildsAndRunsThePackagedGoSXArtifact(t *testing.T) {
	blueprint, err := os.ReadFile("render.yaml")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(blueprint)
	for _, required := range []string{
		"runtime: go",
		"numInstances: 1",
		"buildCommand: ./scripts/render-build.sh",
		"startCommand: ./dist/run.sh",
		"healthCheckPath: /api/health",
		"key: GOSX_ENV\n        value: production",
		"key: STUDIO_SERVER_ONLY\n        value: \"1\"",
		"key: STUDIO_DEMO_MODE\n        value: \"1\"",
		"key: SESSION_SECRET\n        generateValue: true",
		"key: STUDIO_ACTION_TOKEN\n        generateValue: true",
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("Render Blueprint is missing production contract %q", required)
		}
	}
	if strings.Contains(contents, "buildCommand: go build") {
		t.Fatal("Render Blueprint must use the GoSX packager so Scene3D runtime assets are deployed")
	}

	buildScript, err := os.ReadFile("scripts/render-build.sh")
	if err != nil {
		t.Fatal(err)
	}
	buildScriptInfo, err := os.Stat("scripts/render-build.sh")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && buildScriptInfo.Mode().Perm()&0o111 == 0 {
		t.Fatal("Render build script must be executable")
	}
	buildContents := string(buildScript)
	for _, required := range []string{
		"#!/bin/sh",
		`studio_tinygo_version="0.41.1"`,
		"sha256sum --check --status",
		`tinygo${studio_tinygo_version}.linux-${studio_tinygo_arch}.tar.gz`,
		"go run m31labs.dev/gosx/cmd/gosx@v0.54.0 build --prod .",
	} {
		if !strings.Contains(buildContents, required) {
			t.Fatalf("Render build script is missing reproducibility contract %q", required)
		}
	}
	for _, forbidden := range []string{"go build ./", "go run .", "build --dev", "cmd/gosx@v0.54.0 build ."} {
		if strings.Contains(buildContents, forbidden) {
			t.Fatalf("Render build script contains non-production fallback %q", forbidden)
		}
	}

	routeConfig, err := os.ReadFile("app/route.config.json")
	if err != nil {
		t.Fatal(err)
	}
	var dynamicRoute struct {
		Prerender *bool `json:"prerender"`
		Cache     struct {
			NoStore *bool `json:"noStore"`
		} `json:"cache"`
	}
	if err := json.Unmarshal(routeConfig, &dynamicRoute); err != nil {
		t.Fatalf("decode app/route.config.json: %v", err)
	}
	if dynamicRoute.Prerender == nil || *dynamicRoute.Prerender {
		t.Fatal("stateful Studio page must not be prerendered with a session-bound CSRF token")
	}
	if dynamicRoute.Cache.NoStore == nil || !*dynamicRoute.Cache.NoStore {
		t.Fatal("stateful Studio page must remain no-store")
	}
}

func TestWindowsProductionPackagingProvisionsPinnedTinyGo(t *testing.T) {
	workflow, err := os.ReadFile(".github/workflows/windows.yml")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(workflow)
	for _, required := range []string{
		`$version = "0.41.1"`,
		"tinygo$version.windows-amd64.zip",
		"56b8ccf2c705b6a5da14b319ecffc73db2850cd5d09681d65022e604311276b5",
		"Get-FileHash -Path $archive -Algorithm SHA256",
		"$env:GITHUB_PATH",
		`..\gosx-cli.exe build --prod --offline .`,
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("Windows production packaging is missing TinyGo contract %q", required)
		}
	}
}
