# WebMCP Challenge demo video script

Target runtime: **2:22**. Keep the final public YouTube upload below three
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
- At 1920×1080, confirm the viewport card reads **Delegate scene busywork.
  Keep creative control.** and its **4 WebMCP tools · 0 commit tools** proof is
  legible without covering the sample scene.
- Wait for the viewport to show the live WebGPU scene and for the compact
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
- Use a clean demo take, not the stress-test acceptance sequence. Do not edit a
  coral material, Discard, or restage on camera; those checks deliberately
  create extra activity that muddies the one-intent story.
- Only say **from one prompt, the browser agent...** when the recorded agent
  visibly receives the prompt and chooses the four calls. If the recording
  invokes native WebMCP tools directly, say **Chrome invokes the four page
  tools...** instead. Do not turn harness coverage into an autonomy claim.

## Exact agent prompt

Use **Copy demo prompt** in the Studio, then paste the copied text unchanged:

> Inspect the current scene, find and focus the object named Board, then
> stage—without committing—a proposal that renames it Launch Board and
> assigns the Brushed Steel material. Explain the revision boundary.

The expected tool sequence is `scene_get_state`, `scene_find_objects`,
`scene_focus_object`, then `scene_preview_actions`. There is deliberately no
agent-callable commit tool.

## Eight-beat run of show

### 1. 0:00-0:17 - Start with the problem in motion

**On screen:** Begin on the clean Studio prepared before capture, with the
non-Board selection created by reset. Hold the plain-language value card for
two seconds, then make one small, controlled orbit across the polished wood
board and glossy coral pieces. Do not point to or select `Board`. Add the
restrained caption **Shared ephemeral demo workspace · WebGPU**.

**Voiceover:**

> 3D scenes are packed with exact objects, materials, and revisions. Asking an
> agent to hunt through pixels is fragile; giving it invisible write access is
> worse. GoSX 3D Studio uses WebMCP so the artist keeps the final decision.

### 2. 0:17-0:27 - Establish the live boundary

**On screen:** Point to `WEBGPU`, **Agent tools ready**, the value-card proof
**150 entities · 4 WebMCP tools · 0 commit tools**, canonical revision `[R]`,
and **Evidence 31/31 · current**. Add the caption **150 entities · 145 meshes ·
4 page tools · 0 commit tools**.

**Voiceover:**

> This is a live, 150-entity, 145-mesh WebGPU scene. The page exposes four
> typed tools—and no commit tool.

### 3. 0:27-0:57 - Inspect, find, focus, and stage

**On screen:** Click **Copy demo prompt**, paste the exact prompt into the
browser agent, and send it. Let the **Inspect -> Find -> Focus -> Stage** rail
fill from real tool results. Do not touch Board. Hold when stable ID `board`
appears and the hierarchy plus Inspector visibly move from the non-Board
selection to Board.

**Voiceover:**

> I start with a coral piece selected, not the Board. From one prompt, the
> browser agent reads the revision, searches structured SceneDoc data, resolves
> Board to stable ID `board`, and moves the hierarchy and Inspector to that same
> object.

### 4. 0:57-1:08 - Show the WebMCP tool receipts

**On screen:** Use a 150% punch-in on **WebMCP tool receipts**. Make all four
concise results readable in order: Inspect, Find, Focus, Stage.

**Voiceover:**

> The trace records each real result: inspect, find, focus, then stage. No pixel
> guessing. No reload.

### 5. 1:08-1:39 - Review the exact proposal and policy

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

Show the caption **Canonical rev [R] unchanged**. Click **Why Arbiter allowed
this proposal**, hold for three seconds, then close it. Do not reload or leave
the review flow.

**Voiceover:**

> The agent proposes two edits: Board becomes Launch Board, and Carved Wood
> becomes Brushed Steel. The viewport renders that reversible result
> immediately, but the badge says not committed and the canonical revision has
> not moved. The review card carries the agent actor, Arbiter's allow decision,
> and a fingerprint bound to this exact result.

### 6. 1:39-1:51 - Orbit the reversible result

**On screen:** Return wide. Make one deliberate orbit of the live Brushed Steel
preview under **Agent preview · not committed**. The glossy coral and warm wood
are visual contrast, not a materials tutorial.

**Voiceover:**

> I can orbit the live preview before deciding.

### 7. 1:51-2:02 - Make the human decision

**On screen:** Move deliberately from the agent surface to **Apply staged
changes** and click exactly once. Do not cut immediately after the click.

**Voiceover:**

> Now I click Apply. That visible human action sits outside the WebMCP tool
> surface.

### 8. 2:02-2:22 - Prove one canonical handoff

**On screen:** Wait until **Evidence 31/31 · current** appears for `[R+1]`.
Then show all of the following in one deliberate sweep:

- `Launch Board` in the hierarchy or Inspector;
- `Brushed Steel` as the applied material;
- canonical revision `[R+1]`, exactly one above the baseline; and
- the `agent://webmcp` preview and `human://webmcp-review` apply entries on the
  same short plan token in **Agent Activity**.

End on the full Studio with the viewport and collaboration surface together.
Add the closing caption: **1 intent -> 4 tools -> 2 edits -> 1 human
approval**.

**Voiceover:**

> Both edits commit through the same revision-safe path, and the revision
> advances once. One intent became four page tools, two edits, and one approval.
> Scene lookup and batch preparation move to the agent; spatial judgment and
> authorship stay with the artist.

## Recovery rules

- If WebMCP discovery does not reach **Agent tools ready** and **4 tools**, stop
  the take, fix the compatible browser, and start again. Do not narrate an
  unverified tool surface.
- If the agent skips a tool, targets anything other than stable ID `board`, or
  stages different operations, stop and restart with the exact prompt.
- If another visitor changes or resets the scene, stop. Reset again, record a
  new `[R]` before recording, leave Board unselected, and restart the take; do
  not claim realtime multiplayer behavior.
- If Apply reports a revision conflict, the safety boundary worked. For the
  main demo, stop, reset, and record a clean take rather than editing around
  the conflict.
- Do not use Undo to repair a take. Undo creates another canonical revision and
  makes the promised `[R] -> [R+1]` proof harder to read.
- Retake if a tool is skipped or duplicated, Board is selected before agent
  focus, another visitor changes the workspace, Apply advances more than once,
  a reload occurs, or final evidence never reaches **31/31 · current**.
- Safe pickups are narration, trace/proposal punch-ins from the same
  uninterrupted master, and a longer closing hold. Do not substitute a
  different scene revision or browser session.

## Final content checklist

- [ ] The spoken edit is about 2:22; the upload is below 3:00.
- [ ] The public video and hosted URL both work while signed out.
- [ ] The opening visibly identifies the hosted sample as one shared ephemeral
      workspace and does not claim realtime multiplayer.
- [ ] **Agent tools ready**, **4 tools**, and all four completed flow steps are
      visible.
- [ ] The opening and closing show the live WebGPU scene plus compact
      **Evidence 31/31 · current** status for the visible revision.
- [ ] **WebMCP tool receipts** shows concise real results for all four steps
      without interrupting the flow for a reload.
- [ ] The exact prompt, stable ID `board`, and baseline `[R]` are shown; Board
      is found and focused by the agent rather than selected in advance.
- [ ] Both staged operations are visible: `Board -> Launch Board` and
      `Carved Wood (board-material) -> Brushed Steel (board-steel-material)`.
- [ ] Arbiter Allow evidence, the fingerprint, and
      **canonical [R] unchanged - approval [R+1]** are readable.
- [ ] The viewport visibly shows the reversible proposal under **Agent preview
      · not committed** while canonical revision remains `[R]`.
- [ ] A visible human click applies the proposal, and canonical state advances
      exactly once to `[R+1]`.
- [ ] The camera holds after Apply until **Evidence 31/31 · current** is
      readable for `[R+1]`; no screenshot or closing frame says recomputing.
- [ ] Agent Activity labels `agent://webmcp` as **PROPOSED** and
      `human://webmcp-review` as **APPROVED**, shows both operation kinds, and
      gives both entries the same short plan token.
- [ ] Trace and proposal punch-ins come from this one continuous browser
      session; no different revision or take is substituted.
- [ ] No credentials, notifications, unsupported capability claims, or private
      information appear.
