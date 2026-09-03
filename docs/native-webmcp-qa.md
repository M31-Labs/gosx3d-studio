# Native WebMCP verification

Completed September 2, 2026 evidence covers the local GoSX `v0.55.1`
release candidate in native Windows Chrome and an earlier deployed
[public demo](https://gosx3d.m31labs.dev) pass. The final immutable image and
public-origin `v0.55.1` replay remain explicit follow-up gates.

## Client

- Google Chrome stable `152.0.7977.65` on Windows
- `WebMCPTesting,DevToolsWebMCPSupport`
- Native `Document.modelContext` getter and native
  `ModelContext.registerTool`; no injected compatibility object
- Public GoSX `v0.55.1` pinned in the local release candidate, including the
  upstream responsive `fillHeight` stabilization
- GoSX `v0.55.0` on the latest completed public-origin verification

This verifies the compatible Google Chrome path accepted by the Challenge.
ChatGPT's in-app browser remains useful optional cross-client assurance.

## WebMCP result

Chrome discovered and invoked exactly these four webpage tools:

1. `scene_get_state`
2. `scene_find_objects`
3. `scene_focus_object`
4. `scene_preview_actions`

The preview tool schema exposed exactly three operation kinds:
`rename-entity`, `set-transform`, and `assign-material`.

The local GoSX `v0.55.1` release-candidate run passed 162/162 acceptance
assertions and proved this same-document flow:

- native state inspection, stable-ID search, and visible focus;
- a two-operation `Board` to `Launch Board` rename plus dedicated `Brushed
  Steel` (`board-steel-material`) board-finish preview;
- all four visible tool-flow steps completing;
- unchanged canonical name, material, and revision before approval;
- a visible Apply action outside the registered tool surface that advanced the
  canonical revision exactly once and applied both reviewed changes;
- a query-free reset that cleared the browser focus and left a deterministic
  non-Board selection, so the recorded agent must genuinely find Board;
- matching short plan tokens for the staged and approved activity entries,
  with both operation kinds retained in their summaries;
- zero runtime exceptions, console errors, failed requests, or HTTP error
  responses during the complete flow;
- non-mutating affine group-scale previews in the broader native QA suite;
- a semantic scale diff (`1.00, 1.00, 1.00 → 1.05, 1.00, 0.95`);
- Arbiter Allow evidence for the bounded group-scale operation;
- adapter input-validation rejection of meaningless light scale, not an
  Arbiter Deny claim;
- discard with no canonical revision change;
- stale proposal rejection and distinct proposal/UI-approval attribution; and
- a coordinate-clicked, browser-confirmed visible reset back to the sample
  scene.

Apply and reset reconciled the hierarchy, Inspector, footer, and Agent
Collaboration panel in place. The primary demo flow stayed in one main
document: it required no page reload or second main-document request, and it
preserved the mounted Scene3D surface and camera. Deliberate full-reload
persistence is a separate historical recovery test documented below, not a
step in the demo path.

## Core Studio result

The same native run also verified that the product surrounding WebMCP is real,
not a protocol-only fixture:

- Scene3D negotiated its WebGPU-first path in native Windows Chrome, reached
  ready/revealed state, and used neither the WebGL capability fallback nor a
  DOM fallback in that run;
- the canonical selected object seeded GoSX's shared selection signal;
- Select displayed no transform helper, while Move, Rotate, and Scale rendered
  three visibly distinct helper sets;
- a real canvas coordinate pick posted through the CSRF-protected selection
  route and synchronized the URL, hierarchy, Inspector, and engine signal; and
- runtime exceptions, console errors, log errors, and HTTP responses at or
  above 400 were all empty.

The local GoSX `v0.55.1` run also confirmed this material contract:

- glossy Coral Pieces use lit Standard PBR with authored
  color `#c8321f`, roughness `0.32`, and clearcoat `0.65`;
- the default polished board face uses Standard PBR with color `#76513a`,
  roughness `0.38`, and clearcoat `0.42`, while the thin Carved Grain Inlay
  retained the portable Selena `CarvedWood` surface program.

### Historical explicit-reload persistence (GoSX v0.54.2)

The public `v0.54.2`-backed build was reloaded with `ignoreCache: true` in
native Windows Chrome 152.0.7977.65. Browser-side health reported `0.54.2`;
Scene3D completed renderer negotiation and settled on its then-current WebGL
path. All four WebMCP tools registered, and Runtime, Log, Network, and HTTP
error collections were empty. This older run remains evidence only for the
explicit recovery boundary; the current WebGPU and material contracts are the
ones listed above.

The same native Windows Chrome 152.0.7977.65 run completed Inspect, Find, Focus,
and Stage, resolved stable ID `board` among 150 entities, and staged the atomic
`Launch Board` plus `Brushed Steel` proposal. Canonical revision 3, name
`Board`, and material `board-material` remained unchanged during review. The
proposal and visible Apply control survived a full same-session reload,
displayed `Arbiter · Allow · 2/2`, and committed once through the visible human
Apply button. The human-attributed receipt advanced revision 3 to 4 and
assigned `board-steel-material`; the smooth machined rim, dark fillet, chassis,
and sockets remained independent. The run then restored a clean scene at
revision 5. Runtime exceptions, console errors, failed requests, and HTTP error
responses were all empty.

## Repository evidence floor

The source tree pinned to the public GoSX `v0.55.1` module passes:

```bash
go run m31labs.dev/gosx/cmd/gosx@v0.55.1 check app/page.gsx
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 fmt internal/studio/rules/webmcp-operations.arb --check
go run m31labs.dev/arbiter/cmd/arbiter@v1.9.0 check internal/studio/rules/webmcp-operations.arb --strict
go mod verify
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./internal/... ./app/... .
go run ./cmd/studio-smoke
go run ./cmd/studio-certify
./scripts/render-build.sh
```

The completed GoSX `v0.55.0` production/TLS run exercised the deployed
`https://gosx3d.m31labs.dev` origin through Chrome's native WebMCP surface with
no injected compatibility object. WebGPU initialized first, exactly four tools
completed inspect/search/focus and a governed Brushed Steel preview, and the
visible human Apply advanced the canonical revision once. The successful path
used in-place reconciliation without a reload or second main-document request;
the live root remained dynamic and `no-store`, with session/CSRF protection.

GoSX `v0.55.1` is the final target because its upstream `fillHeight` fix derives
responsive height from the mount's layout box instead of feeding a stale canvas
aspect ratio back into resize. That keeps grid- and flex-bound Scene3D canvases
inside the workbench during hydration and soft navigation. The local acceptance
run used that pin; this paragraph does not assert a final immutable image or
public-origin `v0.55.1` pass.
