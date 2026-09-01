# WebMCP Challenge demo video script

Target runtime: **2:16**. Keep the final public YouTube upload below three
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
- Reset the shared scene before capture, write down the resulting baseline
  revision `[R]`, and confirm no proposal is staged. Do not record the reset.
- Leave `Board` unselected. The agent must find it through WebMCP and create the
  first visible focus on camera.
- Confirm no one else is testing the shared deployment. A reset or edit affects
  the one canonical demo workspace for every visitor.
- Keep the hierarchy, viewport revision, four-step WebMCP rail, persistent
  typed-call trace, proposal card, and Agent Activity readable at the same
  browser zoom.
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

### 1. 0:00-0:22 - Establish the boundary

**On screen:** Begin on the clean Studio prepared before capture, with a
non-Board selection.
Show canonical revision `[R]`, **Agent tools ready**, **4 tools**, and the
shared ephemeral demo note. Hold on the authority cue that the agent may stage
while only a person may apply. Do not point to or select `Board`.

**Voiceover:**

> GoSX 3D Studio turns this 148-object scene into a shared workspace for a
> person and a browser agent. The page registers four live WebMCP tools:
> inspect, find, focus, and stage. The agent can prepare work, but it has no
> commit tool. We begin at canonical revision [R], with a clean scene and no
> staged proposal.

### 2. 0:22-0:58 - Inspect, find, focus, and stage

**On screen:** Click **Copy demo prompt**, paste the exact prompt into the
browser agent, and send it. Let the **Inspect -> Find -> Focus -> Stage** rail
and persistent typed-call trace fill from real tool results. Hold briefly when
the agent finds stable ID `board` and the hierarchy plus Inspector converge on
that object.

**Voiceover:**

> I'll paste one creative brief. The agent reads revision [R]; Board is not
> selected for it. It searches the canonical SceneDoc, resolves Board to stable
> ID `board`, and asks the visible Studio to focus it. Then it stages a rename
> to Launch Board and a material change to Cobalt Pieces. The typed-call trace
> records each result.

### 3. 0:58-1:42 - Review the proof, then reload it

**On screen:** In **Latest staged proposal**, point deliberately to:

1. `Board -> Launch Board`;
2. `board-material -> player-4-material`, the stable material ID the agent
   resolved from `Cobalt Pieces`;
3. `agent://webmcp`;
4. **Arbiter - Allow - 2/2** and its evidence tooltip;
5. the affected stable ID and result fingerprint; and
6. **canonical [R] unchanged - approval [R+1]**.

Point back to the viewport's still-unchanged revision `[R]`. Then perform a
full browser reload in the same tab. Show that the typed-call trace, proposal,
and human review controls return while the viewport remains at `[R]`.

**Voiceover:**

> Nothing canonical has changed. The review card shows human-readable
> before-and-after values, `board`, the agent actor, policy approval for both
> operations, and a deterministic fingerprint. The boundary is explicit:
> canonical [R] is unchanged; approval creates [R+1]. I'll reload this tab. The
> server-owned proposal, review controls, and typed-call trace return in the
> same session.

### 4. 1:42-2:16 - Human apply, one revision, and why WebMCP fits

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

> Now the decision returns to me. I review the diff, then click Apply staged
> changes—the only commit in this flow. The server executes the exact
> stored transaction through the Studio's revision-safe command path. Board
> becomes Launch Board, its material becomes Cobalt Pieces, and revision
> advances once to [R+1]. Activity separates the agent proposal from my
> approval. One intent, four typed calls, two exact edits, one human decision:
> structured scene access without giving away authorship.

## Recovery rules

- If WebMCP discovery does not reach **Agent tools ready** and **4 tools**, stop
  the take, fix the compatible browser, and start again. Do not narrate an
  unverified tool surface.
- If the agent skips a tool, targets anything other than stable ID `board`, or
  stages different operations, stop and restart with the exact prompt.
- If another visitor changes or resets the scene, stop. Reset again, record a
  new `[R]` before recording, leave Board unselected, and restart the take; do
  not claim realtime multiplayer behavior.
- If the proposal does not return after the full reload, stop. Do not claim
  reload persistence from a different tab or browser session.
- If Apply reports a revision conflict, the safety boundary worked. For the
  main demo, stop, reset, and record a clean take rather than editing around
  the conflict.
- Do not use Undo to repair a take. Undo creates another canonical revision and
  makes the promised `[R] -> [R+1]` proof harder to read.

## Final content checklist

- [ ] The spoken edit is between 2:10 and 2:20; the upload is below 3:00.
- [ ] The public video and hosted URL both work while signed out.
- [ ] The opening visibly identifies the hosted sample as one shared ephemeral
      workspace and does not claim realtime multiplayer.
- [ ] **Agent tools ready**, **4 tools**, and all four completed flow steps are
      visible.
- [ ] The persistent typed-call trace shows concise real results for all four
      steps and survives the same-session reload.
- [ ] The exact prompt, stable ID `board`, and baseline `[R]` are shown; Board
      is found and focused by the agent rather than selected in advance.
- [ ] Both staged operations are visible: `Board -> Launch Board` and
      `Carved Wood (board-material) -> Cobalt Pieces (player-4-material)`.
- [ ] Arbiter Allow evidence, the fingerprint, and
      **canonical [R] unchanged - approval [R+1]** are readable.
- [ ] A full same-session reload restores the proposal before approval.
- [ ] A visible human click applies the proposal, and canonical state advances
      exactly once to `[R+1]`.
- [ ] Agent Activity labels `agent://webmcp` as **PROPOSED** and
      `human://webmcp-review` as **APPROVED**.
- [ ] No credentials, notifications, unsupported capability claims, or private
      information appear.
