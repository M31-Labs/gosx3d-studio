# WebMCP demo shot list

Use this as the recording-day operator sheet. The complete narration is in
[the demo video script](demo-video-script.md).

## Timeline

| Time | Operator action | Proof that must be readable |
| --- | --- | --- |
| 0:00-0:28 | Reset the shared scene, then note `[R]` | `Board`, stable ID `board`, clean sample, baseline revision |
| 0:28-1:08 | Copy and send the exact in-product prompt | **Agent tools ready**, **4 tools**, Inspect/Find/Focus/Stage completion, visible focus |
| 1:08-1:42 | Inspect the staged review card | Rename plus `board-material -> player-4-material`, `agent://webmcp`, Arbiter Allow 2/2, fingerprint, canonical `[R]` unchanged |
| 1:42-2:05 | Reload the same browser tab | Proposal and Apply/Discard return; canonical revision still `[R]` |
| 2:05-2:30 | Human clicks **Apply staged changes** | `Launch Board`, `Cobalt Pieces`, revision `[R+1]` |
| 2:30-2:38 | Show Agent Activity and finish on Studio | Agent preview and human approval attribution together |

## Exact prompt

Click **Copy demo prompt** in the Studio. The copied text must be:

> Inspect the current scene, find and focus the object named Board, then
> stage—without committing—a proposal that renames it Launch Board and
> assigns the Cobalt Pieces material. Explain the revision boundary.

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
- **Layout:** Put the Studio and browser agent in one frame. Use a tested zoom
  level that keeps the hierarchy, viewport revision, collaboration rail,
  proposal card, and Agent Activity readable without live resizing.
- **Privacy:** Enable Do Not Disturb; hide bookmarks, account details,
  extensions, credentials, unrelated tabs, and desktop notifications.

## Recording procedure

1. Confirm the deployed URL loads while signed out.
2. Confirm the compatible browser reports **Agent tools ready** and **4 tools**.
3. Make sure no teammate is using the single shared demo workspace.
4. Record ten seconds of motion and narration; inspect the file at 100% scale.
5. Start the real take with two seconds of stillness.
6. Reset on camera, then write down the resulting baseline `[R]`.
7. Follow the timeline without improvising object names, material names, or
   revision claims.
8. Reload only the current tab so the same session cookie restores the staged
   proposal.
9. Leave two seconds of stillness after the closing frame.

## Edit and upload

- Trim dead tool-waiting time and verbal stumbles, but do not join footage from
  different sessions, resets, or canonical revisions.
- Keep the final timeline between 2:30 and 2:45. Watch the exported MP4 from
  beginning to end with headphones.
- Export 1080p H.264 with 48 kHz AAC audio. Confirm small text remains legible
  after YouTube processing.
- Upload as a **Public** YouTube video, add accurate captions, and verify the
  final link in a signed-out browser.
- Recheck that the video shows exactly one canonical advance from `[R]` to
  `[R+1]` after the human Apply.
