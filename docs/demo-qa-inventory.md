# Demo QA inventory

This is the release evidence checklist for the public WebMCP challenge build. It separates the claims we make from the controls and states a judge can verify.

## Claims and evidence

| Claim | Visible evidence | Functional evidence |
| --- | --- | --- |
| The browser discovers four typed tools | Four-step collaboration rail and ready status | Native `document.modelContext` registry contains exactly `scene_get_state`, `scene_find_objects`, `scene_focus_object`, and `scene_preview_actions` |
| The tools operate on real scene truth | Persistent receipts report revision, object count, stable ID, and operation count | Tool responses come from the same `/api/studio/document` and proposal command path used by the editor |
| Agent work cannot silently commit | Proposal card says canonical revision is unchanged; Apply and Discard follow the exact diff | Preview returns `applied:false`; no WebMCP commit or reset tool exists; only the CSRF-protected human UI calls commit |
| Review is governed and revision-safe | Arbiter decision, reasons, canonical/approval revision, fingerprint, affected IDs, and expiry are visible | Stale revisions fail; proposals are session-owned, one-shot, expiring, and one-active-per-browser |
| A human decision is unambiguous | Apply and Discard are mutually disabled while either request is pending | Concurrent commit/discard attempts allow only one terminal operation |
| The showcase is reproducible | Clean/dirty baseline gate and human-only Prepare clean demo control; reset returns to a query-free, non-Board selection | Exact prompt cannot be copied while the shared canonical scene differs from the baseline |
| The result is visible | Board selection, Inspector name/material, material swatch, viewport, and activity history converge in place after approval | Managed navigation preserves the Scene3D canvas while reconciling committed canonical state; a separate recovery reload cannot resurrect a stale proposal; activity retains both operation kinds and one matching plan token across proposal/approval |
| The workbench stays interactive | Hierarchy focus and viewport selection move immediately; camera orbit survives selection, material edits, Apply, and Discard | Same-origin links and forms reconcile in place; the live Scene3D canvas keeps object identity while safe SceneIR changes dispatch as commands |
| Player one reads as polished coral | Ten player-one spheres are visibly saturated red-orange with crisp clearcoat highlights, not flat white-pink | The dielectric material uses Standard PBR `#c8321f`, roughness `0.32`, and clearcoat `0.65`; deterministic preview evidence enforces at least 600 deeply saturated coral pixels |
| The viewport uses the modern GPU path | Native Windows Chrome reports `WEBGPU` in the status bar and remains responsive through an orbit drag | Studio prefers WebGPU at native display refresh with a 16.7 ms adaptive interaction budget and an explicit WebGL fallback only for unsupported or lost devices |

## Controls

- Prepare/reset the shared demo; confirm destructive scope.
- Copy the exact showcase prompt only from a clean baseline.
- Apply a staged proposal.
- Discard a staged proposal.
- Navigate the hierarchy with keyboard focus.
- Jump directly to Agent review with the skip link.
- Expand “Why allowed” for the full policy rationale.

## Required state coverage

- WebMCP detecting, ready, unsupported, and tool error.
- Clean and dirty shared-demo baseline.
- No proposal, staged proposal, expired proposal, applied proposal, and discarded proposal.
- Stale-revision recovery after another canonical edit.
- Successful exact sequence: inspect → find → focus → live preview → apply, with no page reload or canvas teardown.
- Separate resilience sequence: stage → forced reload → proposal recovery → discard.
- Dirty same-entity Inspector value survives a background reconciliation; switching entities replaces keyed forms so the old draft cannot target the new object.
- Preview Apply and Discard each preserve camera pose and canvas identity; forced command failure remains visibly disclosed and recovers canonical state.
- Native Windows Chrome reports WebGPU, no fallback reason or device loss, and an interaction-frame p95 at or below the 60 Hz target after warmup; idle is intentionally event-driven rather than continuously redrawn.
- No-op rename, material assignment, transform, and canceling operation batch.
- Double-action attempt: Apply/Discard and Discard/Apply.

## Viewports and accessibility

- 1024×768 in-app/split-view target: no horizontal overflow; Agent review fully visible.
- 1280×800 laptop target: complete editor hierarchy remains legible.
- 1440×900 recording target: primary demo frame and proposal evidence remain above the fold.
- Keyboard-only traversal reaches reset, prompt, hierarchy, proposal evidence, policy disclosure, Apply, and Discard.
- Flow steps announce their state in text alternatives, and dynamic status/trace regions use live-region semantics.
- Reduced-motion mode has no essential information encoded only in animation.

## Evidence to retain

- Clean baseline screenshot at each target viewport.
- Staged proposal screenshot showing exact diffs and unchanged canonical revision.
- Applied screenshot showing Launch Board, Brushed Steel, matching steel swatch, and applied activity.
- Native WebMCP registry/tool-call transcript.
- Console/network error check.
- Unit, race, smoke, build, and public-health command output recorded in the release notes.
