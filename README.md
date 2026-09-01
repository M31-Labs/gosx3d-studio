# GoSX 3D Studio

Standalone GoSX application foundation for the animation, game,
simulation, and Scene3D asset workbench.

The current build mounts a Studio-owned Chinese Checkers SceneDoc through typed
Scene3D, supports exact viewport selection and revision-safe human/agent
commands with undo, redo, checkpoint-safe modeling, explicit atomic saves, and
crash-journal recovery, and proves rendering and exact picking in the
browser-free harness. The Diagnostics surface reports certification honestly;
unfinished dimensions and desktop packaging remain explicit rather than being
presented as complete.

## WebMCP collaboration

The Studio now exposes its existing human/agent scene contract directly to a
compatible browser. An agent can inspect canonical scene state, search stable
object IDs, focus the visible hierarchy, and stage a bounded edit preview. The
preview appears in the Studio with its rationale, revision, affected objects,
semantic changes, deterministic fingerprint, and Arbiter Allow/Deny evidence.
It does not change the scene until a person explicitly applies it.

Four tools are registered in `public/studio-webmcp.js`:

- `scene_get_state`
- `scene_find_objects`
- `scene_focus_object`
- `scene_preview_actions`

The implementation uses the standard imperative shape required by the WebMCP
Challenge:

```js
document.modelContext.registerTool({
  name: "scene_preview_actions",
  description: "Validate and visibly stage reversible scene actions for human review.",
  inputSchema: { /* exact revision, title, rationale, and bounded operations */ },
  execute: async (input) => { /* stage a real Studio proposal; never commit */ }
});
```

Agent proposals and human commits converge on the same revision-safe
`Workspace.Execute` path that predates the browser adapter. The server retains
the exact previewed transaction behind an opaque proposal ID, so the approval
step cannot silently substitute different operations. An embedded Arbiter
strategy evaluates every proposed operation before staging and returns an
inspectable decision trace. See
[docs/webmcp-challenge.md](docs/webmcp-challenge.md) for the architecture,
baseline/new-work split, safety boundaries, examples, and compatible-browser
test script.

## Run locally

```bash
cp .env.example .env
go run .
```

Open <http://localhost:8080>. For hot reload:

```bash
gosx dev .
```

To exercise the WebMCP tools, use ChatGPT's in-app browser or Chrome 149+ with
`chrome://flags/#enable-webmcp-testing` enabled. The Agent Collaboration panel
reports whether the browser registered all four tools.

The full local workflow is verified against Chrome for Testing 152.0.7977.64
with its native WebMCP testing experiment enabled. ChatGPT's in-app browser and
the eventual public deployment remain separate pre-submission checks. See
[Native WebMCP verification](docs/native-webmcp-qa.md) for the exact evidence
boundary and exercised workflow.

For a disposable shared judge/demo workspace with a visible, human-only reset:

```bash
STUDIO_DEMO_MODE=1 go run .
```

Demo reset is session/CSRF protected, invalidates pending proposals, and moves
the shared process to a fresh sample scene at a newer revision. It is
intentionally not a WebMCP tool.

On Windows, the application executable launches the GoSX Desktop WebView2 host
by default. For desktop development with the native dialog/clipboard bridge:

```bash
gosx desktop --native-bridge --app-id m31labs.gosx3d-studio dev .
```

Set `STUDIO_SERVER_ONLY=1` only for a Windows server probe or deployment that
must not create a native window.

## Deploy on Render

The root `render.yaml` defines a single-instance native Go web service with an
HTTPS endpoint and `/api/health` readiness check. It runs the pinned GoSX
production packager so the server and hashed Scene3D runtime ship together.
The tracked build script uses TinyGo 0.41.1 when already installed, otherwise
downloads the official Linux release and verifies its published SHA-256 digest:

```bash
./scripts/render-build.sh
./dist/run.sh
```

The Blueprint generates private `SESSION_SECRET` and `STUDIO_ACTION_TOKEN`
values, sets `GOSX_ENV=production` and `STUDIO_DEMO_MODE=1`, and uses Render's
paid `0.5c-512mb` plan to avoid free-tier spin-down during judging. Keep the
service at one instance while the Studio uses process-local collaborative
state. The root route is explicitly dynamic and `no-store`; it is never
prerendered with a session-bound CSRF token.

## Dependencies

`go.mod` pins GoSX `v0.54.0` and Arbiter `v1.9.0`, and `go.sum` checksums the
complete dependency graph. The Studio adopts GoSX v0.54's affine group scale
through SceneDoc compilation, nested prefab lowering, exact picking, preview
evidence, and gizmo commits; non-unit light scale remains rejected because it
has no render meaning. CI, releases, and fresh clones all build those pinned
versions rather than an ambient local checkout.

To edit GoSX or Arbiter next to the Studio, use a workspace:

```bash
cp go.work.example go.work
```

`go.work` is gitignored. It overrides the pinned versions for your working tree
only, so local sibling edits never decide what anyone else builds. Delete it to
return to the pinned versions.

## Current surfaces

- Server-rendered Industrial Void editor shell.
- Project, Hierarchy, mounted viewport, Inspector, timeline, telemetry, and agent
  action regions.
- `GET /api/health` for process health.
- `GET /api/studio/platform` for machine-readable host capability diagnostics.
- `GET /api/studio/manifest` for the agent-readable scaffold and capability
  contract.
- `GET /api/studio/document` for the current canonical SceneDoc snapshot.
- Session- and CSRF-protected `POST /api/studio/webmcp/proposals` for bounded,
  non-mutating WebMCP previews and `POST /api/studio/webmcp/commits` for the
  separate visible human-approval step.
- `GET /api/studio/demo/status` and session/CSRF-protected
  `POST /api/studio/demo/reset` for an opt-in shared, ephemeral judge demo.
- `GET /api/studio/scene-ir` for the shared compiled SceneIR snapshot.
- `GET /api/studio/rig/skin?target=<stable-id>` for revision-tagged,
  browser-free deformed geometry and influence telemetry.
- `GET /api/studio/export/scene3d` and `/export/scene-ir` for
  byte-deterministic exports with machine-readable semantic-loss reports.
- `GET /api/studio/initialize` for the spec §13.1 handshake: protocol identity,
  document identity, authority modes, and the action-surface summary.
- `GET /api/studio/actions`, `/descriptors`, and `/certification` for
  discovery; every mutating action descriptor carries input and output JSON
  Schemas, and the read surface is enumerated as read-authority descriptors.
- Human editing forms for transforms, material assignment and PBR/Selena
  records, sub-object selection, seven deterministic mesh operators, undo and
  redo — all converging on the same revision-safe transactions agents use.
- A live Agent Collaboration panel (WebMCP readiness, semantic proposal diff,
  explicit human approval, session attribution counts) and a command-history
  console fed by real receipts.
- The certification card renders the live deterministic evidence run, cached
  per document revision.
- `GET /api/studio/project/status` plus authenticated
  `POST /api/studio/project/save` for explicit-save and recovery state.
- Scene3D viewport clicks and Hierarchy links converge on canonical workspace
  selection; viewport clicks are confirmed against the exact CPU ray query,
  the canonical result wins disagreements, and GPU/CPU divergences surface as
  machine-readable diagnostics. The Transform Inspector converges on the same
  transaction path as agents.
- Authenticated `POST /api/studio/actions/preview`, `/transactions/call`,
  `/undo`, and `/redo` paths using `STUDIO_ACTION_TOKEN`.
- Preview/direct command contracts with deterministic fingerprints, revision
  conflicts, undo/redo, checksummed append-only journals, corrupt-save
  quarantine, and explicit atomic `.scene3d` saves.
- Browser-free PNG and exact ray evidence through `cmd/studio-smoke`.
- Deterministic 60 Hz physics recording/replay, contact evidence, exact state
  hashes, and shared `simulate-ticks` transactions in the articulated proof.
- Stable rest-relative animation retargeting and Arbiter-compiled state-machine
  transitions with decision traces, shared actions, undo, and human Timeline
  controls.
- Byte-deterministic combined M0/M1/M2-foundation certification through `cmd/studio-certify`,
  including SceneDoc/source-map, incremental equivalence, frame, exact-pick,
  Selena WGSL/GLSL artifacts, topology/assets/prefabs, and deterministic
  articulated rig/animation action checks. Its `releaseStatus` remains `partial`.
- Explicit partial, planned, and uncertified states for unfinished capability.

## Verify

```bash
go run m31labs.dev/gosx/cmd/gosx@v0.54.0 check app/page.gsx
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 fmt internal/studio/rules/webmcp-operations.arb --check
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 check internal/studio/rules/webmcp-operations.arb --strict
go vet ./...
go test ./...
go test -race ./internal/... ./app/... .
go run ./cmd/studio-smoke
go run ./cmd/studio-certify
./scripts/render-build.sh
```

The command bus, the workspace caches, and the background evidence run all
share state across goroutines, so the race detector is part of the floor. The
`-race` run names its packages because `./...` picks up generated `dist/`
copies once a bundle has been built.

See [docs/handoff.md](docs/handoff.md) for the next implementation slice and
[docs/design-spec.md](docs/design-spec.md) for the binding visual system.
Desktop truth is tracked in
[docs/platform-capabilities.md](docs/platform-capabilities.md).

## Product boundary

GoSX 3D Studio is a digital scene, animation, game-development, simulation, and
asset workbench. It is separate from the GoSX website editor and from Kiln's
real-world CAD/manufacturing scope.

## License

GoSX 3D Studio is available under the [MIT License](LICENSE).
