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

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
