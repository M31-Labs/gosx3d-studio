package app

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx/action"
	"m31labs.dev/gosx/route"
	"m31labs.dev/gosx/server"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

var workspaceBinding struct {
	sync.RWMutex
	workspace *studio.Workspace
}

func BindWorkspace(workspace *studio.Workspace) {
	workspaceBinding.Lock()
	workspaceBinding.workspace = workspace
	workspaceBinding.Unlock()
}

func currentDocument() (studio.Document, error) {
	workspaceBinding.RLock()
	workspace := workspaceBinding.workspace
	workspaceBinding.RUnlock()
	if workspace == nil {
		return studio.SampleDocument(), nil
	}
	return workspace.Snapshot()
}

func boundWorkspace() *studio.Workspace {
	workspaceBinding.RLock()
	defer workspaceBinding.RUnlock()
	return workspaceBinding.workspace
}

func executeHumanTransform(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	target := studio.ID(strings.TrimSpace(values["target"]))
	document, err := workspace.Snapshot()
	if err != nil {
		return studio.Receipt{}, err
	}
	entity, ok := document.Entities[target]
	if !ok {
		return studio.Receipt{}, fmt.Errorf("entity %q does not exist", target)
	}
	parse := func(name string) (float64, error) {
		value, parseErr := strconv.ParseFloat(strings.TrimSpace(values[name]), 64)
		if parseErr != nil {
			return 0, fmt.Errorf("%s: %w", name, parseErr)
		}
		return value, nil
	}
	x, err := parse("x")
	if err != nil {
		return studio.Receipt{}, err
	}
	y, err := parse("y")
	if err != nil {
		return studio.Receipt{}, err
	}
	z, err := parse("z")
	if err != nil {
		return studio.Receipt{}, err
	}
	rx, err := parse("rx")
	if err != nil {
		return studio.Receipt{}, err
	}
	ry, err := parse("ry")
	if err != nil {
		return studio.Receipt{}, err
	}
	rz, err := parse("rz")
	if err != nil {
		return studio.Receipt{}, err
	}
	transaction := studio.Transaction{
		ID:               fmt.Sprintf("human-transform-%d", time.Now().UnixNano()),
		Actor:            "human://local-ui",
		Mode:             studio.ModeDirect,
		ExpectedRevision: revision,
		Operations: []studio.Operation{{
			Kind: studio.OpSetTransform, Target: target,
			Transform: &studio.Transform{
				Position: studio.Vec3{X: x, Y: y, Z: z},
				Rotation: studio.Vec3{X: rx, Y: ry, Z: rz},
				Scale:    entity.Transform.Scale,
			},
		}},
	}
	receipt, _, err := workspace.Execute(transaction)
	return receipt, err
}

func executeHumanSolidify(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	target := studio.ID(strings.TrimSpace(values["target"]))
	document, err := workspace.Snapshot()
	if err != nil {
		return studio.Receipt{}, err
	}
	entity, ok := document.Entities[target]
	if !ok || entity.Mesh == nil || entity.Mesh.Geometry.Kind != "indexed-mesh" {
		return studio.Receipt{}, fmt.Errorf("solidify target %q must be an indexed mesh", target)
	}
	thickness, err := strconv.ParseFloat(strings.TrimSpace(values["thickness"]), 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("thickness: %w", err)
	}
	modifierID := studio.ID(strings.TrimSpace(values["modifierId"]))
	if modifierID == "" {
		modifierID = "solidify"
	}
	modifier := studio.Modifier{ID: modifierID, Kind: "solidify", Enabled: true, Thickness: thickness}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-solidify-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect,
		ExpectedRevision: revision,
		Operations:       []studio.Operation{{Kind: studio.OpSetModifier, Target: target, Modifier: &modifier}},
	})
	return receipt, err
}

func executeHumanSubdivision(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	levels, err := strconv.Atoi(strings.TrimSpace(values["levels"]))
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("levels: %w", err)
	}
	modifierID := studio.ID(strings.TrimSpace(values["modifierId"]))
	if modifierID == "" {
		modifierID = "subdivision"
	}
	modifier := studio.Modifier{ID: modifierID, Kind: "subdivision", Enabled: true, Levels: levels}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-subdivision-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect,
		ExpectedRevision: revision,
		Operations:       []studio.Operation{{Kind: studio.OpSetModifier, Target: studio.ID(strings.TrimSpace(values["target"])), Modifier: &modifier}},
	})
	return receipt, err
}

func executeHumanAssetReimport(workspace *studio.Workspace, values map[string]string) (studio.Receipt, studio.AssetRecord, error) {
	if workspace == nil {
		return studio.Receipt{}, studio.AssetRecord{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, studio.AssetRecord{}, fmt.Errorf("expectedRevision: %w", err)
	}
	receipt, _, asset, err := workspace.ReimportAsset(studio.AssetReimportRequest{
		AssetID: studio.ID(strings.TrimSpace(values["assetId"])), Path: strings.TrimSpace(values["path"]),
		Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision,
	})
	return receipt, asset, err
}

func executeHumanAssetImport(workspace *studio.Workspace, values map[string]string) (studio.Receipt, studio.AssetRecord, error) {
	if workspace == nil {
		return studio.Receipt{}, studio.AssetRecord{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, studio.AssetRecord{}, fmt.Errorf("expectedRevision: %w", err)
	}
	receipt, _, asset, err := workspace.ImportAsset(studio.AssetImportRequest{
		Path: strings.TrimSpace(values["path"]), Actor: "human://local-ui",
		Mode: studio.ModeDirect, ExpectedRevision: revision,
	})
	return receipt, asset, err
}

func executeHumanModifierOperation(workspace *studio.Workspace, values map[string]string, kind studio.OperationKind) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	operation := studio.Operation{
		Kind: kind, Target: studio.ID(strings.TrimSpace(values["target"])),
		ModifierID: studio.ID(strings.TrimSpace(values["modifierId"])),
	}
	if kind == studio.OpReorderModifier {
		index, parseErr := strconv.Atoi(strings.TrimSpace(values["modifierIndex"]))
		if parseErr != nil {
			return studio.Receipt{}, fmt.Errorf("modifierIndex: %w", parseErr)
		}
		operation.ModifierIndex = index
	}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-%s-%d", kind, time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect,
		ExpectedRevision: revision, Operations: []studio.Operation{operation},
	})
	return receipt, err
}

func currentSelection(requested string) (studio.Document, studio.ID, error) {
	document, err := currentDocument()
	if err != nil {
		return studio.Document{}, "", err
	}
	workspaceBinding.RLock()
	workspace := workspaceBinding.workspace
	workspaceBinding.RUnlock()
	selected := studio.ID(strings.TrimSpace(requested))
	if selected != "" {
		if _, ok := document.Entities[selected]; !ok {
			selected = ""
		}
	}
	if selected == "" && workspace != nil {
		values := workspace.Selection()
		if len(values) > 0 {
			selected = values[0]
		}
	}
	if selected == "" {
		selected, _ = studio.FirstPickTarget(document)
	}
	if workspace != nil {
		_ = workspace.Select(selected)
	}
	return document, selected, nil
}

func init() {
	if err := route.RegisterFileModuleHere(route.FileModuleOptions{
		Actions: route.FileActions{
			"saveProject": func(ctx *action.Context) error {
				workspace := boundWorkspace()
				if workspace == nil {
					ctx.ValidationFailure("Studio workspace is not bound.", map[string]string{"project": "Workspace unavailable"})
					return nil
				}
				if _, err := workspace.Save(); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"project": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"setTransform": func(ctx *action.Context) error {
				receipt, err := executeHumanTransform(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"transform": err.Error()})
					return nil
				}
				_ = receipt
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
			"setSolidify": func(ctx *action.Context) error {
				receipt, err := executeHumanSolidify(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"solidify": err.Error()})
					return nil
				}
				_ = receipt
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
			"setSubdivision": func(ctx *action.Context) error {
				if _, err := executeHumanSubdivision(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"subdivision": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
			"reimportAsset": func(ctx *action.Context) error {
				if _, _, err := executeHumanAssetReimport(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"asset": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
			"importAsset": func(ctx *action.Context) error {
				if _, _, err := executeHumanAssetImport(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"asset": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"reorderModifier": func(ctx *action.Context) error {
				if _, err := executeHumanModifierOperation(boundWorkspace(), ctx.FormData, studio.OpReorderModifier); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"modifier": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
			"applyModifier": func(ctx *action.Context) error {
				if _, err := executeHumanModifierOperation(boundWorkspace(), ctx.FormData, studio.OpApplyModifier); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"modifier": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
		},
		Load: func(ctx *route.RouteContext, page route.FilePage) (any, error) {
			appName := os.Getenv("APP_NAME")
			if appName == "" {
				appName = "GoSX 3D Studio"
			}
			document, selected, err := currentSelection(ctx.Query("selection"))
			if err != nil {
				return nil, err
			}
			sceneProps, err := studio.CompileSelected(document, selected)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"appName":       appName,
				"projectName":   document.Name,
				"source":        page.Source,
				"scene":         sceneProps,
				"document":      document,
				"hierarchy":     hierarchyView(document, selected),
				"inspector":     inspectorView(document, selected),
				"entityCount":   fmt.Sprint(len(document.Entities)),
				"revision":      fmt.Sprintf("%04d", document.Revision),
				"certification": certificationView(studio.Certification()),
				"project":       projectView(boundWorkspace()),
				"assets":        assetView(document, boundWorkspace()),
				"assetCount":    fmt.Sprint(len(document.Assets)),
			}, nil
		},
		Metadata: func(ctx *route.RouteContext, page route.FilePage, data any) (server.Metadata, error) {
			values, _ := data.(map[string]any)
			appName, _ := values["appName"].(string)
			if appName == "" {
				appName = "GoSX 3D Studio"
			}
			return server.Metadata{
				Title:       server.Title{Default: appName},
				Description: "Standalone local-first GoSX 3D scene, asset, modeling, and deterministic evidence workbench.",
			}, nil
		},
	}); err != nil {
		log.Fatal(err)
	}
}
