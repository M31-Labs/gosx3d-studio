# Architecture map

## Visual System

The binding territory is **Industrial Void**, a compact Dark Elegance tool UI:
near-black canvas, layered graphite work surfaces, restrained structure, and
semantic state color rather than decoration. Orange is authored state, cyan is
observed runtime state, and gold is trusted certification state.

- Typography uses Space Grotesk 600 for display labels, Work Sans 400/600 for
  body/UI copy, and JetBrains Mono 400 for identities and telemetry. The compact
  tool scale follows a 1.2 minor-third ratio.
- The 60/30/10 palette is canvas `#0b0d10`, graphite panels
  `#14181c`/`#23272c`, and authored orange `#ff8a2a`. Primary `#e6e2d6`,
  secondary `#b9b4a8`, and muted `#918d84` text all meet WCAG AA against the
  canvas; primary and secondary meet AAA. Focus uses `#f6a45e`.
- Motion is Minimal: 140 ms control feedback and 200 ms panel transitions with
  `cubic-bezier(0.16, 1, 0.3, 1)` ease-out and
  `cubic-bezier(0.34, 1.56, 0.64, 1)` spring feedback. Reduced-motion collapses
  transitions to 1 ms.
- Spacing uses a compact 4 px base and responsive `xs` through `3xl` tokens.
  Dense tool rows intentionally use the lower half of the scale.

The canonical ready-to-use token contract is:

```css
:root {
  --font-display: "Space Grotesk", "Aptos Display", sans-serif;
  --font-body: "Work Sans", "Aptos", sans-serif;
  --font-mono: "JetBrains Mono", "Cascadia Mono", monospace;
  --type-xs: clamp(0.6875rem, 0.66rem + 0.08vw, 0.75rem);
  --type-sm: clamp(0.75rem, 0.72rem + 0.10vw, 0.8125rem);
  --type-md: clamp(0.875rem, 0.84rem + 0.12vw, 0.9375rem);
  --type-lg: clamp(1rem, 0.94rem + 0.18vw, 1.125rem);
  --type-xl: clamp(1.2rem, 1.08rem + 0.30vw, 1.44rem);
  --type-2xl: clamp(1.44rem, 1.24rem + 0.50vw, 1.728rem);
  --color-canvas: #0b0d10;
  --color-panel: #14181c;
  --color-panel-raised: #23272c;
  --color-panel-sunken: #101317;
  --color-border: #2d3239;
  --color-border-strong: #454c55;
  --color-text: #e6e2d6;
  --color-text-secondary: #b9b4a8;
  --color-text-muted: #918d84;
  --color-author: #ff8a2a;
  --color-author-soft: #5a321c;
  --color-runtime: #2ec7e6;
  --color-runtime-soft: #173b43;
  --color-certified: #d4af37;
  --color-focus: #f6a45e;
  --space-xs: clamp(0.25rem, 0.22rem + 0.08vw, 0.375rem);
  --space-sm: clamp(0.5rem, 0.46rem + 0.10vw, 0.625rem);
  --space-md: clamp(0.75rem, 0.68rem + 0.16vw, 0.875rem);
  --space-lg: clamp(1rem, 0.90rem + 0.24vw, 1.25rem);
  --space-xl: clamp(1.5rem, 1.32rem + 0.40vw, 1.875rem);
  --space-2xl: clamp(2rem, 1.72rem + 0.65vw, 2.5rem);
  --space-3xl: clamp(3rem, 2.5rem + 1vw, 4rem);
  --duration-fast: 140ms;
  --duration-panel: 200ms;
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1);
}
```

```text
GoSX shell (app/)
       │ reads state / submits typed transactions
       ▼
Workspace + command bus (internal/studio)
       │ revision checks, proposal preview, atomic commit, undo
       ▼
SceneDoc v1
       │ deterministic validation + compilation
       ▼
GoSX typed Scene3D ──► shared SceneIR ──► runtime backends
       │
       └─────────────► native harness ──► frame + exact trace + evidence
```

The compiler supports groups, box/plane/sphere/cylinder and retained indexed meshes,
StandardMaterial records, and ambient/directional/point lights. Since gosx
v0.31.18 mesh and model entities carry non-uniform leaf scale end to end.
Since GoSX v0.54.0, group and prefab-group entities also lower through
`scene.Group.Scale`; nested non-uniform scale and rotation retain their exact
affine parent matrix (including shear) across shared SceneIR, browser
renderers, native preview, BVH/raycast queries, and Studio picking. Light scale
remains a deliberate honesty gate because it has no render meaning; validation,
compilation, prefab overrides, and gizmo commits reject it explicitly.

Transform rotation follows the spec contract: a normalized quaternion is the
authoritative value and the euler field is display metadata. Quaternion
composition mirrors the engine's Rz·Ry·Rx convention, so Studio math, skinning
matrices, and scene lowering agree. Legacy euler-only documents and zero-valued
in-code literals migrate losslessly at JSON decode, and encoding canonicalizes
the identity so fingerprints stay stable across save/reopen. Pose and clip
sampling interpolate rotations by shortest-arc slerp, and retargeting applies
rest-relative quaternion deltas. The document-level camera record still stores
an euler rotation; migrating it to the same contract is tracked work.

The command bus implements `set-field`, `set-transform`, create/delete/reparent/
duplicate, material assignment, rename, and checkpoint-safe indexed-mesh
`extrude-faces`, `inset-faces`, `triangulate-faces`, `weld-vertices`, `fill-face`,
`recalculate-normals`, `project-planar-uv`, `dissolve-edges`, `bridge-loops`,
crack-free stable-edge `loop-cut` traversal across quad rings, and a
single-segment `bevel-edges` floor that replaces a two-face interior edge
with a deterministic offset quad (multi-edge bevels, segments, and vertex
bevels remain partial).
Object, vertex, deterministic derived-edge, and face
selection carry the SceneDoc revision; topology changes either preserve valid
stable IDs or downgrade invalidated sub-object selection to its owning object.
Receipts include normalized operator records with selection, coordinate space,
parameters, result IDs, and checkpoint policy.

The compiled viewport attaches the engine `TransformControls` helper to the
current selection at its composed world position, and the toolbar's
Select/Move/Rotate/Scale buttons drive the shared `studio.viewport.gizmoMode`
signal so the helper switches modes live without a page round-trip. The viewport
camera is a Studio-owned rig over the engine's camera signals: the compiled
props expose `studio.viewport.cameraIn`/`cameraOut`, the header view buttons
switch between the authored perspective camera and orthographic front/top/
right authoring views framed on the current selection, and keyboard
pan/dolly nudges write poses through the same input signal. Since
gosx v0.31.17 gizmo drags are interactive: the engine emits `gizmo-commit`
input events with axis-constrained drag math shared with pure-Go helpers,
translate drags preview live client-side, and the end phase commits exactly
one revision-safe `set-transform` transaction (one undo step per drag,
actor `human://viewport-gizmo`). Scale drags commit per-axis multiplicative
factors on mesh, model, group, and prefab-group entities; light scale stays
rejected. Ring rotation composes a Z-axis quaternion delta onto the
authoritative rotation.

Browser viewport clicks are confirmed by the canonical exact CPU query, not
trusted from the GPU picker alone. The click forwards the GPU-reported world
hit; the server probes that point with a short exact ray, the canonical result
decides the selection, and any GPU/CPU divergence is recorded as a
machine-readable disagreement (id-mismatch, position-gap, or no-cpu-hit) on
the selection state and in the `m1-viewport-exact-selection` certification
check. Clients that cannot report a world hit degrade explicitly to id-only
validation. Since gosx v0.31.16 the pick event carries the live camera's
world-space click ray; the client forwards it and confirmation retraces that
exact ray, so orbited-camera clicks confirm exactly. The authored-camera
probe remains the fallback for clients that report only a world point.

`GET /api/studio/geometry/analysis?target=<stable-id>` performs browser-free,
revision-tagged indexed-mesh inspection. It reports topology counts, bounds,
surface area, closed-mesh volume, boundary and non-manifold edges, degenerate
faces, duplicate positions, isolated vertices, and UV completeness, normalized
bounds, tile overflow, and degenerate UV faces. Authored UVs compile through
typed `BufferGeometry.UVs` into shared SceneIR. Findings refer to stable
SceneDoc sub-object IDs and do not imply that interactive rulers or repair
operators are complete.

`nurbs-curve` geometry stores stable weighted control points, degree, clamped
or non-uniform knot vectors, tessellation budgets, and tube radius in SceneDoc.
The compiler evaluates the rational B-spline and emits deterministic positions,
normals, UVs, and indices through typed `BufferGeometry`; it does not introduce
an editor renderer graph. Control-point selection and mutation are revision-safe
and use the same preview/direct/checkpoint command path. Curve analysis reports
the parameter domain, approximate length, and tessellation counts. Cyclic
curves and loft/revolve/sweep surface tools remain explicitly partial.

Indexed meshes may carry an ordered, stable-ID non-destructive modifier stack.
The current evaluator supports subdivision, mirror, array, and solidify, validates parameters before
commit, leaves authored topology untouched, generates deterministic vertex and
face IDs, and feeds the evaluated geometry into the same typed Scene3D compiler.
Solidify computes averaged authored-surface normals, creates symmetric top and
bottom shells, closes boundary loops with stable side faces, and is available
through both the action API and the human Inspector form.
Subdivision performs bounded multi-level Catmull-Clark evaluation. Interior edge
points are shared by both adjacent faces, face points are emitted once, original
vertices follow interior or boundary rules, and every generated quad has a
stable level/face/corner identity. Non-manifold input and million-element budget
overflow fail explicitly; closed manifold input remains crack-free and
manifold. Levels are editable through the shared agent action and human
Inspector path.
Stable modifier IDs can be moved to an exact stack index. Applying a modifier
bakes the authored geometry through that modifier (including every preceding
stack entry), removes the baked prefix, preserves later modifiers, and records a
geometry-checkpoint undo policy. Reorder and apply are exposed through the same
typed agent descriptors and human Inspector actions; proposals and direct
commits are fingerprint-equivalent.
Set/remove actions support proposal preview, direct commit, journal replay, and
checkpoint undo. The broader modifier library remains partial.

CSG operations accept two valid closed manifold indexed meshes under the same
parent, evaluate their modifier stacks, and perform deterministic bounded voxel
union, intersection, or subtraction. The result is a new SceneDoc entity with
stable grid-derived vertex and face IDs and compiles through typed Scene3D and
shared SceneIR. The one-million-cell budget is an explicit diagnostic gate.
Current CSG requires identity rotation/scale and is voxel-accurate at the
authored resolution; analytic polygon booleans and rotated operands remain
partial.

Material records are authored through `set-material` and `delete-material` on
the same proposal/direct transaction path. PBR ranges are validated before
commit; Selena source is compiled before replacement, so invalid source returns
diagnostics without replacing the last valid material state. Receipts carry
material before/after semantics, deletion rejects live references, and accepted
materials compile through the existing typed Scene3D material transport.

Prefab definitions capture stable-ID entity subtrees in SceneDoc. Instances
remain linked to the definition and compile with namespaced runtime IDs; local
transform, material, and visibility overrides do not mutate the definition.
Capture/update, instantiate, override, and reference-safe delete operations use
the shared command path. Source maps connect runtime IDs back to both the
instance and `prefab/local-entity` record, while incremental fingerprints include
definition content so linked changes cannot reuse stale nodes. Nested prefab instances resolve recursively with fully namespaced runtime
IDs, definition reference cycles (including variant base chains) are rejected
with the concrete path, and variants inherit a resolved base definition and
apply per-entity overrides without adding entities of their own. Incremental
fingerprints hash the resolved definition closure so base or nested edits
invalidate every dependent instance. Variants may also add their own entities parented into the inherited tree,
with deterministic collision and orphan rejection at validation. Unpacking
and portable prefab packages remain partial.

Asset imports inspect source bytes before registration and derive identity from
the complete SHA-256 payload. Direct imports atomically place immutable payloads
under `assets/sha256/` in the active project, then register the asset through the
same revision-safe command bus; proposals register only in the preview document.
Integrity audits and the content handler re-hash stored bytes before trusting or
serving them. SceneDoc model entities reference GLB/glTF asset IDs and lower
through typed `scene.Model` into shared SceneIR, including source-map and
incremental-cache dependencies. Dependency reports enumerate sorted direct
entity references, prefab-definition references, and linked instances.
Registered glTF/GLB assets can decode one triangles primitive at a time into
authored stable-ID indexed-mesh geometry through the revision-safe
`DecodeModelGeometry` request: payload bytes are re-hashed before trust,
float32 position/normal/uv0 and 8/16/32-bit index accessors are supported
with a 100k-triangle budget, and Draco, Meshopt, sparse accessors, skinning,
and morph targets fail explicitly or surface as skipped attributes in the
decode report. Reimport inspects a replacement into a new content identity, previews the exact
transaction, stores immutable bytes only after validation, then atomically
retargets ordinary and prefab-local model records; human Inspector and agent API
calls converge on that command. The current foundation does not claim external
glTF dependency packaging, thumbnails, conversion, optimization, automatic file
watching, garbage collection, or native file-dialog import. The Project panel
is no longer a static mock: it reads the current SceneDoc asset map and
dependency report, displays content identity/name/format/bytes/reference counts,
and submits initial human imports through the same inspected `register-asset`
transaction used by agents. Direct imports validate a proposal and exact
revision before writing the immutable payload, so a stale human or agent request
does not leave an unregistered project artifact. In the packaged desktop host,
File → Import Asset and the Project-panel chooser call the trusted
`gosxDesktop.dialog.openFile` bridge with the supported format filter, populate
that same form, and leave confirmation explicit. Server-only mode retains the
manual trusted-path input and an accessible status explanation.

The M2 articulated proof adds stable armature and bone graphs, normalized
one-to-four bone influences per indexed-mesh vertex, rest and pose transforms,
two-bone IK constraint records, and stable transform clips. `set-bone-pose`,
`set-animation-key`, and deterministic CPU `solve-ik` use the same propose/direct revision-safe transaction path
as human-authored state, including semantic rig/animation receipts and undo.
Exact-time sampling is deterministic and writes articulated part transforms
and bone poses into a cloned SceneDoc. Hierarchical rest and pose matrices,
inverse-bind transforms, and normalized one-to-four bone influences evaluate
linear-blend positions and normals without mutating authored geometry. The
deformed indexed mesh preserves stable IDs, faces, and UVs and compiles through
typed `BufferGeometry` and shared SceneIR; incremental fingerprints include the
referenced armature so pose changes cannot reuse stale nodes. Skin/modifier
ordering currently fails explicitly rather than guessing. This foundation
deliberately does not claim weight-paint UI, dual-quaternion skinning,
retargeting, animation state machines, or physics
interaction yet; those remain partial and are named in certification evidence.

The articulated proof also owns a deterministic fixed-step physics profile.
Stable entity components describe dynamic/static bodies and sphere/plane
colliders; the profile fixes tick rate, gravity, and body membership. The pure-Go
reference loop uses semi-implicit Euler integration, tick-addressed impulses,
and deterministic sphere/plane contact response. Recordings contain sorted
inputs, contact events, and exact initial/final state hashes; replay rejects any
hash divergence. `simulate-ticks` uses the shared propose/direct command path,
semantic simulation receipts, journaling, and undo. This is a certified physics
interaction floor, not a claim of general body pairs, constraints, broad phase,
CCD, fields, particles, audio, or cache authoring.

The Timeline panel is data-driven when an articulated project is open. Its pose,
key insertion, IK, and fixed-tick forms read stable armature/clip/profile IDs and
submit the same operations used by agents. The Chinese Checkers-only document
honestly reports that no articulated clip is loaded. Timeline transport,
scrubbing, dope-sheet/curve editing, and state-machine tooling remain partial.

Retarget maps are stable SceneDoc records connecting one source armature bone to
one target bone. Clip transfer is rest-relative: position deltas use an authored
scale or deterministic source/target rest-length ratio, rotation deltas are
applied to target rest orientation, and generated track IDs are stable. The
`retarget-animation` action, semantic receipt, undo, and Timeline form share the
same implementation.

Animation state machines keep stable states, numeric parameters, transitions,
priority, and runtime state in SceneDoc. Transition eligibility is not encoded
as an application `if/else` chain: six operator rules live in the checked,
embedded `rules/animation-transitions.arb` program and execute through Arbiter.
Studio sorts candidate transitions by priority and stable ID, records the exact
matched rule and evaluated values, then samples the selected clip through the
same deterministic animation evaluator and SceneIR path. Parameter and step
operations are revision-safe and shared by agents and Timeline forms. Crossfade,
masks, blend spaces, layered graphs, root motion, and a graph editor remain
partial.

SceneDoc may retain editable render-resource and pass records, but it does not
own a renderer-only graph. `CompileRenderGraph` validates and deterministically
lowers those records into `studio.RenderGraphIR`, which `studio.SceneIR` carries
beside the embedded canonical GoSX SceneIR under the same `renderGraph` key.
GoSX declared identical types until 2026-07-26 and then removed them as dead:
no GoSX renderer, Go or client-side, ever read them, and Studio was the only
writer. Studio owns the plan now. The authored records, the validation, the
schedule, and the portable JSON are unchanged; only the declaring package moved.
The portable plan contains stable resources, a topologically ordered pass
schedule, and exact transient allocation intervals. Resources declare
`imported`, `persistent`, or `transient` ownership. Missing references,
dependency cycles, and transient reads before their first scheduled write are
rejected. Non-overlapping transient lifetimes share deterministic allocation
slots; overlapping lifetimes never do. The revision-safe `set-render-graph`
operation is shared by human adapters and agents. Backend pass execution,
MRT/reflections, human graph editing, leak telemetry, and profiler visualization
remain honestly partial.

glTF and GLB imports pass through a versioned compatibility control plane before
they become content-addressed asset records. GLB inspection reads and validates
the JSON chunk instead of trusting only its twelve-byte header. Used and
required extensions are canonicalized separately, checked against target rows
for SceneIR, native/headless, WebGPU, and WebGL, and hashed into a stable
compatibility fingerprint. A required extension is compatible only when its
target row is `available`; partial, planned, unknown, and migration-only rows
remain incompatible until their actual conversion or decoder path exists.
Optional gaps produce explicit degradation. The stable corpus manifest gives
each equivalence, migration, degradation, or unsupported-required proof a
domain-addressable ID. Matrix and corpus are exposed at
`/api/studio/gltf/capabilities` and `/api/studio/gltf/corpus`.

External `.gltf` buffers and images are packaged rather than left as ambient
filesystem dependencies. Import resolves only portable relative URIs inside the
source directory, checks the resolved real path to prevent traversal and symlink
escape, and rejects remote URLs. Each unique payload becomes a typed
content-addressed asset. The root JSON is deterministically rewritten to those
immutable Studio content URLs before its own hash is computed. Typed dependency
IDs participate in document validation, dependency reports, integrity audits,
and reference-safe deletion. Data URIs remain embedded and require no package
entry.

Unused-asset collection is a fingerprinted two-phase lifecycle action. The plan
starts from model entities and prefab definitions, walks transitive typed asset
dependencies, and orders unreachable records so all dependents precede shared
payloads. Propose mode returns the exact resulting document without mutation.
Direct mode requires the same document revision and 64-hex plan fingerprint,
then checkpoints the document, reclaims payloads, reports filesystem findings,
and clears ordinary undo/redo history. This non-undoable boundary is explicit in
action discovery and in the human Project panel; it is never presented as an
ordinary reversible delete.

Proposals do not mutate. Direct operations require an exact revision and append
a checksummed, fsynced journal record without rewriting the last explicit save.
Save atomically replaces the canonical SceneDoc. Restart selects the newest
valid journal snapshot, reports recovered/dirty state, skips torn records, and
quarantines corrupt canonical bytes before continuing. Undo/redo use the same
journal path.

A journal record carries operations, not a whole SceneDoc. A direct commit's
Transaction already describes the change, so the record replays it onto the
document the record before it produced. Every 32nd record
(`journalSnapshotInterval`) carries a full Document instead, so a crash-time
recovery never replays more than 31 operations past its anchor. Undo and redo
force a snapshot on their own record: their stored Operations are the
original forward edit, not its inverse, so redo alone would be a valid diff
and undo would not — both take the simpler, uniformly correct path instead. A
record written before this format still carries a Document on every line.
Recovery treats any record with one as a snapshot, so an old journal reopens
unchanged.

The reader still has no fixed line budget. A capped `bufio.Scanner` meant that
one snapshot of a scene inside the documented 100k-triangle import budget
wrote a record nothing could read back, and every later open of that project
failed the same way. Trimming the journal to its newest records is an
optimization, so a trim failure reports through `ProjectStatus.compactionWarning`
and never rejects a command whose record is already durable. A trim never cuts
past the newest snapshot a kept record still needs to replay onto, even when
that keeps more than the nominal bound. It streams one record at a time and
fsyncs the directory after its rename.

Undo history is bounded. Each entry retains the complete document before and
after its transaction, so the stack grows by twice the document size per edit;
`ProjectStatus` reports `undoDepth` and `undoLimit` so a dropped step is
visible rather than silent. Storing inverse operations instead of document
pairs would remove this remaining O(document) cost; it stays open.

Cloning a document is a deep copy, not a JSON round trip. The round trip spent
two thirds of its time parsing text it had just produced: 10.4 ms against
1.5 ms for a 1,000-entity scene. The copier is reflection-driven rather than
hand-written, because a hand-written one would have to be updated for every
field added to the document and a forgotten field aliases silently — the
canonical document and a clone would share a slice, and a later edit would
reach back through history. Two tests hold it: one walks both documents in
parallel and fails on any shared slice, map, or pointer; the other requires the
copy to be structurally identical to the JSON round trip it replaced.
`Transform` is the one special case, because its JSON hooks canonicalized on
the way past, and callers have always seen a canonical transform out of a
clone.

The command bus keeps one copy per commit, not three. A transaction clones the
document once to apply operations against; the outgoing document is retained
directly, because nothing mutates a document in place — operations run on the
clone and a commit rebinds rather than edits. A proposal returns its working
document without a further copy, since it never becomes canonical. The
workspace also carries the fingerprint of its current document, so the
fingerprint one transaction produces is the one the next starts from, and read
paths reuse it instead of re-deriving the identity that keys the compiled-graph
cache. On a 1,000-entity scene that took a direct set-transform from 51.8 ms to
22.4 ms, undo from 59.5 ms to 27.1 ms, and a viewport pick from 2.8 ms to
0.15 ms. `perf-budget.json` gates all three.

The HTTP surface is built by `buildStudioApp`, which takes its configuration as
a value rather than reading the environment inline. That is what lets the tests
construct the real router and exercise the real routes: authority is declared
once in `studioRouteAuthority`, `route_authority_test.go` holds that table
against the source with `go/ast`, and `http_surface_test.go` holds the running
server against the table.

Every Studio form is a GoSX managed form (`data-gosx-form`) and the selection
and gizmo bridges route through the built-in page-navigation morph, so
transactions refresh the panels in place without tearing down the Scene3D
mount or reloading the page; a browser without the navigation runtime
degrades to ordinary posts and reloads.

The GSX viewport spreads the same compiled `scene.Props` consumed by the native
harness. No DOM board or editor-only renderer graph is scene truth.

The certification card does not run the evidence suite on the render path. The
suite renders frames and builds workspaces — about 2.3 seconds on the sample
document — and every edit changes the revision, so running it inline stalled
the editor once per edit. It now runs in the background keyed by revision and
fingerprint, and the card reports `certState`: `current` when the evidence
describes the rendered revision, `recomputing` while a newer run is in flight,
`pending` before any run finishes. Only `current` renders in certification
gold, because evidence that describes an older revision is not trusted state.

Evidence that could not be taken never reads as evidence that passed. Checks
compare document fingerprints to prove a proposal and a direct commit produce
the same document, and a discarded capture error yielded an empty string on
both sides, which compared equal. Captures are now a type whose equality
requires both sides to have succeeded, and whose zero-length function field
makes `==` a compile error, so the unsafe comparison cannot be written. A
failed capture renders as `unhashable: <reason>`. The suite also runs on a
background goroutine, where an unrecovered panic would end the process, so a
panic is recovered and published as a failed evidence state.

Validation also bounds what a document may cost to use, not only whether it is
well formed. A record that validates cheaply and then cannot be evaluated is
worse than one rejected outright, because the cheap commit is journaled and
every later attempt to open the project dies on it. Three bounds close that:

- A modifier stack is capped at eight entries, and the stack's projected
  element count at four million. Array and mirror multiply, so three stacked
  arrays of one thousand turned four vertices into four billion — hundreds of
  gigabytes — while the commit itself cost milliseconds.
- NURBS degree is capped at sixteen, because `nurbsBasis` recurses two children
  per level: degree twenty-two already costs about thirty milliseconds for one
  sample. Path and radial segment counts, and their product, are bounded
  because compilation allocates from them and `GET /api/studio/curve/analysis`
  walks them without a credential.
- The CSG voxel budget counts cells in float64 before converting. The previous
  `int64` product wrapped, so a small enough voxel size admitted a grid of
  about 1e28 cells — evaluated inside the workspace write lock, which is never
  released again.

These are a contract change: a stored document that exceeds a bound stops
validating, and therefore stops opening. That is deliberate. Every such
document was already one compile away from taking the process with it.

A SceneDoc that validates is a SceneDoc that persists. `encoding/json` refuses
NaN and infinities, so a non-finite value surviving validation produced a
document the workspace could not clone, fingerprint, journal, or save. Camera,
environment, and light records reached that state. Every float a document can
carry is now either rejected by validation or safe to marshal, and a
reflection walk over both seed documents holds that implication.

`studio-certify` emits the deterministic M0/M1/M2-foundation evidence envelope.
A valid envelope proves its named checks only; it embeds the broader certification
matrix and keeps `releaseStatus: partial` until the complete public-v1 ledger is
closed. The Chinese Checkers SceneDoc owns portable Selena source records for
Carved Wood, Imperial Jade, Midnight Lacquer, and Moon Porcelain; each compiles
through the WGSL/GLSL artifact transport inspected by the harness and retains
physical fallback metadata. Brushed Steel and the board's rim, chassis, and
sockets remain on the Standard PBR path.
