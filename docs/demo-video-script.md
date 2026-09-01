# WebMCP Challenge demo video script

Target runtime: **2:38**. Keep the final public YouTube upload below three
minutes. Record the Studio and its WebMCP-capable browser agent in one readable
frame. Bracketed revision values such as `[R]` are production cues, not words
to read literally.

This demo makes one precise collaboration claim: a person and a browser agent
work against the same canonical SceneDoc through a shared, revision-safe
command path. The deployed demo is a **single-instance shared ephemeral
workspace**, not a realtime multiplayer room system.

For the operator's compact timeline and recording settings, keep
[the demo shot list](demo-shot-list.md) open on a second device.

## Before recording

- Use the exact hosted build and browser that will appear in the video. Wait
  until **Agent Collaboration** reads **Agent tools ready** and **4 tools**.
- Confirm no one else is testing the shared deployment. A reset or edit affects
  the one canonical demo workspace for every visitor.
- Keep the hierarchy, viewport revision, four-step WebMCP rail, proposal card,
  and Agent Activity readable at the same browser zoom.
- Turn on Do Not Disturb. Hide bookmarks, credentials, account menus,
  developer tools, unrelated tabs, and the recording controls.
- Record a ten-second audio and screen test before the real take. Check that
  the cursor is visible, text survives export, and speech peaks around
  `-12 dB` to `-6 dB` without clipping.
- Leave two seconds of stillness at the beginning and end. Dead waiting time
  may be trimmed, but never splice together different browser sessions or
  scene revisions.

## Exact agent prompt

Use **Copy demo prompt** in the Studio, then paste the copied text unchanged:

> Inspect the current scene, find and focus the object named Board, then
> stage—without committing—a proposal that renames it Launch Board and
> assigns the Cobalt Pieces material. Explain the revision boundary.

The expected tool sequence is `scene_get_state`, `scene_find_objects`,
`scene_focus_object`, then `scene_preview_actions`. There is deliberately no
agent-callable commit tool.

## Four-step run of show

### 1. 0:00-0:28 - Reset and establish the boundary

**On screen:** Begin on the full Studio. Click **Reset shared scene**, accept
the confirmation, and wait for the refreshed clean sample. Point to `Board`
with stable ID `board`, then record the new baseline revision as `[R]`. Do not
compare it with the pre-reset revision; reset revisions are monotonic.

**Voiceover:**

> A 3D editor is rich for a person, but ambiguous for an agent. I will reset
> this public demo to a clean sample and record its new baseline, revision
> [R]. To be precise, this deployment is one shared ephemeral server
> workspace, not a realtime multiplayer room system. The collaboration shown
> is between this person and this browser agent over one canonical SceneDoc.

### 2. 0:28-1:08 - Inspect, find, focus, and stage

**On screen:** Point to **Agent tools ready**, **4 tools**, and the
**Inspect -> Find -> Focus -> Stage** rail. Click **Copy demo prompt**, paste
the exact prompt into the browser agent, and send it. As the agent works, let
the rail show all four steps completing. Hold briefly on the focused `board`
row and converged Inspector selection before the proposal replaces the idle
panel.

**Voiceover:**

> This page declares four WebMCP tools: inspect, find, focus, and stage. There
> is no agent-callable commit. I will paste the Studio's exact demo prompt. The
> agent reads canonical revision [R], resolves Board to stable ID `board`,
> focuses that same object in the visible Studio, then stages two bounded
> operations: rename it Launch Board and assign Cobalt Pieces. The four-step
> rail shows each tool completing.

### 3. 1:08-2:05 - Review the proof, then reload it

**On screen:** In **Latest staged proposal**, point deliberately to:

1. `Board -> Launch Board`;
2. `board-material -> player-4-material`, the stable material ID the agent
   resolved from `Cobalt Pieces`;
3. `agent://webmcp`;
4. **Arbiter - Allow - 2/2** and its evidence tooltip;
5. the affected stable ID and result fingerprint; and
6. **canonical [R] unchanged - approval [R+1]**.

Point back to the viewport's still-unchanged revision `[R]`. Then perform a
full browser reload in the same tab. Wait for tool registration and proposal
hydration; show that the same proposal and human review controls return while
the viewport remains at `[R]`.

**Voiceover:**

> The scene has not changed. The card shows both semantic diffs, affected
> object, agent attribution, deterministic result fingerprint, and Arbiter
> Allow evidence for two of two operations. Most importantly, canonical
> revision [R] is unchanged; approval would create [R+1]. The server keeps the
> exact reviewed transaction behind an opaque, session-owned proposal ID. I
> will prove that by reloading the whole page. The same proposal and review
> controls return in this browser session, while the scene remains at [R].

### 4. 2:05-2:38 - Human apply, one revision, and why WebMCP fits

**On screen:** Move the cursor from the agent into the Studio and click
**Apply staged changes** yourself. After the managed refresh, show all of the
following in one deliberate sweep:

- `Launch Board` in the hierarchy or Inspector;
- `Cobalt Pieces` as the applied material;
- canonical revision `[R+1]`, exactly one above the baseline; and
- the `agent://webmcp` preview and `human://webmcp-review` apply entries in
  **Agent Activity**.

End on the Studio with the viewport and collaboration surface together.

**Voiceover:**

> Now I make the only commit by clicking Apply staged changes myself. The
> server applies the exact reviewed operations through the same revision-safe
> command path used by the human editor. Board becomes Launch Board, its
> material becomes Cobalt Pieces, and the canonical revision advances exactly
> once, from [R] to [R+1]. Agent Activity separates the agent preview from the
> human approval. That is why WebMCP fits: the site exposes domain actions
> instead of making agents guess through pixels, while the person keeps the
> final decision.

## Recovery rules

- If WebMCP discovery does not reach **Agent tools ready** and **4 tools**, stop
  the take, fix the compatible browser, and start again. Do not narrate an
  unverified tool surface.
- If the agent skips a tool, targets anything other than stable ID `board`, or
  stages different operations, stop and restart with the exact prompt.
- If another visitor changes or resets the scene, stop. Reset again, record a
  new `[R]`, and restart the take; do not claim realtime multiplayer behavior.
- If the proposal does not return after the full reload, stop. Do not claim
  reload persistence from a different tab or browser session.
- If Apply reports a revision conflict, the safety boundary worked. For the
  main demo, stop, reset, and record a clean take rather than editing around
  the conflict.
- Do not use Undo to repair a take. Undo creates another canonical revision and
  makes the promised `[R] -> [R+1]` proof harder to read.

## Final content checklist

- [ ] The spoken edit is between 2:30 and 2:45; the upload is below 3:00.
- [ ] The public video and hosted URL both work while signed out.
- [ ] The video visibly identifies the demo as one shared ephemeral workspace,
      not realtime multiplayer.
- [ ] **Agent tools ready**, **4 tools**, and all four completed flow steps are
      visible.
- [ ] The exact prompt, stable ID `board`, and baseline `[R]` are shown.
- [ ] Both staged operations are visible: `Board -> Launch Board` and
      `board-material -> player-4-material`; the agent output identifies the
      latter as `Cobalt Pieces`.
- [ ] Arbiter Allow evidence, the fingerprint, and
      **canonical [R] unchanged - approval [R+1]** are readable.
- [ ] A full same-session reload restores the proposal before approval.
- [ ] A visible human click applies the proposal, and canonical state advances
      exactly once to `[R+1]`.
- [ ] Agent Activity distinguishes `agent://webmcp` from
      `human://webmcp-review`.
- [ ] No credentials, notifications, unsupported capability claims, or private
      information appear.
