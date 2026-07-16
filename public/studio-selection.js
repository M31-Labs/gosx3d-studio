(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioSelectionBridge) return;
  window.__gosxStudioSelectionBridge = true;

  function request(url, options) {
    if (window.__gosx && typeof window.__gosx.request === "function") {
      return window.__gosx.request(url, options);
    }
    return fetch(url, options);
  }

  document.addEventListener("gosx:scene3d:input", function (event) {
    var detail = event && event.detail;
    var input = detail && detail.kind === "pick" ? detail.input : null;
    if (!input || input.type !== "click" || !input.selectedID) return;

    var selected = String(input.selectedID);
    var payload = { selected: selected, kind: String(input.selectedKind || "") };
    if (isFinite(input.worldX) && isFinite(input.worldY) && isFinite(input.worldZ)) {
      payload.world = { x: Number(input.worldX), y: Number(input.worldY), z: Number(input.worldZ) };
    }
    if (isFinite(input.targetTriangleIndex)) payload.triangle = Number(input.targetTriangleIndex);
    if (isFinite(input.depth)) payload.depth = Number(input.depth);
    var rayFinite = isFinite(input.rayOriginX) && isFinite(input.rayOriginY) && isFinite(input.rayOriginZ) &&
      isFinite(input.rayDirX) && isFinite(input.rayDirY) && isFinite(input.rayDirZ);
    var rayNonZero = rayFinite && (Number(input.rayDirX) !== 0 || Number(input.rayDirY) !== 0 || Number(input.rayDirZ) !== 0);
    if (rayNonZero) {
      payload.ray = {
        origin: { x: Number(input.rayOriginX), y: Number(input.rayOriginY), z: Number(input.rayOriginZ) },
        direction: { x: Number(input.rayDirX), y: Number(input.rayDirY), z: Number(input.rayDirZ) }
      };
    }
    request("/api/studio/viewport-selection", {
      method: "POST",
      headers: { "Accept": "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    }).then(function (response) {
      if (!response || !response.ok) return;
      return response.json().catch(function () { return null; }).then(function (confirmation) {
        // The canonical CPU query decides the selection; follow its answer so
        // a GPU/CPU disagreement never becomes silent editor selection truth.
        var confirmed = confirmation && confirmation.selected ? String(confirmation.selected) : selected;
        var url = new URL(window.location.href);
        url.searchParams.set("selection", confirmed);
        window.location.assign(url.pathname + url.search);
      });
    }).catch(function (error) {
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("viewport selection", error, {
          scope: "studio", type: "selection", fallback: "hierarchy"
        });
      }
    });
  });
})();
