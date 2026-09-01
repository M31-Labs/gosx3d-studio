(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioGizmoBridge) return;
  window.__gosxStudioGizmoBridge = true;

  var MODE_SIGNAL = "studio.viewport.gizmoMode";
  var SELECTION_SIGNAL = "studio.viewport.selectedID";
  // GoSX 0.54 treats an empty gizmo signal as the default translate mode.
  // Studio uses an explicit non-transform sentinel so Select is visually and
  // behaviorally distinct from Move while retaining the canonical selection.
  var activeMode = "select";

  function setSignal(name, value) {
    var runtime = window.__gosx_runtime_api;
    if (runtime && typeof runtime.setSharedSignalValue === "function") {
      try {
        runtime.setSharedSignalValue(name, value);
        return true;
      } catch (_) {}
    }
    if (typeof window.__gosx_set_shared_signal === "function") {
      try {
        var result = window.__gosx_set_shared_signal(name, JSON.stringify(value));
        if (typeof result !== "string" || result === "") return true;
      } catch (_) {}
    }
    if (typeof window.__gosx_notify_shared_signal === "function") {
      try {
        window.__gosx_notify_shared_signal(name, JSON.stringify(value));
        return true;
      } catch (_) {
        return false;
      }
    }
    return false;
  }

  function currentSelection() {
    var stage = document.querySelector("[data-selection-id]");
    return stage ? String(stage.getAttribute("data-selection-id") || "") : "";
  }

  function syncSelection() {
    return setSignal(SELECTION_SIGNAL, currentSelection());
  }

  function syncEngineState() {
    var selected = syncSelection();
    var mode = setSignal(MODE_SIGNAL, activeMode);
    return selected && mode;
  }

  function reportUnavailable() {
    if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
      window.__gosx.reportFailure("gizmo mode", new Error("shared signal runtime unavailable"), {
        scope: "studio", type: "gizmo", fallback: "inspector-forms"
      });
    }
  }

  function chooseMode(mode) {
    syncSelection();
    if (!setSignal(MODE_SIGNAL, mode)) {
      reportUnavailable();
      return false;
    }
    activeMode = mode;
    syncButtons();
    return true;
  }

  function syncButtons() {
    document.querySelectorAll("[data-gizmo-mode]").forEach(function (button) {
      var active = String(button.getAttribute("data-gizmo-mode") || "") === activeMode;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", active ? "true" : "false");
    });
  }

  document.addEventListener("click", function (event) {
    var button = event.target && typeof event.target.closest === "function"
      ? event.target.closest("[data-gizmo-mode]")
      : null;
    if (!button) return;
    chooseMode(String(button.getAttribute("data-gizmo-mode") || ""));
  });

  function request(url, options) {
    if (window.__gosx && typeof window.__gosx.request === "function") {
      return window.__gosx.request(url, options);
    }
    return fetch(url, options);
  }

  function csrfToken() {
    var field = document.querySelector('input[name="csrf_token"]');
    return field && typeof field.value === "string" ? field.value : "";
  }

  document.addEventListener("gosx:scene3d:input", function (event) {
    var detail = event && event.detail;
    var input = detail && detail.kind === "gizmo-commit" ? detail.input : null;
    if (!input || input.phase !== "end" || !input.target) return;

    var headers = { "Accept": "application/json", "Content-Type": "application/json" };
    var token = csrfToken();
    if (token) headers["X-CSRF-Token"] = token;
    request("/api/studio/gizmo-commit", {
      method: "POST",
      headers: headers,
      body: JSON.stringify(input)
    }).then(function (response) {
      if (!response || !response.ok) return;
      // One committed drag = one transaction = one undo step; morph-refresh
      // so the Inspector, history, and revision-stamped forms update without
      // tearing down the viewport.
      var navigation = window.__gosx && window.__gosx.navigation
        ? window.__gosx.navigation
        : window.__gosx_page_nav;
      if (navigation && typeof navigation.revalidate === "function") {
        navigation.revalidate({ replace: true, preserveScroll: true }).catch(function () { window.location.reload(); });
      } else if (navigation && typeof navigation.navigate === "function") {
        navigation.navigate(window.location.href, {
          replace: true, preserveScroll: true, force: true, revalidate: true
        }).catch(function () { window.location.reload(); });
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

  document.addEventListener("gosx:navigate", function () {
    window.setTimeout(function () {
      syncEngineState();
      syncButtons();
    }, 0);
  });

  document.addEventListener("gosx:ready", function () {
    // Scene3D publishes its empty initial selection while mounting. Seed the
    // canonical Studio selection on the next task, after that initialization.
    window.setTimeout(syncEngineState, 0);
  });

  function initialize() {
    syncEngineState();
    syncButtons();
    if (window.__gosx && window.__gosx.ready) window.setTimeout(syncEngineState, 0);
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", initialize);
  else initialize();
})();
