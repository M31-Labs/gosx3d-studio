(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioSelectionBridge) return;
  window.__gosxStudioSelectionBridge = true;
  var selectionGeneration = 0;
  var selectionController = null;
  var handledHostPopstate = false;

  function selectedURL(selected) {
    var url = new URL(window.location.href);
    url.searchParams.set("selection", selected);
    return url;
  }

  function selectionFeedback(message, retry) {
    var host = document.querySelector("[data-selection-feedback]");
    if (!host) {
      host = document.createElement("div");
      host.className = "selection-feedback";
      host.setAttribute("data-selection-feedback", "");
      host.setAttribute("role", "status");
      document.body.appendChild(host);
    }
    host.textContent = message || "";
    host.hidden = !message;
    host.setAttribute("data-state", retry ? "error" : "loading");
    if (retry) {
      var button = document.createElement("button");
      button.type = "button";
      button.textContent = "Retry selection";
      button.addEventListener("click", retry);
      host.appendChild(button);
      var reload = document.createElement("button");
      reload.type = "button";
      reload.textContent = "Reload workspace";
      reload.addEventListener("click", function () { window.location.reload(); });
      host.appendChild(reload);
    }
  }

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
      // the Inspector reconciles. Prevent a fast Apply click from sending those
      // values to either entity while the local highlight moves immediately.
      markInspectorSelectionPending();
    }
    setSharedSelection(selected);

    document.querySelectorAll("[data-entity-id]").forEach(function (item) {
      var active = String(item.getAttribute("data-entity-id") || "") === selected;
      var row = item.closest("[data-hierarchy-row]");
      if (row) row.classList.toggle("selected", active);
      item.setAttribute("aria-selected", active ? "true" : "false");
      item.setAttribute("aria-current", active ? "true" : "false");
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

  function currentRevision(root) {
    var field = root.querySelector('input[name="expectedRevision"]');
    return field ? String(field.value || field.getAttribute("value") || "") : "";
  }

  function rememberSelection(selected, replace) {
    var url = selectedURL(selected);
    var state = Object.assign({}, window.history.state || {}, { studioSelection: selected });
    if (replace || url.href === window.location.href) window.history.replaceState(state, "", url.href);
    else window.history.pushState(state, "", url.href);
  }

  // Keep the canonical server template as Inspector truth. Fetching HTML is
  // not navigation: only this panel's children reconcile, never the document,
  // head, scene mount, camera, other panes, or bootstrap runtime.
  function selectInPlace(selected, options) {
    selected = String(selected || "");
    options = options || {};
    if (!selected) return Promise.resolve(false);
    var generation = ++selectionGeneration;
    if (selectionController) selectionController.abort();
    selectionController = new AbortController();
    var controller = selectionController;
    var inspector = document.getElementById("inspector-panel");
    var expectedRevision = currentRevision(document);
    applyLocalSelection(selected);
    markInspectorSelectionPending();
    var loading = window.setTimeout(function () {
      if (generation === selectionGeneration) selectionFeedback("Loading selected object…");
    }, 180);
    var timeout = window.setTimeout(function () { controller.abort(); }, 10000);
    return request(selectedURL(selected).href, {
      method: "GET", headers: { Accept: "text/html" },
      cache: "no-store", credentials: "same-origin", signal: controller.signal
    }).then(function (response) {
      if (!response || !response.ok) throw new Error("The selected object's Inspector could not be loaded.");
      return response.text();
    }).then(function (html) {
      if (generation !== selectionGeneration) return false;
      var rendered = new DOMParser().parseFromString(html, "text/html");
      var nextInspector = rendered.getElementById("inspector-panel");
      var nextStage = rendered.querySelector(".scene-stage[data-selection-id]");
      if (!inspector || !nextInspector || !nextStage || nextStage.getAttribute("data-selection-id") !== selected) {
        throw new Error("That object is no longer available. Choose another object or reload the workspace.");
      }
      if (!expectedRevision || currentRevision(nextInspector) !== expectedRevision) {
        throw new Error("The scene changed elsewhere. Reload the workspace before editing.");
      }
      var scrollTop = inspector.scrollTop;
      var disclosure = {};
      inspector.querySelectorAll(":scope > details").forEach(function (item) {
        var summary = item.querySelector("summary");
        if (summary) disclosure[summary.textContent.trim()] = item.open;
      });
      var children = Array.prototype.map.call(nextInspector.childNodes, function (child) {
        return document.importNode(child, true);
      });
      inspector.replaceChildren.apply(inspector, children);
      inspector.querySelectorAll(":scope > details").forEach(function (item) {
        var summary = item.querySelector("summary");
        if (summary && Object.prototype.hasOwnProperty.call(disclosure, summary.textContent.trim())) {
          item.open = disclosure[summary.textContent.trim()];
        }
      });
      inspector.scrollTop = scrollTop;
      var stage = document.querySelector(".scene-stage[data-selection-id]");
      if (stage) ["data-camera-focus-x", "data-camera-focus-y", "data-camera-focus-z"].forEach(function (name) {
        stage.setAttribute(name, nextStage.getAttribute(name) || "0");
      });
      applyLocalSelection(selected, { pending: false });
      clearInspectorSelectionPending();
      selectionFeedback("");
      if (options.history !== false) rememberSelection(selected, options.replace === true);
      document.dispatchEvent(new CustomEvent("studio:selection", { detail: { selected: selected, revision: expectedRevision } }));
      return true;
    }).catch(function (error) {
      if (generation !== selectionGeneration) return false;
      selectionFeedback(error && error.name === "AbortError"
        ? "Selection timed out. Your scene and camera have not been reset."
        : error.message || "Selection failed. Your scene and camera have not been reset.",
      function () { selectInPlace(selected, options); });
      return false;
    }).finally(function () {
      window.clearTimeout(loading);
      window.clearTimeout(timeout);
      if (generation === selectionGeneration) selectionController = null;
    });
  }

  window.__gosxStudioSelection = { apply: applyLocalSelection, select: selectInPlace };

  function confirmViewportSelection(payload) {
    var generation = ++selectionGeneration;
    if (selectionController) selectionController.abort();
    selectionController = new AbortController();
    var controller = selectionController;
    applyLocalSelection(payload.selected);
    markInspectorSelectionPending();
    var headers = { "Accept": "application/json", "Content-Type": "application/json" };
    var token = csrfToken();
    if (token) headers["X-CSRF-Token"] = token;
    var timeout = window.setTimeout(function () { controller.abort(); }, 10000);
    return request("/api/studio/viewport-selection", {
      method: "POST", headers: headers, body: JSON.stringify(payload), signal: controller.signal
    }).then(function (response) {
      if (!response || !response.ok) throw new Error("The scene could not confirm that selection.");
      return response.json();
    }).then(function (confirmation) {
      if (generation !== selectionGeneration) return false;
      // The canonical CPU query decides; never silently treat a GPU hit as
      // authority when the two disagree or an older request finishes late.
      var confirmed = confirmation && confirmation.selected ? String(confirmation.selected) : "";
      if (!confirmed) throw new Error("That object is no longer available.");
      return selectInPlace(confirmed);
    }).catch(function (error) {
      if (generation !== selectionGeneration) return false;
      selectionFeedback(error.name === "AbortError" ? "Selection confirmation timed out." : error.message,
        function () { confirmViewportSelection(payload); });
      return false;
    }).finally(function () {
      window.clearTimeout(timeout);
      if (generation === selectionGeneration) selectionController = null;
    });
  }

  document.addEventListener("click", function (event) {
    var link = event.target && typeof event.target.closest === "function"
      ? event.target.closest("a[data-entity-id]")
      : null;
    if (!link) return;
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey ||
        link.hasAttribute("download") || link.getAttribute("target") === "_blank") return;
    event.preventDefault();
    selectInPlace(link.getAttribute("data-entity-id"));
  }, true);

  function isSelectionHistory(state) {
    return state && state.studioSelection && window.location.pathname === "/";
  }

  document.addEventListener("gosx:popstate", function (event) {
    var state = event.detail && event.detail.state;
    if (!isSelectionHistory(state)) return;
    event.preventDefault();
    handledHostPopstate = true;
    selectInPlace(state.studioSelection, { history: false });
  });

  window.addEventListener("popstate", function (event) {
    if (handledHostPopstate) {
      handledHostPopstate = false;
      return;
    }
    if (!isSelectionHistory(event.state)) return;
    event.stopImmediatePropagation();
    selectInPlace(event.state.studioSelection, { history: false });
  }, true);

  var initialStage = document.querySelector(".scene-stage[data-selection-id]");
  if (initialStage) rememberSelection(initialStage.getAttribute("data-selection-id"), true);

  document.addEventListener("gosx:navigate", function () {
    ++selectionGeneration;
    if (selectionController) selectionController.abort();
    selectionController = null;
    selectionFeedback("");
    window.setTimeout(function () {
      clearInspectorSelectionPending();
      var stage = document.querySelector("[data-selection-id]");
      var selected = stage ? String(stage.getAttribute("data-selection-id") || "") : "";
      if (selected) applyLocalSelection(selected, { pending: false });
      if (selected) rememberSelection(selected, true);
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
    confirmViewportSelection(payload);
  });
})();
