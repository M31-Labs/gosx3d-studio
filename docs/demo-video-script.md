# WebMCP Challenge demo video script

Target runtime: **2:28**. Record this version now in **native Windows Chrome
149+**. The primary take uses the already-tested, cue-driven recording helper
at `scripts/record-public-demo.mjs`. It attaches to an existing native
Windows Chrome tab, invokes the registered WebMCP tools through Chrome's native
protocol, and deliberately leaves Apply to the person on camera. Chrome
DevTools' WebMCP pane is the no-driver backup. The public YouTube upload must
stay below three minutes.

The precise product claim is that a person and a browser agent can work against
the same canonical SceneDoc through a shared, revision-safe path. In this take,
Chrome invokes the four page tools deterministically and the Studio makes every
result visible. Do not say that an agent chose the calls. The deployed demo is
a **single-instance shared ephemeral workspace**, not a realtime multiplayer
room.

Record one uninterrupted Studio/WebMCP session. Trim cue and tool wait time,
and derive editorial punch-ins from that master. Never splice together
different scene revisions. Bracketed values such as `[R]` are production cues,
not words to read literally.

Keep [the operator shot list](demo-shot-list.md) open on a second device.

## Lock the recording path

Use the **cue-driven native Chrome take** below as the primary submission video.
The helper refuses non-Windows browser targets, verifies the public GoSX
`v0.55.1` WebGPU scene and exactly four native tools, invokes them in a fixed
order, and pauses for the visible human Apply. Its terminal stays outside the
browser-window capture.

If the helper is unavailable, use **Application -> WebMCP** in Chrome DevTools
and enter the same four inputs from the shot list. Chrome documents **Run tool**
as the browser's built-in way to test WebMCP tools independently of agent
decision logic.

Only upgrade to the optional natural-language take if a visible WebMCP agent is
already working reliably in Windows Chrome or ChatGPT's in-app browser. Do not
delay the deterministic recording to debug the unavailable client.

## Before recording

- Use native Windows Chrome 149+ with
  `chrome://flags/#enable-webmcp-testing` enabled and its already-validated
  remote-debugging endpoint. Do not use or show a Linux browser. Open exactly
  one public Studio tab.
- Start `node scripts/record-public-demo.mjs` before OBS. A local Node
  process may carry the cues, but the helper must identify its target as native
  Windows Chrome; it will abort otherwise. It never launches a browser.
- At the helper's off-camera prompt, coordinate the shared workspace and type
  `RESET`. Wait for **READY TO RECORD**. The helper confirms `WEBGPU`, 145
  meshes, four native tools, **Evidence 31/31 · current**, a clean Board, and a
  non-Board opening selection. Write down its baseline revision `[R]`.
- In the ready state, rehearse **Application -> WebMCP** without running a tool.
  Confirm exactly four entries under **Available Tools**, then close DevTools.
  The primary helper calls use the exact inputs in the shot list.
- Prepare a signed-out public GitHub tab at
  `github.com/M31-Labs/gosx3d-studio/blob/main/public/studio-webmcp.js#L747-L802`
  for the eight-second source close. Hide bookmarks, account UI, credentials,
  unrelated tabs, notifications, and recording controls.
- Confirm no one else is testing the shared deployment. A reset or edit affects
  the one canonical demo workspace for every visitor.
- Record a ten-second audio and screen test. Check that the cursor is visible,
  text survives export, and speech peaks around `-12 dB` to `-6 dB`.
- Use a clean demo take. Do not edit a coral material, Discard, restage, or Undo
  on camera; those checks muddy the one-intent story.
- Leave two seconds of stillness at the beginning and end. Dead cue and wait
  time may be trimmed, but tool results must remain in their real order.

## The exact user intent

Keep the in-product prompt visible briefly; do not claim it was sent in the
deterministic take:

> Inspect the current scene, find and focus the object named Board, then
> stage—without committing—a proposal that renames it Launch Board and
> assigns the Brushed Steel material. Explain the revision boundary.

The exact tool sequence is `scene_get_state`, `scene_find_objects`,
`scene_focus_object`, then `scene_preview_actions`. There is deliberately no
agent-callable commit tool.

## Ten-beat run of show

### 1. 0:00-0:16 — Start with the problem in motion

**On screen:** Begin on the clean Studio, with a non-Board selection. Hold the
value card for two seconds, then make one controlled orbit across the polished
wood board and glossy coral pieces. Do not point to or select `Board`. Caption:
**Shared ephemeral demo workspace · WebGPU**.

**Voiceover:**

> 3D scenes are dense with exact objects, materials, and revisions. Asking an
> agent to hunt through pixels is fragile; giving it invisible write access is
> worse. GoSX 3D Studio keeps the artist in the final decision.

### 2. 0:16-0:28 — Establish the live boundary

**On screen:** Point to `WEBGPU`, **Agent tools ready**, **150 entities · 4
WebMCP tools · 0 commit tools**, canonical revision `[R]`, and **Evidence 31/31
· current**. Caption: **150 entities · 145 meshes · 4 tools · 0 commit tools**.

**Voiceover:**

> This live WebGPU scene has 150 entities and 145 meshes. Its page exposes four
> typed WebMCP tools—and no commit tool.

### 3. 0:28-0:40 — Show the task and the honest invocation mode

**On screen:** Hold the in-product task card. Use a short same-session pickup of
**Application -> WebMCP** showing exactly four entries under **Available
Tools**; return to the full Studio before invoking them.

**Voiceover:**

> A browser agent can turn this one request into four calls. For this
> deterministic take, Chrome invokes those same native page tools, while its
> WebMCP pane exposes the schemas an agent discovers.

### 4. 0:40-1:07 — Inspect, find, focus, and stage

**On screen:** Follow the recording helper's off-capture cues: press Enter for
Inspect, Find, Focus, and Stage. Let the four-step rail and receipts fill from
the real Chrome calls. Keep the Studio visible as the hierarchy and Inspector
move from the coral selection to stable ID `board`, then as the staged preview
appears. Trim cue and wait time, not result order.

**Voiceover:**

> Chrome reads the current revision, searches structured SceneDoc data, finds
> Board as stable ID `board`, and focuses the same object in the visible Studio.
> The fourth tool validates and stages two edits at that exact revision. Nothing
> navigates or reloads.

### 5. 1:07-1:18 — Show the receipts

**On screen:** With DevTools closed, use a 150% punch-in on **WebMCP tool
receipts**. Make Inspect, Find, Focus, and Stage readable in order. Caption:
**4 completed calls · same document · 0 reloads**.

**Voiceover:**

> The page records each real result: inspect, find, focus, then stage. No pixel
> guessing, and no hidden second scene.

### 6. 1:18-1:43 — Review the proposal and policy

**On screen:** Use a 165% punch-in on **Latest staged proposal**. Point to:

1. `Board -> Launch Board`;
2. `board-material -> board-steel-material`;
3. `agent://webmcp`;
4. **Arbiter · Allow · 2/2** and its visible reasons;
5. the affected stable ID and result fingerprint; and
6. **canonical [R] unchanged · approval [R+1]**.

Open **Why Arbiter allowed this proposal**, hold for two seconds, then close it.
Caption: **Reversible preview · canonical rev [R] unchanged**.

**Voiceover:**

> The proposal renames Board to Launch Board and previews Brushed Steel. The
> viewport renders the reversible result now, but the badge says not committed
> and canonical revision R has not moved. The review also carries policy
> evidence, affected IDs, and a fingerprint bound to this exact result.

### 7. 1:43-1:53 — Exercise human judgment

**On screen:** Return wide and orbit the Brushed Steel preview once under
**Agent preview · not committed**.

**Voiceover:**

> The artist can inspect the proposed material spatially before deciding.

### 8. 1:53-2:03 — Make the human decision

**On screen:** Move deliberately to **Apply staged changes** and click exactly
once. Do not cut immediately after the click.

**Voiceover:**

> I choose Apply. That human action is deliberately outside the WebMCP tool
> surface.

### 9. 2:03-2:20 — Prove one canonical handoff

**On screen:** Wait for **Evidence 31/31 · current** at `[R+1]`, then show
`Launch Board`, `Brushed Steel`, canonical `[R+1]`, and the
`agent://webmcp` **PROPOSED** plus `human://webmcp-review` **APPROVED** entries
on the same short plan token. Caption: **1 intent -> 4 tools -> 2 edits -> 1
human approval**.

**Voiceover:**

> Both edits commit through the same revision-safe engine, and the revision
> advances exactly once. Scene search and batch preparation can move to an
> agent; spatial judgment and authorship stay with the artist.

### 10. 2:20-2:28 — Close on public implementation

**On screen:** Switch to the prepared signed-out GitHub source tab. Hold on the
visible `document.modelContext.registerTool` registrations and public repository
URL. Caption: **Live demo + MIT source · links below**.

**Voiceover:**

> The live demo and complete MIT-licensed implementation are public. This is
> GoSX 3D Studio.

## Optional natural-language upgrade

Use this only if the visible agent works reliably before recording time. Keep
all other beats unchanged and do not run the cue helper for the tool calls.

- In beat 3, click **Copy demo prompt**, paste it unchanged into the visible
  agent, and send it.
- Let the agent choose all four calls. Stop if it skips, duplicates, reorders,
  or changes one.
- Replace the last sentence of beat 3 with: **“From one prompt, the browser
  agent chooses the four registered tools.”**
- Replace the start of beat 4 with: **“The agent reads the current revision…”**
- Do not show the cue helper or describe Chrome as manually invoking the calls
  in this version.

Never mix deterministic Chrome invocation footage with narration that says the
agent chose the calls.

## Recovery rules

- If the helper does not print **Preflight passed** and **READY TO RECORD**, or
  Chrome does not show **Agent tools ready**, **4 tools**, and exactly four
  Available Tools in **Application -> WebMCP**, stop and fix the compatible
  Windows browser before recording.
- If any tool errors, targets anything other than stable ID `board`, or stages
  different operations, stop. Reset, record a fresh `[R]`, and restart.
- If another visitor changes the workspace, stop and reset. Do not claim
  realtime multiplayer behavior.
- If Apply reports a revision conflict, the safety boundary worked. For the
  main demo, restart rather than editing around it.
- Do not use Undo to repair a take; it creates another canonical revision.
- Retake if Board is selected before focus, a reload occurs, Apply advances
  more than once, or final evidence never reaches **31/31 · current**.
- Safe edits are dead-time trims, narration pickups, and punch-ins from the
  same uninterrupted master. Never substitute a different revision.

## Final content checklist

- [ ] The finished video runs about 2:28 and is under 3:00 with clear audio.
- [ ] The public YouTube video, live app, and public repo work while signed out.
- [ ] The video uses native Windows Chrome 149+ and never shows a Linux browser.
- [ ] Manual narration says Chrome invokes the tools; it never says an agent
      chose them.
- [ ] The opening identifies the single shared ephemeral workspace and does not
      claim realtime multiplayer.
- [ ] `WEBGPU`, **Agent tools ready**, **4 tools**, **0 commit tools**, and
      **Evidence 31/31 · current** are readable.
- [ ] Chrome's WebMCP pane shows exactly four Available Tools, and the Studio
      records four completed native invocations in the intended order.
- [ ] Stable ID `board`, both staged operations, Arbiter Allow, fingerprint,
      and unchanged canonical `[R]` are readable.
- [ ] The reversible preview orbits before one visible human Apply.
- [ ] Canonical state advances exactly once to `[R+1]`, with matching
      **PROPOSED** and **APPROVED** activity.
- [ ] The flow visibly stays in one document with no reload.
- [ ] The close shows the public repository, live source registrations, and MIT
      caption without private account details.
- [ ] No credentials, notifications, unsupported claims, stale evidence, or
      private information appear.
