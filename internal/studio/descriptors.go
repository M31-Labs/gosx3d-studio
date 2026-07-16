package studio

import (
	"encoding/json"

	"m31labs.dev/gosx/scene/cert"
)

type ActionDescriptor struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Access          string            `json:"access"`
	AuthorityModes  []TransactionMode `json:"authorityModes"`
	SupportsPreview bool              `json:"supportsPreview"`
	SupportsBatch   bool              `json:"supportsBatch"`
	UndoPolicy      string            `json:"undoPolicy"`
	Headless        string            `json:"headless"`
	InputSchema     map[string]any    `json:"inputSchema"`
	Endpoint        string            `json:"endpoint"`
}

func ActionDescriptors() []ActionDescriptor {
	descriptions := map[OperationKind]string{OpSetField: "Set a descriptor-backed entity field.", OpSetTransform: "Replace an entity local transform.", OpAssignMaterial: "Assign an existing material to a mesh entity.", OpRenameEntity: "Rename an entity without changing identity.", OpCreateEntity: "Create a stable-ID entity.", OpDeleteEntity: "Delete an entity subtree.", OpReparentEntity: "Move an entity while preserving identity.", OpDuplicateEntity: "Duplicate an entity subtree with deterministic descendant IDs.", OpExtrudeFaces: "Extrude stable face IDs with a geometry checkpoint.", OpInsetFaces: "Inset stable polygon face IDs.", OpTriangulateFaces: "Triangulate stable polygon face IDs.", OpWeldVertices: "Weld stable vertex IDs into a deterministic survivor.", OpFillFace: "Fill a polygon from stable vertex IDs.", OpRecalculateNormals: "Recalculate indexed-mesh vertex normals.", OpProjectPlanarUV: "Project stable vertices into normalized planar UV space.", OpDissolveEdges: "Dissolve a stable shared edge and merge its adjacent polygons.", OpBridgeLoops: "Bridge two stable closed vertex loops with deterministic quad IDs.", OpLoopCut: "Traverse a quad ring from a stable edge and split it without cracks.", OpSetCurveControlPoint: "Set a stable NURBS control-point position or rational weight."}
	descriptions[OpSetModifier] = "Add or replace a stable non-destructive mesh modifier."
	descriptions[OpRemoveModifier] = "Remove a stable non-destructive mesh modifier."
	descriptions[OpReorderModifier] = "Move a stable modifier to an exact stack index."
	descriptions[OpApplyModifier] = "Bake the modifier stack through a stable modifier checkpoint."
	descriptions[OpCSGBoolean] = "Create a deterministic voxel union, intersection, or subtraction from closed meshes."
	descriptions[OpSetMaterial] = "Add or replace a validated PBR or Selena material record."
	descriptions[OpDeleteMaterial] = "Delete an unreferenced material record."
	descriptions[OpCapturePrefab] = "Capture an authored entity subtree as a linked prefab definition."
	descriptions[OpInstantiatePrefab] = "Create a linked prefab instance with stable runtime IDs."
	descriptions[OpSetPrefabOverride] = "Set or clear a stable local-entity prefab override."
	descriptions[OpDeletePrefab] = "Delete an unreferenced prefab definition."
	descriptions[OpRegisterAsset] = "Register inspected content-addressed asset metadata."
	descriptions[OpDeleteAsset] = "Delete an unreferenced asset record."
	descriptions[OpReimportAsset] = "Replace an asset revision and atomically retarget its model references."
	descriptions[OpSetBonePose] = "Pose a stable armature bone and its bound articulated entity."
	descriptions[OpSetAnimationKey] = "Insert or replace a deterministic transform key on a stable animation track."
	descriptions[OpSolveIK] = "Solve a stable two-bone IK constraint with the deterministic CPU reference evaluator."
	descriptions[OpSetPhysicsBody] = "Set a validated rigid-body and collider component on a stable entity."
	descriptions[OpSimulateTicks] = "Advance a deterministic fixed-step simulation with recorded inputs."
	descriptions[OpRetargetAnimation] = "Retarget a stable bone clip through a validated armature map."
	descriptions[OpSetAnimationParameter] = "Set a declared animation-state-machine parameter."
	descriptions[OpStepAnimationMachine] = "Advance an Arbiter-governed animation state machine and sample its active clip."
	descriptions[OpSetRenderGraph] = "Replace the retained render/resource graph after deterministic schedule and lifetime validation."
	descriptions[OpCollectUnusedAssets] = "Plan or checkpoint deletion of unreachable asset records and payload bytes."
	required := map[OperationKind][]string{OpSetField: {"target", "field", "value"}, OpSetTransform: {"target", "transform"}, OpAssignMaterial: {"target", "material"}, OpRenameEntity: {"target", "name"}, OpCreateEntity: {"entity"}, OpDeleteEntity: {"target"}, OpReparentEntity: {"target", "parent"}, OpDuplicateEntity: {"target", "newId"}, OpExtrudeFaces: {"target", "faces", "distance"}, OpInsetFaces: {"target", "faces", "amount"}, OpTriangulateFaces: {"target", "faces"}, OpWeldVertices: {"target", "vertices", "tolerance"}, OpFillFace: {"target", "vertices", "newId"}, OpRecalculateNormals: {"target"}, OpProjectPlanarUV: {"target", "projection"}, OpDissolveEdges: {"target", "edges"}, OpBridgeLoops: {"target", "loops", "newId", "closed"}, OpLoopCut: {"target", "edges", "amount", "newId"}, OpSetCurveControlPoint: {"target", "controlPoint"}}
	required[OpSetModifier] = []string{"target", "modifier"}
	required[OpRemoveModifier] = []string{"target", "modifierId"}
	required[OpReorderModifier] = []string{"target", "modifierId", "modifierIndex"}
	required[OpApplyModifier] = []string{"target", "modifierId"}
	required[OpCSGBoolean] = []string{"target", "operand", "boolean", "voxelSize", "newId"}
	required[OpSetMaterial] = []string{"materialRecord"}
	required[OpDeleteMaterial] = []string{"materialId"}
	required[OpCapturePrefab] = []string{"target", "prefabId", "name"}
	required[OpInstantiatePrefab] = []string{"prefabId", "newId"}
	required[OpSetPrefabOverride] = []string{"target", "prefabEntity"}
	required[OpDeletePrefab] = []string{"prefabId"}
	required[OpRegisterAsset] = []string{"asset"}
	required[OpDeleteAsset] = []string{"assetId"}
	required[OpReimportAsset] = []string{"assetId", "asset"}
	required[OpSetBonePose] = []string{"armatureId", "boneId", "transform"}
	required[OpSetAnimationKey] = []string{"clipId", "trackId", "key"}
	required[OpSolveIK] = []string{"armatureId", "constraintId"}
	required[OpSetPhysicsBody] = []string{"target", "physics"}
	required[OpSimulateTicks] = []string{"simulationId", "ticks"}
	required[OpRetargetAnimation] = []string{"retargetMapId", "sourceClipId", "newId", "name"}
	required[OpSetAnimationParameter] = []string{"machineId", "parameter", "number"}
	required[OpStepAnimationMachine] = []string{"machineId", "deltaTime"}
	required[OpSetRenderGraph] = []string{"renderGraph"}
	required[OpCollectUnusedAssets] = []string{"actor", "mode", "expectedRevision"}
	out := make([]ActionDescriptor, 0, len(descriptions))
	for _, capability := range ActionCatalog() {
		kind := OperationKind(capability.ID)
		endpoint, undoPolicy, supportsBatch := "/api/studio/transactions/call", "inverse-or-checkpoint", true
		if kind == OpCollectUnusedAssets {
			endpoint = "/api/studio/assets/garbage-collect"
			undoPolicy = "explicit-checkpoint"
			supportsBatch = false
		}
		out = append(out, ActionDescriptor{Name: capability.ID, Version: "1.0.0", Description: descriptions[kind], Access: "mutate", AuthorityModes: []TransactionMode{ModePropose, ModeDirect}, SupportsPreview: true, SupportsBatch: supportsBatch, UndoPolicy: undoPolicy, Headless: "available", Endpoint: endpoint, InputSchema: map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "required": required[kind], "properties": operationSchemaProperties()}})
	}
	return out
}

func operationSchemaProperties() map[string]any {
	properties := map[string]any{
		"target": map[string]any{"type": "string"}, "field": map[string]any{"type": "string"}, "value": map[string]any{},
		"transform": map[string]any{"type": "object"}, "material": map[string]any{"type": "string"}, "name": map[string]any{"type": "string", "minLength": 1},
		"entity": map[string]any{"type": "object"}, "parent": map[string]any{"type": "string"}, "newId": map[string]any{"type": "string", "minLength": 1},
		"faces":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1},
		"vertices": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 2},
		"edges":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1, "maxItems": 1},
		"loops":    map[string]any{"type": "array", "minItems": 2, "maxItems": 2, "items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 2}},
		"closed":   map[string]any{"const": true}, "distance": map[string]any{"type": "number", "not": map[string]any{"const": 0}},
		"amount": map[string]any{"type": "number", "exclusiveMinimum": 0, "exclusiveMaximum": 1}, "tolerance": map[string]any{"type": "number", "exclusiveMinimum": 0},
		"coordinateSpace": map[string]any{"type": "string", "enum": []string{"object"}}, "projection": map[string]any{"type": "string", "enum": []string{"xy", "xz", "yz"}},
		"controlPoint": map[string]any{"type": "string", "minLength": 1}, "position": map[string]any{"type": "object"}, "weight": map[string]any{"type": "number", "exclusiveMinimum": 0},
		"modifier": map[string]any{"type": "object"}, "modifierId": map[string]any{"type": "string", "minLength": 1}, "operand": map[string]any{"type": "string", "minLength": 1},
		"modifierIndex": map[string]any{"type": "integer", "minimum": 0},
		"boolean":       map[string]any{"type": "string", "enum": []string{"union", "intersection", "subtract"}}, "voxelSize": map[string]any{"type": "number", "exclusiveMinimum": 0},
		"materialRecord": map[string]any{"type": "object"}, "materialId": map[string]any{"type": "string", "minLength": 1},
		"prefabId": map[string]any{"type": "string", "minLength": 1}, "prefabEntity": map[string]any{"type": "string", "minLength": 1}, "prefabOverride": map[string]any{"type": "object"},
		"asset": map[string]any{"type": "object"}, "assetId": map[string]any{"type": "string", "minLength": 1},
		"armatureId": map[string]any{"type": "string", "minLength": 1}, "boneId": map[string]any{"type": "string", "minLength": 1},
		"clipId": map[string]any{"type": "string", "minLength": 1}, "trackId": map[string]any{"type": "string", "minLength": 1}, "key": map[string]any{"type": "object"},
		"constraintId": map[string]any{"type": "string", "minLength": 1},
		"physics":      map[string]any{"type": "object"}, "simulationId": map[string]any{"type": "string", "minLength": 1}, "ticks": map[string]any{"type": "integer", "minimum": 1, "maximum": 1000000}, "inputs": map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
		"retargetMapId": map[string]any{"type": "string", "minLength": 1}, "sourceClipId": map[string]any{"type": "string", "minLength": 1}, "machineId": map[string]any{"type": "string", "minLength": 1},
		"parameter": map[string]any{"type": "string", "minLength": 1}, "number": map[string]any{"type": "number"}, "deltaTime": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 3600},
		"renderGraph": map[string]any{"type": "object"},
		"actor":       map[string]any{"type": "string", "minLength": 1}, "mode": map[string]any{"type": "string", "enum": []string{"propose", "direct"}}, "expectedRevision": map[string]any{"type": "integer", "minimum": 0}, "confirmPlan": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"},
	}
	return properties
}

type FieldDescriptor struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Editable bool   `json:"editable"`
	Semantic string `json:"semantic,omitempty"`
}
type ComponentDescriptor struct {
	Type    string            `json:"type"`
	Version uint32            `json:"version"`
	Fields  []FieldDescriptor `json:"fields"`
}

func ComponentCatalog() []ComponentDescriptor {
	return []ComponentDescriptor{
		{Type: "transform", Version: 2, Fields: []FieldDescriptor{{Name: "position", Type: "vec3", Editable: true, Semantic: "authoring"}, {Name: "quaternion", Type: "quaternion", Editable: true, Semantic: "authoring"}, {Name: "rotation", Type: "euler", Editable: true, Semantic: "display"}, {Name: "scale", Type: "vec3", Editable: true, Semantic: "authoring"}}},
		{Type: "mesh", Version: 1, Fields: []FieldDescriptor{{Name: "geometry", Type: "geometry", Editable: true, Semantic: "authoring"}, {Name: "material", Type: "material-ref", Editable: true, Semantic: "authoring"}, {Name: "pickable", Type: "bool", Editable: true, Semantic: "runtime"}}},
		{Type: "material", Version: 1, Fields: []FieldDescriptor{{Name: "pbr", Type: "standard-material", Editable: true, Semantic: "authoring"}, {Name: "selena", Type: "selena-source", Editable: true, Semantic: "shader"}}},
		{Type: "armature", Version: 1, Fields: []FieldDescriptor{{Name: "bones", Type: "bone-graph", Editable: true, Semantic: "rigging"}, {Name: "pose", Type: "bone-transforms", Editable: true, Semantic: "animation"}, {Name: "constraints", Type: "constraint-list", Editable: true, Semantic: "runtime"}}},
		{Type: "skin", Version: 1, Fields: []FieldDescriptor{{Name: "armature", Type: "armature-ref", Editable: true, Semantic: "rigging"}, {Name: "weights", Type: "vertex-influences", Editable: true, Semantic: "deformation"}}},
		{Type: "animation", Version: 1, Fields: []FieldDescriptor{{Name: "tracks", Type: "transform-tracks", Editable: true, Semantic: "animation"}, {Name: "duration", Type: "seconds", Editable: true, Semantic: "animation"}}},
		{Type: "physics", Version: 1, Fields: []FieldDescriptor{{Name: "kind", Type: "enum", Editable: true, Semantic: "simulation"}, {Name: "mass", Type: "number", Editable: true, Semantic: "simulation"}, {Name: "velocity", Type: "vec3", Editable: true, Semantic: "runtime"}, {Name: "collider", Type: "collider", Editable: true, Semantic: "simulation"}}},
		{Type: "simulation", Version: 1, Fields: []FieldDescriptor{{Name: "tickRate", Type: "integer", Editable: true, Semantic: "simulation"}, {Name: "gravity", Type: "vec3", Editable: true, Semantic: "simulation"}, {Name: "bodies", Type: "entity-refs", Editable: true, Semantic: "simulation"}}},
		{Type: "retarget-map", Version: 1, Fields: []FieldDescriptor{{Name: "source", Type: "armature-ref", Editable: true, Semantic: "animation"}, {Name: "target", Type: "armature-ref", Editable: true, Semantic: "animation"}, {Name: "bones", Type: "bone-map", Editable: true, Semantic: "animation"}}},
		{Type: "animation-state-machine", Version: 1, Fields: []FieldDescriptor{{Name: "states", Type: "state-map", Editable: true, Semantic: "animation"}, {Name: "transitions", Type: "arbiter-transitions", Editable: true, Semantic: "governed-runtime"}, {Name: "parameters", Type: "number-map", Editable: true, Semantic: "runtime"}}},
		{Type: "render-graph", Version: 1, Fields: []FieldDescriptor{{Name: "resources", Type: "resource-map", Editable: true, Semantic: "authoring"}, {Name: "passes", Type: "pass-dag", Editable: true, Semantic: "runtime"}, {Name: "allocations", Type: "lifetime-plan", Editable: false, Semantic: "observed"}}},
	}
}

type CertificationReport struct {
	Schema     string                `json:"schema"`
	Status     string                `json:"status"`
	Dimensions map[string]Capability `json:"dimensions"`
	Framework  cert.Report           `json:"framework"`
}

func Certification() CertificationReport {
	return CertificationReport{Schema: "gosx3d.studio.certification/v1", Status: "partial", Framework: cert.BuildReport(), Dimensions: map[string]Capability{
		"sceneDoc":                 {ID: "sceneDoc", Status: "available", Evidence: "validation, stable IDs, deterministic fingerprints, migration"},
		"undoReplay":               {ID: "undoReplay", Status: "available", Evidence: "command, undo/redo, checksummed journal recovery tests"},
		"projectLifecycle":         {ID: "projectLifecycle", Status: "partial", Evidence: "startup project open, explicit atomic save, dirty/recovered state, torn-tail recovery, and corrupt-save quarantine; native file dialog pending"},
		"sourceMap":                {ID: "sourceMap", Status: "partial", Evidence: "entity and linked-prefab runtime provenance are available; field-level provenance pending"},
		"inspector":                {ID: "inspector", Status: "partial", Evidence: "descriptor catalog, live reads, and revision-safe transform writes; remaining component editors pending"},
		"agentAPI":                 {ID: "agentAPI", Status: "partial", Evidence: "authenticated discovery, preview, direct, undo and redo JSON APIs"},
		"desktopHost":              {ID: "desktopHost", Status: "partial", Evidence: "Windows native entrypoint and bridge cross-build; staged MSIX payload exists, but live Windows/install/sign/update evidence is pending"},
		"headless":                 {ID: "headless", Status: "available", Evidence: "cmd/studio-smoke frame and exact trace"},
		"shaderValidation":         {ID: "shaderValidation", Status: "partial", Evidence: "SceneDoc Selena source compiles to validated WGSL/GLSL artifact hashes in M0 evidence; source diagnostics and last-good fallback pending"},
		"modeling":                 {ID: "modeling", Status: "partial", Evidence: "stable vertex/edge/face selection and ten deterministic indexed-mesh/UV operators are certified through shared commands; remaining M1 modeling scope pending"},
		"curves":                   {ID: "curves", Status: "partial", Evidence: "stable rational NURBS controls, deterministic tube tessellation, SceneIR transport, analysis, shared actions, and headless evidence; remaining curve/surface tools pending"},
		"modifiers":                {ID: "modifiers", Status: "partial", Evidence: "ordered mirror/array stacks evaluate deterministically through shared SceneIR commands and evidence; remaining modifier library and UI pending"},
		"csg":                      {ID: "csg", Status: "partial", Evidence: "bounded deterministic voxel union/intersection/subtraction is certified for closed manifold meshes; analytic and interactive CSG tooling pending"},
		"materials":                {ID: "materials", Status: "partial", Evidence: "validated PBR/Selena shared actions and last-good shader preservation are certified; texture/node/bake/human tooling pending"},
		"prefabs":                  {ID: "prefabs", Status: "partial", Evidence: "linked stable instances, overrides, provenance, incremental invalidation, and shared actions are certified; nested/variant/package tooling pending"},
		"riggingAnimation":         {ID: "riggingAnimation", Status: "partial", Evidence: "stable armature, normalized skin weights, hierarchical linear-blend deformation, deterministic two-bone IK, transform clips, exact-time sampling, shared pose/key/IK actions, incremental invalidation, typed SceneIR lowering, retarget maps, governed state transitions, and a recorded physics interaction are certified; richer editors, blend spaces, layered clips, dual-quaternion skinning, and advanced physics remain pending"},
		"simulation":               {ID: "simulation", Status: "partial", Evidence: "stable physics bodies, sphere/plane colliders, fixed-step gravity/contact response, tick-addressed impulses, exact snapshots, recording/replay, shared simulate actions, and browser-free evidence are available; general rigid-body pairs, constraints, broad phase, CCD, fields, particles, audio, caches, and runtime editors remain pending"},
		"retargetingStateMachines": {ID: "retargetingStateMachines", Status: "partial", Evidence: "stable one-to-one armature maps, rest-relative scaled clip transfer, Arbiter-compiled transition rules, deterministic priority/ID selection, transition traces, exact clip sampling, shared actions, and browser-free evidence are available; masks, blend spaces, crossfades, layered graphs, root motion, and full editors remain pending"},
		"renderResourceGraph":      {ID: "renderResourceGraph", Status: "partial", Evidence: "stable retained resources/passes, deterministic DAG scheduling, transient read-before-write diagnostics, lifetime intervals, alias slots, shared SceneIR transport, revision-safe agent action, and browser-free tests are available; backend execution, MRT/reflections, human graph editing, leak telemetry, and profiler UI remain pending"},
		"gltfCompatibility":        {ID: "gltfCompatibility", Status: "partial", Evidence: "versioned extension/compression matrix, deterministic GLB JSON inspection, required/optional target verdicts, compatibility fingerprints, stable migration corpus IDs, confined content-addressed external dependency packaging, imported-asset diagnostics, public endpoints, tests, and browser-free evidence are available; Draco/Meshopt/KTX2 execution, HDR/EXR IBL decoding, export loss reports, and the full downloadable corpus remain pending"},
	}}
}

const LegacySceneDocSchema = "gosx3d.scene-document/v1"

func MigrateDocument(document Document) (Document, error) {
	if document.Schema == LegacySceneDocSchema {
		document.Schema = SceneDocSchema
		if document.Metadata == nil {
			document.Metadata = map[string]string{}
		}
		document.Metadata["migratedFrom"] = LegacySceneDocSchema
	}
	if err := document.Validate(); err != nil {
		return Document{}, err
	}
	return document, nil
}

func BoolValue(value bool) json.RawMessage { data, _ := json.Marshal(value); return data }
