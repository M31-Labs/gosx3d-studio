# Native WebMCP verification

Verified August 31, 2026 against the local demo server and the current source
tree.

## Client

- Chrome for Testing `152.0.7977.64`
- `enable-webmcp-testing@1`
- Native `Document.modelContext` getter and native
  `ModelContext.registerTool`; no injected compatibility object
- GoSX `v0.54.0`

This verifies the Chrome path accepted by the Challenge. It does not claim a
test in ChatGPT's in-app browser or against the not-yet-deployed public URL.

## WebMCP result

Chrome discovered and invoked exactly these four webpage tools:

1. `scene_get_state`
2. `scene_find_objects`
3. `scene_focus_object`
4. `scene_preview_actions`

The run proved:

- state inspection, stable-ID search, and visible focus;
- non-mutating rename and affine group-scale previews;
- a semantic scale diff (`1.00, 1.00, 1.00 → 1.05, 1.00, 0.95`);
- Arbiter Allow evidence for the bounded group-scale operation;
- rejection of meaningless light scale;
- discard with no canonical revision change;
- a coordinate-clicked human Apply that advanced the revision exactly once;
- stale proposal rejection and distinct agent/human attribution; and
- a coordinate-clicked, human-confirmed reset back to the sample scene.

The Apply and reset both refreshed the rendered hierarchy, Inspector, footer,
and collaboration state through managed GoSX navigation. Neither required a
page reload or a second main-document request.

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

## Repository evidence floor

The final source tree also passes:

```bash
go run m31labs.dev/gosx/cmd/gosx@v0.54.0 check app/page.gsx
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

A second production/TLS run used Chrome's native WebMCP surface with no
injected compatibility object. It discovered exactly four tools, completed
state/search/focus, staged a governed rename preview through the secure
session/CSRF boundary, and discarded it with a real coordinate click. The
proposal POST returned 200, Arbiter reported Allow, canonical revision 1 and
`Board` remained unchanged, and runtime, page, console, and HTTP-error lists
were empty.
