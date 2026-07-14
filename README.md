# GoSX 3D Studio

Initial standalone GoSX application scaffold for the animation, game,
simulation, and Scene3D asset workbench.

The current slice establishes the Industrial Void editor shell, file-routed
GoSX application, validated SceneDoc core, revision-safe command workspace,
shared Scene3D compilation, native evidence harness, capability manifest, and
the viewport seam where the compiled scene will mount next.

## Run locally

```bash
cp .env.example .env
go run .
```

Open <http://localhost:8080>. For hot reload:

```bash
gosx dev .
```

The module uses the sibling `../gosx` checkout through a development `replace`
directive. Remove or change that directive when the application moves to a
released GoSX version.

## Current surfaces

- Server-rendered Industrial Void editor shell.
- Project, Hierarchy, viewport seam, Inspector, timeline, telemetry, and agent
  action regions.
- `GET /api/health` for process health.
- `GET /api/studio/manifest` for the agent-readable scaffold and capability
  contract.
- `GET /api/studio/document` for the current canonical SceneDoc snapshot.
- `GET /api/studio/scene-ir` for the shared compiled SceneIR snapshot.
- Proposal/direct command contracts with atomic validation, deterministic
  fingerprints, revision conflicts, and undo.
- Browser-free PNG and exact ray evidence through `cmd/studio-smoke`.
- Explicit disabled and uncertified states for functionality that has not been
  wired yet.

## Verify

```bash
gosx check app/page.gsx
go test ./...
go vet ./...
go run ./cmd/studio-smoke
```

See [docs/handoff.md](docs/handoff.md) for the next implementation slice and
[docs/design-spec.md](docs/design-spec.md) for the binding visual system.

## Product boundary

GoSX 3D Studio is a digital scene, animation, game-development, simulation, and
asset workbench. It is separate from the GoSX website editor and from Kiln's
real-world CAD/manufacturing scope.
