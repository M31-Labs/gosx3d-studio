# GoSX 3D Studio agent contract

## Scope

This repository owns the standalone digital-scene, animation, game-development,
simulation, and asset workbench. It does not own the GoSX website editor or
Kiln's manufacturing/CAD product model.

## Architecture boundaries

- `app/` is the GoSX presentation shell. It must not become scene truth.
- `internal/studio/` owns SceneDoc, command, workspace, compilation, and
  capability contracts until package extraction is justified.
- SceneDoc compiles through GoSX typed Scene3D and shared SceneIR. Do not create
  an editor-only renderer graph.
- Human UI and agent operations must converge on the same revision-safe command
  path.
- Browser-free harness evidence is mandatory for scene behavior; browser or GPU
  screenshots are supplemental.
- Generated `build/` and `dist/` output is never edited by hand.

## Honesty gate

Do not mark a surface available because markup, a field, or a screenshot exists.
Capabilities remain `planned` until authoring, runtime, headless evidence, tests,
diagnostics, and documentation are complete or explicitly not applicable.

## Required verification

```bash
gosx check app/page.gsx
go vet ./...
go test ./...
node --test scripts/studio-webmcp-preview.test.js
go test -race ./internal/... ./app/... .
go run ./cmd/studio-smoke
gosx build .
```

Do not add a `replace` directive to `go.mod` for sibling development. `go.mod`
pins released versions and `go.sum` checksums them; a replace switches version
resolution off for everyone and hides upstream removals until the build breaks.
Use `go.work` (see `go.work.example`), which stays in your working tree.

Preserve the Industrial Void semantics: orange is authored state, cyan is
observed runtime state, and gold is trusted certification state.
