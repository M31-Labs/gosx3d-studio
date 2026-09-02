# WebMCP Challenge demo video script

Target runtime: **2:10**. Keep the final public YouTube upload below three
minutes. Record one continuous Studio/WebMCP session, then use editorial
punch-ins from that same capture so the trace and proposal survive YouTube
compression. Bracketed revision values such as `[R]` are production cues, not
words to read literally.

This demo makes one precise collaboration claim: a person and a browser agent
work against the same canonical SceneDoc through a shared, revision-safe
command path. The deployed demo is a **single-instance shared ephemeral
workspace**, not a realtime multiplayer room system.

For the operator's compact timeline and recording settings, keep
[the demo shot list](demo-shot-list.md) open on a second device.

## Before recording

- Use the exact hosted build and browser that will appear in the video. Wait
  until **Agent Collaboration** reads **Agent tools ready** and **4 tools**.
- Wait for the viewport to show the live WebGL scene and for the compact
  **Evidence 31/31 · current · rev [R]** status in the Agent Activity bar. Do
  not open on a negotiating renderer or stale/recomputing evidence state.
- Reset the shared scene before capture, write down the resulting baseline
  revision `[R]`, and confirm no proposal is staged. Do not record the reset.
- Confirm reset removed the `selection` query and left a non-Board object
  selected. The agent must find `Board` through WebMCP and create the first
  visible Board focus on camera.
- Confirm no one else is testing the shared deployment. A reset or edit affects
  the one canonical demo workspace for every visitor.
- Keep the hierarchy, viewport revision, four-step WebMCP rail, and Agent
  Activity readable at the recording zoom. Plan a 150% trace punch-in and a
  165% proposal-card punch-in rather than asking every proof surface to be
  readable in one wide frame.
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
> assigns the Brushed Steel material. Explain the revision boundary.

The expected tool sequence is `scene_get_state`, `scene_find_objects`,
`scene_focus_object`, then `scene_preview_actions`. There is deliberately no
agent-callable commit tool.

## Four-step run of show

### 1. 0:00-0:15 - Start with the problem and boundary

**On screen:** Begin on the clean Studio prepared before capture, with the
non-Board selection created by reset. Show canonical revision `[R]`, **Agent
tools ready**, **4 tools**, and the authority cue. Add a two-line editorial
caption: **Shared ephemeral demo workspace** and **150 entities · 145 meshes ·
4 typed tools · 0 agent commit tools**. Do not point to or select `Board`.

**Voiceover:**

> GoSX 3D Studio is a 150-entity scene with 145 compiled meshes where a browser
> agent uses scene truth, not pixels. It can inspect, find, focus, and stage.
> Only I can apply.

### 2. 0:15-0:48 - Inspect, find, focus, and stage

**On screen:** Click **Copy demo prompt**, paste the exact prompt into the
browser agent, and send it. Let the **Inspect -> Find -> Focus -> Stage** rail
and persistent typed-call trace fill from real tool results. Hold when stable
ID `board` appears and the hierarchy plus Inspector visibly move from the
non-Board selection to Board. Use a 150% punch-in on the completed trace and
caption it **Found stable ID board among 150 entities**.

**Voiceover:**

> I haven't selected Board. From one brief, the agent reads revision [R],
> searches the canonical SceneDoc, resolves Board to stable ID `board`, and
> moves the Studio's focus. Then it stages a rename and material assignment.
> These are real WebMCP calls; the trace records every result.

### 3. 0:48-1:27 - Review the proof, then reload it

**On screen:** Use a 165% punch-in on **Latest staged proposal**. Point
deliberately to:

1. `Board -> Launch Board`;
2. `board-material -> board-steel-material`, the stable material ID the agent
   resolved from `Brushed Steel`;
3. `agent://webmcp`;
4. **Arbiter - Allow - 2/2**, then click **Why Arbiter allowed this proposal**
   so the policy reasons are visible without depending on a native tooltip;
5. the affected stable ID and result fingerprint; and
6. **canonical [R] unchanged - approval [R+1]**.

Show two short editorial captions in sequence: **Canonical rev [R] stays
unchanged**, then **Only visible human Apply creates rev [R+1]**. Point back to
the viewport's still-unchanged revision `[R]`. Perform a full browser reload in
the same tab and compress the settled reload proof to about six seconds. Show
that the trace, proposal, and review controls return while the viewport remains
at `[R]`.

**Voiceover:**

> The card shows both edits, the agent actor, and Arbiter's reasons. Revision
> [R] hasn't changed; only approval creates [R+1]. The fingerprint binds that
> review to one result. Reloaded: same session, same proposal, no canonical
> change.

### 4. 1:27-2:10 - Human apply, one revision, and why the handoff matters

**On screen:** Move the cursor from the agent into the Studio and click
**Apply staged changes** yourself. After the managed refresh, show all of the
following in one deliberate sweep:

- `Launch Board` in the hierarchy or Inspector;
- `Brushed Steel` as the applied material;
- canonical revision `[R+1]`, exactly one above the baseline; and
- the `agent://webmcp` preview and `human://webmcp-review` apply entries on the
  same short plan token in **Agent Activity**.

End on the full Studio with the viewport and collaboration surface together.
Add the closing caption: **1 intent -> 4 page tools -> 2 exact edits -> 1 human
approval**.

**Voiceover:**

> Now the decision returns to me. I click Apply—the only commit in this flow.
> The stored transaction runs through the Studio's revision-safe command path.
> Launch Board and Brushed Steel appear together; revision advances once.
> Activity pairs the agent proposal with my approval on one plan. Simple edits,
> hard handoff: useful automation without surrendering authorship.

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

- [ ] The spoken edit is between 2:05 and 2:15; the upload is below 3:00.
- [ ] The public video and hosted URL both work while signed out.
- [ ] The opening visibly identifies the hosted sample as one shared ephemeral
      workspace and does not claim realtime multiplayer.
- [ ] **Agent tools ready**, **4 tools**, and all four completed flow steps are
      visible.
- [ ] The opening and closing show the live WebGL scene plus compact
      **Evidence 31/31 · current** status for the visible revision.
- [ ] The persistent typed-call trace shows concise real results for all four
      steps and survives the same-session reload.
- [ ] The exact prompt, stable ID `board`, and baseline `[R]` are shown; Board
      is found and focused by the agent rather than selected in advance.
- [ ] Both staged operations are visible: `Board -> Launch Board` and
      `Carved Wood (board-material) -> Brushed Steel (board-steel-material)`.
- [ ] Arbiter Allow evidence, the fingerprint, and
      **canonical [R] unchanged - approval [R+1]** are readable.
- [ ] A full same-session reload restores the proposal before approval.
- [ ] A visible human click applies the proposal, and canonical state advances
      exactly once to `[R+1]`.
- [ ] Agent Activity labels `agent://webmcp` as **PROPOSED** and
      `human://webmcp-review` as **APPROVED**, shows both operation kinds, and
      gives both entries the same short plan token.
- [ ] Trace and proposal punch-ins come from this one continuous browser
      session; no different revision or take is substituted.
- [ ] No credentials, notifications, unsupported capability claims, or private
      information appear.
