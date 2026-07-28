package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
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
	projectDir := getenv("STUDIO_PROJECT_DIR", filepath.Join(root, ".studio"))
	workspace, err := studio.OpenWorkspace(projectDir, studio.SampleDocument())
	if err != nil {
		log.Fatal(err)
	}
	app, err := buildStudioApp(studioConfig{
		root:        root,
		appName:     appName,
		workspace:   workspace,
		actionToken: os.Getenv("STUDIO_ACTION_TOKEN"),
		// desktopHost decides whether native-host-only routes answer at all.
		desktopHost:   desktopMode(runtime.GOOS, os.Getenv("STUDIO_DESKTOP"), os.Getenv("STUDIO_SERVER_ONLY")),
		sessionSecret: os.Getenv("SESSION_SECRET"),
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := runApplication(app, appName, port); err != nil {
		log.Fatal(err)
	}
}

// studioConfig is everything buildStudioApp needs from the environment. Taking
// it as a value rather than reading os.Getenv inline is what lets a test build
// the real router and exercise the real routes, instead of asserting on
// helpers the routes might not use.
type studioConfig struct {
	root          string
	appName       string
	workspace     *studio.Workspace
	actionToken   string
	desktopHost   bool
	sessionSecret string
}

// buildStudioApp wires the Studio HTTP surface. Route authority is declared in
// studioRouteAuthority and checked against this file by the tests in
// route_authority_test.go.
func buildStudioApp(config studioConfig) (*server.App, error) {
	root := config.root
	appName := config.appName
	workspace := config.workspace
	actionToken := config.actionToken
	desktopHost := config.desktopHost

	sessionSecret, err := resolveSessionSecret(config.sessionSecret)
	if err != nil {
		return nil, err
	}
	sessions, err := session.New(sessionSecret, session.Options{})
	if err != nil {
		return nil, err
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
		return nil, err
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
	app.API("GET /api/studio/certification/state", func(ctx *server.Context) (any, error) {
		// Deliberately cheap: the card polls this while a background run is
		// in flight, so it must not compute evidence or clone the document.
		ctx.NoStore()
		var revision uint64
		if err := workspace.Read(func(document *studio.Document) error {
			revision = document.Revision
			return nil
		}); err != nil {
			return nil, err
		}
		return studioapp.CertificationEvidenceStateFor(revision), nil
	})
	app.API("GET /api/studio/certification/evidence", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		// Cached by document identity. This route carries read authority, so
		// it answers without a credential, and the suite costs about a second
		// of CPU per run. Repeat requests at the same revision must not each
		// buy that second.
		return studio.CertifyCurrentCached(document)
	})
	app.API("GET /api/studio/export/scene3d", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		payload, report, err := studio.ExportSceneDoc(document)
		if err != nil {
			return nil, err
		}
		return map[string]any{"report": report, "document": json.RawMessage(payload)}, nil
	})
	app.API("GET /api/studio/export/glb", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		payload, report, err := studio.ExportGLB(document)
		if err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(payload)
		return map[string]any{"report": report, "glbBase64": encoded}, nil
	})
	app.API("GET /api/studio/export/go", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		packageName := ctx.Request.URL.Query().Get("package")
		if packageName == "" {
			packageName = "scene"
		}
		payload, report, err := studio.ExportEmbeddedGo(document, packageName)
		if err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		return map[string]any{"report": report, "source": string(payload)}, nil
	})
	app.API("GET /api/studio/export/scene-ir", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		document, err := workspace.Snapshot()
		if err != nil {
			return nil, err
		}
		payload, report, err := studio.ExportSceneIR(document)
		if err != nil {
			return nil, err
		}
		return map[string]any{"report": report, "artifact": json.RawMessage(payload)}, nil
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
		// This route exists so the trusted native picker can hand back a path
		// the browser could not have chosen on its own. The session CSRF
		// middleware is what authenticates it; the intent header only names
		// the caller. Outside the desktop host there is no native picker, so
		// the route must not answer at all — a server-only deployment that
		// accepted a "native dialog" claim would be accepting a lie.
		if !desktopHost {
			return nil, statusError{http.StatusForbidden, fmt.Errorf("native project dialog is unavailable: this process has no desktop host")}
		}
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
		// Confirming a click reads the document under the workspace read lock
		// and reuses the recorded fingerprint, so it neither copies the scene
		// nor re-derives its identity. The selection write happens after.
		confirmation, revision, err := workspace.ConfirmViewportSelection(request)
		if err != nil {
			return nil, statusError{http.StatusConflict, err}
		}
		if err := workspace.SelectAtRevision(revision, confirmation.Selected); err != nil {
			return nil, commandError(err)
		}
		workspace.RecordViewportConfirmation(confirmation)
		return confirmation, nil
	})
	app.API("POST /api/studio/gizmo-commit", func(ctx *server.Context) (any, error) {
		ctx.NoStore()
		var request studio.GizmoCommit
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		receipt, err := studio.ApplyGizmoCommit(workspace, request)
		if err != nil {
			return nil, commandError(err)
		}
		return receipt, nil
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
		// The pick reads the scene under the workspace read lock and reuses
		// the recorded fingerprint; the optional selection write runs after
		// that lock is released.
		result, revision, err := workspace.ExactPick(request)
		if err != nil {
			return nil, err
		}
		if request.Select && result.Selected != "" {
			if err := workspace.SelectAtRevision(revision, result.Selected); err != nil {
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
	app.API("POST /api/studio/assets/decode-geometry", func(ctx *server.Context) (any, error) {
		if err := authorizeAction(ctx.Request, actionToken); err != nil {
			return nil, err
		}
		ctx.NoStore()
		var request studio.GeometryDecodeRequest
		decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			return nil, statusError{http.StatusBadRequest, err}
		}
		receipt, document, report, err := workspace.DecodeModelGeometry(request)
		if err != nil {
			return nil, commandError(err)
		}
		return map[string]any{"receipt": receipt, "document": document, "report": report}, nil
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
		return nil, err
	}
	app.Mount("/api/studio/assets/content/", assetContentHandler(workspace))
	app.Mount("/", rootHandler)
	return app, nil
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

// studioAuthority names who may call a Studio route. Authority used to be
// implicit in whether a handler happened to call authorizeAction, so a new
// mutating route could pick the wrong one and nothing would notice. This
// table is the contract; TestEveryStudioRouteDeclaresAuthority reads the
// registered routes out of this file and fails on any route missing here.
type studioAuthority string

const (
	// authorityRead is an unauthenticated read. It must not mutate.
	authorityRead studioAuthority = "read"
	// authoritySession is a browser path. The session CSRF middleware covers
	// it; the GoSX client transport supplies X-CSRF-Token on mutating methods.
	authoritySession studioAuthority = "session"
	// authorityToken requires the STUDIO_ACTION_TOKEN bearer credential and
	// bypasses CSRF, because agents hold no session.
	authorityToken studioAuthority = "token"
)

// studioRouteAuthority declares the authority of every registered route.
var studioRouteAuthority = map[string]studioAuthority{
	"GET /api/health":                            authorityRead,
	"GET /api/studio/platform":                   authorityRead,
	"GET /api/studio/manifest":                   authorityRead,
	"GET /api/studio/gltf/capabilities":          authorityRead,
	"GET /api/studio/gltf/corpus":                authorityRead,
	"GET /api/studio/document":                   authorityRead,
	"GET /api/studio/scene-ir":                   authorityRead,
	"GET /api/studio/artifact":                   authorityRead,
	"GET /api/studio/actions":                    authorityRead,
	"GET /api/studio/descriptors":                authorityRead,
	"GET /api/studio/certification":              authorityRead,
	"GET /api/studio/certification/evidence":     authorityRead,
	"GET /api/studio/certification/state":        authorityRead,
	"GET /api/studio/export/scene3d":             authorityRead,
	"GET /api/studio/export/glb":                 authorityRead,
	"GET /api/studio/export/go":                  authorityRead,
	"GET /api/studio/export/scene-ir":            authorityRead,
	"GET /api/studio/initialize":                 authorityRead,
	"GET /api/studio/selection":                  authorityRead,
	"GET /api/studio/geometry/analysis":          authorityRead,
	"GET /api/studio/curve/analysis":             authorityRead,
	"GET /api/studio/rig/skin":                   authorityRead,
	"GET /api/studio/project/status":             authorityRead,
	"GET /api/studio/assets/audit":               authorityRead,
	"GET /api/studio/assets/dependencies":        authorityRead,
	"POST /api/studio/selection":                 authorityToken,
	"POST /api/studio/pick":                      authorityToken,
	"POST /api/studio/actions/preview":           authorityToken,
	"POST /api/studio/transactions/call":         authorityToken,
	"POST /api/studio/undo":                      authorityToken,
	"POST /api/studio/redo":                      authorityToken,
	"POST /api/studio/project/save":              authorityToken,
	"POST /api/studio/project/open":              authorityToken,
	"POST /api/studio/assets/import":             authorityToken,
	"POST /api/studio/assets/decode-geometry":    authorityToken,
	"POST /api/studio/assets/reimport":           authorityToken,
	"POST /api/studio/assets/garbage-collect":    authorityToken,
	"POST /api/studio/viewport-selection":        authoritySession,
	"POST /api/studio/gizmo-commit":              authoritySession,
	"POST /api/studio/project/open-from-desktop": authoritySession,
}

type statusError struct {
	status int
	err    error
}

func (e statusError) Error() string   { return e.err.Error() }
func (e statusError) Unwrap() error   { return e.err }
func (e statusError) StatusCode() int { return e.status }

// sharedSecretPlaceholders are the values .env.example ships. They exist so a
// fresh checkout runs; they must never authenticate anything, because they are
// published in this repository.
var sharedSecretPlaceholders = map[string]bool{
	"gosx-app-session-secret":                 true,
	"replace-with-a-local-development-secret": true,
}

// resolveSessionSecret refuses to sign sessions with a value anyone can read
// out of the repository. Session cookies and CSRF tokens both derive from this
// secret, so a published default would let an attacker mint the CSRF token
// that guards every browser-authority route. Local development without a
// configured secret gets a per-process random one: sessions do not survive a
// restart, which is correct, because a development session is not durable
// state.
func resolveSessionSecret(configured string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured != "" && !sharedSecretPlaceholders[configured] {
		return configured, nil
	}
	reason := "SESSION_SECRET is not set"
	if configured != "" {
		reason = "SESSION_SECRET is still the published placeholder"
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GOSX_ENV")), "production") {
		return "", fmt.Errorf("%s; production requires a private session secret", reason)
	}
	ephemeral := make([]byte, 32)
	if _, err := rand.Read(ephemeral); err != nil {
		return "", fmt.Errorf("generate a development session secret: %w", err)
	}
	log.Printf("%s; using a random per-process secret. Sessions will not survive a restart.", reason)
	return hex.EncodeToString(ephemeral), nil
}

// bearerMatches compares the Authorization header against the configured
// action token in constant time. A byte-by-byte string comparison leaks the
// length of the matching prefix through timing.
func bearerMatches(header, token string) bool {
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte("Bearer "+token)) == 1
}

func authorizeAction(request *http.Request, token string) error {
	if token == "" {
		return statusError{http.StatusServiceUnavailable, fmt.Errorf("STUDIO_ACTION_TOKEN is not configured")}
	}
	if !bearerMatches(request.Header.Get("Authorization"), token) {
		return statusError{http.StatusUnauthorized, fmt.Errorf("invalid Studio action credentials")}
	}
	return nil
}

func studioCSRF(sessions *session.Manager, token string) server.Middleware {
	protected := sessions.Protect
	return func(next http.Handler) http.Handler {
		csrf := protected(next)
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/studio/") && bearerMatches(request.Header.Get("Authorization"), token) {
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
	// Every other Studio handler bounds its body. This one did not, so an
	// undo request could stream unbounded bytes into the decoder.
	decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 64<<10))
	if err := decoder.Decode(&payload); err != nil {
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
