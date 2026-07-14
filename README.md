# GoSX 3D Studio

Standalone GoSX application foundation for the animation, game,
simulation, and Scene3D asset workbench.

The current M0 slice mounts a Studio-owned Chinese Checkers SceneDoc through
typed Scene3D, supports exact viewport selection and revision-safe human/agent
commands with undo, redo, checkpoint-safe extrusion, explicit atomic saves, and
crash-journal recovery, and proves rendering and exact picking in the
browser-free harness. Remaining component inspectors, full certification, and
desktop packaging remain open.

## Run locally

```bash
cp .env.example .env
go run .
```

Open <http://localhost:8080>. For hot reload:

```bash
gosx dev .
```

On Windows, the application executable launches the GoSX Desktop WebView2 host
by default. For desktop development with the native dialog/clipboard bridge:

```bash
gosx desktop --native-bridge --app-id m31labs.gosx3d-studio dev .
```

Set `STUDIO_SERVER_ONLY=1` only for a Windows server probe or deployment that
must not create a native window.

The module uses the sibling `../gosx` checkout through a development `replace`
directive. Remove or change that directive when the application moves to a
released GoSX version.

## Current surfaces

- Server-rendered Industrial Void editor shell.
- Project, Hierarchy, mounted viewport, Inspector, timeline, telemetry, and agent
  action regions.
- `GET /api/health` for process health.
- `GET /api/studio/platform` for machine-readable host capability diagnostics.
- `GET /api/studio/manifest` for the agent-readable scaffold and capability
  contract.
- `GET /api/studio/document` for the current canonical SceneDoc snapshot.
- `GET /api/studio/scene-ir` for the shared compiled SceneIR snapshot.
- `GET /api/studio/rig/skin?target=<stable-id>` for revision-tagged,
  browser-free deformed geometry and influence telemetry.
- `GET /api/studio/actions`, `/descriptors`, and `/certification` for discovery.
- `GET /api/studio/project/status` plus authenticated
  `POST /api/studio/project/save` for explicit-save and recovery state.
- Scene3D viewport clicks and Hierarchy links converge on canonical workspace
  selection; the Transform Inspector converges on the same transaction path as
  agents.
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
gosx check app/page.gsx
go test ./...
go vet ./...
go run ./cmd/studio-smoke
go run ./cmd/studio-certify
gosx build .
```

See [docs/handoff.md](docs/handoff.md) for the next implementation slice and
[docs/design-spec.md](docs/design-spec.md) for the binding visual system.
Desktop truth is tracked in
[docs/platform-capabilities.md](docs/platform-capabilities.md).

## Product boundary

GoSX 3D Studio is a digital scene, animation, game-development, simulation, and
asset workbench. It is separate from the GoSX website editor and from Kiln's
real-world CAD/manufacturing scope.
