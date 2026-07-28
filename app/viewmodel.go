package app

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func certificationView(report studio.CertificationReport) map[string]any {
	keys := make([]string, 0, len(report.Dimensions))
	for key := range report.Dimensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	dimensions := make([]map[string]any, 0, len(keys))
	available := 0
	for _, key := range keys {
		dimension := report.Dimensions[key]
		if dimension.Status == "available" {
			available++
		}
		dimensions = append(dimensions, map[string]any{"id": dimension.ID, "status": dimension.Status, "evidence": dimension.Evidence})
	}
	return map[string]any{
		"status": report.Status, "dimensions": dimensions,
		"available": fmt.Sprint(available), "total": fmt.Sprint(len(dimensions)),
	}
}

func projectView(workspace *studio.Workspace) map[string]any {
	if workspace == nil {
		return map[string]any{"state": "unavailable", "dirty": "false", "recovered": "false"}
	}
	status := workspace.ProjectStatus()
	state := "saved"
	if status.Recovered {
		state = "recovered"
	} else if status.Dirty {
		state = "modified"
	}
	return map[string]any{
		"state": state, "dirty": fmt.Sprint(status.Dirty), "recovered": fmt.Sprint(status.Recovered),
		"directory": status.Directory, "savedRevision": fmt.Sprintf("%04d", status.SavedRevision),
	}
}

func assetView(document studio.Document, workspace *studio.Workspace) []map[string]any {
	dependencies := map[studio.ID]studio.AssetDependencyEntry{}
	if workspace != nil {
		for _, entry := range workspace.AssetDependencies().Assets {
			dependencies[entry.Asset] = entry
		}
	}
	ids := make([]string, 0, len(document.Assets))
	for id := range document.Assets {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	out := make([]map[string]any, 0, len(ids))
	for _, value := range ids {
		asset := document.Assets[studio.ID(value)]
		dependency := dependencies[asset.ID]
		out = append(out, map[string]any{
			"id": string(asset.ID), "shortId": shortID(asset.ID), "name": asset.SourceName,
			"kind": asset.Kind, "format": strings.ToUpper(asset.Format), "bytes": formatBytes(asset.Bytes),
			"dependencies": fmt.Sprintf("%d direct · %d instances", dependency.DirectCount, len(dependency.Instances)),
		})
	}
	return out
}

func assetGCView(document studio.Document) map[string]any {
	plan, err := studio.PlanAssetGarbage(document)
	if err != nil {
		return map[string]any{"available": false, "count": "0", "bytes": "0 B", "fingerprint": "", "status": err.Error()}
	}
	return map[string]any{"available": len(plan.Assets) > 0, "count": fmt.Sprint(len(plan.Assets)), "bytes": formatBytes(plan.Bytes), "fingerprint": plan.Fingerprint, "status": "Explicit checkpoint: clears undo history and deletes payload bytes."}
}

func timelineView(document studio.Document) map[string]any {
	view := map[string]any{"state": "No articulated clip loaded", "armatureId": "", "boneId": "", "boneName": "—", "constraintId": "", "clipId": "", "clipName": "No clip", "trackId": "", "duration": "0", "keyTime": "0.500", "rx": "0", "ry": "0", "rz": "0", "simulationId": "", "simulationName": "No simulation", "tickRate": "0", "ticks": "60", "retargetMapId": "", "retargetName": "No retarget map", "retargetClipId": "retargeted-clip", "machineId": "", "machineName": "No state machine", "machineState": "—", "machineParameter": "", "machineValue": "0", "deltaTime": "0.016667"}
	armatureIDs := sortedMapIDs(document.Armatures)
	if len(armatureIDs) > 0 {
		armature := document.Armatures[armatureIDs[0]]
		view["armatureId"] = string(armature.ID)
		boneIDs := sortedMapIDs(armature.Bones)
		if len(boneIDs) > 0 {
			bone := armature.Bones[boneIDs[0]]
			pose := bone.Rest
			if value, ok := armature.Pose[bone.ID]; ok {
				pose = value
			}
			view["boneId"], view["boneName"] = string(bone.ID), bone.Name
			view["rx"], view["ry"], view["rz"] = number(pose.Euler.X), number(pose.Euler.Y), number(pose.Euler.Z)
		}
		if len(armature.Constraints) > 0 {
			view["constraintId"] = string(armature.Constraints[0].ID)
		}
	}
	clipIDs := sortedMapIDs(document.Animations)
	if len(clipIDs) > 0 {
		clip := document.Animations[clipIDs[0]]
		view["clipId"], view["clipName"], view["duration"] = string(clip.ID), clip.Name, number(clip.Duration)
		trackIDs := sortedMapIDs(clip.Tracks)
		if len(trackIDs) > 0 {
			view["trackId"] = string(trackIDs[0])
		}
		view["state"] = fmt.Sprintf("%d tracks · %.3fs", len(clip.Tracks), clip.Duration)
	}
	simulationIDs := sortedMapIDs(document.Simulations)
	if len(simulationIDs) > 0 {
		profile := document.Simulations[simulationIDs[0]]
		view["simulationId"], view["simulationName"], view["tickRate"] = string(profile.ID), profile.Name, fmt.Sprint(profile.TickRate)
	}
	retargetIDs := sortedMapIDs(document.RetargetMaps)
	if len(retargetIDs) > 0 {
		mapping := document.RetargetMaps[retargetIDs[0]]
		view["retargetMapId"], view["retargetName"] = string(mapping.ID), mapping.Name
	}
	machineIDs := sortedMapIDs(document.AnimationMachines)
	if len(machineIDs) > 0 {
		machine := document.AnimationMachines[machineIDs[0]]
		view["machineId"], view["machineName"], view["machineState"] = string(machine.ID), machine.Name, string(machine.Current)
		parameterNames := make([]string, 0, len(machine.Parameters))
		for name := range machine.Parameters {
			parameterNames = append(parameterNames, name)
		}
		sort.Strings(parameterNames)
		if len(parameterNames) > 0 {
			view["machineParameter"], view["machineValue"] = parameterNames[0], number(machine.Parameters[parameterNames[0]])
		}
	}
	return view
}

func sortedMapIDs[T any](values map[studio.ID]T) []studio.ID {
	ids := make([]studio.ID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func formatBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	if value < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(value)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(value)/(1024*1024))
}

func hierarchyView(document studio.Document, selected studio.ID) []map[string]any {
	items := make([]map[string]any, 0, len(document.Entities))
	var walk func(studio.ID, int)
	walk = func(id studio.ID, depth int) {
		entity := document.Entities[id]
		class := fmt.Sprintf("depth-%d", min(depth, 2))
		if id == selected {
			class += " selected"
		}
		kind := "group"
		if entity.Mesh != nil {
			kind = "mesh"
		} else if entity.Model != nil {
			kind = "model"
		} else if entity.Light != nil {
			kind = "light"
		}
		items = append(items, map[string]any{"id": string(id), "name": entity.Name, "class": class, "kind": kind, "code": shortID(id)})
		for _, child := range entity.Children {
			walk(child, depth+1)
		}
	}
	for _, root := range document.RootIDs {
		walk(root, 0)
	}
	return items
}

func inspectorView(document studio.Document, selected studio.ID) map[string]any {
	entity, ok := document.Entities[selected]
	if !ok {
		return map[string]any{"id": "", "name": "No selection", "material": "—"}
	}
	materialName := "—"
	materialID := ""
	shader := "Standard PBR"
	roughness := "—"
	transmission := "—"
	geometry := "Group"
	modifierID := "solidify"
	subdivisionID := "subdivision"
	subdivisionLevels := "1"
	activeModifierID := ""
	thickness := "0.100"
	modifierStatus := "Requires an indexed mesh"
	assetID := ""
	assetName := "No model asset selected"
	materialColor := "#888888"
	metalness := "0"
	clearcoat := "0"
	emissive := "0"
	selenaSource := ""
	if entity.Mesh != nil {
		materialID = string(entity.Mesh.Material)
		geometry = entity.Mesh.Geometry.Kind
		if entity.Mesh.Geometry.Kind == "indexed-mesh" {
			modifierStatus = fmt.Sprintf("%d modifiers · ready", len(entity.Mesh.Modifiers))
			if len(entity.Mesh.Modifiers) > 0 {
				activeModifierID = string(entity.Mesh.Modifiers[0].ID)
			}
			for _, modifier := range entity.Mesh.Modifiers {
				if modifier.Kind == "solidify" {
					modifierID = string(modifier.ID)
					thickness = number(modifier.Thickness)
					break
				}
			}
			for _, modifier := range entity.Mesh.Modifiers {
				if modifier.Kind == "subdivision" {
					subdivisionID = string(modifier.ID)
					subdivisionLevels = fmt.Sprint(modifier.Levels)
					break
				}
			}
		}
		if material, exists := document.Materials[entity.Mesh.Material]; exists {
			materialName = material.Name
			roughness = number(material.Roughness)
			transmission = number(material.Transmission)
			materialColor = material.Color
			metalness = number(material.Metalness)
			clearcoat = number(material.Clearcoat)
			emissive = number(material.Emissive)
			if material.Selena != nil {
				shader = "Selena · " + material.Selena.Material
				selenaSource = material.Selena.Source
			}
		}
	} else if entity.Model != nil {
		geometry = "Model · " + string(entity.Model.Asset)
		assetID = string(entity.Model.Asset)
		if asset, exists := document.Assets[entity.Model.Asset]; exists {
			assetName = asset.SourceName
		}
	} else if entity.Light != nil {
		geometry = capitalizeASCII(entity.Light.Kind) + " light"
	}
	return map[string]any{"id": string(entity.ID), "name": entity.Name, "kind": geometry, "x": number(entity.Transform.Position.X), "y": number(entity.Transform.Position.Y), "z": number(entity.Transform.Position.Z), "rx": number(entity.Transform.Euler.X), "ry": number(entity.Transform.Euler.Y), "rz": number(entity.Transform.Euler.Z), "material": materialName, "materialId": materialID, "shader": shader, "roughness": roughness, "transmission": transmission, "visible": fmt.Sprint(entity.Visible), "locked": fmt.Sprint(entity.Locked), "modifierId": modifierID, "activeModifierId": activeModifierID, "thickness": thickness, "subdivisionId": subdivisionID, "subdivisionLevels": subdivisionLevels, "modifierStatus": modifierStatus, "assetId": assetID, "assetName": assetName, "materialColor": materialColor, "metalness": metalness, "clearcoat": clearcoat, "emissive": emissive, "selenaSource": selenaSource}
}

func shortID(id studio.ID) string {
	value := string(id)
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
func number(value float64) string { return fmt.Sprintf("%.3f", value) }

// historyView projects the workspace receipt history into console lines so
// command attribution is visible without replaying the journal.
func historyView(workspace *studio.Workspace) []map[string]any {
	if workspace == nil {
		return []map[string]any{}
	}
	receipts := workspace.RecentReceipts(8)
	out := make([]map[string]any, 0, len(receipts))
	for _, receipt := range receipts {
		kind := "transaction"
		if len(receipt.Changes) > 0 {
			kind = string(receipt.Changes[0].Kind)
		}
		actorLabel := "AGENT"
		if strings.HasPrefix(receipt.Actor, "human://") {
			actorLabel = "AUTHOR"
		}
		out = append(out, map[string]any{
			"afterRevision": fmt.Sprintf("%04d", receipt.AfterRevision),
			"actorLabel":    actorLabel,
			"summary":       fmt.Sprintf("%s %s (%s, %d op) %s", receipt.Actor, kind, receipt.Mode, receipt.Operations, receipt.TransactionID),
		})
	}
	return out
}

// liveCertificationView reports the deterministic evidence suite for the
// current document without running it on the render path. The suite renders
// frames and builds workspaces; measured on the sample document it costs
// about 2.3 seconds, and every edit changes the revision, so running it
// inline stalled the editor once per edit. It now runs in the background and
// the card reports which document the evidence belongs to.
//
// certState is "current" when the evidence matches the rendered revision,
// "recomputing" when older evidence is shown while the suite runs, and
// "pending" before any run has finished. The card must never present stale
// evidence as if it described the document on screen.
// capitalizeASCII upper-cases the first letter of a light kind for display.
// Light kinds are a closed ASCII set ("ambient", "directional", "point"), so
// this needs none of what strings.Title was deprecated for, and none of the
// x/text/cases dependency that replaces it.
func capitalizeASCII(value string) string {
	if value == "" {
		return value
	}
	first := value[0]
	if first >= 'a' && first <= 'z' {
		return string(first-32) + value[1:]
	}
	return value
}

// certifyCurrent is indirected so a test can prove the recover in
// runCertification actually contains a panic from the suite.
var certifyCurrent = studio.CertifyCurrent

var liveCertCache struct {
	sync.Mutex
	fingerprint string         // document the completed view describes
	revision    uint64         // revision the completed view describes
	view        map[string]any // published: never mutated after storing
	running     string         // fingerprint a background run is computing
}

func liveCertificationView(document studio.Document) map[string]any {
	fingerprint, err := document.Fingerprint()
	if err != nil {
		view := certificationView(studio.Certification())
		view["certState"], view["certRevision"] = "error: "+err.Error(), ""
		return view
	}

	liveCertCache.Lock()
	cached, cachedRevision := liveCertCache.view, liveCertCache.revision
	current := cached != nil && liveCertCache.fingerprint == fingerprint && cachedRevision == document.Revision
	if !current && liveCertCache.running != fingerprint {
		liveCertCache.running = fingerprint
		// The document is a per-request Snapshot clone that nothing mutates,
		// so the run may read it directly instead of cloning it again.
		go runCertification(document, fingerprint)
	}
	liveCertCache.Unlock()

	if cached == nil {
		view := certificationView(studio.Certification())
		view["certState"], view["certRevision"] = "pending", ""
		view["liveChecksPass"], view["liveChecksTotal"], view["releaseStatus"] = "0", "0", "measuring"
		return view
	}
	view := make(map[string]any, len(cached)+2)
	for key, value := range cached {
		view[key] = value
	}
	view["certRevision"] = fmt.Sprintf("%04d", cachedRevision)
	if current {
		view["certState"] = "current"
	} else {
		view["certState"] = "recomputing"
	}
	return view
}

// runCertification computes one evidence view and publishes it. It clears the
// running marker last so a failed run cannot wedge the card in "recomputing".
//
// The suite runs on its own goroutine, and an unrecovered panic on any
// goroutine takes the whole process with it. The suite is exactly the code
// most likely to panic: it drives edge cases, indexes results, and asserts on
// shapes it did not produce. The editor must not die because a check could not
// be taken, so a panic becomes a reported failure like any other.
func runCertification(document studio.Document, fingerprint string) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		log.Printf("certification evidence panicked: %v", recovered)
		view := certificationView(studio.Certification())
		view["liveChecksPass"], view["liveChecksTotal"] = "0", "0"
		view["releaseStatus"] = fmt.Sprintf("error: evidence suite panicked: %v", recovered)
		liveCertCache.Lock()
		defer liveCertCache.Unlock()
		liveCertCache.revision, liveCertCache.fingerprint, liveCertCache.view = document.Revision, fingerprint, view
		if liveCertCache.running == fingerprint {
			liveCertCache.running = ""
		}
	}()

	view := certificationView(studio.Certification())
	report, err := certifyCurrent(document)
	if err != nil {
		view["liveChecksPass"], view["liveChecksTotal"], view["releaseStatus"] = "0", "0", "error: "+err.Error()
	} else {
		pass := 0
		for _, check := range report.Checks {
			if check.Status == "pass" {
				pass++
			}
		}
		view = certificationView(report.Certification)
		view["liveChecksPass"], view["liveChecksTotal"], view["releaseStatus"] = fmt.Sprint(pass), fmt.Sprint(len(report.Checks)), report.ReleaseStatus
	}

	liveCertCache.Lock()
	defer liveCertCache.Unlock()
	liveCertCache.revision, liveCertCache.fingerprint, liveCertCache.view = document.Revision, fingerprint, view
	if liveCertCache.running == fingerprint {
		liveCertCache.running = ""
	}
}

// imageAssetsView lists image assets for texture slot selects.
func imageAssetsView(document studio.Document) []map[string]any {
	ids := sortedMapIDs(document.Assets)
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		asset := document.Assets[id]
		if asset.Kind == "image" {
			out = append(out, map[string]any{"id": string(id), "name": asset.SourceName})
		}
	}
	return out
}

// materialsView lists all document materials for the assign-material select.
func materialsView(document studio.Document) []map[string]any {
	ids := make([]string, 0, len(document.Materials))
	for id := range document.Materials {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		material := document.Materials[studio.ID(id)]
		out = append(out, map[string]any{"id": id, "name": material.Name})
	}
	return out
}

// modelingView summarizes the selected indexed mesh for the modeling forms:
// element counts, the live sub-object selection, and starter IDs so a human
// can act without hand-copying identifiers.
func modelingView(document studio.Document, workspace *studio.Workspace, selected studio.ID) map[string]any {
	view := map[string]any{"available": false, "status": "Select an indexed-mesh entity to model", "vertices": "0", "edges": "0", "faces": "0", "selectionSummary": "none", "firstFace": "", "firstVertex": "", "firstEdge": ""}
	entity, ok := document.Entities[selected]
	if !ok || entity.Mesh == nil || entity.Mesh.Geometry.Kind != "indexed-mesh" {
		return view
	}
	geometry := entity.Mesh.Geometry
	edges := studio.MeshEdges(geometry)
	view["available"] = true
	view["status"] = fmt.Sprintf("Indexed mesh · deterministic operators share the agent transaction path")
	view["vertices"], view["edges"], view["faces"] = fmt.Sprint(len(geometry.Vertices)), fmt.Sprint(len(edges)), fmt.Sprint(len(geometry.Faces))
	if len(geometry.Faces) > 0 {
		view["firstFace"] = string(geometry.Faces[0].ID)
	}
	if len(geometry.Vertices) > 0 {
		view["firstVertex"] = string(geometry.Vertices[0].ID)
	}
	if len(edges) > 0 {
		view["firstEdge"] = string(edges[0].ID)
	}
	if workspace != nil {
		state := workspace.SelectionState()
		if state.Object == selected && state.Mode != studio.SelectionObject {
			view["selectionSummary"] = fmt.Sprintf("%s · %d selected", state.Mode, len(state.IDs))
		}
	}
	return view
}

// agentView projects real automation state into the Agent Actions panel: the
// latest propose-mode receipt, session activity counts, and whether the
// bearer-token mutation surface is configured at all.
func agentView(workspace *studio.Workspace, tokenConfigured bool) map[string]any {
	view := map[string]any{
		"tokenConfigured": fmt.Sprint(tokenConfigured),
		"authority":       "read: open GET APIs · propose/direct: bearer token",
		"proposalPresent": false,
		"proposalSummary": "No proposals this session. POST /api/studio/actions/preview validates a transaction without mutating.",
		"proposalRevision": "—", "proposalAffected": "—", "proposalFingerprint": "—", "proposalActor": "—",
		"agentCount": "0", "humanCount": "0",
	}
	if !tokenConfigured {
		view["authority"] = "read: open GET APIs · propose/direct: DISABLED (set STUDIO_ACTION_TOKEN)"
	}
	if workspace == nil {
		return view
	}
	agents, humans := 0, 0
	for _, receipt := range workspace.RecentReceipts(32) {
		if strings.HasPrefix(receipt.Actor, "human://") {
			humans++
			continue
		}
		agents++
		if receipt.Mode == studio.ModePropose && view["proposalPresent"] == false {
			view["proposalPresent"] = true
			view["proposalSummary"] = fmt.Sprintf("%s · %d operation(s) validated without mutation", receipt.TransactionID, receipt.Operations)
			view["proposalRevision"] = fmt.Sprintf("%04d", receipt.BeforeRevision)
			view["proposalAffected"] = fmt.Sprint(len(receipt.Affected))
			view["proposalFingerprint"] = shortID(studio.ID(receipt.AfterFingerprint))
			view["proposalActor"] = receipt.Actor
		}
	}
	view["agentCount"], view["humanCount"] = fmt.Sprint(agents), fmt.Sprint(humans)
	return view
}

// prefabsView lists prefab definitions for the Prefabs panel forms.
func prefabsView(document studio.Document) []map[string]any {
	ids := sortedMapIDs(document.Prefabs)
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		definition := document.Prefabs[id]
		kind := "definition"
		if definition.Base != "" {
			kind = "variant of " + string(definition.Base)
		}
		out = append(out, map[string]any{"id": string(id), "name": definition.Name, "kind": kind, "entities": fmt.Sprint(len(definition.Entities))})
	}
	return out
}

// playView reports the play-mode session for the transport controls.
func playView(workspace *studio.Workspace) map[string]any {
	if workspace == nil {
		return map[string]any{"active": "false", "tick": "0", "simulation": ""}
	}
	state := workspace.PlayState()
	return map[string]any{"active": fmt.Sprint(state.Active), "tick": fmt.Sprint(state.Tick), "simulation": string(state.Simulation), "diffs": fmt.Sprint(len(workspace.PlayDiff()))}
}
