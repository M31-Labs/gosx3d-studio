# M0 implementation handoff

## What exists

- A conventional GoSX file-routed application in `app/`.
- Industrial Void visual tokens and a dense desktop workbench shell.
- A typed scaffold manifest in `internal/studio`.
- SceneDoc v1 validation, deterministic fingerprints, typed Scene3D compilation,
  revision-safe proposal/direct transactions, and undo.
- Health, manifest, SceneDoc, and compiled SceneIR JSON endpoints.
- A pure-Go smoke harness with visible-frame and exact-ray gates.
- Original approved visual references in `design/`.
- A sibling-module development link to `../gosx`.

## What does not exist yet

The Hierarchy, Inspector, timeline, telemetry, and displayed proposal remain
static shell data. The canonical SceneDoc is mounted into the Scene3D viewport,
but those panels are not yet generated from descriptor/workspace state. Exact
interactive selection enrichment, semantic proposal diffs, strict Studio
certification, and desktop packaging remain unimplemented.

## First vertical slice

1. Bind browser selection to the canonical exact CPU query and surface disagreements.
2. Generate Hierarchy and Inspector UI from live document and descriptors.
3. Add semantic diffs, inverse IDs, telemetry correlation, and action JSON Schemas.
4. Integrate editor-facing `scene/cert` dimensions and Selena evidence.
5. Launch, package, and recovery-test the same app through `gosx desktop` on Windows.

## Non-negotiable invariants

- Studio, runtime, native harness, and exported applications share SceneIR
  semantics.
- Unavailable capabilities remain explicit and machine-readable.
- Agent actions use the same command path as human UI operations.
- Orange is authored state, cyan is observed runtime state, and gold is trusted
  certification state.
- The application must not depend on Chrome or browser GPU access for scene
  evidence and certification.
