# Submission recording runbook

Use this runbook for the final public YouTube take. The target is one truthful,
uninterrupted master under three minutes, recorded from the public GoSX 3D Studio in native
Windows Chrome. The cue driver invokes the four registered WebMCP tools through
Chrome's native protocol, but it never clicks **Apply 2 exact edits**.

Keep the [detailed narration card](demo-narration-silent-master.md) on a second
device. The
[recording sheet](demo-shot-list.md) has the detailed shot table, exact fallback
tool inputs, and OBS settings. The [full script](demo-video-script.md) contains
the complete production language and claim boundaries.

## Non-negotiable claims

- Say **“Chrome invokes the four page tools”** during the deterministic take.
  Do not say an agent chose the calls unless a visible agent actually did so.
- Call this one shared, ephemeral workspace with revision-conflict safety. Do
  not call it realtime multiplayer.
- Show four tools and zero commit tools. The visible human-facing Apply control
  is deliberately outside the WebMCP tool surface.
- Record one scene revision from clean `[R]` through exactly one approval at
  `[R+1]`. Never splice footage from different revisions.

## One-time setup

### 1. Start isolated native Windows Chrome

Close any earlier recording-profile Chrome window. In Windows PowerShell, run:

```powershell
$chrome = "C:\Program Files\Google\Chrome\Application\chrome.exe"
$profile = Join-Path $env:TEMP "gosx3d-webmcp-recording"
& $chrome `
  --user-data-dir="$profile" `
  --remote-debugging-port=9336 `
  --enable-features=WebMCPTesting,DevToolsWebMCPSupport `
  --no-first-run `
  --no-default-browser-check `
  --new-window "https://gosx3d.m31labs.dev/"
```

Open exactly one Studio tab. A separate signed-out source tab is allowed and
should point to:

```text
https://github.com/M31-Labs/gosx3d-studio/blob/main/public/studio-webmcp.js#L747-L802
```

Hide bookmarks and account UI, turn on Do Not Disturb, and close unrelated
tabs. Do not open Chrome DevTools for the primary take.

### 2. Bridge Chrome's loopback endpoint into WSL

Modern Chrome keeps remote debugging on Windows loopback. In WSL terminal A,
from the repository root, run:

```bash
recording_host="$(ip route show default | awk '{print $3; exit}')"
/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe \
  -NoProfile -ExecutionPolicy Bypass \
  -File "$(wslpath -w scripts/windows-cdp-proxy.ps1)" \
  -ListenAddress "$recording_host"
```

Leave terminal A open. The bridge listens only on the current WSL virtual
gateway at port 9337 and forwards to Chrome at `127.0.0.1:9336`. Stop it with
Ctrl+C when recording is complete.

In WSL terminal B, confirm that Chrome is reachable:

```bash
recording_host="$(ip route show default | awk '{print $3; exit}')"
curl -fsS "http://${recording_host}:9337/json/version"
```

The response must identify Google Chrome. Do not continue if this check fails.

### 3. Configure OBS

- Canvas and output: 1920×1080, 30 fps, H.264.
- Capture: the Chrome application window and microphone, with cursor visible.
- Recording: Hybrid MP4, or MKV followed by **File → Remux Recordings**.
- Video quality: CQP/CQ 18–22 or 10–16 Mbps.
- Audio: 48 kHz AAC at 160–192 kbps; speech peaks between -12 dB and -6 dB.
- Privacy: no terminal, OBS recursion, notifications, credentials, or private
  browser UI in frame.

Record and inspect a ten-second screen/audio test before the real take. Text
must remain readable at 100% playback scale.

## Preflight and clean reset

In interactive WSL terminal B, run:

```bash
recording_host="$(ip route show default | awk '{print $3; exit}')"
CDP_ENDPOINT="http://${recording_host}:9337" \
  node scripts/record-public-demo.mjs
```

The helper attaches to Chrome; it does not launch or close it. At its off-camera
prompt:

1. Confirm nobody else is exercising the shared public workspace.
2. Type `RESET` exactly once.
3. Wait for **Preflight passed** and **READY TO RECORD**.
4. Write down the printed baseline revision `[R]`.
5. Confirm the opening state is a Coral Piece selection, canonical `Board`,
   WebGPU, 145 meshes, four native page tools, and **Evidence 31/31 · current**.

If any condition differs, type `ABORT`, stop OBS if it is running, choose the
explicit reset cleanup, and start a new take.

## Record the uninterrupted master

Start OBS only at the helper's **START OBS** cue. Capture the browser window;
the two terminals stay off camera.

| Time | Operator action and spoken point |
| --- | --- |
| 0:00–0:22 | Hold the clean scene and establish why pixel hunting and invisible write access are both inadequate. |
| 0:22–0:38 | Show WebGPU, the four WebMCP tools, zero commit tools, and current evidence. |
| 0:38–0:55 | Advance **INSPECT** and **FIND**; leave Coral Piece selected until the structured result resolves stable ID `board`. |
| 0:55–1:10 | Advance **FOCUS** and **STAGE**; let the hierarchy, Inspector, and reversible viewport proposal settle. |
| 1:10–1:45 | Show the two exact edits, not-committed badge, review checks, fingerprint, and unchanged canonical revision. |
| 1:45–2:15 | Keep the live Brushed Steel proposal and human-facing approval boundary readable. |
| 2:15–2:25 | At **MANUAL HUMAN APPROVAL**, click **Apply 2 exact edits** exactly once. Do not press Enter in the terminal yet. |
| 2:25–2:46 | Hold persistent **Human approved**, `Launch Board`, `Brushed Steel`, `[R+1]`, current evidence, and paired PROPOSED/APPROVED activity. |
| 2:46–2:54 | Hold the final state and disclose the Cedar-generated narration. |

Stop OBS before returning to terminal B. Only after recording has stopped, press
Enter at the helper's approval cue. A usable take ends with a line equivalent
to:

```text
Recording verified: revision [R] → [R+1] · Launch Board · Brushed Steel · paired plan … · Evidence 31/31 · 0 document reloads · same mounted WebGPU canvas …
```

Reject the take if the helper does not print **Recording verified**.

## Abort and recovery

Stop and retake if any of these occurs:

- tool discovery shows anything other than four tools;
- a call errors, skips, duplicates, reorders, or targets anything but `board`;
- Board is already selected before Focus;
- another visitor changes the shared workspace;
- the Studio reloads or navigates as a new document;
- the staged proposal is not exactly Board → Launch Board plus Carved Wood →
  Brushed Steel;
- Apply conflicts, advances more than once, or lacks the persistent approval;
- evidence does not return to **31/31 · current**; or
- private information or an unsupported claim appears in frame.

On failure, stop OBS first. Type `ABORT` at the next helper cue and then type
`RESET` when offered cleanup. Never use Undo to repair a take because that
creates another canonical revision.

## Edit and export

1. Use the single uninterrupted master. Trim dead cue latency and verbal
   stumbles, but never reorder calls or combine different revisions.
2. Add the script's restrained 150% receipts and 165% proposal punch-ins from
   that same master. Return wide for orbit, Apply, and final evidence.
3. Keep only the captions named in the full script.
4. Export a 1920×1080 H.264 MP4 with 48 kHz AAC audio.
5. Keep the final runtime strictly below 3:00. The verified Cedar master is
   2:54.
6. Watch the export end to end with headphones at 100% scale.

If using the generated voiceover, follow the exact timed text in the detailed
narration card, identify it as OpenAI text-to-speech with the Cedar voice, and
upload the matching `.srt` captions with the video.

## Publish and verify

Use this title:

```text
GoSX 3D Studio — Human-Gated 3D Editing with WebMCP
```

Upload the video to YouTube as **Public**, add accurate captions, and put both
of these links in the description:

```text
Live demo: https://gosx3d.m31labs.dev
MIT source: https://github.com/M31-Labs/gosx3d-studio
```

After HD processing finishes, open the YouTube URL in a signed-out/private
window. Confirm public playback, clear audio, readable text, a runtime below
three minutes, and working description links. Then send the public URL for the
final Devpost readiness check.

After the recording is verified, return the public Studio to a clean judge
state with **Prepare clean demo** and its visible confirmation. Finally, stop
the temporary bridge in terminal A with Ctrl+C and close the isolated recording
profile.
