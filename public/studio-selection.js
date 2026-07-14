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
    request("/api/studio/viewport-selection", {
      method: "POST",
      headers: { "Accept": "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ selected: selected, kind: String(input.selectedKind || "") })
    }).then(function (response) {
      if (!response || !response.ok) return;
      var url = new URL(window.location.href);
      url.searchParams.set("selection", selected);
      window.location.assign(url.pathname + url.search);
    }).catch(function (error) {
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("viewport selection", error, {
          scope: "studio", type: "selection", fallback: "hierarchy"
        });
      }
    });
  });
})();
