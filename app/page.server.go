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
			Transform: func() *studio.Transform {
				value := studio.TransformFromEuler(studio.Vec3{X: x, Y: y, Z: z}, studio.Vec3{X: rx, Y: ry, Z: rz}, entity.Transform.Scale)
				return &value
			}(),
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

func executeHumanAssetGC(workspace *studio.Workspace, values map[string]string) (studio.AssetGCResult, error) {
	if workspace == nil {
		return studio.AssetGCResult{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.AssetGCResult{}, fmt.Errorf("expectedRevision: %w", err)
	}
	return workspace.CollectAssetGarbage(studio.AssetGCRequest{Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, ConfirmPlan: strings.TrimSpace(values["confirmPlan"])})
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

func executeHumanBonePose(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	document, err := workspace.Snapshot()
	if err != nil {
		return studio.Receipt{}, err
	}
	armatureID, boneID := studio.ID(strings.TrimSpace(values["armatureId"])), studio.ID(strings.TrimSpace(values["boneId"]))
	armature, ok := document.Armatures[armatureID]
	if !ok {
		return studio.Receipt{}, fmt.Errorf("armature %q does not exist", armatureID)
	}
	bone, ok := armature.Bones[boneID]
	if !ok {
		return studio.Receipt{}, fmt.Errorf("bone %q does not exist", boneID)
	}
	pose := bone.Rest
	if value, exists := armature.Pose[boneID]; exists {
		pose = value
	}
	pose.Euler, err = parseHumanVec3(values, "rx", "ry", "rz")
	if err != nil {
		return studio.Receipt{}, err
	}
	pose.Rotation = studio.QuaternionFromEuler(pose.Euler)
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-bone-pose-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpSetBonePose, ArmatureID: armatureID, BoneID: boneID, Transform: &pose}}})
	return receipt, err
}

func executeHumanAnimationKey(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	document, err := workspace.Snapshot()
	if err != nil {
		return studio.Receipt{}, err
	}
	clipID, trackID := studio.ID(strings.TrimSpace(values["clipId"])), studio.ID(strings.TrimSpace(values["trackId"]))
	clip, ok := document.Animations[clipID]
	if !ok {
		return studio.Receipt{}, fmt.Errorf("animation %q does not exist", clipID)
	}
	track, ok := clip.Tracks[trackID]
	if !ok || len(track.Keys) == 0 {
		return studio.Receipt{}, fmt.Errorf("track %q does not exist", trackID)
	}
	keyTime, err := strconv.ParseFloat(strings.TrimSpace(values["time"]), 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("time: %w", err)
	}
	transform := track.Keys[0].Transform
	transform.Euler, err = parseHumanVec3(values, "rx", "ry", "rz")
	if err != nil {
		return studio.Receipt{}, err
	}
	transform.Rotation = studio.QuaternionFromEuler(transform.Euler)
	key := studio.TransformKey{Time: keyTime, Transform: transform}
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-animation-key-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpSetAnimationKey, ClipID: clipID, TrackID: trackID, Key: &key}}})
	return receipt, err
}

func executeHumanSolveIK(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-solve-ik-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpSolveIK, ArmatureID: studio.ID(strings.TrimSpace(values["armatureId"])), ConstraintID: studio.ID(strings.TrimSpace(values["constraintId"]))}}})
	return receipt, err
}

func executeHumanSimulation(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	ticks, err := strconv.ParseUint(strings.TrimSpace(values["ticks"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("ticks: %w", err)
	}
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-simulate-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpSimulateTicks, SimulationID: studio.ID(strings.TrimSpace(values["simulationId"])), Ticks: ticks}}})
	return receipt, err
}

func executeHumanRetarget(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-retarget-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpRetargetAnimation, RetargetMapID: studio.ID(strings.TrimSpace(values["retargetMapId"])), SourceClipID: studio.ID(strings.TrimSpace(values["sourceClipId"])), NewID: studio.ID(strings.TrimSpace(values["newId"])), Name: strings.TrimSpace(values["name"])}}})
	return receipt, err
}

func executeHumanAnimationParameter(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(values["number"]), 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("number: %w", err)
	}
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-animation-parameter-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpSetAnimationParameter, MachineID: studio.ID(strings.TrimSpace(values["machineId"])), Parameter: strings.TrimSpace(values["parameter"]), Number: value}}})
	return receipt, err
}

func executeHumanAnimationMachineStep(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	deltaTime, err := strconv.ParseFloat(strings.TrimSpace(values["deltaTime"]), 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("deltaTime: %w", err)
	}
	receipt, _, err := workspace.Execute(studio.Transaction{ID: fmt.Sprintf("human-animation-machine-step-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{{Kind: studio.OpStepAnimationMachine, MachineID: studio.ID(strings.TrimSpace(values["machineId"])), DeltaTime: deltaTime}}})
	return receipt, err
}

func parseHumanVec3(values map[string]string, xName, yName, zName string) (studio.Vec3, error) {
	parse := func(name string) (float64, error) {
		value, err := strconv.ParseFloat(strings.TrimSpace(values[name]), 64)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", name, err)
		}
		return value, nil
	}
	x, err := parse(xName)
	if err != nil {
		return studio.Vec3{}, err
	}
	y, err := parse(yName)
	if err != nil {
		return studio.Vec3{}, err
	}
	z, err := parse(zName)
	if err != nil {
		return studio.Vec3{}, err
	}
	return studio.Vec3{X: x, Y: y, Z: z}, nil
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
	// Selecting takes the workspace write lock, so only do it when the
	// selection actually changes. Rendering a page is a read; taking the
	// write lock on every render serialized reads behind it for no reason.
	// The error is deliberately ignored: a selection that no longer resolves
	// is a stale link, not a render failure, and the page shows the canonical
	// selection either way.
	if workspace != nil && selected != "" {
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
			"setPhysics": func(ctx *action.Context) error {
				receipt, err := executeHumanPhysics(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"physics": err.Error()})
					return nil
				}
				ctx.Redirect(fmt.Sprintf("/?selection=%s&applied=%s", ctx.FormData["target"], receipt.TransactionID))
				return nil
			},
			"playOp": func(ctx *action.Context) error {
				if err := executeHumanPlayOp(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"play": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"prefabOp": func(ctx *action.Context) error {
				receipt, err := executeHumanPrefabOp(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"prefab": err.Error()})
					return nil
				}
				ctx.Redirect(fmt.Sprintf("/?selection=%s&applied=%s", ctx.FormData["target"], receipt.TransactionID))
				return nil
			},
			"entityOp": func(ctx *action.Context) error {
				receipt, err := executeHumanEntityOp(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"hierarchy": err.Error()})
					return nil
				}
				selection := ctx.FormData["target"]
				if ctx.FormData["op"] == "delete" {
					selection = ""
				}
				ctx.Redirect(fmt.Sprintf("/?selection=%s&applied=%s", selection, receipt.TransactionID))
				return nil
			},
			"selectSubobjects": func(ctx *action.Context) error {
				if err := executeHumanSubobjectSelection(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"modeling": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["target"])
				return nil
			},
			"meshOperator": func(ctx *action.Context) error {
				receipt, err := executeHumanMeshOperator(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"modeling": err.Error()})
					return nil
				}
				ctx.Redirect(fmt.Sprintf("/?selection=%s&applied=%s", ctx.FormData["target"], receipt.TransactionID))
				return nil
			},
			"assignMaterial": func(ctx *action.Context) error {
				receipt, err := executeHumanAssignMaterial(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"material": err.Error()})
					return nil
				}
				ctx.Redirect(fmt.Sprintf("/?selection=%s&applied=%s", ctx.FormData["target"], receipt.TransactionID))
				return nil
			},
			"setMaterial": func(ctx *action.Context) error {
				receipt, err := executeHumanSetMaterial(boundWorkspace(), ctx.FormData)
				if err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"material": err.Error()})
					return nil
				}
				ctx.Redirect(fmt.Sprintf("/?selection=%s&applied=%s", ctx.FormData["selection"], receipt.TransactionID))
				return nil
			},
			"undoCommand": func(ctx *action.Context) error {
				return executeHumanHistory(ctx, true)
			},
			"redoCommand": func(ctx *action.Context) error {
				return executeHumanHistory(ctx, false)
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
			"collectAssets": func(ctx *action.Context) error {
				if _, err := executeHumanAssetGC(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"assetGC": err.Error()})
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
			"setBonePose": func(ctx *action.Context) error {
				if _, err := executeHumanBonePose(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"rig": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"setAnimationKey": func(ctx *action.Context) error {
				if _, err := executeHumanAnimationKey(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"animation": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"solveIK": func(ctx *action.Context) error {
				if _, err := executeHumanSolveIK(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"rig": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"simulateTicks": func(ctx *action.Context) error {
				if _, err := executeHumanSimulation(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"simulation": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"retargetAnimation": func(ctx *action.Context) error {
				if _, err := executeHumanRetarget(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"retarget": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"setAnimationParameter": func(ctx *action.Context) error {
				if _, err := executeHumanAnimationParameter(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"stateMachine": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
				return nil
			},
			"stepAnimationMachine": func(ctx *action.Context) error {
				if _, err := executeHumanAnimationMachineStep(boundWorkspace(), ctx.FormData); err != nil {
					ctx.ValidationFailure(err.Error(), map[string]string{"stateMachine": err.Error()})
					return nil
				}
				ctx.Redirect("/?selection=" + ctx.FormData["selection"])
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
				"certification": liveCertificationView(document),
				"materials":     materialsView(document),
				"modeling":      modelingView(document, boundWorkspace(), selected),
				"prefabs":       prefabsView(document),
				"imageAssets":   imageAssetsView(document),
				"play":          playView(boundWorkspace()),
				"agent":         agentView(boundWorkspace(), strings.TrimSpace(os.Getenv("STUDIO_ACTION_TOKEN")) != ""),
				"cameraHome":    fmt.Sprintf("%g,%g,%g", document.Camera.Position.X, document.Camera.Position.Y, document.Camera.Position.Z),
				"history":        historyView(boundWorkspace()),
				"historySummary": fmt.Sprintf("%d entities · revision %04d · command history below", len(document.Entities), document.Revision),
				"project":       projectView(boundWorkspace()),
				"assets":        assetView(document, boundWorkspace()),
				"assetCount":    fmt.Sprint(len(document.Assets)),
				"assetGC":       assetGCView(document, boundWorkspace()),
				"timeline":      timelineView(document),
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

// executeHumanHistory routes the menu Undo/Redo forms through the same
// workspace history path agents use, with the same revision safety.
func executeHumanHistory(ctx *action.Context, undo bool) error {
	workspace := boundWorkspace()
	if workspace == nil {
		ctx.ValidationFailure("Studio workspace is not bound.", map[string]string{"history": "Workspace unavailable"})
		return nil
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(ctx.FormData["expectedRevision"]), 10, 64)
	if err != nil {
		ctx.ValidationFailure("expectedRevision: "+err.Error(), map[string]string{"history": err.Error()})
		return nil
	}
	if undo {
		_, _, err = workspace.Undo(revision, "human://local-ui")
	} else {
		_, _, err = workspace.Redo(revision, "human://local-ui")
	}
	if err != nil {
		ctx.ValidationFailure(err.Error(), map[string]string{"history": err.Error()})
		return nil
	}
	ctx.Redirect("/?selection=" + ctx.FormData["selection"])
	return nil
}

// executeHumanAssignMaterial routes the Inspector assign-material form through
// the shared transaction path.
func executeHumanAssignMaterial(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-assign-material-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect,
		ExpectedRevision: revision,
		Operations:       []studio.Operation{{Kind: studio.OpAssignMaterial, Target: studio.ID(strings.TrimSpace(values["target"])), Material: studio.ID(strings.TrimSpace(values["materialId"]))}},
	})
	return receipt, err
}

// executeHumanSetMaterial edits the selected entity's material record —
// PBR ranges validate before commit and invalid Selena keeps the last valid
// material, identically to the agent path.
func executeHumanSetMaterial(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	document, err := workspace.Snapshot()
	if err != nil {
		return studio.Receipt{}, err
	}
	materialID := studio.ID(strings.TrimSpace(values["materialId"]))
	material, ok := document.Materials[materialID]
	if !ok {
		return studio.Receipt{}, fmt.Errorf("material %q does not exist", materialID)
	}
	parse := func(name string, into *float64) error {
		value, parseErr := strconv.ParseFloat(strings.TrimSpace(values[name]), 64)
		if parseErr != nil {
			return fmt.Errorf("%s: %w", name, parseErr)
		}
		*into = value
		return nil
	}
	if color := strings.TrimSpace(values["color"]); color != "" {
		material.Color = color
	}
	for name, into := range map[string]*float64{"roughness": &material.Roughness, "metalness": &material.Metalness, "clearcoat": &material.Clearcoat, "transmission": &material.Transmission, "emissive": &material.Emissive} {
		if err := parse(name, into); err != nil {
			return studio.Receipt{}, err
		}
	}
	channels := []string{"color", "normal", "roughness", "metalness", "emissive"}
	textures := map[string]studio.TextureSlot{}
	for _, channel := range channels {
		if asset := strings.TrimSpace(values["texture-"+channel]); asset != "" {
			textures[channel] = studio.TextureSlot{Asset: studio.ID(asset)}
		}
	}
	if len(textures) > 0 {
		material.Textures = textures
	} else {
		material.Textures = nil
	}
	source := strings.TrimSpace(values["selenaSource"])
	switch {
	case source == "":
		material.Selena = nil
	case material.Selena != nil:
		material.Selena = &studio.SelenaShader{Material: material.Selena.Material, Source: source}
	default:
		material.Selena = &studio.SelenaShader{Material: string(material.ID), Source: source}
	}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-set-material-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect,
		ExpectedRevision: revision,
		Operations:       []studio.Operation{{Kind: studio.OpSetMaterial, MaterialRecord: &material}},
	})
	return receipt, err
}

// executeHumanSubobjectSelection routes the Inspector sub-object selection
// form through the same revision-safe selection contract agents use.
func executeHumanSubobjectSelection(workspace *studio.Workspace, values map[string]string) error {
	if workspace == nil {
		return fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return fmt.Errorf("expectedRevision: %w", err)
	}
	return workspace.SelectSubobjects(studio.SelectionRequest{
		ExpectedRevision: revision,
		Mode:             studio.SelectionMode(strings.TrimSpace(values["mode"])),
		Object:           studio.ID(strings.TrimSpace(values["target"])),
		IDs:              parseIDList(values["ids"]),
	})
}

func parseIDList(value string) []studio.ID {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
	out := make([]studio.ID, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, studio.ID(trimmed))
		}
	}
	return out
}

// executeHumanMeshOperator maps the Inspector modeling form onto the shared
// deterministic operator transactions. Blank ID lists fall back to the current
// sub-object selection so viewport/Hierarchy selection and forms converge.
func executeHumanMeshOperator(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	target := studio.ID(strings.TrimSpace(values["target"]))
	ids := parseIDList(values["ids"])
	if len(ids) == 0 {
		if state := workspace.SelectionState(); state.Object == target {
			ids = append(ids, state.IDs...)
		}
	}
	parseFloat := func(name string) (float64, error) {
		raw := strings.TrimSpace(values[name])
		if raw == "" {
			return 0, nil
		}
		value, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("%s: %w", name, parseErr)
		}
		return value, nil
	}
	distance, err := parseFloat("distance")
	if err != nil {
		return studio.Receipt{}, err
	}
	amount, err := parseFloat("amount")
	if err != nil {
		return studio.Receipt{}, err
	}
	tolerance, err := parseFloat("tolerance")
	if err != nil {
		return studio.Receipt{}, err
	}
	operation := studio.Operation{Target: target, Distance: distance, Amount: amount, Tolerance: tolerance, Projection: strings.TrimSpace(values["projection"])}
	switch strings.TrimSpace(values["operator"]) {
	case "extrude-faces":
		operation.Kind, operation.Faces = studio.OpExtrudeFaces, ids
	case "inset-faces":
		operation.Kind, operation.Faces = studio.OpInsetFaces, ids
	case "triangulate-faces":
		operation.Kind, operation.Faces = studio.OpTriangulateFaces, ids
	case "weld-vertices":
		operation.Kind, operation.Vertices = studio.OpWeldVertices, ids
	case "dissolve-edges":
		operation.Kind, operation.Edges = studio.OpDissolveEdges, ids
	case "bevel-edges":
		operation.Kind, operation.Edges, operation.NewID = studio.OpBevelEdges, ids, studio.ID(fmt.Sprintf("bevel-face-%d", time.Now().UnixNano()))
	case "recalculate-normals":
		operation.Kind = studio.OpRecalculateNormals
	case "project-planar-uv":
		operation.Kind = studio.OpProjectPlanarUV
	default:
		return studio.Receipt{}, fmt.Errorf("unsupported modeling operator %q", values["operator"])
	}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-mesh-%s-%d", operation.Kind, time.Now().UnixNano()), Actor: "human://local-ui",
		Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{operation},
	})
	return receipt, err
}

// executeHumanEntityOp routes the Hierarchy panel's entity operations
// (rename, duplicate, delete, reparent) through the shared revision-safe
// transaction path.
func executeHumanEntityOp(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	target := studio.ID(strings.TrimSpace(values["target"]))
	if target == "" {
		return studio.Receipt{}, fmt.Errorf("select an entity first")
	}
	operation := studio.Operation{Target: target}
	switch strings.TrimSpace(values["op"]) {
	case "rename":
		operation.Kind = studio.OpRenameEntity
		operation.Name = strings.TrimSpace(values["name"])
	case "duplicate":
		operation.Kind = studio.OpDuplicateEntity
		operation.NewID = studio.ID(fmt.Sprintf("%s-copy-%d", target, time.Now().UnixNano()))
	case "delete":
		operation.Kind = studio.OpDeleteEntity
	case "reparent":
		operation.Kind = studio.OpReparentEntity
		operation.Parent = studio.ID(strings.TrimSpace(values["parent"]))
	default:
		return studio.Receipt{}, fmt.Errorf("unsupported entity operation %q", values["op"])
	}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-entity-%s-%d", values["op"], time.Now().UnixNano()), Actor: "human://local-ui",
		Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{operation},
	})
	return receipt, err
}

// executeHumanPrefabOp routes the Prefabs panel operations through the shared
// transaction path: capture selection, instantiate, set a simple override, or
// delete an unreferenced definition.
func executeHumanPrefabOp(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	operation := studio.Operation{}
	switch strings.TrimSpace(values["op"]) {
	case "capture":
		operation.Kind = studio.OpCapturePrefab
		operation.Target = studio.ID(strings.TrimSpace(values["target"]))
		operation.PrefabID = studio.ID(strings.TrimSpace(values["prefabId"]))
		operation.Name = strings.TrimSpace(values["name"])
	case "instantiate":
		operation.Kind = studio.OpInstantiatePrefab
		operation.PrefabID = studio.ID(strings.TrimSpace(values["prefabId"]))
		operation.NewID = studio.ID(fmt.Sprintf("%s-instance-%d", operation.PrefabID, time.Now().UnixNano()))
		operation.Parent = studio.ID(strings.TrimSpace(values["parent"]))
	case "override":
		operation.Kind = studio.OpSetPrefabOverride
		operation.Target = studio.ID(strings.TrimSpace(values["target"]))
		operation.PrefabEntity = studio.ID(strings.TrimSpace(values["prefabEntity"]))
		override := studio.PrefabEntityOverride{}
		if material := strings.TrimSpace(values["material"]); material != "" {
			override.Material = studio.ID(material)
		}
		if visible := strings.TrimSpace(values["visible"]); visible != "" {
			value := visible == "true" || visible == "on"
			override.Visible = &value
		}
		operation.PrefabOverride = &override
	case "delete":
		operation.Kind = studio.OpDeletePrefab
		operation.PrefabID = studio.ID(strings.TrimSpace(values["prefabId"]))
	default:
		return studio.Receipt{}, fmt.Errorf("unsupported prefab operation %q", values["op"])
	}
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-prefab-%s-%d", values["op"], time.Now().UnixNano()), Actor: "human://local-ui",
		Mode: studio.ModeDirect, ExpectedRevision: revision, Operations: []studio.Operation{operation},
	})
	return receipt, err
}

// executeHumanPlayOp drives play mode: enter clones the document into a
// runtime world, step advances fixed ticks, exit discards runtime state.
func executeHumanPlayOp(workspace *studio.Workspace, values map[string]string) error {
	if workspace == nil {
		return fmt.Errorf("Studio workspace is not bound")
	}
	switch strings.TrimSpace(values["op"]) {
	case "enter":
		return workspace.EnterPlay(studio.ID(strings.TrimSpace(values["simulationId"])))
	case "step":
		ticks, err := strconv.ParseUint(strings.TrimSpace(values["ticks"]), 10, 64)
		if err != nil || ticks == 0 {
			ticks = 1
		}
		return workspace.StepPlay(ticks, nil)
	case "exit":
		return workspace.ExitPlay()
	}
	return fmt.Errorf("unsupported play operation %q", values["op"])
}

// executeHumanPhysics routes the Inspector physics form through the shared
// set-physics-body transaction.
func executeHumanPhysics(workspace *studio.Workspace, values map[string]string) (studio.Receipt, error) {
	if workspace == nil {
		return studio.Receipt{}, fmt.Errorf("Studio workspace is not bound")
	}
	revision, err := strconv.ParseUint(strings.TrimSpace(values["expectedRevision"]), 10, 64)
	if err != nil {
		return studio.Receipt{}, fmt.Errorf("expectedRevision: %w", err)
	}
	parse := func(name string) (float64, error) {
		raw := strings.TrimSpace(values[name])
		if raw == "" {
			return 0, nil
		}
		value, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("%s: %w", name, parseErr)
		}
		return value, nil
	}
	body := studio.PhysicsBody{Kind: strings.TrimSpace(values["kind"])}
	if body.Mass, err = parse("mass"); err != nil {
		return studio.Receipt{}, err
	}
	if body.Friction, err = parse("friction"); err != nil {
		return studio.Receipt{}, err
	}
	if body.Restitution, err = parse("restitution"); err != nil {
		return studio.Receipt{}, err
	}
	if body.GravityScale, err = parse("gravityScale"); err != nil {
		return studio.Receipt{}, err
	}
	collider := studio.Collider{Kind: strings.TrimSpace(values["colliderKind"]), Sensor: strings.TrimSpace(values["sensor"]) != ""}
	if collider.Radius, err = parse("radius"); err != nil {
		return studio.Receipt{}, err
	}
	if collider.HalfHeight, err = parse("halfHeight"); err != nil {
		return studio.Receipt{}, err
	}
	switch collider.Kind {
	case "box":
		extents := studio.Vec3{}
		if extents.X, err = parse("extentX"); err != nil {
			return studio.Receipt{}, err
		}
		if extents.Y, err = parse("extentY"); err != nil {
			return studio.Receipt{}, err
		}
		if extents.Z, err = parse("extentZ"); err != nil {
			return studio.Receipt{}, err
		}
		collider.HalfExtents = &extents
	case "plane":
		collider.Normal = studio.Vec3{Y: 1}
	}
	body.Collider = collider
	receipt, _, err := workspace.Execute(studio.Transaction{
		ID: fmt.Sprintf("human-physics-%d", time.Now().UnixNano()), Actor: "human://local-ui", Mode: studio.ModeDirect,
		ExpectedRevision: revision,
		Operations:       []studio.Operation{{Kind: studio.OpSetPhysicsBody, Target: studio.ID(strings.TrimSpace(values["target"])), Physics: &body}},
	})
	return receipt, err
}
