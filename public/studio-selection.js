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

  function csrfToken() {
    var field = document.querySelector('input[name="csrf_token"]');
    return field && typeof field.value === "string" ? field.value : "";
  }

  function setSharedSelection(selected) {
    var runtime = window.__gosx_runtime_api;
    if (runtime && typeof runtime.setSharedSignalValue === "function") {
      try {
        runtime.setSharedSignalValue("studio.viewport.selectedID", selected);
        return true;
      } catch (_) {}
    }
    if (typeof window.__gosx_set_shared_signal === "function") {
      try {
        var result = window.__gosx_set_shared_signal(
          "studio.viewport.selectedID",
          JSON.stringify(selected)
        );
        if (typeof result !== "string" || result === "") return true;
      } catch (_) {}
    }
    if (typeof window.__gosx_notify_shared_signal === "function") {
      try {
        window.__gosx_notify_shared_signal(
          "studio.viewport.selectedID",
          JSON.stringify(selected)
        );
        return true;
      } catch (_) {}
    }
    return false;
  }

  function markInspectorSelectionPending() {
    document.querySelectorAll("form[data-selection-bound]").forEach(function (form) {
      if (!form || form.hasAttribute("data-selection-pending")) return;
      form.setAttribute("data-selection-pending", "true");
      form.setAttribute("aria-busy", "true");
      form.querySelectorAll('button[type="submit"], input[type="submit"]').forEach(function (control) {
        if (control.disabled) return;
        control.disabled = true;
        control.setAttribute("data-selection-pending-enabled", "true");
      });
    });
  }

  function clearInspectorSelectionPending() {
    document.querySelectorAll("form[data-selection-pending]").forEach(function (form) {
      form.removeAttribute("data-selection-pending");
      form.removeAttribute("aria-busy");
      form.querySelectorAll("[data-selection-pending-enabled]").forEach(function (control) {
        control.removeAttribute("data-selection-pending-enabled");
        if (!control.closest("[data-webmcp-review-locked]")) control.disabled = false;
      });
    });
  }

  // Selection is local-first. The shared signal updates the outline and gizmo
  // in the already-mounted Scene3D surface on the same input frame; the
  // server-rendered Inspector then reconciles behind it. This keeps canonical
  // server selection without making the GPU viewport wait for navigation.
  function applyLocalSelection(selected, options) {
    selected = String(selected || "");
    if (!selected) return false;
    var stage = document.querySelector("[data-selection-id]");
    var previous = stage ? String(stage.getAttribute("data-selection-id") || "") : "";
    if ((!options || options.pending !== false) && previous && previous !== selected) {
      // Inspector values still describe the previous server selection until
      // navigation reconciles. Prevent a fast Apply click from sending those
      // values to either entity while the local highlight moves immediately.
      markInspectorSelectionPending();
    }
    setSharedSelection(selected);

    document.querySelectorAll("[data-entity-id]").forEach(function (item) {
      var active = String(item.getAttribute("data-entity-id") || "") === selected;
      var row = item.closest("[data-hierarchy-row]");
      if (row) row.classList.toggle("selected", active);
      item.setAttribute("aria-selected", active ? "true" : "false");
      item.setAttribute("tabindex", active ? "0" : "-1");
    });
    document.querySelectorAll("[data-selection-id]").forEach(function (stage) {
      stage.setAttribute("data-selection-id", selected);
    });
    document.querySelectorAll('input[type="hidden"][name="selection"]').forEach(function (field) {
      field.value = selected;
      field.setAttribute("value", selected);
    });
    return true;
  }

  window.__gosxStudioSelection = {
    apply: applyLocalSelection
  };

  document.addEventListener("click", function (event) {
    var link = event.target && typeof event.target.closest === "function"
      ? event.target.closest("a[data-entity-id]")
      : null;
    if (!link) return;
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey ||
        link.hasAttribute("download") || link.getAttribute("target") === "_blank") return;
    applyLocalSelection(link.getAttribute("data-entity-id"));
  });

  document.addEventListener("gosx:navigate", function () {
    window.setTimeout(function () {
      clearInspectorSelectionPending();
      var stage = document.querySelector("[data-selection-id]");
      var selected = stage ? String(stage.getAttribute("data-selection-id") || "") : "";
      if (selected) applyLocalSelection(selected, { pending: false });
    }, 0);
  });

  document.addEventListener("gosx:scene3d:input", function (event) {
    var detail = event && event.detail;
    var input = detail && detail.kind === "pick" ? detail.input : null;
    if (!input || (input.type !== "select" && input.type !== "click") || !input.selectedID) return;

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
    var headers = { "Accept": "application/json", "Content-Type": "application/json" };
    var token = csrfToken();
    if (token) headers["X-CSRF-Token"] = token;
    request("/api/studio/viewport-selection", {
      method: "POST",
      headers: headers,
      body: JSON.stringify(payload)
    }).then(function (response) {
      if (!response || !response.ok) return;
      return response.json().catch(function () { return null; }).then(function (confirmation) {
        // The canonical CPU query decides the selection; follow its answer so
        // a GPU/CPU disagreement never becomes silent editor selection truth.
        var confirmed = confirmation && confirmation.selected ? String(confirmation.selected) : selected;
        applyLocalSelection(confirmed);
        var url = new URL(window.location.href);
        url.searchParams.set("selection", confirmed);
        var target = url.pathname + url.search;
        if (window.__gosx_page_nav && typeof window.__gosx_page_nav.navigate === "function") {
          // Morph in place: panels refresh, the Scene3D mount stays alive.
          window.__gosx_page_nav.navigate(target, { preserveScroll: true });
        } else {
          window.location.assign(target);
        }
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
