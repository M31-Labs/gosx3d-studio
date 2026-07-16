package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"m31labs.dev/gosx/desktop"
	"m31labs.dev/gosx/session"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

func TestViewportSelectionBridgeConsumesSceneMountInput(t *testing.T) {
	asset, err := os.ReadFile("public/studio-selection.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"gosx:scene3d:input", "input.selectedID", "/api/studio/viewport-selection", "input.worldX", "payload.world", "confirmation.selected", "input.rayOriginX", "payload.ray"} {
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
	for _, required := range []string{"studio.viewport.gizmoMode", "__gosx_set_shared_signal", "data-gizmo-mode", "gizmo-commit", "/api/studio/gizmo-commit", "input.phase"} {
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
	unauthorized := httptest.NewRequest(http.MethodPost, "/api/studio/transactions/call", nil)
	unauthorized.Header.Set("Accept", "application/json")
	unauthorizedResult := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResult, unauthorized)
	if unauthorizedResult.Code != http.StatusForbidden {
		t.Fatalf("unauthorized status = %d", unauthorizedResult.Code)
	}
}

func TestCameraRigBridgeDrivesCameraSignals(t *testing.T) {
	asset, err := os.ReadFile("public/studio-camera.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"studio.viewport.cameraIn", "studio.viewport.cameraOut", "orthographic", "data-camera-view", "data-camera-focus-x"} {
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
