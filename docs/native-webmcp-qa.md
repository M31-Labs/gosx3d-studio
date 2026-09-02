# Native WebMCP verification

Verified September 2, 2026 across the current source tree, the immutable
release image, the canonical local server in native Windows Chrome, and a
separate deployed [public demo](https://gosx3d.m31labs.dev) run.

## Client

- Google Chrome stable `152.0.7977.65`
- `WebMCPTesting,DevToolsWebMCPSupport`
- Native `Document.modelContext` getter and native
  `ModelContext.registerTool`; no injected compatibility object
- GoSX `v0.54.1` for the post-release local run; the earlier deployed-origin
  pass used `v0.54.0` before the public app rollout

This verifies the public Google Chrome path accepted by the Challenge. A final
manual replay in ChatGPT's in-app browser remains a separate submission gate.

## WebMCP result

Chrome discovered and invoked exactly these four webpage tools:

1. `scene_get_state`
2. `scene_find_objects`
3. `scene_focus_object`
4. `scene_preview_actions`

The preview tool schema exposed exactly three operation kinds:
`rename-entity`, `set-transform`, and `assign-material`.

The latest release-candidate run proved:

- native state inspection, stable-ID search, and visible focus;
- a two-operation `Board` to `Launch Board` rename plus `Cobalt Pieces`
  material preview;
- all four visible tool-flow steps completing;
- unchanged canonical name, material, and revision before approval;
- restoration of the same session-owned proposal and Apply/Discard controls
  after a full page reload;
- a visible Apply action outside the registered tool surface that advanced the
  canonical revision exactly once and applied both reviewed changes;
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

The Apply and reset both refreshed the rendered hierarchy, Inspector, footer,
and Agent Collaboration panel through managed GoSX navigation. Neither required
a page reload or a second main-document request.

## Core Studio result

The same native run also verified that the product surrounding WebMCP is real,
not a protocol-only fixture:

- Scene3D mounted a marked `623 × 518` canvas through WebGL with GPU rendering,
  ready/revealed state, and no DOM fallback;
- the canonical selected object seeded GoSX's shared selection signal;
- Select displayed no transform helper, while Move, Rotate, and Scale rendered
  three visibly distinct helper sets;
- a real canvas coordinate pick posted through the CSRF-protected selection
  route and synchronized the URL, hierarchy, Inspector, and engine signal; and
- runtime exceptions, console errors, log errors, and HTTP responses at or
  above 400 were all empty.

Three best-effort client-event telemetry requests were canceled by managed DOM
morphs. They produced no HTTP error response and did not affect the application
or the verification result.

### GoSX v0.54.1 solid-material regression proof

After the governed `v0.54.1` release, the canonical local server was restarted
against the public module and reloaded with `ignoreCache: true` in native
Windows Chrome 152. Browser-side health reported `0.54.1`, Scene3D was
ready/revealed on WebGPU, all four WebMCP tools registered, and Runtime, Log,
Network, and HTTP error collections were empty.

The exact runtime census proved that all 133 typed standard-material objects
carried `wireframe: false` and triangle geometry: the board had 576 vertices,
all 121 sockets were filled cylinders, and all ten Player 4 pieces were filled
spheres. The ten custom Selena Player 1 pieces remained valid and solid. The
same run completed Inspect, Find, and Focus, staged the atomic `Launch Board`
plus `Cobalt Pieces` proposal without mutating revision 1, displayed
`Arbiter · Allow · 2/2`, and committed once through a real pointer click on the
visible Apply button. The human-attributed receipt advanced revision 1 to 2;
the final board remained a solid 576-vertex PBR surface.

## Repository evidence floor

The final source tree also passes:

```bash
go run m31labs.dev/gosx/cmd/gosx@v0.54.1 check app/page.gsx
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

The packaged production-mode server then passed a clean 1280 × 800 Chrome load
behind local TLS with a real WebGL canvas, exact page fit, and no page, console,
or network errors. The live root was dynamic and `no-store`, issued a
`Secure; HttpOnly; SameSite=Lax` session cookie, and accepted the
CSRF-protected canvas selection without a reload. No session-bound page is
included in the static export.

A second production/TLS run exercised the deployed
`https://gosx3d.m31labs.dev` origin through Chrome's native WebMCP surface with
no injected compatibility object. It discovered exactly four tools, completed
inspect/search/focus, and staged a governed two-operation preview through the
secure session/CSRF boundary. The same proposal and visible Apply control
survived a full reload. Apply changed `Board` to `Launch Board`, assigned
`Cobalt Pieces`, and advanced the canonical revision exactly once. The run then
reset the shared scene, and its runtime, page, console, network-failure, and
HTTP-error lists were empty.
