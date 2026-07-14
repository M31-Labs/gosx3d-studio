package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/env"
	"m31labs.dev/gosx/route"
	"m31labs.dev/gosx/server"
	"m31labs.dev/gosx/session"
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
	sessions, err := session.New(getenv("SESSION_SECRET", "gosx-app-session-secret"), session.Options{})
	if err != nil {
		log.Fatal(err)
	}
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		log.Fatal(err)
	}

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
	app.Use(sessions.Protect)
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
	app.API("GET /api/studio/manifest", func(ctx *server.Context) (any, error) {
		ctx.CachePublic(30 * time.Second)
		ctx.CacheTag("studio-manifest")
		return studio.DefaultManifest(), nil
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
	rootHandler, err := router.BuildChecked()
	if err != nil {
		log.Fatal(err)
	}
	app.Mount("/", rootHandler)

	log.Printf("%s listening on http://localhost:%s", appName, port)
	log.Fatal(app.ListenAndServe(":" + port))
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
