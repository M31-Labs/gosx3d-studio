# WebMCP demo shot list

Use this as the recording-day operator sheet. The complete narration is in
[the demo video script](demo-video-script.md).

## Timeline

| Time | Operator action | Proof that must be readable |
| --- | --- | --- |
| 0:00-0:15 | Establish the clean scene prepared before capture without selecting Board | Baseline `[R]`, **Agent tools ready**, **4 tools**, captions: **Shared ephemeral demo workspace** and **150 entities · 145 meshes · 4 typed tools · 0 agent commit tools** |
| 0:15-0:48 | Copy and send the exact in-product prompt; punch into the completed trace at 150% | Inspect/Find/Focus/Stage completion, stable ID `board`, visible focus moving from the reset selection |
| 0:48-1:15 | Punch into the proposal at 165%; open **Why Arbiter allowed this proposal**, then orbit the live preview slightly | Exact rename/material diff, `agent://webmcp`, Arbiter Allow 2/2 plus reasons, fingerprint, canonical `[R]` unchanged, **Agent preview · not committed** |
| 1:15-1:42 | Human reviews, then visibly clicks **Apply staged changes** | `Launch Board`, `Brushed Steel`, revision `[R+1]`, no canvas blink |
| 1:42-1:55 | Show Agent Activity and finish on the full Studio | Both operation kinds, proposed/approved actors, matching plan token, compact current evidence status, closing 1→4→2→1 caption |

## Exact prompt

Click **Copy demo prompt** in the Studio. The copied text must be:

> Inspect the current scene, find and focus the object named Board, then
> stage—without committing—a proposal that renames it Launch Board and
> assigns the Brushed Steel material. Explain the revision boundary.

## OBS setup

- **Canvas:** 1920x1080.
- **Output:** 1920x1080, 30 fps, H.264. Use hardware encoding when it is
  stable on the recording machine.
- **Recording format:** MKV during capture so a crash does not destroy the
  take; use **File -> Remux Recordings** to produce MP4 afterward.
- **Video quality:** CQP/CQ around 18-22, or 10-16 Mbps if using constant
  bitrate. Favor readable UI text over a smaller source file.
- **Audio:** 48 kHz, mono or stereo AAC on export, roughly 160-192 kbps. Set
  microphone gain so normal speech peaks between `-12 dB` and `-6 dB`.
- **Sources:** Capture the browser/application window and microphone. Keep the
  cursor enabled. Avoid capturing the OBS window itself.
- **Layout:** Capture the Studio and browser agent in one continuous frame at a
  tested zoom. In the edit, derive a 150% trace crop and a 165% proposal crop
  from that same capture; return to the full frame for the human Apply and
  closing state. Do not resize the browser during the take.
- **Privacy:** Enable Do Not Disturb; hide bookmarks, account details,
  extensions, credentials, unrelated tabs, and desktop notifications.

## Recording procedure

1. Confirm the deployed URL loads while signed out.
2. Confirm the compatible browser reports **Agent tools ready** and **4 tools**;
   wait for the live WebGL scene and compact **Evidence 31/31 · current** status
   for the visible revision.
3. Make sure no teammate is using the single shared demo workspace.
4. Reset the shared scene before recording, write down baseline `[R]`, confirm
   no proposal is staged, confirm the URL has no `selection` query, and verify
   the deterministic reset selection is not Board.
5. Record ten seconds of motion and narration; inspect the file at 100% scale.
6. Start the real take with two seconds of stillness. Do not reset on camera.
7. Follow the timeline without improvising object names, material names, or
   revision claims.
8. Click the visible Arbiter disclosure rather than hovering for a tooltip.
9. Leave two seconds of stillness after the closing frame.

## Edit and upload

- Trim dead tool-waiting time and verbal stumbles, but do not join footage from
  different sessions, resets, or canonical revisions.
- Keep the final timeline between 1:50 and 2:05. Watch the exported MP4 from
  beginning to end with headphones.
- Export 1080p H.264 with 48 kHz AAC audio. Confirm small text remains legible
  after YouTube processing.
- Upload as a **Public** YouTube video, add accurate captions, and verify the
  final link in a signed-out browser.
- Recheck that the video shows exactly one canonical advance from `[R]` to
  `[R+1]` after the human Apply.
