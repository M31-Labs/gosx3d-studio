# Current implementation handoff

## What exists

- A standalone GoSX v0.54 application with a dense desktop workbench shell,
  typed `SceneDoc`, shared `SceneIR`, and Scene3D rendering.
- Live hierarchy selection, transform and material editing, modeling operators,
  timeline controls, diagnostics, and command receipts backed by canonical
  workspace state rather than display-only fixtures.
- Revision-safe preview and direct transactions, undo/redo, explicit atomic
  saves, checksummed journals, recovery, and asset cleanup planning.
- Deterministic browser-free frame, exact-pick, simulation, rigging, animation,
  export, and certification evidence.
- Four browser-native WebMCP tools for scene inspection, search, visible focus,
  and bounded edit proposals. Every proposal is evaluated by Arbiter and stays
  non-mutating until a person applies the exact staged transaction.
- A pinned production build, Render Blueprint, Linux evidence workflow, and a
  Windows workflow that stages an offline bundle and unsigned MSIX package.

## Honest limits

- The hosted demonstration uses one process and process-local scene state. It
  is not yet an account-backed, durable, realtime multiplayer service.
- WebMCP intentionally cannot delete, reparent, switch projects, clean assets,
  commit a proposal, undo, or redo. Those operations remain human-only until
  their review and authority contracts are equally strong.
- Native Windows window, bridge, picker, recovery, install, and update behavior
  still needs verification on a real Windows host. Signing identity is pending.
- Certification remains `partial`; the capability matrix is the authority for
  what is available, partial, planned, or unsupported.

## Next product slices

1. Deploy and verify the HTTPS judge instance in an external compatible
   browser, including ChatGPT's in-app browser when available.
2. Add durable private workspaces, identity, backups, and revision history
   without moving scene truth into the presentation layer.
3. Add presence, soft claims, and conflict-free collaboration over the stable
   scene IDs and revision metadata already in `SceneDoc`.
4. Complete native Windows runtime evidence, signing, installed launch,
   recovery, and update verification.
5. Expand the agent action surface only through the shared command path, with
   semantic previews, explicit authority, deterministic receipts, and tests.

## Substrate work still open

- Undo retains a document pair per entry, so its stack costs O(document) per
  edit. Inverse operations or copy-on-write entities would reduce that cost.
- `Document.Fingerprint` re-marshals the full document. Per-entity hashes with
  a Merkle root would make fingerprints and dependent caches incremental.
- `Document.Clone` now uses a reflection deep copy instead of a JSON round trip,
  but direct commits still clone twice to preserve the current return contract.

## Non-negotiable invariants

- Studio, runtime, native harness, and exported applications share SceneIR
  semantics.
- Unavailable capabilities remain explicit and machine-readable.
- Human UI and agent operations converge on the same revision-safe command
  path.
- Orange is authored state, cyan is observed runtime state, and gold is trusted
  certification state.
- Scene evidence and certification must not depend on Chrome or browser GPU
  access.
