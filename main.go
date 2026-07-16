package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/desktop"
	"m31labs.dev/gosx/env"
	"m31labs.dev/gosx/route"
	"m31labs.dev/gosx/server"
	"m31labs.dev/gosx/session"
	studioapp "m31labs.dev/gosx3d-studio/app"
	"m31labs.dev/gosx3d-studio/internal/studio"
	_ "m31labs.dev/gosx3d-studio/modules"
)

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	root := server.ResolveAppRoot(thisFile)
	if err := env.LoadDir(root, ""); err != nil {
		log.Fatal(err)
	}

	appName := getenv("APP_NAME", "GoSX 3D Studio")
	port := getenv("PORT", "8080")
	actionToken := os.Getenv("STUDIO_ACTION_TOKEN")
	sessions, err := session.New(getenv("SESSION_SECRET", "gosx-app-session-secret"), session.Options{})
	if err != nil {
		log.Fatal(err)
	}
	projectDir := getenv("STUDIO_PROJECT_DIR", filepath.Join(root, ".studio"))
	workspace, err := studio.OpenWorkspace(projectDir, studio.SampleDocument())
	if err != nil {
		log.Fatal(err)
	}
	studioapp.BindWorkspace(workspace)

	router := route.NewRouter()
	router.SetLayout(func(ctx *route.RouteContext, body gosx.Node) gosx.Node {
		ctx.SetMetadata(server.Metadata{
			Links: []server.LinkTag{
				{Rel: "stylesheet", Href: "/styles.css"},
			},
		})
		return server.HTMLDocument(ctx.Title(appName), ctx.Head(), body)
	})
	if err := router.AddDir(filepath.Join(root, "app"), route.FileRoutesOptions{}); err != nil {
		log.Fatal(err)
	}

	app := server.New()
	app.EnableISR()
	app.EnableNavigation()
	app.Use(sessions.Middleware)
	app.Use(studioCSRF(sessions, actionToken))
	app.SetPublicDir(filepath.Join(root, "public"))
	app.API("GET /api/health", func(ctx *server.Context) (any, error) {
		ctx.CachePublic(30 * time.Second)
		ctx.CacheTag("health")
		return map[string]any{
			"ok":      true,
			"app":     appName,
			"version": gosx.Version,
			"time":    time.Now().Format(time.RFC3339),
		}, nil
	})
	app.API("GET /api/studio/platform", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return currentPlatformReport(), nil
	})
	app.API("GET /api/studio/manifest", func(ctx *server.Context) (any, error) {
		ctx.CachePublic(30 * time.Second)
		ctx.CacheTag("studio-manifest")
		return studio.DefaultManifest(), nil
	})
	app.API("GET /api/studio/gltf/capabilities", func(ctx *server.Context) (any, error) {
		ctx.CachePublic(30 * time.Second)
		ctx.CacheTag("studio-gltf-capabilities")
		return studio.DefaultGLTFCapabilityMatrix(), nil
	})
	app.API("GET /api/studio/gltf/corpus", func(ctx *server.Context) (any, error) {
		ctx.CachePublic(30 * time.Second)
		ctx.CacheTag("studio-gltf-corpus")
		return studio.DefaultGLTFCorpusManifest(), nil
	})
	app.API("GET /api/studio/document", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return workspace.Snapshot()
	})
	app.API("GET /api/studio/scene-ir", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		ir, err := studio.CompileIR(document)
		if err != nil {
			return nil, err
		}
		return ir, nil
	})
	app.API("GET /api/studio/artifact", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		return studio.CompileWithSourceMap(document)
	})
	app.API("GET /api/studio/actions", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return studio.ActionDescriptors(), nil
	})
	app.API("GET /api/studio/descriptors", func(ctx *server.Context) (any, error) {
		ctx.CachePublic(30 * time.Second)
		return studio.ComponentCatalog(), nil
	})
	app.API("GET /api/studio/certification", func(ctx *server.Context) (any, error) { ctx.NoStore(); return studio.Certification(), nil })
	app.API("GET /api/studio/certification/evidence", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		return studio.CertifyCurrent(document)
	})
	app.API("GET /api/studio/initialize", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		return studio.BuildInitializeReport(document)
	})
	app.API("GET /api/studio/selection", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return map[string]any{"selection": workspace.Selection(), "state": workspace.SelectionState(), "viewportConfirmation": workspace.ViewportConfirmation()}, nil
	})
	app.API("GET /api/studio/geometry/analysis", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		return studio.AnalyzeEntityGeometry(document, studio.ID(ctx.Request.URL.Query().Get("target")))
	})
	app.API("GET /api/studio/curve/analysis", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		return studio.AnalyzeEntityCurve(document, studio.ID(ctx.Request.URL.Query().Get("target")))
	})
	app.API("GET /api/studio/rig/skin", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		return studio.InspectSkinDeformation(document, studio.ID(ctx.Request.URL.Query().Get("target")))
	})
	app.API("POST /api/studio/selection", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		ctx.NoStore()
		var request studio.SelectionRequest
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		if err := workspace.SelectSubobjects(request); err != nil {
			return nil, commandError(err)
		}
		return workspace.SelectionState(), nil
	})
	app.API("GET /api/studio/project/status", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return workspace.ProjectStatus(), nil
	})
	app.API("POST /api/studio/project/save", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		ctx.NoStore()
		return workspace.Save()
	})
	app.API("POST /api/studio/project/open", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		return switchProjectRequest(ctx, workspace)
	})
	app.API("POST /api/studio/project/open-from-desktop", func(ctx *server.Context) (any, error) {
		if ctx.Request.Header.Get("X-GoSX-Desktop-Intent") != "native-dialog" {
			return nil, statusError{http.StatusForbidden, fmt.Errorf("native desktop project intent is required")}
		}
		return switchProjectRequest(ctx, workspace)
	})
	app.API("POST /api/studio/viewport-selection", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		var request studio.ViewportPick
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		confirmation, err := studio.ConfirmViewportSelection(document, request)
		if err != nil {
			return nil, statusError{http.StatusConflict, err}
		}
		if err := workspace.SelectAtRevision(document.Revision, confirmation.Selected); err != nil {
			return nil, commandError(err)
		}
		workspace.RecordViewportConfirmation(confirmation)
		return confirmation, nil
	})
	app.API("POST /api/studio/pick", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		var request studio.PickRequest
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		result, err := studio.ExactPick(document, request)
		if err != nil {
			return nil, err
		}
		if request.Select && result.Selected != "" {
			if err := workspace.SelectAtRevision(document.Revision, result.Selected); err != nil {
				return nil, commandError(err)
			}
		}
		return result, nil
	})
	app.API("POST /api/studio/actions/preview", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		transaction, err := studio.DecodeTransaction(ctx.Request.Body)
		if err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		transaction.Mode = studio.ModePropose
		receipt, preview, err := workspace.Execute(transaction)
		if err != nil {
			return nil, commandError(err)
		}
		return map[string]any{"receipt": receipt, "document": preview}, nil
	})
	app.API("POST /api/studio/transactions/call", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		transaction, err := studio.DecodeTransaction(ctx.Request.Body)
		if err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		transaction.Mode = studio.ModeDirect
		receipt, document, err := workspace.Execute(transaction)
		if err != nil {
			return nil, commandError(err)
		}
		return map[string]any{"receipt": receipt, "document": document}, nil
	})
	app.API("POST /api/studio/assets/import", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		ctx.NoStore()
		var request studio.AssetImportRequest
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		receipt, document, asset, err := workspace.ImportAsset(request)
		if err != nil {
			return nil, commandError(err)
		}
		return map[string]any{"receipt": receipt, "document": document, "asset": asset}, nil
	})
	app.API("POST /api/studio/assets/reimport", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		ctx.NoStore()
		var request studio.AssetReimportRequest
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		receipt, document, asset, err := workspace.ReimportAsset(request)
		if err != nil {
			return nil, commandError(err)
		}
		return map[string]any{"receipt": receipt, "document": document, "asset": asset}, nil
	})
	app.API("GET /api/studio/assets/audit", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return workspace.AuditAssets(), nil
	})
	app.API("GET /api/studio/assets/dependencies", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		return workspace.AssetDependencies(), nil
	})
	app.API("POST /api/studio/assets/garbage-collect", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		ctx.NoStore()
		var request studio.AssetGCRequest
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		result, err := workspace.CollectAssetGarbage(request)
		if err != nil {
			return nil, commandError(err)
		}
		return result, nil
	})
	app.API("POST /api/studio/undo", func(ctx *server.Context) (any, error) { return historyAction(ctx, workspace, actionToken, true) })
	app.API("POST /api/studio/redo", func(ctx *server.Context) (any, error) { return historyAction(ctx, workspace, actionToken, false) })
	rootHandler, err := router.BuildChecked()
	if err != nil {
		log.Fatal(err)
	}
	app.Mount("/api/studio/assets/content/", assetContentHandler(workspace))
	app.Mount("/", rootHandler)

	if err := runApplication(app, appName, port); err != nil {
		log.Fatal(err)
	}
}

func assetContentHandler(workspace *studio.Workspace) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := studio.ID(strings.TrimPrefix(request.URL.Path, "/api/studio/assets/content/"))
		if id == "" || strings.Contains(string(id), "/") {
			http.NotFound(writer, request)
			return
		}
		path, asset, err := workspace.AssetContentPath(id)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != asset.ContentHash || int64(len(data)) != asset.Bytes {
			http.Error(writer, "asset integrity check failed", http.StatusConflict)
			return
		}
		writer.Header().Set("Content-Type", asset.MediaType)
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		writer.Header().Set("ETag", `"sha256-`+asset.ContentHash+`"`)
		http.ServeContent(writer, request, asset.SourceName, time.Time{}, bytes.NewReader(data))
	})
}

func runApplication(app *server.App, appName, port string) error {
	if !desktopMode(runtime.GOOS, os.Getenv("STUDIO_DESKTOP"), os.Getenv("STUDIO_SERVER_ONLY")) {
		log.Printf("%s listening on http://localhost:%s", appName, port)
		return app.ListenAndServe(":" + port)
	}
	if err := desktop.Available(); err != nil {
		return fmt.Errorf("Studio desktop host: %w", err)
	}
	address := "127.0.0.1:" + port
	baseURL := "http://" + address
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- app.ListenAndServe(address) }()
	if err := waitForLocalStudio(baseURL+"/api/health", serverErrors, 15*time.Second); err != nil {
		return err
	}
	log.Printf("%s desktop host using internal %s", appName, baseURL)
	host, err := desktop.New(desktop.Options{
		Title: appName, Width: 1600, Height: 1000,
		AppID: "m31labs.gosx3d-studio", Version: getenv("STUDIO_VERSION", "0.1.0"),
		URL: baseURL, NativeBridge: true, SingleInstance: true,
		DPIAwareness:  desktop.DPIAwarenessPerMonitorV2,
		Accessibility: desktop.AccessibilityOptions{Enabled: true, Name: appName, Description: "Go-native 3D scene, animation, game, and simulation workbench"},
	})
	if err != nil {
		return err
	}
	if err := host.SetMenuBar(studioDesktopMenu(host)); err != nil {
		return fmt.Errorf("Studio desktop menu: %w", err)
	}
	return host.Run()
}

func studioDesktopMenu(host *desktop.App) desktop.Menu {
	return desktop.Menu{Items: []desktop.MenuItem{
		{Label: "File", Submenu: &desktop.Menu{Items: []desktop.MenuItem{
			{ID: "file.open-project", Label: "Open Project…", OnClick: func() { _ = host.ExecuteScript(`document.getElementById("studio-open-project")?.click()`) }},
			{ID: "file.import-asset", Label: "Import Asset…", OnClick: func() { _ = host.ExecuteScript(`document.getElementById("studio-choose-asset")?.click()`) }},
			{ID: "file.save-project", Label: "Save Project", OnClick: func() { _ = host.ExecuteScript(`document.querySelector(".menu-save-form")?.requestSubmit()`) }},
			{Separator: true},
			{ID: "file.exit", Label: "Exit", OnClick: func() { _ = host.Close() }},
		}}},
		{Label: "Window", Submenu: &desktop.Menu{Items: []desktop.MenuItem{
			{ID: "window.minimize", Label: "Minimize", OnClick: func() { _ = host.Minimize() }},
			{ID: "window.maximize", Label: "Maximize", OnClick: func() { _ = host.Maximize() }},
			{ID: "window.restore", Label: "Restore", OnClick: func() { _ = host.Restore() }},
		}}},
	}}
}

func desktopMode(goos, desktopValue, serverOnlyValue string) bool {
	if strings.TrimSpace(serverOnlyValue) == "1" {
		return false
	}
	if strings.TrimSpace(desktopValue) != "" {
		return strings.TrimSpace(desktopValue) == "1"
	}
	return goos == "windows"
}

type platformReport struct {
	Schema               string `json:"schema"`
	OS                   string `json:"os"`
	Architecture         string `json:"architecture"`
	DesktopBackend       string `json:"desktopBackend"`
	NativeBridgeContract string `json:"nativeBridgeContract"`
	ProjectSemantics     string `json:"projectSemantics"`
	Limitation           string `json:"limitation,omitempty"`
}

func currentPlatformReport() platformReport {
	report := platformReport{
		Schema: "gosx3d.studio.platform/v1", OS: runtime.GOOS, Architecture: runtime.GOARCH,
		DesktopBackend: "unavailable", NativeBridgeContract: "implemented", ProjectSemantics: "portable",
	}
	if err := desktop.Available(); err == nil {
		report.DesktopBackend = "available"
	} else {
		report.Limitation = err.Error()
	}
	return report
}

func waitForLocalStudio(url string, serverErrors <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		select {
		case err := <-serverErrors:
			return fmt.Errorf("Studio internal server: %w", err)
		default:
		}
		response, err := client.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("Studio internal server did not become ready within %s", timeout)
}

type statusError struct {
	status int
	err    error
}

func (e statusError) Error() string   { return e.err.Error() }
func (e statusError) Unwrap() error   { return e.err }
func (e statusError) StatusCode() int { return e.status }

func authorizeAction(request *http.Request, token string) error {
	if token == "" {
		return statusError{http.StatusServiceUnavailable, fmt.Errorf("STUDIO_ACTION_TOKEN is not configured")}
	}
	if request.Header.Get("Authorization") != "Bearer "+token {
		return statusError{http.StatusUnauthorized, fmt.Errorf("invalid Studio action credentials")}
	}
	return nil
}

func studioCSRF(sessions *session.Manager, token string) server.Middleware {
	protected := sessions.Protect
	return func(next http.Handler) http.Handler {
		csrf := protected(next)
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/studio/") && token != "" && request.Header.Get("Authorization") == "Bearer "+token {
				next.ServeHTTP(writer, request)
				return
			}
			csrf.ServeHTTP(writer, request)
		})
	}
}

func commandError(err error) error {
	if errors.Is(err, studio.ErrRevisionConflict) {
		return statusError{http.StatusConflict, err}
	}
	if errors.Is(err, studio.ErrUnsavedChanges) {
		return statusError{http.StatusConflict, err}
	}
	return statusError{http.StatusUnprocessableEntity, err}
}

func historyAction(ctx *server.Context, workspace *studio.Workspace, token string, undo bool) (any, error) {
	if err := authorizeAction(ctx.Request, token); err != nil {
		return nil, err
	}
	var payload struct {
		ExpectedRevision uint64 `json:"expectedRevision"`
		Actor            string `json:"actor"`
	}
	if err := json.NewDecoder(ctx.Request.Body).Decode(&payload); err != nil {
		return nil, statusError{http.StatusBadRequest, err}
	}
	var receipt studio.Receipt
	var document studio.Document
	var err error
	if undo {
		receipt, document, err = workspace.Undo(payload.ExpectedRevision, payload.Actor)
	} else {
		receipt, document, err = workspace.Redo(payload.ExpectedRevision, payload.Actor)
	}
	if err != nil {
		return nil, commandError(err)
	}
	return map[string]any{"receipt": receipt, "document": document}, nil
}

func switchProjectRequest(ctx *server.Context, workspace *studio.Workspace) (any, error) {
	ctx.NoStore()
	var request struct {
		Path             string `json:"path"`
		ExpectedRevision uint64 `json:"expectedRevision"`
		DiscardUnsaved   bool   `json:"discardUnsaved,omitempty"`
	}
	decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return nil, statusError{http.StatusBadRequest, err}
	}
	status, err := workspace.SwitchProjectFile(request.Path, request.ExpectedRevision, request.DiscardUnsaved)
	if err != nil {
		return nil, commandError(err)
	}
	studioapp.BindWorkspace(workspace)
	return status, nil
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
