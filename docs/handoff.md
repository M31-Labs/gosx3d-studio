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

1. ~~Bind browser selection to the canonical exact CPU query and surface
   disagreements.~~ Done: viewport clicks confirm through `ExactPick`, the
   canonical result wins, and disagreements are machine-readable on selection
   state and in certification (`m1-viewport-exact-selection`).
2. Generate Hierarchy and Inspector UI from live document and descriptors.
3. Add semantic diffs, inverse IDs, telemetry correlation, and action JSON Schemas.
4. ~~Integrate editor-facing `scene/cert` dimensions and Selena evidence.~~
   Superseded: GoSX removed `scene/cert` as dead code on 2026-07-26. Studio now
   records the linked framework module and version and does not republish a
   framework matrix it cannot re-derive. See
   [[decisions/0002-dependency-model-pinned-versions-not-replace]] in the
   `gosx3d-studio` Hyphae space.
5. Launch, package, and recovery-test the same app through `gosx desktop` on Windows.

## Substrate work still open

Named here so it is not rediscovered. None of it blocks the slices above.

- The journal writes a whole SceneDoc per record, and undo retains a document
  pair per entry. Operations with periodic snapshots, and inverse operations in
  undo, remove the remaining O(document) cost.
- `Document.Fingerprint` re-marshals the whole document. Per-entity content
  hashes with a Merkle root would make it incremental and fix every caller,
  including the compiled-graph and certification caches, at once.
- The certification card reports `recomputing` until the next render. There is
  no push to refresh it when the background run finishes.

## Non-negotiable invariants

- Studio, runtime, native harness, and exported applications share SceneIR
  semantics.
- Unavailable capabilities remain explicit and machine-readable.
- Agent actions use the same command path as human UI operations.
- Orange is authored state, cyan is observed runtime state, and gold is trusted
  certification state.
- The application must not depend on Chrome or browser GPU access for scene
  evidence and certification.
