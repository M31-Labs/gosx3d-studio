(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioGizmoBridge) return;
  window.__gosxStudioGizmoBridge = true;

  var SIGNAL = "studio.viewport.gizmoMode";

  function setMode(mode) {
    if (typeof window.__gosx_set_shared_signal === "function") {
      window.__gosx_set_shared_signal(SIGNAL, mode);
      return true;
    }
    return false;
  }

  function bind() {
    var buttons = document.querySelectorAll("[data-gizmo-mode]");
    if (!buttons.length) return;
    buttons.forEach(function (button) {
      button.addEventListener("click", function () {
        var applied = setMode(String(button.getAttribute("data-gizmo-mode") || ""));
        if (!applied && window.__gosx && typeof window.__gosx.reportFailure === "function") {
          window.__gosx.reportFailure("gizmo mode", new Error("shared signal runtime unavailable"), {
            scope: "studio", type: "gizmo", fallback: "inspector-forms"
          });
          return;
        }
        buttons.forEach(function (other) { other.classList.remove("active"); });
        button.classList.add("active");
      });
    });
  }

  function request(url, options) {
    if (window.__gosx && typeof window.__gosx.request === "function") {
      return window.__gosx.request(url, options);
    }
    return fetch(url, options);
  }

  document.addEventListener("gosx:scene3d:input", function (event) {
    var detail = event && event.detail;
    var input = detail && detail.kind === "gizmo-commit" ? detail.input : null;
    if (!input || input.phase !== "end" || !input.target) return;

    request("/api/studio/gizmo-commit", {
      method: "POST",
      headers: { "Accept": "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(input)
    }).then(function (response) {
      if (!response || !response.ok) return;
      // One committed drag = one transaction = one undo step; morph-refresh
      // so the Inspector, history, and revision-stamped forms update without
      // tearing down the viewport.
      if (window.__gosx_page_nav && typeof window.__gosx_page_nav.navigate === "function") {
        window.__gosx_page_nav.navigate(window.location.href, { replace: true, preserveScroll: true });
      } else {
        window.location.reload();
      }
    }).catch(function (error) {
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("gizmo commit", error, {
          scope: "studio", type: "gizmo", fallback: "inspector-forms"
        });
      }
    });
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
