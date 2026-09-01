# WebMCP Challenge demo video script

Target runtime: **2:43**. Keep the final public YouTube upload under three
minutes and record the application plus the WebMCP-capable agent in one frame.
The bracketed revision values are cues, not lines to read literally.

## Before recording

- Start the hosted build in demo mode. Click **Reset shared scene**, confirm the
  warning, and verify that the hierarchy contains `Board` with stable ID
  `board`. Do this before recording so the take starts from a known revision.
- Open the Studio in the compatible browser you will demonstrate. Wait until
  **Agent Collaboration** reads **Agent tools ready** and **4 tools**.
- Keep **REVISION**, **Latest staged proposal**, and **Agent Activity** visible
  or reachable without hunting. Record the starting revision as `[R]`.
- Put the three prompts below in a scratchpad for reliable copy/paste. Hide the
  scratchpad, notifications, credentials, and developer tooling before capture.
- Use a 1080p canvas, a readable browser zoom, a visible cursor, and one clear
  microphone. Leave two seconds of stillness at the beginning and end.

## One-take run of show

### 0:00–0:20 — The human problem

**On screen:** Begin on the full Studio: dense Scene Hierarchy, 3D viewport,
Inspector, **Agent Collaboration**, and the current **REVISION**.

**Voiceover:**

> A dense 3D scene is easy for a person to see, but hard for an agent to
> understand safely. This scene has a board, pieces, lights, materials,
> transforms, and stable identities; a visual guess can target the wrong thing.
> The agent should not guess from pixels, and it should not silently edit the
> scene. Here, both work against one canonical SceneDoc, with the human keeping
> commit authority.

### 0:20–0:37 — Discover the WebMCP surface

**On screen:** Point to **Agent tools ready** and **4 tools**. In the agent tool
list, briefly show `scene_get_state`, `scene_find_objects`,
`scene_focus_object`, and `scene_preview_actions`.

**Voiceover:**

> The Studio exposes four structured WebMCP tools: inspect scene state, find
> objects, focus one object, and preview bounded actions. There is deliberately
> no agent-callable commit tool. Because these are webpage-declared contracts,
> the agent discovers them in context.

### 0:37–0:59 — Inspect, then find

**On screen:** Send this exact prompt:

> Inspect the current scene with `scene_get_state`. Then use
> `scene_find_objects` to find visible mesh objects matching "board". Do not
> propose changes yet.

Show the returned revision `[R]` and stable ID `board`.

**Voiceover:**

> I will ask the agent to inspect the scene, then find the visible mesh named
> board. It reads revision [R] and returns the stable object ID `board`, instead
> of inferring either from the interface.

### 0:59–1:18 — Establish shared visible context

**On screen:** Send:

> Use `scene_focus_object` on object ID `board`. Do not propose or mutate
> anything.

Show the focused row in **Scene Hierarchy**, the **Inspector** selection, and
the unchanged viewport **REVISION** `[R]`.

**Voiceover:**

> Now I ask it to focus that object. The Scene Hierarchy and Inspector move to
> Board, so the person and agent share the same visible target. Focus changes
> only UI selection; the canonical scene is still revision [R].

### 1:18–1:49 — Stage a non-mutating preview

**On screen:** Send:

> At the exact current revision, use `scene_preview_actions` to stage one
> `rename-entity` operation: target `board`, name `Hero Board`. Title:
> `Clarify the board in the hierarchy`. Rationale: `A more descriptive name
> makes the shared focal object easier to find.` Do not do anything else.

In **Latest staged proposal**, point to the rationale, semantic change
`Board → Hero Board`, **Actor**, **Policy**, **Revision**, **Affected**, and
**Result fingerprint**. Then point back to viewport **REVISION** `[R]`.

**Voiceover:**

> Next, I ask for a rename preview: Board to Hero Board, with a short
> rationale. The Studio shows the staged proposal, semantic change, affected
> object, proposed revision, and result fingerprint. Arbiter visibly allows the
> reversible operation before the receipt is staged as `agent://webmcp`.
> Notice the canonical revision remains [R].

### 1:49–2:04 — Exercise human rejection

**On screen:** Click **Discard**. Hold on the cleared proposal card, the status
message, and unchanged **REVISION** `[R]`.

**Voiceover:**

> I will discard this first proposal. The review card clears, and the message
> confirms the canonical scene was never changed. Revision remains [R].

### 2:04–2:31 — Restage and accept

**On screen:** Send the same preview prompt again. When the proposal appears,
move the cursor from the agent pane into the Studio and click
**Apply staged changes** yourself. After refresh, show `Hero Board` and
**REVISION** `[R+1]`.

**Voiceover:**

> I will ask for the same proposal again. This time, I—not the agent—click
> Apply staged changes. The server commits the exact operations behind the
> reviewed opaque proposal ID. The page refreshes: Board is now Hero Board, and
> the canonical revision advances once, from [R] to [R+1].

### 2:31–2:43 — Close on attribution and safety

**On screen:** In **Agent Activity**, frame the `agent://webmcp` propose entry
and the `human://webmcp-review` direct entry together. End on the Studio.

**Voiceover:**

> Agent Activity attributes the preview to `agent://webmcp` and the accepted
> change to `human://webmcp-review`. If the scene changed before approval, the
> expected-revision check would reject the stale commit. The collaboration is
> inspect, align, propose, and review—not guess, click, and hope.

## Backup notes

- If tool discovery does not complete, stop the take, confirm the browser's
  WebMCP support is enabled, reload, and wait for **Agent tools ready**. Do not
  claim a browser was verified unless that exact recording proves it.
- If a tool call times out or the page reloads before review, restage from the
  current revision. Never splice a response from a different revision into the
  take.
- **Discard** clears the visible review without changing the canonical scene.
  If you accidentally click **Apply staged changes** on the first proposal,
  stop the take and use **Reset shared scene** before recording again; do not
  use Undo because that adds another canonical revision.
- If `Board` was renamed before recording, use **Reset shared scene** before
  capture. Do not improvise a second name; the prepared prompts and visible
  semantic change should agree.
- If the agent adds commentary, let it finish, then show only the structured
  result needed for the shot. Accuracy matters more than filling every second.

## Final shot checklist

- [ ] Spoken runtime is below 2:45; uploaded video remains below 3:00.
- [ ] Audio is clear, cursor is visible, and all text shown is readable.
- [ ] **Agent tools ready** and **4 tools** appear on screen.
- [ ] All four exact WebMCP tool names are visible or clearly invoked.
- [ ] Starting revision `[R]`, stable ID `board`, and focused UI context appear.
- [ ] The first preview shows rationale, semantic change, Arbiter policy,
      fingerprint, and `agent://webmcp`, then **Discard** leaves revision `[R]`
      unchanged.
- [ ] The second proposal is accepted by a visible human click on
      **Apply staged changes**.
- [ ] `Hero Board` and exactly one canonical advance to `[R+1]` are visible.
- [ ] **Agent Activity** shows both `agent://webmcp` and
      `human://webmcp-review` attribution.
- [ ] No credentials, localhost-only claims, unsupported features, or private
      notifications appear.
