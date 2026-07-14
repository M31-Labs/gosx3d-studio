package app

import (
	"log"
	"os"

	"m31labs.dev/gosx/route"
	"m31labs.dev/gosx/server"
)

func init() {
	if err := route.RegisterFileModuleHere(route.FileModuleOptions{
		Load: func(ctx *route.RouteContext, page route.FilePage) (any, error) {
			appName := os.Getenv("APP_NAME")
			if appName == "" {
				appName = "GoSX 3D Studio"
			}
			return map[string]string{
				"appName": appName,
				"source":  page.Source,
			}, nil
		},
		Metadata: func(ctx *route.RouteContext, page route.FilePage, data any) (server.Metadata, error) {
			values, _ := data.(map[string]string)
			appName := values["appName"]
			if appName == "" {
				appName = "GoSX 3D Studio"
			}
			return server.Metadata{
				Title:       server.Title{Default: appName},
				Description: "Initial standalone GoSX 3D Studio application scaffold and agent-readable workbench contract.",
			}, nil
		},
	}); err != nil {
		log.Fatal(err)
	}
}
