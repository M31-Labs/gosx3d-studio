# WebMCP Challenge implementation

GoSX 3D Studio was already designed around one collaborative scene truth. Human
forms and agent actions both entered the same revision-safe transaction engine,
used stable SceneDoc IDs, and produced attributed receipts. The WebMCP Challenge
work gives browser agents a standards-based seat in that existing collaboration:
they can inspect the scene, help a person find and focus objects, and stage an
exact edit preview inside the open editor. A visible Studio Apply action, which
is not exposed as a WebMCP tool, is what turns that preview into canonical scene
state.

This is a better fit for WebMCP than pixel-level UI automation or a private
agent integration. A compatible browser can discover typed, domain-specific
tools directly from the page through `document.modelContext`, while the artist
keeps the spatial viewport, hierarchy, proposal receipt, and final decision.
Both participants work against the same scene revision rather than exchanging a
prompt and a detached file.

## What existed before the Challenge

The comparison baseline is commit
`a216194ebc0f415a011aa780386773c0750bccf9` from July 29, 2026, before the
August 25 submission window opened. That baseline already contained:

- a server-rendered GoSX 3D scene editor with a viewport, hierarchy, inspector,
  timeline, telemetry, and agent collaboration panel;
- a canonical SceneDoc owned by `internal/studio`, with stable entity IDs and a
  monotonically increasing revision;
- one `Workspace.Execute` command path for human and agent operations;
- `propose` transactions that compute a validated preview without rebinding the
  canonical document, and `direct` transactions that commit;
- exact expected-revision checks, deterministic before/after fingerprints,
  actor-attributed receipts, undo/redo, and bounded receipt history;
- authenticated agent HTTP endpoints and machine-readable action descriptors.

In other words, the Studio's human/agent collaboration model is pre-existing.
The Challenge work is not claiming those foundations as new WebMCP work.

## What was added from August 25 to September 3

The Challenge slice turns those foundations into an agent-native browser
experience:

- `public/studio-webmcp.js` registers four structured tools with
  `document.modelContext.registerTool(...)` and adapts their inputs and outputs
  to the existing Studio APIs and SceneDoc model.
- `webmcp.go` adds a deliberately narrow, session-authorized proposal service
  over `Workspace.Execute`; it never creates a second scene model.
- `internal/studio/rules/webmcp-operations.arb` makes the reversible operation
  allowlist executable policy. Successfully staged operations carry passed
  review checks and a trace; a denial prevents staging.
- `app/page.gsx`, `public/studio-webmcp-ui.js`, and `public/styles.css` expose
  tool readiness, agent-requested focus, and a visible proposal-review surface
  outside the registered WebMCP tools in the existing editor.
- `demo.go` provides the shared hosted demo with a visible reset action, outside
  the WebMCP tool surface, into a fresh temporary generation at a monotonically
  newer revision.
- The WebMCP, policy, demo, and browser-contract tests verify the allowlist,
  policy traces, non-mutating preview, exact stored commit, expiry, reset,
  revision behavior, and HTTP authority boundary.

The resulting collaboration loop is:

```text
compatible browser agent
        |
        | document.modelContext tools
        v
same-origin WebMCP adapter ---- read ----> canonical SceneDoc snapshot
        |
        | stage bounded operations at expectedRevision
        v
Workspace.Execute(mode=propose) ----> preview receipt + opaque proposal ID
                                            |
                                            | visible review in Studio
                                            v
                                      visible Apply action
                                      (not a WebMCP tool)
                                            |
                                            v
Workspace.Execute(mode=direct) -----> canonical SceneDoc next revision
```

## Browser tool surface

The tool surface is intentionally small enough for an agent to use reliably and
for a person to understand.

| Tool | What the agent can do | Canonical scene effect |
| --- | --- | --- |
| `scene_get_state` | Read the scene identity and revision, object/component counts, roots, materials, camera, environment, and current selection. | None |
| `scene_find_objects` | Search stable IDs and names, with optional component, visibility, and result-limit filters. | None |
| `scene_focus_object` | Ask the visible Studio UI to focus a known object so the human and agent can discuss the same thing. | None |
| `scene_preview_actions` | Validate and stage 1–12 reversible edits at an exact scene revision, with a human-facing title and rationale. | None until the visible Studio Apply action is used |

The preview tool accepts only three operation kinds:

- `rename-entity`
- `set-transform`
- `assign-material`

Destructive and broad operations such as delete, reparent, project switching,
asset garbage collection, arbitrary field writes, undo, and redo are not in the
WebMCP surface.

A representative registration in `public/studio-webmcp.js` has the shape the
Challenge expects:

```js
document.modelContext.registerTool({
  name: "scene_preview_actions",
  description: "Validate and visibly stage reversible scene actions for human review.",
  inputSchema: { /* exact revision, title, rationale, and bounded operations */ },
  execute: async (input) => { /* validate, preview, and surface the receipt */ }
});
```

## Example collaboration

First, the agent inspects the current revision:

```json
{}
```

Call that input with `scene_get_state`. It returns a concise scene summary rather
than asking the model to infer structure from pixels.

The agent can then locate the board:

```json
{
  "query": "board",
  "types": ["mesh"],
  "visibleOnly": true,
  "limit": 10
}
```

Call that input with `scene_find_objects`, then use the returned stable ID with
`scene_focus_object`:

```json
{
  "objectId": "board"
}
```

Finally, the agent can stage a small change. Replace the example revision `1`
with the integer returned by `scene_get_state`:

```json
{
  "expectedRevision": 1,
  "title": "Clarify the board in the hierarchy",
  "rationale": "Make the focal surface unmistakable while keeping both edits reviewable.",
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

The tool returns a non-applied receipt and an opaque proposal ID. The Studio
displays the title, rationale, actor, revision delta, affected IDs, and
fingerprint. Discarding it leaves the canonical scene unchanged. Applying it
sends only that opaque ID, and the server commits the exact transaction it
previously previewed.

## Trust and safety boundaries

The browser integration is designed as a proposal boundary, not as a new direct
mutation authority.

- **Commit stays out of WebMCP.** There is no agent-callable commit tool.
  Only the visible Studio review UI calls the commit endpoint.
- **Preview is genuinely non-mutating.** The agent transaction uses
  `ModePropose`; the canonical workspace remains at the same revision until a
  visible UI Apply action uses `ModeDirect`.
- **The reviewed payload cannot be swapped in the browser.** The server stores
  the exact validated transaction and gives the page an opaque 128-bit proposal
  ID. Apply submits the ID, not a rewritten operation list.
- **Stale work fails closed.** The stored transaction retains its original
  `expectedRevision`. If somebody changes the scene before approval, the normal
  workspace revision check rejects the commit.
- **Authority requires the same-origin session and CSRF token.** Proposal and
  commit requests use the existing browser session and CSRF protection. The
  adapter uses same-origin requests with same-origin credentials and does not
  expose the Studio's bearer action token to page JavaScript.
- **Inputs are bounded twice.** The adapter validates object IDs, component
  filters, finite vectors, existing objects/materials, locked objects, and JSON
  shapes. The server independently rejects unknown fields, bodies beyond its
  JSON limit, empty or oversized edit batches, unsupported operation kinds,
  and oversized title/rationale fields.
- **The operation boundary is executable and fail-closed.** An embedded rules
  strategy passes only the three reversible operation kinds. A missing,
  malformed, inconsistent, or denying decision stops staging; successful
  proposals return visible check evidence in the review surface.
- **Pending authority is short-lived and bounded.** Proposals expire after 15
  minutes, the in-memory store retains at most 64, and a successful proposal ID
  is removed so it cannot be committed twice.
- **Attribution remains visible.** Preview receipts use `agent://webmcp`; the
  visible approval path uses `human://webmcp-review`. Both flow through the same
  receipt and fingerprint machinery as the rest of the Studio.

The current Challenge deployment is a single-instance shared canonical
workspace with revision-conflict safety. Proposal storage is process-local and
session-owned; it does not claim live presence, CRDT synchronization, or durable
concurrent-user infrastructure. A visible reset action outside the WebMCP tool
surface restores the sample in a fresh temporary generation, advances the
revision to prevent stale-work ABA, and invalidates every staged proposal. The
WebMCP operation allowlist is intentionally smaller than the Studio's internal
command catalog. Those constraints make the demonstrated authority boundary
concrete and testable.

## Why this improves the experience

Before this adapter, a browser agent would have needed to interpret a dense 3D
editor visually, manipulate controls heuristically, or receive a separate API
token and work outside the page. That is fragile for exact scene IDs,
transforms, materials, and revisions—and it removes the person from the moment
where spatial judgment matters most.

With WebMCP, the division of labor is explicit:

- the agent is good at inventorying a complex scene, finding objects by
  semantics, preparing consistent batches, and explaining its rationale;
- the human is good at judging the semantic proposal in its visible spatial
  context and deciding whether it belongs in the scene;
- the Studio is responsible for validation, revision conflicts, deterministic
  receipts, and canonical state.

That makes the web app meaningfully better for both participants. The agent no
longer guesses at controls, and the human does not have to surrender authorship
or copy agent output between tools.

## Local compatible-browser test

The project pins the current GoSX release, `v0.55.1`. From the repository root:

```bash
cp .env.example .env
STUDIO_DEMO_MODE=1 go run .
```

If `.env` already exists, keep it rather than overwriting it. Then:

1. Use Google Chrome 149 or newer.
2. Enable `chrome://flags/#enable-webmcp-testing` and restart Chrome.
3. Open the [local Studio](http://localhost:8080) in a WebMCP-capable agent/browser session. Use
   **Reset shared scene** if another demo run has changed the sample.
4. Confirm the Agent Collaboration panel reports four available tools.
5. Ask the agent to inspect the scene, find `board`, focus it, and stage the
   rename-plus-material example above using the returned revision.
6. Confirm the proposal card and **Agent preview · not committed** badge appear,
   the board finish changes live, and the canonical scene revision does not.
7. Orbit the still-mounted viewport, then choose **Apply staged changes** and
   confirm the name and material reconcile in place while the revision advances
   exactly once.
8. As a separate resilience check, stage a fresh proposal, deliberately reload
   the tab, and confirm the same-session proposal returns without changing the
   canonical revision; discard it before the next demo run.

For the hosted build, the same flow can be tested through ChatGPT's in-app
browser or Chrome with WebMCP testing enabled. The hosted URL must be HTTPS and
reachable without local network access.

### Verified compatible client

The immutable [public deployment](https://gosx3d.m31labs.dev) passed a 162/162
stress run plus 139/139 clean-recording runs at both 1920×1080 and 1440×900 in
Google Chrome 152 on Windows with
`WebMCPTesting,DevToolsWebMCPSupport` enabled. Chrome exposed
its native `Document.modelContext` getter and `ModelContext.registerTool`,
discovered exactly the four Studio tools, and completed inspect, find, focus, a
two-operation non-mutating preview, and visible-UI Apply. Canonical name,
material, and revision stayed unchanged before approval; Apply advanced the
revision exactly once; and the clean flow reset the shared scene afterward.
WebGPU initialized, each run issued zero reload commands and one main-document
request. No failed assertion, runtime exception, console/log error, or HTTP
error was recorded; the loading-failure log contained only canceled best-effort
`/_gosx/client-events` telemetry uploads.

The public health endpoint reported GoSX `0.55.1`. The image was built from
commit `1920e05447bfd5d98bee6b0c2576e9302734d46f` and pinned by digest
`sha256:0ec822b383c8d75536351f7cd6118961340dc93267691b4b67399c74f4774e10`.
Broader tests cover discard, stale rejection, group-scale preview, client-side
light-scale validation, and shared reset. This is native Chrome WebMCP
evidence; it is not a claim that ChatGPT's in-app browser has already been
tested.

Browser-free verification remains part of the repository's evidence floor:

```bash
go run m31labs.dev/gosx/cmd/gosx@v0.55.1 check app/page.gsx
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 fmt internal/studio/rules/webmcp-operations.arb --check
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 check internal/studio/rules/webmcp-operations.arb --strict
go vet ./...
go test ./...
go test -race ./internal/... ./app/... .
go run ./cmd/studio-smoke
go run ./cmd/studio-certify
./scripts/render-build.sh
```

## Fit against the judging criteria

### WebMCP leverage

The page exposes four complementary tools rather than a decorative protocol
hook. They cover discovery, structured search, visible human/agent grounding,
and a non-trivial revision-safe proposal workflow. Typed inputs map to real 3D
domain entities and the implementation uses WebMCP's browser discovery surface
as the bridge into the existing command model.

### Execution

The WebMCP layer sits inside a coherent working editor rather than a standalone
tool-call demo. An agent's proposal appears beside the same hierarchy,
viewport, Inspector, transaction engine, and receipt history a person uses.
Automated tests exercise the server and authority boundaries, including the
executable policy and shared-demo reset, while the repository retains
browser-free scene evidence.

### Potential impact

Scene creators can delegate tedious inspection and exact batch preparation
without giving an agent silent write access. The pattern also generalizes to
other high-context web tools where an agent can prepare work but a person must
make the consequential decision in context.

### Creativity and ambition

The collaboration is not a chatbot bolted beside a canvas. The website itself
becomes the shared protocol surface: agent and human point at the same stable
objects, reason from the same revision, and turn a machine-authored preview into
a visible-UI commit, outside the WebMCP tool surface, with an auditable handoff.
