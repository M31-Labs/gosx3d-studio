# WebMCP demo recording sheet

Primary take: use `scripts/record-public-demo.mjs` to cue an existing
**native Windows Chrome 149+** tab. The helper never launches a browser, refuses
a non-Windows target, invokes the four native WebMCP tools in a fixed order,
and deliberately leaves Apply to the person. Chrome DevTools' manual **Run
tool** flow is the immediate backup. Full narration is in
[the demo video script](demo-video-script.md).

## Timeline

| Time | Operator action | Proof that must be readable |
| --- | --- | --- |
| 0:00-0:16 | Hold the clean value card, then orbit once without selecting Board | Polished board, glossy coral, **Shared ephemeral demo workspace · WebGPU** |
| 0:16-0:28 | Establish the live boundary | `[R]`, `WEBGPU`, **Agent tools ready**, **150 entities · 4 WebMCP tools · 0 commit tools**, **Evidence 31/31 · current** |
| 0:28-0:40 | Hold the in-product task; use a short same-session **Application -> WebMCP** pickup | Exact user intent and exactly four **Available Tools** |
| 0:40-1:07 | Follow the off-capture helper cues for state, find, focus, and preview; trim only cue/wait time | Four completed native results; stable ID `board`; hierarchy/Inspector focus; staged preview |
| 1:07-1:18 | With DevTools closed, punch into **WebMCP tool receipts** at 150% | Inspect/Find/Focus/Stage, same document, no reload |
| 1:18-1:43 | Punch into the proposal at 165%; open Arbiter reasons for two seconds | Exact rename/material diff, actor, Allow 2/2, fingerprint, canonical `[R]` unchanged |
| 1:43-1:53 | Return wide and orbit Brushed Steel preview once | **Agent preview · not committed**, unchanged revision |
| 1:53-2:03 | Click **Apply staged changes** exactly once | Visible human decision outside the four-tool surface |
| 2:03-2:20 | Wait for evidence, then show applied state and paired activity | `Launch Board`, `Brushed Steel`, `[R+1]`, **Evidence 31/31 · current**, matching proposed/approved token |
| 2:20-2:28 | Switch to signed-out public GitHub source tab | Public repo URL, live `registerTool` code, **MIT source** caption |

## Primary sequence — cue-driven Windows Chrome

1. Start the already-validated native Windows Chrome with WebMCP and remote
   debugging enabled. Open exactly one tab at
   `https://gosx3d.m31labs.dev`. Do not start or attach to a Linux browser.
2. In the operator terminal, run:

   ```bash
   node scripts/record-public-demo.mjs
   ```

   The terminal is only the control channel. The helper prints the detected
   browser and aborts unless its target is Windows Chrome.
3. At the off-camera prompt, make sure nobody else is using the shared
   workspace, then type `RESET`. The helper performs its only navigation and
   reset before recording.
4. Wait for **Preflight passed** and **READY TO RECORD**. Write down the printed
   baseline revision `[R]`.
5. Start OBS only when cued. Capture the browser window, not the terminal. Do
   beats 1-3, then press Enter once to finish the helper's **START OBS** cue.
6. Press Enter once at each **INSPECT**, **FIND**, **FOCUS**, and **STAGE**
   cue. Do not type any other response. The helper invokes the exact inputs
   below and validates each visible result.
7. At **REVIEW / ORBIT**, show the receipts, proposal, policy reasons,
   fingerprint, and reversible material in the Studio. Press Enter after the
   orbit to advance to the approval cue.
8. At **MANUAL HUMAN APPROVAL**, click **Apply staged changes** yourself exactly
   once. The helper intentionally does not click or verify this action.
9. Wait for **Evidence 31/31 · current** at `[R+1]`, show the applied proof,
   switch to the public source tab, leave two seconds still, and stop OBS.
10. Only after OBS stops, return to the terminal and press Enter. On an aborted
    take, stop OBS first; use the helper's explicit `RESET` cleanup only if you
    intend to clear the shared state.

The helper performs one off-camera `Page.navigate` before **READY TO RECORD**
and never calls `Page.reload`. There is no navigation or reload during the
recorded Studio flow.

## Exact WebMCP inputs

These are the helper inputs and the no-driver DevTools fallback inputs.

### 1. `scene_get_state`

Use no parameters. The result supplies the canonical revision `[R]`.

### 2. `scene_find_objects`

Use:

```json
{
  "query": "board",
  "visibleOnly": true,
  "limit": 10
}
```

Run it and hold long enough to show the result for stable ID `board`.

### 3. `scene_focus_object`

Use:

```json
{
  "objectId": "board"
}
```

Run it and show the hierarchy plus Inspector moving from the reset selection to
Board. The canonical revision must remain `[R]`.

### 4. `scene_preview_actions`

Replace the example `123` with the integer returned by `scene_get_state`:

```json
{
  "expectedRevision": 123,
  "title": "Prepare Launch Board",
  "rationale": "Resolve Board and Brushed Steel by stable ID, show the exact reversible viewport diff, and leave canonical authority with the human reviewer.",
  "operations": [
    {
      "kind": "rename-entity",
      "target": "board",
      "name": "Launch Board"
    },
    {
      "kind": "assign-material",
      "target": "board",
      "material": "board-steel-material"
    }
  ]
}
```

Confirm the output says the proposal is staged for human review and the
canonical revision was not changed. Do not click Apply until the scripted human
decision beat.

## No-driver backup — Chrome DevTools

If the cue helper cannot run, stay in native Windows Chrome. Open DevTools,
select **Application**, then select **WebMCP** at the top level of the
Application sidebar. Under **Available Tools**, choose each tool, enter the
inputs above, and click **Run tool** in order. Read `[R]` from the first output
before filling the fourth. Chrome documents this flow as its built-in way to
test registered tools independently of agent decision logic.

Keep the same narration: **Chrome invokes the four page tools.** Do not say the
agent chose them. Retain each completed result in Chrome's **Invoked Tools** log
and the Studio's visible receipt rail. Apply remains a visible human click.

## Fast preflight — do this once

- [ ] Use the verified native Windows Chrome installation; never use Linux.
- [ ] Enable `chrome://flags/#enable-webmcp-testing`, relaunch Chrome, and load
      `https://gosx3d.m31labs.dev` signed out.
- [ ] Run the helper, coordinate the shared reset, and wait for **Preflight
      passed** plus **READY TO RECORD**. Record its `[R]`; do not reset again.
- [ ] Rehearse opening **Application -> WebMCP** and verify four Available Tools
      without invoking one. Close it again before the tool-call beat.
- [ ] Prepare the signed-out source tab:
      `https://github.com/M31-Labs/gosx3d-studio/blob/main/public/studio-webmcp.js#L747-L802`.
- [ ] Keep this operator sheet on the second device. Do not expose a text
      editor, clipboard manager, terminal, or private notes in the capture.
- [ ] Record ten seconds of UI motion and narration; inspect it at 100% scale.

## OBS setup

- **Canvas/output:** 1920x1080, 30 fps, H.264. Use stable hardware encoding.
- **Capture:** Browser/application window plus microphone; cursor visible.
- **Recording:** Hybrid MP4 (crash-recoverable and upload-ready). If that OBS
  option is unavailable, record MKV and use **File -> Remux Recordings**.
- **Quality:** CQP/CQ 18-22, or 10-16 Mbps. Optimize for readable UI text.
- **Audio:** 48 kHz AAC, 160-192 kbps; normal speech peaks at `-12 dB` to
  `-6 dB` without clipping.
- **DevTools layout:** Start full-width. During beat 3, open DevTools docked
  right and make the WebMCP pane wide enough to read without covering the
  Studio. Close it before beat 4. Do not resize the browser window or reload
  the page.
- **Editorial crops:** 150% for receipts, 165% for proposal. Derive both from
  the same master. Return wide for orbit, Apply, and final evidence.
- **Privacy:** Do Not Disturb on; no bookmarks, accounts, extensions,
  credentials, unrelated tabs, desktop notifications, or OBS recursion.

## Real-take procedure

1. Start with two seconds of stillness on the clean Studio. Do not reset on
   camera.
2. Follow the timeline and narration without improvising names or claims.
3. Use the off-capture helper cues to run each tool. Let the visible rail and
   receipts complete; trim cue latency in post, never result order.
4. Keep the Studio visible when Focus runs so Board's UI selection is obvious.
5. Keep DevTools closed after the beat-3 tool-list pickup; do not refresh or
   navigate the Studio.
6. Open the Arbiter disclosure with a click, hold two seconds, then close it.
7. Orbit the reversible preview before approval.
8. Click **Apply staged changes** once, then wait for **Evidence 31/31 ·
   current** at `[R+1]`.
9. Show the paired `agent://webmcp` **PROPOSED** and
   `human://webmcp-review` **APPROVED** entries with the same short plan token.
10. Switch to the prepared public source tab for eight seconds. Leave two
    seconds of stillness at the end.

## Edit, upload, and verify

- Trim cue latency, tool latency, and verbal stumbles; do not join different
  scene revisions or reorder calls.
- Add only the captions named in the script. Keep them above YouTube controls
  and away from product evidence.
- Export a 2:20-2:30 1080p H.264 MP4 with 48 kHz AAC audio.
- Watch the exported file end-to-end with headphones at 100% scale.
- Upload to YouTube as **Public**, add accurate captions, and verify the video
  signed out after HD processing completes.
- Put both links in the description:
  `https://gosx3d.m31labs.dev` and
  `https://github.com/M31-Labs/gosx3d-studio`.
- Confirm the final edit says **Chrome invokes** for the deterministic take,
  shows exactly one `[R] -> [R+1]` advance, and never claims agent-selected
  calls.

## Abort conditions

Stop, reset, and restart if any of these occurs:

- WebMCP discovery shows anything other than four tools.
- A call errors, is skipped, duplicated, reordered, or targets a non-`board`
  object.
- Board is already selected before Focus.
- Another visitor changes the shared workspace.
- A reload or second Studio navigation occurs.
- The proposal differs from the two scripted operations.
- Apply conflicts, advances more than once, or final evidence stays stale.
- A private account detail, notification, credential, or unsupported claim is
  captured.

Do not use Undo as a repair; it creates another canonical revision. Record a
fresh clean take instead.
