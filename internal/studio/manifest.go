// Package studio defines the handoff contract for the first GoSX 3D Studio
// vertical slice. It deliberately describes capability state without pretending
// that a visible scaffold is an implemented authoring feature.
package studio

const ManifestSchema = "gosx3d.studio.manifest/v1"

type Manifest struct {
	Schema          string             `json:"schema"`
	Product         string             `json:"product"`
	VisualDirection string             `json:"visualDirection"`
	Surfaces        []Surface          `json:"surfaces"`
	Capabilities    []Capability       `json:"capabilities"`
	Actions         []ActionCapability `json:"actions"`
	NextSlice       []string           `json:"nextSlice"`
}

type Surface struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
}

type Capability struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
}

func DefaultManifest() Manifest {
	return Manifest{
		Schema:          ManifestSchema,
		Product:         "GoSX 3D Studio",
		VisualDirection: "Industrial Void",
		Surfaces: []Surface{
			{ID: "project", Label: "Project / Assets", Status: "partial"},
			{ID: "hierarchy", Label: "Scene Hierarchy", Status: "available"},
			{ID: "viewport", Label: "Scene3D Viewport", Status: "available"},
			{ID: "inspector", Label: "Inspector", Status: "partial"},
			{ID: "timeline", Label: "Timeline", Status: "partial"},
			{ID: "telemetry", Label: "Telemetry", Status: "scaffolded"},
			{ID: "agent-actions", Label: "Agent Actions", Status: "api-partial"},
		},
		Capabilities: []Capability{
			{ID: "server-shell", Status: "available", Evidence: "GET / and GET /api/health"},
			{ID: "agent-manifest", Status: "available", Evidence: "GET /api/studio/manifest"},
			{ID: "scene-document", Status: "available", Evidence: "GET /api/studio/document and internal/studio document tests"},
			{ID: "scene3d-compile", Status: "available", Evidence: "GET /api/studio/scene-ir and shared SceneIR tests"},
			{ID: "scene3d-mount", Status: "available", Evidence: "typed Scene3D mount in app/page.gsx and gosx build"},
			{ID: "native-harness", Status: "available", Evidence: "go run ./cmd/studio-smoke"},
			{ID: "m0-certification", Status: "available", Evidence: "go run ./cmd/studio-certify produces byte-deterministic SceneDoc, source-map, incremental, frame, exact-pick, and Selena checks while retaining partial public-v1 release status"},
			{ID: "selena-artifact", Status: "available", Evidence: "SceneDoc-owned Selena source compiles through canonical WGSL/GLSL transport and browser-free artifact hashes"},
			{ID: "command-journal", Status: "available", Evidence: "checksummed append-only journal, torn-tail recovery, corrupt-save quarantine, and explicit atomic save tests"},
			{ID: "project-save-recovery", Status: "available", Evidence: "human Save action and authenticated save/status API preserve last explicit save while journaled edits recover on restart"},
			{ID: "topology-extrude", Status: "available", Evidence: "indexed SceneDoc geometry, SceneIR compilation, checkpoint-safe undo test"},
			{ID: "subobject-selection", Status: "available", Evidence: "revision-safe object/vertex/derived-edge/face selection state, authenticated API, topology reconciliation, and deterministic stable-ID tests"},
			{ID: "topology-core-floor", Status: "partial", Evidence: "extrude, inset, triangulate, weld, fill, dissolve, bridge, crack-free quad-ring loop cut, normal recalculation, and planar UV projection have typed preview/direct actions, normalized operator records, stable IDs, compilation, and checkpoint tests; bevel, broader dissolve cases, and remaining M1 workflows pending"},
			{ID: "uv-authoring-inspection", Status: "available", Evidence: "typed SceneDoc UVs, normalized planar projection action, BufferGeometry/SceneIR transport, completeness/bounds/tile/degeneracy inspection, undo, and deterministic certification"},
			{ID: "nurbs-curve-foundation", Status: "available", Evidence: "stable weighted control points, knot/degree validation, revision-safe selection/action, rational B-spline evaluation, deterministic tube tessellation with normals/UVs, SceneIR compilation, length analysis, checkpoint undo, and certification"},
			{ID: "curves-surface-tools", Status: "partial", Evidence: "open rational B-spline tube authoring is available; cyclic curves, handles, loft/revolve/sweep surfaces, conversion, snapping, and full human tools remain pending"},
			{ID: "modifier-stack-foundation", Status: "available", Evidence: "ordered stable-ID subdivision, mirror, array, and solidify records; exact-index reorder; deterministic apply-through baking; validation; shared human/agent set/remove/reorder/apply actions; preview/direct parity; SceneIR compilation; checkpoint undo; and certification"},
			{ID: "solidify-modifier", Status: "available", Evidence: "deterministic normal-offset shells, boundary closure, stable generated topology, closed/manifold analysis, shared agent action, human Inspector action, preview/direct parity, undo, SceneIR compilation, and browser-free certification"},
			{ID: "subdivision-modifier", Status: "available", Evidence: "bounded Catmull-Clark subdivision uses shared edge points, face points, boundary rules, stable IDs, crack-free closed/manifold output, multi-level evaluation, human/agent set-modifier parity, undo, SceneIR compilation, and deterministic certification"},
			{ID: "modifier-library", Status: "partial", Evidence: "subdivision, mirror, array, solidify, reorder, and apply-through are available; bevel, decimate, normal, boolean modifier form, curve, lattice, shrinkwrap, parameter animation, enablement profiles, and richer visual stack tools remain pending"},
			{ID: "voxel-csg", Status: "available", Evidence: "closed-manifold union/intersection/subtraction, modifier evaluation, bounded deterministic voxelization, stable surface IDs, preview/direct result creation, analysis, SceneIR compilation, undo, and certification"},
			{ID: "csg-tooling", Status: "partial", Evidence: "deterministic voxel CSG is available; analytic polygon booleans, rotated operands, adaptive resolution, live modifier form, and full human tools remain pending"},
			{ID: "material-authoring", Status: "available", Evidence: "validated PBR/Selena set and reference-safe delete actions, preview/direct parity, semantic material receipts, invalid-Selena last-good preservation, SceneIR compilation, undo, and certification"},
			{ID: "material-tools", Status: "partial", Evidence: "PBR and Selena records are authorable through shared actions; texture slots, node graphs, baking, live human editors, and richer diagnostics remain pending"},
			{ID: "linked-prefabs", Status: "available", Evidence: "subtree capture/update, stable linked instances, transform/material/visibility overrides, reference-safe deletion, namespaced runtime IDs, prefab source-map provenance, incremental invalidation, shared actions, undo, and certification"},
			{ID: "prefab-tools", Status: "partial", Evidence: "linked definitions and overrides are available; nested prefabs, variants, unpacking, human editors, and cross-project packages remain pending"},
			{ID: "content-addressed-assets", Status: "available", Evidence: "inspected GLB/glTF, image, audio, video, and OBJ imports use SHA-256 identities, immutable project payload paths, shared propose/direct import and atomic reimport actions, semantic receipts, dependency reports, integrity audits, authenticated APIs, and deterministic certification"},
			{ID: "model-assets", Status: "available", Evidence: "SceneDoc model entities reference content-addressed GLB/glTF records and compile through typed Scene3D Model into shared SceneIR with source-map and incremental invalidation coverage"},
			{ID: "asset-dependencies-reimport", Status: "available", Evidence: "sorted direct entity, prefab-definition, and linked-instance dependency reports; content-addressed reimport; atomic reference migration; human/agent propose-direct parity; reference-safe deletion; audit; SceneIR compilation; and deterministic certification"},
			{ID: "human-asset-database", Status: "available", Evidence: "Project panel renders live SceneDoc assets with content IDs, source names, formats, byte sizes, and dependency counts; human initial import and model reimport use the same revision-safe inspected commands as agents; trusted desktop chooser fills the same form"},
			{ID: "native-asset-import", Status: "partial", Evidence: "File > Import Asset and Project chooser use gosxDesktop.dialog.openFile with typed asset filters, accessible cancel/failure/manual-path fallback, and shared human import confirmation; live Windows dialog evidence pending"},
			{ID: "asset-tools", Status: "partial", Evidence: "content addressing, byte inspection, audits, HTTP serving, live Project panel, human/native-chooser import contract, deterministic confined external glTF dependency packaging, dependencies, reimport/retarget, reference-safe garbage collection, and GLB/glTF model entities are available; thumbnails, optimization, conversion, and automatic file watching remain pending"},
			{ID: "asset-garbage-collection", Status: "available", Evidence: "deterministic transitive reachability, shared-dependency-safe delete ordering, cycle validation, exact revision and plan confirmation, propose preview, explicit non-undoable checkpoint, payload reclamation diagnostics, authenticated typed agent endpoint, human Project action, tests, and browser-free certification"},
			{ID: "gltf-dependency-packaging", Status: "available", Evidence: "relative external buffers and images are source-directory confined with symlink/traversal/remote rejection, imported as typed content-addressed assets, deterministically rewritten to immutable content URLs, reference-validated, dependency-reported, deletion-safe, integrity-audited, tested, and browser-free certified"},
			{ID: "gltf-capability-matrix", Status: "partial", Evidence: "versioned deterministic GLB/glTF extension inspection, required-vs-optional distinction, stable compatibility fingerprints, four target verdicts, Draco/Meshopt/KTX2 honesty diagnostics, stable corpus case IDs, public matrix/corpus endpoints, import metadata, and browser-free certification are available; decoder execution, external dependencies, HDR/EXR, export loss reports, and full corpus assets remain pending"},
			{ID: "indexed-mesh-analysis", Status: "available", Evidence: "browser-free revision-tagged bounds, area, closed-volume, topology counts, manifold/boundary, degeneracy, duplicate-position, and isolated-vertex diagnostics with stable IDs"},
			{ID: "measurement-validation", Status: "partial", Evidence: "indexed-mesh analysis endpoint and deterministic tests are available; interactive rulers, repair actions, UV checks, and broader asset validation remain pending"},
			{ID: "viewport-exact-selection", Status: "available", Evidence: "Scene3D mount-input bridge, stale/pickable validation, workspace selection, and strict harness agreement tests"},
			{ID: "inspector-transform", Status: "available", Evidence: "human form executes the same revision-safe set-transform transaction, semantic receipt, journal, and undo path as agents"},
			{ID: "agent-actions", Status: "partial", Evidence: "authenticated discovery, semantic preview/direct receipts, undo and redo APIs"},
			{ID: "desktop-host", Status: "partial", Evidence: "Windows-default WebView2 entrypoint, native project/asset dialogs, native menus, bridge wiring, loopback server, offline PE cross-build, and staged MSIX manifest; Windows runtime, MakeAppx, signing, install, update, and recovery evidence pending"},
			{ID: "native-project-open", Status: "partial", Evidence: "native gosxDesktop OpenFile bridge, canonical-path validation, dirty-discard policy, revision safety, agent endpoint, and switch tests; live Windows dialog evidence pending"},
			{ID: "rig-animation-foundation", Status: "partial", Evidence: "stable three-bone armature graph, normalized four-influence skin contract, hierarchical linear-blend deformation, deterministic two-bone CPU IK, bone pose/animation-key/IK actions, semantic receipts, deterministic transform sampling, recorded fixed-step physics interaction, armature-aware incremental invalidation, typed SceneIR lowering, undo, and browser-free certification; richer editors, retargeting, state machines, dual-quaternion skinning, and advanced physics remain pending"},
			{ID: "linear-blend-skinning", Status: "available", Evidence: "hierarchical rest/pose matrices, inverse bind transforms, normalized one-to-four influence blending, normal deformation, stable topology/UV preservation, non-mutating evaluation, armature-aware incremental invalidation, typed BufferGeometry/SceneIR lowering, and browser-free deterministic certification"},
			{ID: "fixed-step-physics", Status: "available", Evidence: "stable body/profile schemas, 60 Hz semi-implicit Euler reference, gravity, sphere/plane contact response, tick-addressed impulse recording, exact state hashes, replay divergence detection, shared propose/direct simulate action, semantic receipt, undo, human timeline control, and browser-free certification"},
			{ID: "human-rig-timeline", Status: "partial", Evidence: "live SceneDoc armature/clip/simulation view plus human set-pose, insert-key, solve-IK, and advance-ticks forms converge on shared revision-safe actions; playhead scrubbing, dope sheet, curve editor, transport, selection, and richer diagnostics remain pending"},
			{ID: "animation-retargeting", Status: "available", Evidence: "stable validated one-to-one armature maps, automatic or authored scale, rest-relative position/rotation transfer, stable target track IDs, shared propose/direct action, semantic receipt, undo, human timeline form, and browser-free certification"},
			{ID: "animation-state-machine", Status: "partial", Evidence: "stable states/parameters/transitions, six numeric operators compiled from embedded .arb rules, deterministic priority/ID arbitration, decision traces, exact active-clip sampling, shared parameter/step actions, preview parity, undo, human timeline forms, and browser-free certification; crossfades, masks, blend spaces, layers, root motion, and graph editor remain pending"},
			{ID: "render-resource-graph", Status: "partial", Evidence: "stable retained resource/pass records, deterministic dependency scheduling, cycle and transient read-before-write diagnostics, exact lifetime intervals, non-overlap alias planning, revision-safe agent action, and shared SceneIR transport; backend execution, MRT/reflections, human editing, leak telemetry, and profiler UI remain pending"},
		},
		Actions: ActionCatalog(),
		NextSlice: []string{
			"complete bevel, broader dissolve cases, and topology repair",
			"complete cyclic curves, curve surface tools, topology repair, richer modifiers, and interactive measurement",
			"complete asset thumbnails, optimization, conversion, and automatic file watching",
			"bind M1 human tools through the normalized action catalog",
			"package and recovery-test the Windows desktop host",
			"complete timeline transport/scrubbing, retarget-map editing, state-machine graph editing, and advanced physics authoring",
			"complete animation crossfades, masks, blend spaces, layers, root motion, and graph editing",
		},
	}
}
