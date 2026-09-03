# Title

GoSX 3D Studio

## One-line Summary

A 3D studio where browser agents find and stage exact scene edits while artists
preview and approve them in place.

## Problem

3D editors are dense and stateful. An artist can see which object matters; an
agent needs stable identities, exact properties, and the current revision.
Pixel automation guesses through controls, while a detached API pulls the work
out of the artist's spatial context. That leaves a bad choice: manually repeat
the agent's suggestions, or trust an invisible write. The real problem is
keeping person and agent aligned before anything becomes canonical.

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
the visible Studio, and stage a rename plus material assignment. The final
deployed Studio names all four native tools in-app. During review, a
presentation-size viewport card shows the reversible before/after diff and
unchanged canonical revision while the artist orbits the live result. The full
review retains affected IDs, actor, policy evidence, and fingerprint; its
sticky **Apply 2 exact edits** and **Discard** actions remain human-only. There
is intentionally no WebMCP commit tool.

Apply sends only an opaque proposal ID. The server commits the exact previewed
transaction through the existing revision-safe engine. Stale work is rejected.
A persistent viewport outcome records the `R → R+1` handoff, while activity
distinguishes `agent://webmcp` proposal authorship from
`human://webmcp-review` approval. WebMCP turns detached advice into a shared,
inspectable object in the artist's workspace.

### Existing project and Challenge-period work

GoSX 3D Studio existed before August 25, 2026. It already had its SceneDoc
model, 3D workbench, stable IDs, fingerprints, undo/redo, revision conflicts,
and a shared transaction path for human and agent commands. Those foundations
are not claimed as Challenge work.

During the submission period we added the WebMCP adapter; structured
inspection, search, and focus; session-owned non-canonical proposals; Arbiter
policy; the visible review and exact-commit boundary; typed-call evidence;
shared-demo reset; focused tests; and deployment materials. We also added
same-document Scene3D reconciliation, WebGPU-first rendering, a compact
evidence dock, and a refined PBR sample scene. The final polish serializes demo
reset/status checks and keeps camera shortcuts from stealing hierarchy keyboard
actions.

## Why This Matters

- **WebMCP Leverage:** WebMCP spans the whole handoff: discovery, canonical
  inspection, stable-ID search, visible focus, a typed proposal, and structured
  receipts. It is the product interaction model, not a badge or generic CRUD
  wrapper.
- **Execution:** The tools operate on the real SceneDoc, hierarchy, Inspector,
  live viewport, command history, and transaction engine. Focus, preview,
  approval, and discard reconcile in the same document without tearing down
  the Scene3D canvas. There is no parallel demo scene or canned response path.
- **Potential Impact:** Technical artists can delegate hierarchy search and
  batch preparation without surrendering spatial judgment or repeating every
  proposed edit by hand.
- **Creativity & Ambition:** This is not a chatbot beside a canvas. Person and
  agent point at the same stable object, reason from the same revision, and
  create an auditable handoff from machine-prepared preview to human-approved
  canonical state.

## How We Used AI

A WebMCP-capable browser agent interprets the artist's goal, chooses the page
tools, finds relevant objects, and prepares a typed proposal with rationale.
The Studio retains schema validation, revision checks, policy, fingerprints,
authority, and canonical state; it does not hide a second chat backend or give
the agent a privileged write path.

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
- In-app native-tool disclosure, a presentation-size reversible viewport diff,
  sticky human-only approval, and a persistent revision outcome.
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

1. Open [gosx3d.m31labs.dev](https://gosx3d.m31labs.dev) in ChatGPT's in-app
   browser, or in Chrome 149+ with
   `chrome://flags/#enable-webmcp-testing` enabled. No credentials are required.
2. Click **Reset shared scene**, confirm the warning, and wait for **Agent tools
   ready · 4 tools**.
3. In an agent-enabled client, click **Copy demo prompt** and send it unchanged.
   In Chrome alone, open DevTools → Application → WebMCP and invoke
   `scene_get_state`, `scene_find_objects`, `scene_focus_object`, and
   `scene_preview_actions` in that order; use the returned revision and the
   schemas shown in the panel.
4. Confirm **Inspect → Find → Focus → Stage** completes and the hierarchy
   and Inspector focus stable ID `board`.
5. Confirm the staged card shows both edits, `agent://webmcp`, Arbiter Allow,
   the fingerprint, and canonical revision `R` unchanged.
6. Orbit the live Brushed Steel preview, then click **Apply 2 exact edits**.
7. Confirm `Launch Board`, `Brushed Steel`, revision `R+1`, and separate
   agent proposal and human approval activity.

The preceding immutable public GoSX `v0.55.1` build (commit
`1920e05447bfd5d98bee6b0c2576e9302734d46f`, image digest
`sha256:0ec822b383c8d75536351f7cd6118961340dc93267691b4b67399c74f4774e10`)
passed a **162/162 stress run** plus **139/139 clean-recording runs** at both
1920×1080 and 1440×900 in native Windows Chrome 152. Chrome exposed native
`document.modelContext`, discovered exactly four tools, used WebGPU, stayed in
one main document with zero reload commands, applied the reviewed transaction
exactly once through the visible human action, and returned to a clean shared
scene. No failed assertion, runtime exception, console/log error, or HTTP error
was recorded.

The preceding final-polish deployment passed a separate public acceptance in native
Windows Chrome 152 at 1920×1080 and 1440×900. It used WebGPU, rendered 145
meshes, and exposed exactly four tools. From revision 1 it staged exactly
`Board → Launch Board` and
`Carved Wood (board-material) → Brushed Steel (board-steel-material)`; the
visible human action advanced the scene to revision 2 and left a persistent
approval outcome. Evidence remained **31/31 · current**, PROPOSED and APPROVED
activity stayed paired, and the same canvas remained mounted through five
managed same-document transitions. The guarded flow recorded zero new
top-level document loads, zero camera writes from hierarchy keyboard input, and
no failures. This acceptance claim is for native Windows Chrome; ChatGPT's
in-app browser was not used for it.

The current compact-review deployment passed a fresh native Windows Chrome
`152.0.7977.76` acceptance at 1920×1080. Chrome exposed exactly four native
tools, WebGPU rendered 145 meshes, and Inspect, Find, Focus, and Stage prepared
the exact two-edit proposal while canonical revision 2 remained unchanged.
Visible approval advanced the scene once to revision 3, retained matching
PROPOSED/APPROVED activity and **Evidence 31/31 · current**, and preserved the
same mounted canvas with zero document reloads across three managed
same-document transitions. A visible confirmed reset restored the clean Board
at revision 4 for judges.

Local run and the full automated verification floor are documented in the
public README and `docs/native-webmcp-qa.md`.

## Public Demo Link

[GoSX 3D Studio live demo](https://gosx3d.m31labs.dev)

The live URL serves the immutable GoSX `v0.55.1` image built from main merge
commit `1e68684cbe6d5a8a10257c1ea9b85e40c9046125`, pinned by digest
`sha256:facffa246e9377d49db38c5d0c008e4e6db1a593f10e1e5ebfaff8f2475e9b9c`.
Its public health endpoint reports `0.55.1`.

## Public Repository Link

[M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio) — MIT

The repository is public on its default `main` branch, and the required source,
MIT license, gallery assets, and testing instructions are available without
authentication. The deployed judge-clarity implementation is main merge commit
`1e68684cbe6d5a8a10257c1ea9b85e40c9046125`.

## Demo Video

`[OWNER INPUT REQUIRED: PUBLIC YOUTUBE URL — UNDER 3 MINUTES WITH AUDIO]`

The 2:28 plan in `docs/demo-video-script.md` covers four native page tools,
reversible preview, policy, visible Apply, and separate agent/human attribution
in one session. Its Windows-only helper invokes the four tools but never Apply,
then rejects the take unless it observes the exact `R+1` scene, **Evidence
31/31 · current**, matching **PROPOSED/APPROVED** activity on one proposal
token, the same mounted WebGPU canvas, and zero new top-level document loads.

Suggested YouTube title: **GoSX 3D Studio — Human-Gated 3D Editing with
WebMCP**

Suggested YouTube description:

> GoSX 3D Studio turns a 3D editor into a shared surface for artists and
> browser agents. Four typed WebMCP tools inspect, find, focus, and stage an
> exact revision-safe proposal; the artist reviews the live result and keeps
> the only Apply action. Live demo: [GoSX 3D Studio](https://gosx3d.m31labs.dev)
> · MIT source: [M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio)

## Screenshot Shot List

These high-resolution frames come from the immutable public deployment in
native Windows Chrome:

1. [`docs/assets/webmcp-clean.png`](docs/assets/webmcp-clean.png) — viewport,
   hierarchy, Inspector, revision, WebGPU, and **Agent tools ready · 4 tools**.
   Caption: “One scene, visible to both human and browser agent.”
2. [`docs/assets/webmcp-staged-proposal.png`](docs/assets/webmcp-staged-proposal.png)
   — semantic diff, Arbiter Allow, affected ID, fingerprint, canonical `R`
   unchanged, and Apply/Discard. Caption: “The agent proposes; Apply stays in
   the visible Studio UI.”
3. [`docs/assets/webmcp-human-applied.png`](docs/assets/webmcp-human-applied.png)
   — Launch Board, Brushed Steel, `R+1`, and both actors in Agent Activity.
   Caption: “One reviewed proposal becomes one attributed canonical change.”

## Submission Readiness Notes

- [x] Publish the live Devpost project as **GoSX 3D Studio** with its title,
      tagline, description, Built With list, live URL, public repository, and
      clean-state thumbnail.
- [ ] Upload the three prepared gallery images through Devpost's browser form;
      gallery uploads are not supported by the Devpost MCP.
- [x] Commit, merge, and deploy the current compact-review polish. It keeps
      temporarily locked Save/Undo/Redo controls visible, aligns the proposal
      cleanly at 1024×768 and 1280×800, and keeps the complete four-call rail,
      receipts, diff, and human decision visible at 1440×900.
- [x] Repeat the short public native-WebMCP acceptance after deployment:
      revision 2 → 3, WebGPU, four tools, paired activity, current evidence,
      same canvas, zero document reloads, then a clean reset at revision 4.
- [ ] Record and publish the 2:28 YouTube demo with audio, then verify the app,
      repository, and video while signed out.
- [ ] Supply the six owner-confirmed answers clearly marked below.
- [x] Document existing foundations separately from Challenge-period work.
- [x] Verify local GoSX `v0.55.1` in native Windows Chrome: 162/162, WebGPU,
      four tools, zero reload commands, one main-document request, clean reset.
- [x] Publish the live URL, public repository, and root MIT License.
- [x] Deploy the immutable `v0.55.1` image and repeat the public Windows Chrome
      flow: 162/162 stress and 139/139 clean-recording assertions.
- [x] Capture the three final gallery frames from the public run.
- [x] Commit and merge the final polish to the public default branch.
- [x] Implement the final judge-facing UI and recording guards on the polish
      branch.
- [x] Deploy the final-polish image and repeat public native Windows Chrome
      acceptance at 1920×1080 and 1440×900: WebGPU, 145 meshes, four tools,
      revision 1 → 2, exact staged diffs, persistent approval, Evidence 31/31,
      paired activity, the same mounted canvas, zero document loads, five
      managed same-document transitions, zero keyboard camera writes, and no
      failures.

The public Devpost project is populated at
[devpost.com/software/gosx-3d-studio](https://devpost.com/software/gosx-3d-studio),
but it has not been submitted to The WebMCP Challenge.

## Known Limitations

- Canonical scene and pending proposal state are process-local. The hosted demo
  runs one replica; restart loses that ephemeral state.
- The demo is one shared workspace with revision-conflict safety, not a private
  account workspace or realtime CRDT multiplayer room.
- WebMCP exposes only three reversible operation kinds, and every staged edit
  requires visible human approval.
- The broader editor includes capabilities whose certification state remains
  partial or planned; the demo claim is limited to the verified scene workflow.
- The public YouTube upload and owner-confirmed form answers remain outstanding.

## TODO Official Form Fields

These labels were copied from the live form for The WebMCP Challenge on
September 3, 2026. `OWNER INPUT REQUIRED` marks answers Codex cannot truthfully
choose for the submitter.

| ID | Official field | Paste-ready answer / status |
| --- | --- | --- |
| 28249 | **Submitter Type** (required: Individual / Team of Individuals / Organization) | `[OWNER INPUT REQUIRED: choose one]` |
| 28250 | **Country of residence of yourself and team members if applicable** (required, multi-select) | `[OWNER INPUT REQUIRED: select every applicable country exactly as listed]` |
| 28251 | **If submitting on behalf of an organization, what is the organization name?** (optional) | `[OWNER INPUT REQUIRED IF ORGANIZATION: enter the legal organization name; otherwise leave blank]` |
| 28252 | **App Status** (required: New / Existing) | `Existing` |
| 28253 | **If Existing, explain what you updated during the submission period. (We recommend explaining this in your text description, too!)** (optional) | `GoSX 3D Studio existed before August 25 with its SceneDoc workbench and revision-safe human/agent transaction engine. During the Challenge we added four webpage-registered WebMCP tools for canonical inspection, stable-ID search, visible focus, and bounded non-canonical proposals; session-owned review; executable Arbiter policy evidence; a visible human-only exact-commit boundary; typed-call receipts and actor attribution; same-document Scene3D reconciliation; WebGPU-first rendering; and a polished PBR sample scene. The final deployed polish adds an in-app native-tool disclosure, a presentation-size reversible diff, sticky human-only approval, a persistent revision outcome, race-safe demo reset, conflict-free hierarchy keyboard control, and a recording verifier for the exact handoff. We do not claim the pre-existing editor or transaction foundations as Challenge work.` |
| 28254 | **Live URL that judges can access using ChatGPT's in-app browser or Google Chrome with WebMCP enabled** (required) | [GoSX 3D Studio live demo](https://gosx3d.m31labs.dev) |
| 28255 | **If applicable, testing instructions for application - If you have credentials for your URL, you can put them here.** (optional/private) | `No credentials required. Reset the shared scene and wait for Agent tools ready · 4 tools. In an agent-enabled client, use Copy demo prompt. In Chrome, the same four tools can be invoked from DevTools → Application → WebMCP. Confirm Inspect → Find → Focus → Stage, inspect the non-committed preview at canonical revision R, then click the visible Apply 2 exact edits action and confirm Launch Board, Brushed Steel, and revision R+1.` |
| 28256 | **URL to your PUBLIC Code Repo (on Github, Gitlab, or Bitbucket)** (required) | [Public GitHub repository](https://github.com/M31-Labs/gosx3d-studio) |
| 28257 | **Which agent(s) or client(s) did you test your WebMCP tools with?** (required) | `Google Chrome 152 on Windows with WebMCPTesting and DevToolsWebMCPSupport. The preceding immutable public GoSX v0.55.1 build passed a 162/162 stress run plus 139/139 clean-recording runs. The preceding final-polish deployment passed a separate acceptance at 1920×1080 and 1440×900 using native document.modelContext, WebGPU, 145 meshes, exactly four tools, an exact revision 1→2 staged-and-approved handoff, current 31/31 evidence, paired activity, the same mounted canvas, zero new document loads, five managed same-document transitions, zero hierarchy-keyboard camera writes, and no failures. The current compact-review deployment then passed a fresh 1920×1080 acceptance with revision 2→3, the same four tools/WebGPU/145-mesh boundary, paired activity, current evidence, the same canvas, zero document reloads, three managed same-document transitions, and a clean reset at revision 4. ChatGPT in-app testing was not completed and is not claimed.` |
| 28258 | **Which AI tools have you leveraged while working on this project?** (required) | `OpenAI Codex for repository analysis, implementation, test authoring, debugging, verification, documentation, and submission preparation. Buckley generated and pushed the final structured commit, and Buckbot ran the pre-merge review. [OWNER INPUT REQUIRED: confirm this is the complete AI-tool list, or add every other AI tool used.]` |
| 28259 | **Describe the level of learning you/your team derived from the project** (required: None / Moderate / Significant) | `[OWNER INPUT REQUIRED: choose one]` |
| 28260 | **Did you gain AI value that you can use in your career?** (required: Yes / No) | `[OWNER INPUT REQUIRED: choose one]` |
| — | **Public YouTube demo** (required deliverable, under 3 minutes with audio) | `[OWNER INPUT REQUIRED: paste the public YouTube URL]` |
