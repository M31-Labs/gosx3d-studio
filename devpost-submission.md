# Title

GoSX 3D Studio

## One-line Summary

An agent-native 3D scene workbench where browser agents inspect and stage
revision-safe edits while approval stays in a visible Studio UI action outside
the four-tool WebMCP surface.

## Links

- **Live app:** [GoSX 3D Studio](https://gosx3d.m31labs.dev)
- **Source:** [M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio)
- **License:** MIT

## Problem

3D editors are dense, stateful environments. A person can look at a viewport
and understand which board, light, material, or game piece matters, but an agent
needs stable object identities, exact transforms, and the current scene revision
to act reliably. Pixel-level automation asks the agent to guess its way through
controls. A detached API avoids the pixels but pulls the work away from the
artist's spatial context and can make agent changes feel invisible or unsafe.

That leaves scene creators with a poor choice: perform every mechanical lookup
and edit themselves, or give an agent broad write authority and hope it touched
the intended object. The difficult part is not generating a rename or transform.
It is keeping the human and agent aligned on the same object, the same revision,
and the same proposed result before anything becomes canonical.

## Solution

GoSX 3D Studio exposes four focused tools directly from the webpage through
`document.modelContext.registerTool(...)`:

- `scene_get_state` reads a concise canonical scene overview and revision.
- `scene_find_objects` searches stable IDs and names with typed filters.
- `scene_focus_object` moves the visible hierarchy and Inspector to the object
  the agent is discussing, without changing canonical scene state.
- `scene_preview_actions` validates and stages 1–12 bounded, reversible scene
  operations at an exact revision.

The result is one visible collaboration loop. An agent can inventory a complex
scene, locate the right object, bring it into shared visual focus, and prepare a
proposal with a rationale. The Studio shows the semantic change, affected IDs,
revision delta, actor, and deterministic result fingerprint. The proposal is
non-mutating. A person can discard it or explicitly click **Apply staged
changes**. There is intentionally no WebMCP commit tool.

A persistent typed-call trace shows what the browser agent accomplished at each
step: the inspected revision and entity count, the stable object it found, the
non-mutating focus request, and the number of operations staged. A 148-object
scene becomes one intent, four typed calls, two exact edits, and one human
approval. Technical artists can delegate hierarchy search and batch preparation
without surrendering scene authority.

When a person approves, the browser submits only an opaque proposal ID. The
server commits the exact operations it previously previewed through the same
revision-safe transaction engine used by the rest of the Studio. If the scene
changed in the meantime, the expected-revision check rejects the stale work.
Before staging, an embedded Arbiter policy evaluates every operation.
Successful proposals carry visible Allow evidence; a Deny, invalid decision, or
inconsistent policy result prevents staging.

GoSX 3D Studio is an **existing project**. Before August 25, 2026, it already
had the SceneDoc model, 3D workbench, shared human/agent transaction engine,
propose/direct modes, stable IDs, fingerprints, attributed receipts, revision
conflicts, and undo/redo. The Challenge work does not claim those foundations
as new. During the submission period, the project added the four-tool WebMCP
adapter, session-scoped proposal and review boundary, visible browser-agent
focus and review experience, bounded proposal storage, WebMCP-specific tests,
an executable Arbiter policy, a safe shared-demo reset, documentation, and
deployment configuration. The Studio was also updated to GoSX v0.54.0 and its
new affine group-scale path is exercised through SceneDoc, nested prefabs,
preview evidence, exact picking, and gizmo commits.

## Why This Matters

- **WebMCP Leverage:** This is a four-tool domain workflow, not a decorative
  protocol hook. WebMCP handles discovery, structured scene inspection, typed
  search, visible human-agent grounding, and a revision-safe proposal handoff.
- **Execution:** The tools operate inside a substantial 3D workbench and feed
  its real SceneDoc, hierarchy, Inspector, proposal card, command history, and
  transaction engine. The browser integration does not maintain a second demo
  scene or return canned edits.
- **Potential Impact:** In the hosted 148-object scene, one intent becomes four
  typed calls, two exact edits, and one human approval. Technical artists can
  delegate hierarchy search and batch preparation without surrendering the
  moment of spatial judgment. The same proposal boundary could help other
  high-context web tools where agents prepare consequential work and people
  approve it in place.
- **Creativity & Ambition:** The agent is not a chatbot bolted beside a canvas.
  The webpage becomes the shared protocol surface. Human and agent point at the
  same stable object, reason from the same revision, and create an auditable
  handoff from machine-authored preview to the visible Studio Apply action.

## How We Used AI

The product does not embed a model or hide an autonomous process behind the UI.
It lets a WebMCP-capable external agent use its reasoning where it helps: turn a
human goal into structured inspection, find relevant objects by meaning, and
prepare a small typed proposal with a plain-language rationale. The Studio
keeps deterministic responsibilities—schema validation, revision checks,
fingerprints, authority, and canonical state—outside the model.

In the prepared demo flow, the agent reads the scene revision, finds the mesh
named `Board`, focuses stable ID `board`, and stages two changes: rename it
`Launch Board` and assign the `Cobalt Pieces` material. The same browser
session recovers the persistent typed-call trace and exact proposal after a full
reload, then the person uses the visible Apply action. No registered WebMCP tool
can commit a proposal.

Native browser QA used Google Chrome 152.0.7977.64 with
`WebMCPTesting,DevToolsWebMCPSupport` enabled. Chrome's native
`document.modelContext` discovered and invoked exactly the four declared tools.
The complete inspect, find, focus, two-operation preview, reload recovery, and
visible Apply flow produced no runtime exceptions, console errors, failed
requests, or HTTP error responses. Before approval, the canonical name,
material, and revision were unchanged. Apply advanced the revision exactly once
and kept agent-preview/UI-approval attribution distinct. The test suite also
covers discard, stale-proposal rejection, policy behavior, and clean shared
reset. ChatGPT's in-app browser has not yet been tested.

## How We Used Codex

The human supplied the product direction: GoSX 3D Studio already let people and
agents converge on revisioned scene truth. For this Challenge, that became a
single-instance shared canonical workspace with revision-conflict safety, and
WebMCP gives browser agents a standards-based way to participate. Codex helped
turn that direction into a bounded implementation by:

- inspecting the existing repository and identifying `Workspace.Execute` as
  the correct convergence point instead of creating a parallel state model;
- implementing and reviewing the browser adapter, proposal service, visible
  review UI, session/CSRF boundary, and deployment scaffolding;
- writing tests for the tool registrations, operation allowlist, non-mutating
  preview, exact stored commit, session ownership, expiry, and authority split;
- running repository checks during iteration and tightening error handling,
  browser status reporting, and documentation; and
- drafting the architecture notes, one-take demo script, and this submission
  packet while keeping unverified claims and missing assets explicit.

Codex did not submit or update the project on Devpost in this drafting pass.

## Key Features

- Four webpage-registered WebMCP tools with strict JSON Schemas and structured
  results.
- Canonical SceneDoc inspection instead of visual guessing.
- Stable-ID object search with component, visibility, and result-limit filters.
- Agent-requested focus that aligns the Scene Hierarchy and Inspector without
  mutating the scene.
- A persistent typed-call trace with concise, real results for inspect, find,
  focus, and stage.
- Preview support for `rename-entity`, `set-transform`, and `assign-material`;
  destructive, structural, and broad operations are excluded.
- A visible review UI, outside the registered WebMCP tool surface, with title,
  rationale, semantic diff, affected IDs, proposed revision, and deterministic
  fingerprint.
- Executable Arbiter policy with visible Allow evidence for every successfully
  staged operation; policy denial, failure, or inconsistency prevents staging.
- Opaque 128-bit proposal IDs: approval commits the exact server-stored
  transaction, not a browser-rewritten payload.
- Expected-revision conflict rejection, one-use proposals, 15-minute expiry,
  session ownership, a 64-proposal cap, and a 12-operation cap.
- Same-origin session and CSRF protection without exposing the Studio's bearer
  automation token to browser JavaScript.
- Visible attribution: preview receipts use `agent://webmcp`; accepted changes
  use `human://webmcp-review`.
- Visible shared-demo reset with revision monotonicity and staged-proposal
  invalidation; it is not exposed as a registered WebMCP tool.

## Architecture

```text
WebMCP-capable browser agent
        |
        | scene_get_state / scene_find_objects / scene_focus_object
        | scene_preview_actions
        v
public/studio-webmcp.js
        |
        | same-origin reads + session/CSRF proposal POST
        v
webmcp.go proposal service
        |
        | Arbiter DecideWebMCPOperation (fail closed + trace)
        v
        | Workspace.Execute(mode=propose, expectedRevision=R)
        v
non-applied receipt + opaque proposal ID ----> visible Studio review card
                                                    |
                                                    | Apply UI action
                                                    | (not a WebMCP tool)
                                                    v
                                          session/CSRF commit POST
                                                    |
                                                    v
                                  Workspace.Execute(mode=direct, revision=R)
                                                    |
                                                    v
                                      canonical SceneDoc revision R+1
```

The pre-Challenge comparison point is commit
`a216194ebc0f415a011aa780386773c0750bccf9` from July 29, 2026. At that point,
`internal/studio` already owned the canonical SceneDoc and transaction engine,
while `app/` rendered the human workbench.

The submission-period slice adds:

- `public/studio-webmcp.js` for the four WebMCP registrations, input
  normalization, same-origin API calls, and structured results;
- `webmcp.go` plus routes in `main.go` for bounded, session-owned staging and an
  exact commit from the visible review UI;
- `webmcp_policy.go` and `internal/studio/rules/webmcp-operations.arb` for the
  executable, trace-producing operation boundary;
- `demo.go` for a visible reset action, outside the WebMCP tool surface, into a
  fresh ephemeral project generation;
- `public/studio-webmcp-ui.js`, `app/page.gsx`, and `public/styles.css` for tool
  readiness, shared focus, proposal review, and activity attribution; and
- the WebMCP, policy, demo, and browser-contract tests for server, authority,
  reset, and public adapter coverage.

Both preview and commit reuse `Workspace.Execute`. The WebMCP layer is an
adapter and authority boundary over the existing scene model, not a replacement
for it.

## Testing Instructions

### Hosted judge flow

The same flow has passed locally through native Chrome 152 WebMCP. Reset first
because the public deployment is one shared ephemeral workspace.

1. Open the [live GoSX 3D Studio](https://gosx3d.m31labs.dev) in ChatGPT's in-app browser or Chrome
   149+ with `chrome://flags/#enable-webmcp-testing` enabled.
2. Click **Reset shared scene**, confirm the warning, then confirm **Agent
   Collaboration** reports **Agent tools ready** and **4 tools**.
3. Click **Copy demo prompt** and give the copied text to the browser agent:
   “Inspect the current scene, find and focus the object named Board, then
   stage—without committing—a proposal that renames it Launch Board and
   assigns the Cobalt Pieces material. Explain the revision boundary.” Record
   the baseline revision `R`.
4. Confirm the **Inspect → Find → Focus → Stage** rail completes, the
   persistent typed-call trace records the real result of each step, and **Scene
   Hierarchy** plus **Inspector** visibly converge on stable ID `board`.
5. Confirm **Latest staged proposal** shows a non-applied
   `agent://webmcp` receipt, Arbiter Allow evidence, semantic diff, affected
   object, proposed revision, and result fingerprint while the canonical
   revision remains `R`.
6. Reload the same browser tab. Confirm the typed-call trace, proposal, and
   Apply/Discard controls return while the canonical scene still reads `R`.
7. Click **Apply staged changes** yourself.
8. Confirm the hierarchy reads `Launch Board`, the Inspector reads `Cobalt
   Pieces`, canonical revision is `R+1`, and
   **Agent Activity** separately shows `agent://webmcp` and
   `human://webmcp-review`.

### Local run

```bash
cp .env.example .env  # only when .env does not already exist
STUDIO_DEMO_MODE=1 go run .
```

Open `http://localhost:8080` in a compatible browser and follow the same flow.

### Automated verification

Run the final clean diff through the repository's evidence floor:

```bash
go run m31labs.dev/gosx/cmd/gosx@v0.54.0 check app/page.gsx
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 fmt internal/studio/rules/webmcp-operations.arb --check
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 check internal/studio/rules/webmcp-operations.arb --strict
go vet ./...
go test ./...
go test -race ./internal/... ./app/... .
go run ./cmd/studio-smoke
go run ./cmd/studio-certify
./scripts/render-build.sh
```

The WebMCP tests specifically cover registration of exactly four tools, the
absence of an agent-callable commit path, the narrow operation allowlist,
non-mutating preview, exact one-revision commit, one-use and expiring proposal
IDs, cross-session rejection, CSRF/session authority, and production cookie
attributes. Demo tests cover reset revision monotonicity, stale and concurrent
requests, path ownership, cross-browser behavior, proposal invalidation, and
non-demo denial. The final source tree passed the complete verification sequence
listed above. The deployed HTTPS origin also passed native four-tool discovery,
inspect/search/focus, a governed non-mutating two-operation preview, full-reload
proposal restoration, visible-UI Apply, exactly one canonical revision advance,
secure session/CSRF transport, reset, and a clean real-WebGL browser smoke.

## Public Demo Link

[GoSX 3D Studio live demo](https://gosx3d.m31labs.dev)

## Public Repository Link

[M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio)

The public default branch contains the complete source, run and verification
instructions, WebMCP implementation, and a repository-root MIT License.

## Demo Video

`[TODO: PUBLIC YOUTUBE URL, UNDER 3 MINUTES, WITH AUDIO]`

The recording plan is in `docs/demo-video-script.md`. It targets a 2:16
one-take demo: problem, four-tool discovery, inspect/find/focus, persistent
typed-call evidence, a two-operation non-mutating preview, same-session reload
recovery, visible-UI Apply, one canonical revision advance, and distinct
proposal/approval attribution. The shared scene is reset before recording so
the opening belongs to the agent's object discovery rather than demo setup.

## Screenshot Shot List

1. **Full workbench and tool readiness** — Scene Hierarchy, 3D viewport,
   Inspector, current **REVISION**, and **Agent Collaboration** showing
   **Agent tools ready · 4 tools**. Caption: “One scene, visible to both human
   and browser agent.”
2. **Structured discovery** — agent results and the persistent typed-call trace
   from `scene_get_state` and `scene_find_objects`, showing revision `R`, object
   name `Board`, and stable ID `board`. Caption: “The agent reads scene truth
   instead of guessing from pixels.”
3. **Shared focus** — highlighted `board` row beside the Inspector selection and
   viewport. Caption: “A WebMCP focus request gives the person and agent the
   same visible referent without mutating the scene.”
4. **Reviewable proposal** — **Latest staged proposal** with rationale,
   `Board → Launch Board` and `board-material → player-4-material` semantic
   changes, `agent://webmcp`, affected ID, Arbiter Allow evidence, revision
   boundary, fingerprint, **Discard**, and **Apply staged changes**.
   Caption: “The agent proposes; Apply stays in the visible Studio UI, outside
   the WebMCP tool surface.”
5. **Auditable handoff** — `Launch Board`, `Cobalt Pieces`, revision `R+1`, and
   **Agent Activity** showing both `agent://webmcp` propose and
   `human://webmcp-review` direct entries. Caption: “One reviewed proposal
   becomes one attributed canonical change.”

Do not capture credentials, localhost-only evidence for the public gallery, a
mock `modelContext` console, or any reset UI until those surfaces are final.

## Submission Readiness Notes

- [x] Existing-project baseline and Challenge-period additions are documented
      separately.
- [x] Four WebMCP registrations, typed schemas, proposal UI, server proposal
      path, visible Apply boundary outside WebMCP, and focused tests exist
      locally.
- [x] README and Challenge implementation notes contain the required
      `document.modelContext.registerTool(...)` shape.
- [x] The Devpost account is registered for The WebMCP Challenge.
- [x] Add a complete MIT `LICENSE` at the repository root.
- [x] Implement and exercise the visible shared-demo reset outside the WebMCP
      tool surface, including revision monotonicity and staged-proposal
      invalidation.
- [x] Complete native WebMCP QA in Google Chrome 152.0.7977.64 with the WebMCP
      testing experiment enabled; exact four-tool discovery and the complete
      inspect/find/focus/two-operation-preview/reload/apply flow passed with no
      browser or HTTP errors.
- [x] Run the full clean verification sequence, including GoSX and Arbiter
      checks, module verification, vet, tests, race tests, deterministic smoke
      and certification evidence, production build, and packaged-server browser
      smoke.
- [x] Deploy one HTTPS instance at `gosx3d.m31labs.dev` and fill the public demo
      URL.
- [x] Publish the final source repository, verify anonymous access and license
      detection, and fill the repository URL.
- [ ] Capture screenshots, record the under-three-minute video with audio,
      publish it on YouTube, and fill the video URL.
- [ ] Fill every required official form answer and update the existing Devpost
      project only after the URLs and proof are ready.

This drafting pass made no create, update, or submit call on Devpost. The live
project list currently shows an `Untitled` project in
`submission_pre_draft` state for this event, not a completed project entry.
This draft did not modify it; do not assume Devpost contains any of the
material above.

## Known Limitations

- Native WebMCP behavior is verified in Google Chrome 152.0.7977.64. ChatGPT's
  in-app browser remains a separate manual submission check.
- Canonical scene and pending proposal state are process-local. The Render
  configuration intentionally uses one instance; a restart loses that state.
- The hosted demo is a single-instance shared canonical workspace with
  revision-conflict safety. It does not implement live presence, CRDT
  synchronization, accounts, or durable cloud workspaces.
- The canonical workspace is shared by the running demo process even though
  pending proposal IDs are session-owned. It is not a private per-user studio.
- WebMCP intentionally exposes only three reversible operation kinds:
  `rename-entity`, `set-transform`, and `assign-material`. Delete,
  reparent, project switching, asset cleanup, arbitrary field writes, undo, and
  redo remain outside the browser-agent surface.
- There is no commit tool in the four-tool WebMCP surface. Every staged edit
  requires the visible Studio Apply action.
- The default demonstration uses the bundled Chinese Checkers scene; the
  broader editor contains capabilities whose certification status remains
  explicitly partial or planned.
- The source release uses the MIT License. The public YouTube video and final
  Devpost form review remain outstanding.

## TODO Official Form Fields

The following fields come from the live submission form for The WebMCP
Challenge. Values in brackets require the submitter's confirmation.

| Official field | Draft answer / status |
| --- | --- |
| **Submitter Type** (required) | `[TODO: choose Individual, Team of Individuals, or Organization]` |
| **Country of residence of yourself and team members if applicable** (required) | `[TODO: select every applicable country exactly as listed in the form]` |
| **If submitting on behalf of an organization, what is the organization name?** (optional) | `[TODO if applicable; otherwise leave blank]` |
| **App Status** (required) | `Existing` |
| **If Existing, explain what you updated during the submission period** | `GoSX 3D Studio existed before August 25 with its SceneDoc editor and shared revision-safe human/agent transaction engine. During the submission period we added a browser WebMCP adapter registering four tools; structured scene inspection, search, and visible focus; a bounded session-owned non-mutating proposal service; an executable Arbiter allow/deny policy with decision traces; a visible review and exact-commit UI outside the WebMCP tool surface; a visible shared-demo reset that is not a WebMCP tool; actor attribution and semantic receipts; WebMCP, policy, reset, authority, server, and adapter tests; a GoSX v0.54.0 upgrade with affine group-scale integration; Challenge documentation; and deployment configuration. The pre-existing editor and transaction foundations are documented separately and are not claimed as new work.` |
| **Live URL that judges can access using ChatGPT's in-app browser or Google Chrome with WebMCP enabled** (required) | [GoSX 3D Studio live demo](https://gosx3d.m31labs.dev) |
| **If applicable, testing instructions for application** (private to Devpost and judges) | `No credentials are required. Open the live URL in a compatible browser, click Reset shared scene and confirm the warning, confirm Agent Collaboration reports four tools, then use Copy demo prompt and follow the Hosted judge flow in this draft.` |
| **URL to your PUBLIC Code Repo** (required) | [M31-Labs/gosx3d-studio](https://github.com/M31-Labs/gosx3d-studio) |
| **Which agent(s) or client(s) did you test your WebMCP tools with?** (required) | `Google Chrome 152.0.7977.64 with WebMCPTesting and DevToolsWebMCPSupport enabled. Chrome exposed its native document.modelContext implementation, discovered all four webpage tools, and completed inspect, search, focus, a two-operation preview, same-session reload recovery, and visible-UI Apply with exactly one canonical revision advance. Separate automated coverage exercises discard, conflict rejection, client validation, policy behavior, and reset. We have not yet tested ChatGPT's in-app browser.` |
| **Which AI tools have you leveraged while working on this project?** (required) | `OpenAI Codex for repository analysis, implementation, test authoring, debugging, verification support, documentation, and submission preparation. [TODO: add any other AI tools actually used; do not list planned tools.]` |
| **Describe the level of learning you/your team derived from the project** (required) | `[TODO: choose None, Moderate, or Significant]` |
| **Did you gain AI value that you can use in your career?** (required) | `[TODO: choose Yes or No]` |

The remaining required deliverable outside those custom questions is a public
YouTube demo under three minutes with audio. The working live URL, public
MIT-licensed source repository, and truthful project description are ready.
