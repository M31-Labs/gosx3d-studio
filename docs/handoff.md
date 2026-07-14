# Initial scaffold handoff

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

The DOM board, timeline, telemetry, and displayed proposal are labeled scaffold
data. A canonical sample SceneDoc now exists behind read-only APIs and compiles
to shared SceneIR, but it is not mounted into the DOM viewport. Persistence,
create/delete/reparent/duplicate commands, topology operations, interactive
selection, and desktop packaging remain unimplemented here.

## First vertical slice

1. Replace the compact bootstrap document with full Chinese Checkers conversion
   while preserving stable IDs and deterministic fingerprints.
2. Mount its compiled Scene3D props in the existing viewport region.
3. Connect viewport selection to the same exact CPU query used by the harness.
4. Expose the existing proposal/direct transaction path through authenticated
   typed server and desktop actions.
5. Add create/delete/reparent/duplicate commands and durable journal storage.
6. Save, reopen, replay, and recover one material/transform transaction.
7. Implement one checkpoint-safe topology operation (`ExtrudeFaces`).
8. Launch the same application through `gosx desktop` without forking app code.

## Non-negotiable invariants

- Studio, runtime, native harness, and exported applications share SceneIR
  semantics.
- Unavailable capabilities remain explicit and machine-readable.
- Agent actions use the same command path as human UI operations.
- Orange is authored state, cyan is observed runtime state, and gold is trusted
  certification state.
- The application must not depend on Chrome or browser GPU access for scene
  evidence and certification.
