# Title

GoSX 3D Studio

## One-line Summary

A 3D studio where browser agents find and stage exact scene edits while artists
preview and approve them in place.

## Problem

3D editors are dense and stateful. A person can see which object matters; an
agent needs stable identities, exact properties, and the current revision.
Pixel automation guesses through controls, while a detached API removes work
from the artist's spatial context. The real challenge is keeping person and
agent aligned before anything becomes canonical.

## Solution

GoSX 3D Studio turns the webpage itself into a shared collaboration surface.
It registers four focused tools with `document.modelContext.registerTool(...)`:

- `scene_get_state` reads canonical scene state and its revision.
- `scene_find_objects` searches stable IDs and names with typed filters.
- `scene_focus_object` brings the hierarchy and Inspector to the object under
  discussion without changing canonical scene state.
- `scene_preview_actions` validates and stages 1–12 reversible operations at an
  exact revision.

An agent can inspect 150 entities, resolve `Board` to stable ID `board`, focus
the visible Studio, and stage a rename plus material assignment. The viewport
renders the result under **Agent preview · not committed**; its review card
shows the semantic diff, affected IDs, revision, actor, policy evidence, and
fingerprint. **Apply staged changes** and **Discard** are visible Studio
actions. There is intentionally no WebMCP commit tool.

Apply sends only an opaque proposal ID. The server commits the exact previewed
transaction through the existing revision-safe engine. Stale work is rejected,
and the activity log distinguishes `agent://webmcp` proposal authorship from
`human://webmcp-review` approval.

### Existing project and Challenge-period work

GoSX 3D Studio existed before August 25, 2026. It already had its SceneDoc
model, 3D workbench, stable IDs, fingerprints, undo/redo, revision conflicts,
and a shared transaction path for human and agent commands. Those foundations
are not claimed as Challenge work.

During the submission period we added the WebMCP adapter; structured inspection,
search, and focus; session-owned non-mutating proposals; Arbiter policy; the
visible review and exact-commit boundary; typed-call evidence; shared-demo
reset; focused tests; and deployment materials. Polish added same-document
Scene3D reconciliation, WebGPU-first rendering, a compact evidence dock, and a
refined PBR sample scene.

## Why This Matters

- **WebMCP Leverage:** WebMCP carries discovery, inspection, stable-ID search,
  visible focus, typed proposal, and structured results.
- **Execution:** The tools operate on the real SceneDoc, hierarchy, Inspector,
  viewport, command history, and transaction engine. There is no parallel demo
  scene or canned response path.
- **Potential Impact:** Technical artists can delegate scene search and batch
  preparation without surrendering spatial judgment.
- **Creativity & Ambition:** This is not a chatbot beside a canvas. Person and
  agent point at the same stable object, reason from the same revision, and
  create an auditable handoff from machine-prepared preview to human-approved
  canonical state.

## How We Used AI

An external agent interprets the goal, chooses the page tools, finds relevant
objects, and prepares a typed proposal with rationale. The Studio retains
schema validation, revision checks, policy, fingerprints, authority, and state.

The demo prompt asks the agent to inspect the scene, find and focus `Board`,
then stage—without committing—the Launch Board and Brushed Steel edits. The
person can orbit the reversible result before choosing Apply or Discard.

## How We Used Codex

The human supplied product direction and final judgment. OpenAI Codex helped
trace the architecture, connect WebMCP to `Workspace.Execute`, implement and
review boundaries, author tests, debug native-browser behavior, run checks, and
prepare documentation and demo materials.

## Key Features

- Exactly four webpage-registered tools with strict JSON Schemas and structured
  results; zero commit tools.
- Stable-ID search and non-mutating focus shared by the visible Studio.
- Bounded preview operations: `rename-entity`, `set-transform`, and
  `assign-material` only.
- One-use session proposals with expiry, revision checks, CSRF protection, and
  exact stored-transaction commit.
- Fail-closed Arbiter policy with visible Allow evidence for each staged edit.
- WebGPU-first Scene3D rendering, in-place updates, and WebGL fallback.
- Shared-demo reset that invalidates proposals and advances revision; reset is
  deliberately outside the WebMCP surface.

## Architecture

```text
browser agent -> four WebMCP tools -> same-origin proposal service
              -> Arbiter policy -> Workspace.Execute(propose, revision R)
              -> visible non-applied review card
              -> human Apply UI -> exact stored transaction
              -> Workspace.Execute(direct, revision R) -> canonical R+1
```

Both paths reuse `Workspace.Execute`; WebMCP is an adapter over the existing
scene model. The pre-Challenge comparison point is commit
`a216194ebc0f415a011aa780386773c0750bccf9` (July 29, 2026).

## Built With

Go, GoSX, WebMCP, WebGPU, JavaScript, and Arbiter. The hosted release can run on
Kubernetes, but Kubernetes is deployment infrastructure rather than a product
requirement.

## Testing Instructions

1. Open [gosx3d.m31labs.dev](https://gosx3d.m31labs.dev) in Chrome 149+ with
   `chrome://flags/#enable-webmcp-testing` enabled, or in another compatible
   WebMCP client. No credentials are required.
2. Click **Reset shared scene**, confirm the warning, and wait for **Agent tools
   ready · 4 tools**.
3. Click **Copy demo prompt** and give the copied prompt to the browser agent.
4. Confirm **Inspect → Find → Focus → Stage** completes and the hierarchy and
   Inspector focus stable ID `board`.
5. Confirm the staged card shows both edits, `agent://webmcp`, Arbiter Allow,
   the fingerprint, and canonical revision `R` unchanged.
6. Orbit the live Brushed Steel preview, then click **Apply staged changes**.
7. Confirm `Launch Board`, `Brushed Steel`, revision `R+1`, and separate
   agent proposal and human approval activity.

Local GoSX `v0.55.1` release-candidate acceptance passed **162/162 assertions**
in native Windows Chrome 152 with WebGPU, exact four-tool discovery, zero reload
commands, one main-document request, and a clean reset. The final immutable
public `v0.55.1` deployment and public-origin replay are still pending. A
ChatGPT in-app-browser pass would provide useful cross-client assurance, but is
not required in addition to the compatible Chrome path.

Local run and the full automated verification floor are documented in the
public README and `docs/native-webmcp-qa.md`.

## Public Demo Link

[GoSX 3D Studio live demo](https://gosx3d.m31labs.dev)

The URL is live; deploying and replaying the final immutable GoSX `v0.55.1`
release candidate remains a readiness item.

## Public Repository Link

[M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio) — MIT

The repository is public. The final release-candidate changes still need to be
merged to its default branch and anonymously rechecked before the deadline.

## Demo Video

`[TODO: PUBLIC YOUTUBE URL — UNDER 3 MINUTES WITH AUDIO]`

The 2:22 plan in `docs/demo-video-script.md` covers four native page tools,
typed-call evidence, reversible preview, policy, visible Apply, and separate
agent/human attribution in one continuous session.

## Screenshot Shot List

Replace these three placeholders with high-resolution public-origin Windows
Chrome frames after the final deployment replay:

1. `[TODO GALLERY 1: CLEAN WORKBENCH]` — viewport, hierarchy, Inspector,
   revision, WebGPU, and **Agent tools ready · 4 tools**. Caption: “One scene,
   visible to both human and browser agent.”
2. `[TODO GALLERY 2: STAGED REVIEW]` — semantic diff, Arbiter Allow, affected
   ID, fingerprint, canonical `R` unchanged, and Apply/Discard. Caption: “The
   agent proposes; Apply stays in the visible Studio UI.”
3. `[TODO GALLERY 3: APPLIED HANDOFF]` — Launch Board, Brushed Steel, `R+1`,
   and both actors in Agent Activity. Caption: “One reviewed proposal becomes
   one attributed canonical change.”

## Submission Readiness Notes

- [x] Document existing foundations separately from Challenge-period work.
- [x] Verify local GoSX `v0.55.1` in native Windows Chrome: 162/162, WebGPU,
      four tools, zero reload commands, one main-document request, clean reset.
- [x] Publish the live URL, public repository, and root MIT License.
- [ ] Commit and merge the release candidate to the public default branch.
- [ ] Deploy its immutable image and repeat the public Windows Chrome flow.
- [ ] Capture the three final gallery frames from that public run.
- [ ] Record and publish the 2:22 YouTube demo with audio.
- [ ] Confirm the remaining official form answers below.

Nothing in this draft has been sent to or updated on Devpost.

## Known Limitations

- Canonical scene and pending proposal state are process-local. The hosted demo
  runs one replica; restart loses that ephemeral state.
- The demo is one shared workspace with revision-conflict safety, not a private
  account workspace or realtime CRDT multiplayer room.
- WebMCP exposes only three reversible operation kinds, and every staged edit
  requires visible human approval.
- The broader editor includes capabilities whose certification state remains
  partial or planned; the demo claim is limited to the verified scene workflow.
- The final public `v0.55.1` replay, gallery, YouTube upload, default-branch
  merge, and official form review remain outstanding. ChatGPT in-app testing is
  optional cross-client assurance and has not been completed.

## TODO Official Form Fields

These labels come from the live form for The WebMCP Challenge. Bracketed values
require the submitter's confirmation.

| Official field | Draft answer / status |
| --- | --- |
| **Submitter Type** (required) | `[TODO: choose Individual, Team, or Organization]` |
| **Country of residence of yourself and team members if applicable** (required) | `[TODO: select every applicable country exactly as listed]` |
| **Organization name** (optional when submitting for an organization) | `[TODO if applicable; otherwise leave blank]` |
| **App Status** (required) | `Existing` |
| **If Existing, explain what you updated during the submission period** | `GoSX 3D Studio existed before August 25 with its SceneDoc workbench and revision-safe human/agent transaction engine. During the Challenge we added four webpage-registered WebMCP tools for scene inspection, stable-ID search, visible focus, and bounded non-mutating proposals; a session-owned review service; executable Arbiter policy evidence; a visible human-only exact-commit boundary; typed-call and actor attribution; focused WebMCP, authority, policy, and reset tests; and deployment and demo materials. We also added same-document reconciliation that preserves the Scene3D canvas and camera, WebGPU-first rendering with WebGL fallback, and a polished PBR sample scene. The pre-existing editor and transaction foundations are not claimed as new work.` |
| **Live URL** (required) | [https://gosx3d.m31labs.dev](https://gosx3d.m31labs.dev) |
| **Testing instructions** (optional/private) | `No credentials required. Reset the shared scene, confirm four ready tools, use Copy demo prompt, inspect the staged non-mutating proposal, then click the visible Apply action and confirm revision R+1. See the Hosted judge flow above.` |
| **Public code repository** (required) | [https://github.com/M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio) |
| **Tested WebMCP clients** (required) | `Google Chrome 152 on Windows with WebMCPTesting and DevToolsWebMCPSupport. Local GoSX v0.55.1 acceptance passed 162/162 assertions with native document.modelContext, WebGPU, exactly four tools, zero reload commands, one main-document request, and clean reset. Final public-origin replay remains pending; ChatGPT in-app testing is optional assurance and has not been completed.` |
| **AI tools used** (required) | `OpenAI Codex for repository analysis, implementation, test authoring, debugging, verification, documentation, and submission preparation. [TODO: add any other AI tools actually used.]` |
| **Learning level** (required: None / Moderate / Significant) | `[TODO: choose one]` |
| **Career AI value** (required: Yes / No) | `[TODO: choose one]` |
| **Public YouTube demo** (required, under 3 minutes with audio) | `[TODO: paste public YouTube URL]` |
