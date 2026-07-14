# Architecture map

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

The initial compiler supports groups, box/plane/sphere/cylinder meshes,
StandardMaterial records, and ambient/directional/point lights. Non-unit scale
fails explicitly because the current typed Scene3D mesh/group contract does not
carry scale. This is a deliberate honesty gate, not an implicit omission.

The command bus currently implements `set-transform`, `assign-material`, and
`rename-entity` in proposal and direct modes. Proposals return a complete
preview without mutating the workspace. Direct transactions require an exact
revision, commit atomically, produce deterministic fingerprints, and can be
undone while preserving monotonic revisions.
