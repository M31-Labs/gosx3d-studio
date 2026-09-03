(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioWebMCPUI) return;
  window.__gosxStudioWebMCPUI = true;

  var pendingProposal = null;
  var proposalHydration = 0;
  var focusedEntityId = "";
  var pendingFocusNavigation = null;
  var MAX_FOCUS_NAVIGATION_ATTEMPTS = 3;
  var demoClean = false;
  var demoResetInFlight = false;
  var demoStatusInFlight = false;
  var demoStatusGeneration = 0;
  var reviewInFlight = false;
  var reviewFocusTarget = null;
  var activeScenePreview = null;
  var latestApprovalOutcome = null;
  var scenePreviewChain = Promise.resolve();
  var traceEntries = [];
  var toolStates = {
    scene_get_state: "idle",
    scene_find_objects: "idle",
    scene_focus_object: "idle",
    scene_preview_actions: "idle"
  };
  var toolStateStorageKey = "gosx3d:webmcp-flow:v1";
  var traceStorageKey = "gosx3d:webmcp-trace:v1";
  try {
    var storedToolStates = JSON.parse(window.sessionStorage.getItem(toolStateStorageKey) || "null");
    Object.keys(toolStates).forEach(function (tool) {
      if (storedToolStates && ["idle", "running", "complete", "error"].indexOf(storedToolStates[tool]) >= 0) {
        toolStates[tool] = storedToolStates[tool];
      }
    });
  } catch (_) {}
  try {
    var storedTrace = JSON.parse(window.sessionStorage.getItem(traceStorageKey) || "[]");
    if (Array.isArray(storedTrace)) {
      traceEntries = storedTrace.filter(function (entry) {
        return entry && typeof entry === "object" && typeof entry.callId === "string" && typeof entry.message === "string";
      }).slice(-8);
    }
  } catch (_) {}
  var latestStatus = {
    state: "detecting",
    label: "Detecting browser support",
    message: "The complete human editing surface remains available while WebMCP initializes.",
    toolCount: 0
  };

  function one(selector) {
    return document.querySelector(selector);
  }

  function setText(selector, value) {
    var element = one(selector);
    if (!element) return;
    var next = value == null ? "" : String(value);
    if (element.textContent !== next) element.textContent = next;
  }

  function csrfToken() {
    var meta = one('meta[name="csrf-token"]');
    if (meta) return String(meta.getAttribute("content") || "");
    var input = one('input[name="csrf_token"]');
    return input ? String(input.value || "") : "";
  }

  function request(url, options) {
    var requestOptions = Object.assign({}, options || {});
    var headers = Object.assign({}, requestOptions.headers || {});
    var method = String(requestOptions.method || "GET").toUpperCase();
    var mutating = method === "POST" || method === "PUT" || method === "PATCH" || method === "DELETE";
    var token = mutating ? csrfToken() : "";
    if (token && !headers["X-CSRF-Token"]) headers["X-CSRF-Token"] = token;
    requestOptions.headers = headers;
    if (window.__gosx && typeof window.__gosx.request === "function") {
      return window.__gosx.request(url, requestOptions);
    }
    return fetch(url, requestOptions);
  }

  function responsePayload(response) {
    if (!response || typeof response.json !== "function") return Promise.resolve(null);
    return response.json().catch(function () { return null; });
  }

  function responseError(response, payload) {
    var message = payload && (payload.error || payload.message);
    if (message && typeof message === "object") message = message.message;
    if (!message) message = "Request failed with status " + (response ? response.status : "unknown");
    var error = new Error(String(message));
    error.status = response ? Number(response.status || 0) : 0;
    return error;
  }

  function statusLabel(state) {
    switch (state) {
      case "ready": return "Agent tools ready";
      case "proposal": return "Human review requested";
      case "committing": return "Applying reviewed proposal";
      case "applied": return "Human-approved change applied";
      case "unavailable": return "Browser support unavailable";
      case "error": return "WebMCP needs attention";
      default: return "Detecting browser support";
    }
  }

  function renderStatus() {
    var panel = one("[data-webmcp-status-panel]");
    if (panel) panel.setAttribute("data-state", latestStatus.state);
    var agentPanel = one(".agent-panel");
    if (agentPanel) agentPanel.setAttribute("data-webmcp-state", latestStatus.state);
    setText("[data-webmcp-status-label]", latestStatus.label || statusLabel(latestStatus.state));
    setText("[data-webmcp-status-message]", latestStatus.message);
    setText("[data-webmcp-tool-count]", latestStatus.toolCount + (latestStatus.toolCount === 1 ? " tool" : " tools"));
  }

  function renderToolFlow() {
    Object.keys(toolStates).forEach(function (tool) {
      var item = one('[data-webmcp-flow-tool="' + tool + '"]');
      if (item) {
        item.setAttribute("data-state", toolStates[tool]);
        item.setAttribute("aria-label", String(item.getAttribute("data-label") || tool) + ": " + toolStates[tool]);
      }
    });
  }

  function persistTrace() {
    try { window.sessionStorage.setItem(traceStorageKey, JSON.stringify(traceEntries)); } catch (_) {}
  }

  function traceItem(entry) {
    var item = document.createElement("li");
    item.setAttribute("data-state", entry.state === "error" ? "error" : "complete");
    item.textContent = String(entry.message || entry.tool || "WebMCP call");
    return item;
  }

  function renderTrace() {
    var host = one("[data-webmcp-trace]");
    if (!host) return;
    host.setAttribute("aria-busy", "true");
    host.setAttribute("aria-live", "off");
    while (host.firstChild) host.removeChild(host.firstChild);
    if (!traceEntries.length) {
      var empty = document.createElement("p");
      empty.className = "webmcp-trace-empty";
      empty.textContent = "Typed-call receipts will appear here.";
      host.appendChild(empty);
    } else {
      var list = document.createElement("ol");
      list.className = "webmcp-trace-list";
      traceEntries.forEach(function (entry) { list.appendChild(traceItem(entry)); });
      host.appendChild(list);
    }
    var restoreAnnouncements = function () {
      host.removeAttribute("aria-busy");
      host.setAttribute("aria-live", "polite");
    };
    if (typeof window.requestAnimationFrame === "function") {
      window.requestAnimationFrame(restoreAnnouncements);
    } else {
      window.setTimeout(restoreAnnouncements, 0);
    }
  }

  function appendTrace(detail) {
    detail = detail || {};
    var callId = String(detail.callId || "");
    if (!callId || traceEntries.some(function (entry) { return entry.callId === callId; })) return;
    var entry = {
      callId: callId,
      tool: String(detail.tool || ""),
      state: detail.state === "error" ? "error" : "complete",
      message: String(detail.message || "WebMCP call completed"),
      timestamp: String(detail.timestamp || "")
    };
    traceEntries.push(entry);
    traceEntries = traceEntries.slice(-8);
    persistTrace();
    var host = one("[data-webmcp-trace]");
    var list = host && host.querySelector(".webmcp-trace-list");
    if (!host) return;
    if (!list && traceEntries.length > 1) {
      renderTrace();
      return;
    }
    if (!list) {
      while (host.firstChild) host.removeChild(host.firstChild);
      list = document.createElement("ol");
      list.className = "webmcp-trace-list";
      host.appendChild(list);
    }
    host.removeAttribute("aria-busy");
    host.setAttribute("aria-live", "polite");
    list.appendChild(traceItem(entry));
    while (list.children.length > traceEntries.length) list.removeChild(list.firstElementChild);
    host.scrollTop = host.scrollHeight;
  }

  function clearTrace() {
    traceEntries = [];
    persistTrace();
    renderTrace();
  }

  function persistToolFlow() {
    try { window.sessionStorage.setItem(toolStateStorageKey, JSON.stringify(toolStates)); } catch (_) {}
  }

  function clearToolFlow() {
    Object.keys(toolStates).forEach(function (tool) { toolStates[tool] = "idle"; });
    persistToolFlow();
    renderToolFlow();
  }

  function updateToolFlow(detail) {
    detail = detail || {};
    var tool = String(detail.tool || "");
    if (!Object.prototype.hasOwnProperty.call(toolStates, tool)) return;
    var message = String(detail.message || "");
    if (detail.state === "error") toolStates[tool] = "error";
    else if (message.indexOf("Running ") === 0) toolStates[tool] = "running";
    else toolStates[tool] = "complete";
    persistToolFlow();
    renderToolFlow();
  }

  function updateStatus(detail) {
    detail = detail || {};
    updateToolFlow(detail);
    latestStatus = {
      state: String(detail.state || latestStatus.state || "detecting"),
      label: String(detail.label || statusLabel(detail.state || latestStatus.state)),
      message: String(detail.message || latestStatus.message || ""),
      toolCount: Number.isFinite(Number(detail.toolCount)) ? Number(detail.toolCount) : latestStatus.toolCount
    };
    renderStatus();
  }

  function vectorSummary(value) {
    if (!value || typeof value !== "object") return "";
    return ["x", "y", "z"].map(function (axis) {
      var number = Number(value[axis]);
      return Number.isFinite(number) ? number.toFixed(2) : "?";
    }).join(", ");
  }

  function dispatchSceneCommands(commands) {
    if (!Array.isArray(commands) || commands.length === 0) return Promise.resolve(false);
    var mount = one("[data-gosx-scene3d]");
    var api = window.__gosx && window.__gosx.scene3d;
    if (api && typeof api.dispatchCommands === "function") {
      return Promise.resolve(api.dispatchCommands(mount, commands, { timeoutMS: 10000 })).then(function () { return true; });
    }
    var bridge = window.__gosx_scene3d_command_bridge;
    if (bridge && typeof bridge.dispatchCommands === "function") {
      return Promise.resolve(bridge.dispatchCommands(mount, commands, { timeoutMS: 10000 })).then(function () { return true; });
    }
    if (mount && mount.__gosxScene3DHandle && typeof mount.__gosxScene3DHandle.applyCommands === "function") {
      return Promise.resolve(mount.__gosxScene3DHandle.applyCommands(commands)).then(function () { return true; });
    }
    return Promise.reject(new Error("The live Scene3D command bridge is unavailable."));
  }

  function markScenePreview(active) {
    var stage = one(".scene-stage");
    var badge = one("[data-webmcp-preview-badge]");
    if (stage) {
      if (active) stage.setAttribute("data-webmcp-preview", "true");
      else stage.removeAttribute("data-webmcp-preview");
    }
    if (badge) badge.hidden = !active;
  }

  function renderApprovalOutcome() {
    var outcome = one("[data-webmcp-approval-outcome]");
    if (!outcome) return;
    outcome.hidden = !latestApprovalOutcome;
    if (latestApprovalOutcome) {
      setText("[data-webmcp-approval-outcome-copy]", latestApprovalOutcome.copy);
    }
  }

  function clearApprovalOutcome() {
    latestApprovalOutcome = null;
    renderApprovalOutcome();
  }

  function showApprovalOutcome(copy) {
    latestApprovalOutcome = { copy: String(copy || "Canonical scene updated") };
    renderApprovalOutcome();
  }

  function activateScenePreview(proposal) {
    if (!proposal || !proposal.proposalId) return Promise.resolve(false);
    var unchanged = false;
    var phase = "queued";
    var previousPreview = null;
    var proposalMount = null;
    scenePreviewChain = scenePreviewChain.then(function () {
      phase = "inspect";
      var previous = activeScenePreview;
      previousPreview = previous;
      var currentMount = one("[data-gosx-scene3d]");
      if (previous && previous.__scenePreviewMount !== currentMount) {
        // A remount already discarded the old imperative overlay. Do not send
        // its reverse commands to the fresh canonical engine; just reapply the
        // still-pending proposal below.
        activeScenePreview = null;
        previous = null;
      }
      if (previous && previous.proposalId === proposal.proposalId) {
        unchanged = true;
        // Navigation reconciliation restores the server-authored hidden badge
        // even when it correctly reuses the live Scene3D mount. Reassert the
        // disclosure for the imperative overlay that is still active.
        markScenePreview(previous.__scenePreviewApplied === true || previous.__scenePreviewUncertain === true);
        if (previous.__scenePreviewUncertain === true) {
          updateStatus({
            state: "error",
            message: "The live preview could not be confirmed. Apply or discard the staged proposal to reconcile the canonical scene."
          });
        }
        return false;
      }
      if (!previous || !Array.isArray(previous.reverseSceneCommands) || previous.reverseSceneCommands.length === 0) return false;
      phase = "reverse-previous";
      return dispatchSceneCommands(previous.reverseSceneCommands).then(function (reversed) {
        activeScenePreview = null;
        phase = "previous-reversed";
        return reversed;
      });
    }).then(function () {
      if (unchanged) return false;
      proposalMount = one("[data-gosx-scene3d]");
      phase = "apply-proposal";
      return dispatchSceneCommands(proposal.sceneCommands || []);
    }).then(function (applied) {
      if (unchanged) return applied;
      proposal.__scenePreviewMount = proposalMount || one("[data-gosx-scene3d]");
      proposal.__scenePreviewApplied = applied === true;
      proposal.__scenePreviewUncertain = false;
      activeScenePreview = proposal;
      markScenePreview(applied === true);
      phase = "active";
      return applied;
    }).catch(function (error) {
      if (phase === "reverse-previous" && previousPreview) {
        // The old overlay may still be present (or partially reversed). Keep
        // its exact reverse commands and disclosure until a hard remount
        // restores canonical SceneIR; never pretend the superseding proposal
        // is what the human is looking at.
        activeScenePreview = previousPreview;
        previousPreview.__scenePreviewApplied = true;
        previousPreview.__scenePreviewUncertain = true;
        markScenePreview(true);
      } else if (phase === "apply-proposal") {
        // applyCommands can fail after mutating part of a command batch. Retain
        // the proposal and its full reverse diff so Discard can still restore
        // canonical state. Hiding the badge here would turn an uncertain live
        // overlay into an undisclosed apparent commit.
        proposal.__scenePreviewMount = proposalMount || one("[data-gosx-scene3d]");
        proposal.__scenePreviewApplied = false;
        proposal.__scenePreviewUncertain = true;
        activeScenePreview = proposal;
        markScenePreview(true);
        updateStatus({
          state: "error",
          message: "The live preview could not be confirmed. Apply or discard the staged proposal to reconcile the canonical scene."
        });
      }
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("WebMCP live scene preview", error, {
          scope: "studio", type: "webmcp",
          fallback: phase === "reverse-previous" ? "canonical-remount" : "human-review"
        });
      }
      if (phase === "reverse-previous") {
        return requireCanonicalReload("The previous agent preview could not be reversed safely; restoring canonical SceneIR before showing the next proposal.");
      }
      return false;
    });
    return scenePreviewChain;
  }

  function revertScenePreview(proposalId) {
    var matchedPreview = null;
    scenePreviewChain = scenePreviewChain.then(function () {
      var preview = activeScenePreview;
      if (!preview || proposalId && preview.proposalId !== proposalId) return true;
      matchedPreview = preview;
      activeScenePreview = null;
      if (!Array.isArray(preview.reverseSceneCommands) || preview.reverseSceneCommands.length === 0) return true;
      return dispatchSceneCommands(preview.reverseSceneCommands || []);
    }).catch(function (error) {
      if (matchedPreview) activeScenePreview = matchedPreview;
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("WebMCP live scene preview rollback", error, {
          scope: "studio", type: "webmcp", fallback: "page-reconcile"
        });
      }
      return false;
    }).then(function (restored) {
      if (matchedPreview) markScenePreview(restored ? false : true);
      return restored;
    });
    return scenePreviewChain;
  }

  function vectorsEqual(left, right) {
    if (!left || !right) return left === right;
    return ["x", "y", "z"].every(function (axis) {
      return Number(left[axis]) === Number(right[axis]);
    });
  }

  function materialSummary(id, materials) {
    if (!id) return "none";
    var material = materials && materials[id];
    var name = material && typeof material === "object" ? material.name : material;
    return name ? String(name) + " (" + String(id) + ")" : String(id);
  }

  function changePresentation(change, materials) {
    change = change || {};
    var kind = String(change.kind || "scene edit");
    var target = String(change.target || "scene");
    var before = change.before || {};
    var after = change.after || {};
    if (kind === "rename-entity" && (before.name || after.name)) {
      return {
        label: "Rename",
        target: target,
        before: String(before.name || "unnamed"),
        after: String(after.name || "unnamed")
      };
    }
    if (kind === "assign-material") {
      var beforeMaterial = before.material || before.mesh && before.mesh.material;
      var afterMaterial = after.material || after.mesh && after.mesh.material;
      if (beforeMaterial || afterMaterial) {
        return {
          label: "Material",
          target: target,
          before: materialSummary(beforeMaterial, materials),
          after: materialSummary(afterMaterial, materials)
        };
      }
    }
    if (kind === "set-transform" && after.transform) {
      var beforeTransform = before.transform || {};
      var afterTransform = after.transform || {};
      var fields = ["position", "rotation", "scale"].filter(function (field) {
        return !vectorsEqual(beforeTransform[field], afterTransform[field]);
      });
      return {
        label: "Transform",
        target: target,
        before: fields.length ? fields.map(function (field) {
          return field + " " + vectorSummary(beforeTransform[field]);
        }).join(" · ") : "current transform",
        after: fields.length ? fields.map(function (field) {
          return field + " " + vectorSummary(afterTransform[field]);
        }).join(" · ") : "updated transform"
      };
    }
    return {
      label: kind.replace(/-/g, " "),
      target: target,
      before: "current state",
      after: "staged change"
    };
  }

  function renderChangeList(list, changes, materials) {
    while (list.firstChild) list.removeChild(list.firstChild);
    changes.forEach(function (change) {
      var presentation = changePresentation(change, materials);
      var item = document.createElement("li");
      var kind = document.createElement("span");
      var target = document.createElement("code");
      var values = document.createElement("span");
      var before = document.createElement("del");
      var arrow = document.createElement("span");
      var after = document.createElement("ins");
      kind.className = "webmcp-change-kind";
      target.className = "webmcp-change-target";
      values.className = "webmcp-change-values";
      arrow.className = "webmcp-change-arrow";
      kind.textContent = presentation.label;
      target.textContent = presentation.target;
      before.textContent = presentation.before;
      arrow.textContent = "→";
      arrow.setAttribute("aria-hidden", "true");
      after.textContent = presentation.after;
      values.appendChild(before);
      values.appendChild(arrow);
      values.appendChild(after);
      item.appendChild(kind);
      item.appendChild(target);
      item.appendChild(values);
      item.setAttribute(
        "aria-label",
        presentation.label + " " + presentation.target + ": " + presentation.before + " to " + presentation.after
      );
      list.appendChild(item);
    });
    list.hidden = changes.length === 0;
  }

  function renderChanges(receipt, materials) {
    var changes = receipt && Array.isArray(receipt.changes) ? receipt.changes : [];
    var lists = document.querySelectorAll("[data-webmcp-proposal-changes], [data-webmcp-preview-changes]");
    Array.prototype.forEach.call(lists, function (list) {
      renderChangeList(list, changes, materials);
    });
  }

  function shortFingerprint(value) {
    value = String(value || "");
    if (value.length <= 22) return value;
    return value.slice(0, 10) + "…" + value.slice(-8);
  }

  function reviewButtons(disabled) {
    var actions = document.querySelectorAll("[data-webmcp-commit], [data-webmcp-discard]");
    var group = one("[data-webmcp-review-actions]");
    if (group) {
      if (disabled === true) group.setAttribute("aria-busy", "true");
      else group.removeAttribute("aria-busy");
    }
    if (disabled === true && document.activeElement) {
      Array.prototype.some.call(actions, function (action) {
        if (action !== document.activeElement && !action.contains(document.activeElement)) return false;
        reviewFocusTarget = action;
        return true;
      });
    }
    Array.prototype.forEach.call(actions, function (action) { action.disabled = disabled === true; });
    if (
      disabled !== true && pendingProposal && reviewFocusTarget &&
      reviewFocusTarget.isConnected && !reviewFocusTarget.closest("[hidden]")
    ) {
      try { reviewFocusTarget.focus({ preventScroll: true }); }
      catch (_) { reviewFocusTarget.focus(); }
      reviewFocusTarget = null;
    }
  }

  function lockSceneMutationControls(locked) {
    if (locked) {
      var selectMode = one('[data-gizmo-mode="select"]');
      if (selectMode && selectMode.getAttribute("aria-pressed") !== "true" && !selectMode.disabled) {
        selectMode.click();
      }
    }
    document.querySelectorAll('form[data-gosx-form], [data-gizmo-mode]').forEach(function (surface) {
      var controls = surface.matches && surface.matches("[data-gizmo-mode]")
        ? [surface]
        : Array.prototype.slice.call(surface.querySelectorAll('button[type="submit"], input[type="submit"]'));
      if (locked) {
        surface.setAttribute("data-webmcp-review-locked", "true");
        surface.setAttribute("aria-busy", "true");
        controls.forEach(function (control) {
          if (!control.disabled || control.hasAttribute("data-selection-pending-enabled")) {
            control.disabled = true;
            control.setAttribute("data-webmcp-review-enabled", "true");
          }
        });
        return;
      }
      surface.removeAttribute("data-webmcp-review-locked");
      surface.removeAttribute("aria-busy");
      controls.forEach(function (control) {
        if (!control.hasAttribute("data-webmcp-review-enabled")) return;
        control.removeAttribute("data-webmcp-review-enabled");
        if (!control.closest("[data-selection-pending]")) control.disabled = false;
      });
    });
  }

  function requireCanonicalReload(message) {
    updateStatus({
      state: "error",
      message: message || "The live preview could not be reversed safely; restoring the canonical scene."
    });
    window.setTimeout(function () { window.location.reload(); }, 0);
    return false;
  }

  function renderProposalExpiry() {
    var expiry = one("[data-webmcp-proposal-expiry]");
    if (!expiry) return;
    if (!pendingProposal || !pendingProposal.expiresAt) {
      expiry.textContent = "not staged";
      return;
    }
    var expiresAt = Date.parse(pendingProposal.expiresAt);
    if (!Number.isFinite(expiresAt)) {
      expiry.textContent = "review window unavailable";
      return;
    }
    var remaining = expiresAt - Date.now();
    if (remaining <= 0) {
      expiry.textContent = "expired · restoring canonical scene";
      reviewButtons(true);
      if (!reviewInFlight) discardProposal(null);
      return;
    }
    var minutes = Math.max(1, Math.ceil(remaining / 60000));
    expiry.textContent = "expires in " + minutes + " min";
  }

  function renderProposal() {
    if (!pendingProposal) return;
    lockSceneMutationControls(true);
    var receipt = pendingProposal.receipt || {};
    var affected = Array.isArray(receipt.affected) ? receipt.affected : [];
    var operationCount = Number(receipt.operations || (
      Array.isArray(pendingProposal.sceneCommands) ? pendingProposal.sceneCommands.length : 0
    ));
    var editSummary = operationCount === 1 ? "1 exact edit" : operationCount + " exact edits";
    var rationale = one("[data-webmcp-proposal-rationale]");
    var agentPanel = one(".agent-panel");
    if (agentPanel) agentPanel.classList.add("has-pending-proposal");
    setText("[data-webmcp-proposal-summary]", pendingProposal.title || "Agent-authored scene proposal");
    if (rationale) {
      rationale.textContent = pendingProposal.rationale || "The agent staged a reversible preview for human review.";
      rationale.hidden = false;
    }
    setText("[data-webmcp-proposal-actor]", receipt.actor || "agent://webmcp");
    var governance = Array.isArray(pendingProposal.governance) ? pendingProposal.governance : [];
    var allowed = governance.filter(function (decision) { return decision && decision.allowed === true; }).length;
    var policy = one("[data-webmcp-proposal-policy]");
    if (policy) {
      policy.textContent = governance.length ? "Arbiter · Allow · " + allowed + "/" + governance.length : "Arbiter · evidence unavailable";
      var reasons = governance.map(function (decision) { return decision && decision.reason; }).filter(Boolean);
      policy.setAttribute("title", reasons.length ? reasons.join(" · ") : "Every operation is evaluated by the server policy before preview.");
      setText("[data-webmcp-proposal-policy-reasons]", reasons.length ? reasons.join(" · ") : "Every operation is evaluated by the server policy before preview.");
    }
    var canonicalRevision = receipt.beforeRevision == null ? "?" : String(receipt.beforeRevision);
    var approvalRevision = pendingProposal.preview && pendingProposal.preview.revision != null
      ? String(pendingProposal.preview.revision)
      : (Number.isFinite(Number(receipt.beforeRevision)) ? String(Number(receipt.beforeRevision) + 1) : "?");
    setText("[data-webmcp-proposal-revision]", "canonical " + canonicalRevision + " unchanged · approval " + approvalRevision);
    setText("[data-webmcp-preview-revision]", "Canonical rev " + canonicalRevision + " unchanged · human Apply creates rev " + approvalRevision);
    setText(
      "[data-webmcp-proposal-affected]",
      affected.length ? affected.join(", ") : String(receipt.operations || 0) + " operations"
    );
    var fingerprint = one("[data-webmcp-proposal-fingerprint]");
    var fingerprintValue = receipt.afterFingerprint || "preview ready";
    if (fingerprint) {
      fingerprint.textContent = shortFingerprint(fingerprintValue);
      fingerprint.setAttribute("title", fingerprintValue);
      fingerprint.setAttribute("aria-label", "Full preview fingerprint " + fingerprintValue);
    }
    renderChanges(receipt, pendingProposal.materials || {});
    renderProposalExpiry();
    var actions = one("[data-webmcp-review-actions]");
    if (actions) actions.hidden = false;
    setText("[data-webmcp-review-gate]", "Human-only approval · creates revision " + approvalRevision);
    var commit = one("[data-webmcp-commit]");
    if (commit) {
      commit.textContent = "Apply " + editSummary;
      commit.setAttribute("aria-label", "Apply " + editSummary + " and create canonical revision " + approvalRevision);
    }
    reviewButtons(reviewInFlight);
    renderProposalExpiry();
    updateStatus({
      state: "proposal",
      tool: "scene_preview_actions",
      message: (operationCount > 0 ? editSummary + " prepared. " : "Exact edits prepared. ") +
        "Canonical revision " + canonicalRevision +
        " is unchanged and awaiting your review. Orbit and selection stay live."
    });
    var proposalCard = one("[data-webmcp-proposal]");
    if (proposalCard && agentPanel && typeof agentPanel.scrollTo === "function") {
      window.setTimeout(function () {
        var reducedMotion = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        var panelTop = agentPanel.getBoundingClientRect().top;
        var proposalTop = proposalCard.getBoundingClientRect().top;
        agentPanel.scrollTo({
          top: Math.max(0, agentPanel.scrollTop + proposalTop - panelTop),
          behavior: reducedMotion ? "auto" : "smooth"
        });
      }, 0);
    }
  }

  function clearProposal(message) {
    var agentPanel = one(".agent-panel");
    var rationale = one("[data-webmcp-proposal-rationale]");
    if (rationale) rationale.hidden = true;
    var actions = one("[data-webmcp-review-actions]");
    var restoreReviewFocus = Boolean(
      reviewFocusTarget || actions && document.activeElement && actions.contains(document.activeElement)
    );
    pendingProposal = null;
    reviewInFlight = false;
    lockSceneMutationControls(false);
    if (actions) actions.hidden = true;
    if (agentPanel) agentPanel.classList.remove("has-pending-proposal");
    reviewButtons(false);
    reviewFocusTarget = null;
    renderChanges(null, null);
    setText("[data-webmcp-proposal-summary]", message || "No staged WebMCP proposal is awaiting review.");
    setText("[data-webmcp-proposal-actor]", "none");
    setText("[data-webmcp-proposal-policy]", "Arbiter · awaiting proposal");
    setText("[data-webmcp-proposal-revision]", "current");
    setText("[data-webmcp-proposal-affected]", "0 entities");
    setText("[data-webmcp-proposal-fingerprint]", "not staged");
    setText("[data-webmcp-proposal-policy-reasons]", "Stage a proposal to see the policy decision for every operation.");
    setText("[data-webmcp-proposal-expiry]", "not staged");
    setText("[data-webmcp-preview-revision]", "Canonical revision unchanged · human Apply only");
    setText("[data-webmcp-review-gate]", "Human-only approval · creates the next revision");
    var commit = one("[data-webmcp-commit]");
    if (commit) {
      commit.textContent = "Apply staged changes";
      commit.removeAttribute("aria-label");
    }
    if (restoreReviewFocus && agentPanel && typeof agentPanel.focus === "function") {
      try { agentPanel.focus({ preventScroll: true }); }
      catch (_) { agentPanel.focus(); }
    }
  }

  function scrollAgentPanelTop() {
    var agentPanel = one(".agent-panel");
    if (!agentPanel) return;
    if (typeof agentPanel.scrollTo === "function") {
      agentPanel.scrollTo({ top: 0, behavior: "auto" });
      return;
    }
    agentPanel.scrollTop = 0;
  }

  function discoverPendingProposal() {
    var generation = ++proposalHydration;
    request("/api/studio/webmcp/proposals/current", {
      method: "GET",
      headers: { Accept: "application/json" },
      cache: "no-store",
      credentials: "same-origin"
    }).then(function (response) {
      return responsePayload(response).then(function (payload) {
        if (!response || !response.ok) throw responseError(response, payload);
        return payload;
      });
    }).then(function (payload) {
      if (generation !== proposalHydration) return;
      var proposal = payload && payload.proposal;
      if (!proposal || !proposal.proposalId) {
        if (pendingProposal) {
          var missingProposalID = pendingProposal.proposalId;
          revertScenePreview(missingProposalID).then(function (restored) {
            if (!restored) {
              requireCanonicalReload("The staged preview disappeared before it could be reversed; restoring the canonical scene.");
              return;
            }
            clearProposal("No staged WebMCP proposal is awaiting review in this browser session.");
            refreshPage();
          });
        }
        return;
      }
      if (!pendingProposal || pendingProposal.proposalId !== proposal.proposalId) clearApprovalOutcome();
      pendingProposal = proposal;
      toolStates.scene_preview_actions = "complete";
      renderToolFlow();
      renderProposal();
      activateScenePreview(proposal);
    }).catch(function (error) {
      if (generation !== proposalHydration) return;
      updateStatus({ state: "error", message: error && error.message ? error.message : "The staged proposal could not be restored." });
    });
  }

  function highlightFocusedEntity(id) {
    if (!id) return;
    var previous = document.querySelectorAll(".hierarchy-tree li.webmcp-focused");
    Array.prototype.forEach.call(previous, function (item) { item.classList.remove("webmcp-focused"); });
    var links = document.querySelectorAll(".hierarchy-tree [data-entity-id]");
    var match = null;
    Array.prototype.some.call(links, function (link) {
      if (String(link.getAttribute("data-entity-id")) !== id) return false;
      match = link;
      return true;
    });
    if (!match) return;
    var item = match.closest("li");
    if (item) item.classList.add("webmcp-focused");
    if (typeof match.scrollIntoView === "function") match.scrollIntoView({ block: "nearest" });
    match.focus({ preventScroll: true });
  }

  function selectionFromURL(value) {
    try {
      return String(new URL(value || window.location.href, window.location.href).searchParams.get("selection") || "");
    } catch (_) {
      return "";
    }
  }

  function clearAgentFocus() {
    pendingFocusNavigation = null;
    focusedEntityId = "";
    var focused = document.querySelectorAll(".hierarchy-tree li.webmcp-focused");
    Array.prototype.forEach.call(focused, function (item) { item.classList.remove("webmcp-focused"); });
  }

  function focusNavigationLanded(intent, url) {
    if (!intent || pendingFocusNavigation !== intent || selectionFromURL(url) !== intent.id) return false;
    pendingFocusNavigation = null;
    return true;
  }

  function scheduleFocusNavigation(intent) {
    if (!intent || pendingFocusNavigation !== intent) return;
    if (focusNavigationLanded(intent, window.location.href)) return;
    if (intent.scheduled || intent.inFlight) return;
    if (intent.attempts >= MAX_FOCUS_NAVIGATION_ATTEMPTS) {
      pendingFocusNavigation = null;
      return;
    }

    intent.scheduled = true;
    window.setTimeout(function () {
      if (pendingFocusNavigation !== intent) return;
      intent.scheduled = false;
      if (focusNavigationLanded(intent, window.location.href)) return;

      var targetURL = new URL(window.location.href);
      targetURL.searchParams.set("selection", intent.id);
      var target = targetURL.pathname + targetURL.search;
      var navigation = window.__gosx_page_nav;
      if (!navigation || typeof navigation.navigate !== "function") {
        pendingFocusNavigation = null;
        window.location.assign(target);
        return;
      }

      intent.attempts++;
      intent.inFlight = true;
      var navigationResult;
      try {
        navigationResult = navigation.navigate(target, { preserveScroll: true });
      } catch (_) {
        intent.inFlight = false;
        scheduleFocusNavigation(intent);
        return;
      }
      Promise.resolve(navigationResult).then(function (applied) {
        if (pendingFocusNavigation !== intent) return;
        intent.inFlight = false;
        if (applied !== false && focusNavigationLanded(intent, window.location.href)) return;
        // A managed form redirect can finish after the focus tool starts and
        // supersede this navigation. Reassert the newer focus intent, bounded
        // above so a broken route can never create a navigation loop.
        scheduleFocusNavigation(intent);
      }).catch(function () {
        if (pendingFocusNavigation !== intent) return;
        intent.inFlight = false;
        scheduleFocusNavigation(intent);
      });
    }, 0);
  }

  function focusEntity(detail) {
    var id = detail && detail.id ? String(detail.id) : "";
    if (!id) return;
    focusedEntityId = id;
    var intent = {
      id: id,
      attempts: 0,
      scheduled: false,
      inFlight: false
    };
    pendingFocusNavigation = intent;
    highlightFocusedEntity(id);
    if (window.__gosxStudioSelection && typeof window.__gosxStudioSelection.apply === "function") {
      window.__gosxStudioSelection.apply(id);
    }
    scheduleFocusNavigation(intent);
  }

  function refreshPage(target) {
    var navigation = window.__gosx && window.__gosx.navigation
      ? window.__gosx.navigation
      : window.__gosx_page_nav;
    var hasTarget = typeof target === "string" && target !== "";
    var destination = hasTarget ? target : window.location.href;
    if (!hasTarget && navigation && typeof navigation.revalidate === "function") {
      try {
        return Promise.resolve(navigation.revalidate({ replace: true, preserveScroll: true })).catch(function () {
          window.location.reload();
        });
      } catch (_) {
        window.location.reload();
        return Promise.resolve();
      }
    }
    if (navigation && typeof navigation.navigate === "function") {
      try {
        return Promise.resolve(navigation.navigate(destination, {
          replace: true, preserveScroll: !hasTarget, force: true, revalidate: true
        })).catch(function () {
          if (hasTarget) {
            window.location.assign(destination);
            return;
          }
          window.location.reload();
        });
      } catch (_) {
        if (hasTarget) {
          window.location.assign(destination);
        } else {
          window.location.reload();
        }
        return Promise.resolve();
      }
    }
    if (hasTarget) {
      window.location.assign(destination);
      return Promise.resolve();
    }
    window.location.reload();
    return Promise.resolve();
  }

  function renderedCanonicalRevision() {
    var field = one('input[type="hidden"][name="expectedRevision"]');
    var revision = field ? Number(field.value) : NaN;
    return Number.isSafeInteger(revision) && revision > 0 ? revision : null;
  }

  function proposalBaseRevision(proposal) {
    var receipt = proposal && proposal.receipt;
    var revision = receipt ? Number(receipt.beforeRevision) : NaN;
    return Number.isSafeInteger(revision) && revision > 0 ? revision : null;
  }

  function hideDemoPanel() {
    var panel = one("[data-studio-demo-panel]");
    if (panel) panel.hidden = true;
    demoClean = false;
    var copy = one("[data-webmcp-copy-prompt]");
    if (copy) copy.disabled = true;
  }

  function renderDemoStateFailure() {
    var panel = one("[data-studio-demo-panel]");
    var button = one("[data-studio-demo-reset]");
    var copy = one("[data-webmcp-copy-prompt]");
    demoClean = false;
    if (!panel || !button) return;
    panel.hidden = false;
    panel.setAttribute("data-state", "error");
    button.disabled = true;
    button.textContent = "Demo status unavailable";
    if (copy) copy.disabled = true;
    setText("[data-studio-demo-state]", "Could not verify the shared baseline · retrying automatically.");
    setText("[data-webmcp-copy-status]", "Wait for the shared demo check before running the showcase prompt.");
  }

  function discoverDemoState() {
    if (demoResetInFlight || demoStatusInFlight) return;
    var generation = ++demoStatusGeneration;
    demoStatusInFlight = true;
    request("/api/studio/demo/status", {
      method: "GET",
      headers: { Accept: "application/json" }
    }).then(function (response) {
      return responsePayload(response).then(function (payload) {
        if (!response || !response.ok) throw responseError(response, payload);
        return payload;
      });
    }).then(function (state) {
      if (generation !== demoStatusGeneration || demoResetInFlight) return;
      var panel = one("[data-studio-demo-panel]");
      var button = one("[data-studio-demo-reset]");
      var copy = one("[data-webmcp-copy-prompt]");
      if (!panel || !button || !state || state.enabled !== true) {
        hideDemoPanel();
        return;
      }
      panel.hidden = false;
      demoClean = state.clean === true;
      panel.setAttribute("data-state", demoClean ? "clean" : "dirty");
      button.setAttribute("data-revision", String(state.revision));
      button.disabled = false;
      button.textContent = demoClean ? "Reset clean demo" : "Prepare clean demo";
      if (copy) copy.disabled = !demoClean;
      setText(
        "[data-studio-demo-state]",
        demoClean
          ? "Clean baseline ready · the exact prompt is safe to run."
          : "Demo result preserved · reset when you are ready for another run."
      );
      setText(
        "[data-webmcp-copy-status]",
        demoClean ? "Ready for the browser agent." : "Current result preserved. Reset before another agent run."
      );
    }).catch(function () {
      if (generation !== demoStatusGeneration || demoResetInFlight) return;
      renderDemoStateFailure();
    }).finally(function () {
      if (generation === demoStatusGeneration) demoStatusInFlight = false;
    });
  }

  function resetDemo(button) {
    if (demoResetInFlight) return;
    var expectedRevision = Number(button.getAttribute("data-revision"));
    if (!Number.isSafeInteger(expectedRevision) || expectedRevision < 1) {
      updateStatus({ state: "error", message: "The shared demo revision is unavailable; refresh the view before resetting." });
      return;
    }
    if (!window.confirm("Reset the shared public demo scene for everyone? Pending proposals and current scene edits will be discarded.")) return;
    demoResetInFlight = true;
    demoStatusGeneration++;
    button.disabled = true;
    updateStatus({ state: "committing", message: "Restoring a clean shared demo scene…" });
    request("/api/studio/demo/reset", {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ expectedRevision: expectedRevision })
    }).then(function (response) {
      return responsePayload(response).then(function (payload) {
        if (!response || !response.ok) throw responseError(response, payload);
        return payload;
      });
    }).then(function (payload) {
      proposalHydration++;
      clearAgentFocus();
      clearApprovalOutcome();
      clearToolFlow();
      clearTrace();
      var previewID = pendingProposal && pendingProposal.proposalId;
      return revertScenePreview(previewID).then(function (restored) {
        if (!restored) {
          requireCanonicalReload("The demo reset succeeded, but its local preview could not be reversed; restoring the clean canonical scene.");
          return null;
        }
        clearProposal("The shared demo was restored; no staged proposal is awaiting review.");
        return payload;
      });
    }).then(function (payload) {
      if (!payload) return false;
      updateStatus({
        state: "ready",
        message: "Shared demo restored at scene revision " + String(payload && payload.revision || "current") + "."
      });
      scrollAgentPanelTop();
      return refreshPage(window.location.pathname);
    }).catch(function (error) {
      button.disabled = false;
      updateStatus({ state: "error", message: error && error.message ? error.message : "The shared demo could not be reset." });
    }).finally(function () {
      demoResetInFlight = false;
      demoStatusInFlight = false;
    });
  }

  function commitProposal(button) {
    if (!pendingProposal || !pendingProposal.proposalId) return;
    if (reviewInFlight) return;
    var proposal = pendingProposal;
    reviewInFlight = true;
    reviewButtons(true);
    setText("[data-webmcp-review-gate]", "Applying reviewed edits…");
    updateStatus({ state: "committing", message: "Applying the exact operations from the reviewed preview…" });
    request("/api/studio/webmcp/commits", {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ proposalId: proposal.proposalId })
    }).then(function (response) {
      return responsePayload(response).then(function (payload) {
        if (!response || !response.ok) throw responseError(response, payload);
        return payload;
      });
    }).then(function (payload) {
      proposalHydration++;
      return scenePreviewChain.then(function () {
        var receipt = proposal.receipt || {};
        var beforeRevision = receipt.beforeRevision == null ? "?" : String(receipt.beforeRevision);
        var afterRevision = payload && payload.receipt && payload.receipt.afterRevision != null
          ? String(payload.receipt.afterRevision)
          : "current";
        var operationCount = Number(receipt.operations || (
          Array.isArray(proposal.sceneCommands) ? proposal.sceneCommands.length : 0
        ));
        var editSummary = operationCount === 1 ? "1 exact edit" : operationCount + " exact edits";
        activeScenePreview = null;
        markScenePreview(false);
        clearProposal("The reviewed proposal was applied to the canonical scene.");
        updateStatus({
          state: "applied",
          message: "Human approved " + (operationCount > 0 ? editSummary : "the exact staged edits") +
            " · revision " + beforeRevision + " → " + afterRevision +
            " · same reviewed transaction."
        });
        showApprovalOutcome(
          (operationCount > 0 ? editSummary : "Reviewed edits") +
          " · revision " + beforeRevision + " → " + afterRevision +
          " · same transaction"
        );
        scrollAgentPanelTop();
        return refreshPage();
      });
    }).catch(function (error) {
      if (error && (error.status === 404 || error.status === 409 || error.status === 410)) {
        proposalHydration++;
        return revertScenePreview(proposal.proposalId).then(function (restored) {
          if (!restored) {
            return requireCanonicalReload("The proposal ended while its local preview was active; restoring the current canonical scene.");
          }
          clearProposal(
            error.status === 409
              ? "The scene changed before approval. Ask the agent to inspect the current revision and stage a fresh proposal."
              : "This staged proposal is no longer available. Ask the agent to stage it again."
          );
          updateStatus({
            state: "error",
            message: error.status === 409
              ? "Revision conflict: canonical state was preserved. Inspect the current scene and restage."
              : (error && error.message ? error.message : "The proposal is no longer available.")
          });
          return refreshPage();
        });
      }
      reviewInFlight = false;
      reviewButtons(false);
      renderProposalExpiry();
      setText("[data-webmcp-review-gate]", "Approval still available · canonical revision unchanged");
      updateStatus({ state: "error", message: error && error.message ? error.message : "The proposal could not be applied." });
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("WebMCP proposal commit", error, {
          scope: "studio", type: "webmcp", fallback: "human-review"
        });
      }
    });
  }

  function discardProposal(button) {
    if (!pendingProposal || !pendingProposal.proposalId) return;
    if (reviewInFlight) return;
    var proposalId = pendingProposal.proposalId;
    reviewInFlight = true;
    reviewButtons(true);
    setText("[data-webmcp-review-gate]", "Discarding preview · canonical scene stays unchanged…");
    updateStatus({ state: "committing", message: "Revoking the staged proposal without changing the canonical scene…" });
    request("/api/studio/webmcp/discards", {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ proposalId: proposalId })
    }).then(function (response) {
      return responsePayload(response).then(function (payload) {
        if (!response || !response.ok) throw responseError(response, payload);
        return payload;
      });
    }).then(function () {
      proposalHydration++;
      clearToolFlow();
      return revertScenePreview(proposalId).then(function (restored) {
        if (!restored) {
          return requireCanonicalReload("The proposal was revoked, but its local preview could not be reversed; restoring the canonical scene.");
        }
        clearProposal("Proposal revoked; the canonical scene was never changed.");
        updateStatus({ state: "ready", message: "The staged proposal was revoked without changing the canonical scene." });
        scrollAgentPanelTop();
        return refreshPage();
      });
    }).catch(function (error) {
      if (error && (error.status === 404 || error.status === 410)) {
        proposalHydration++;
        return revertScenePreview(proposalId).then(function (restored) {
          if (!restored) {
            return requireCanonicalReload("The expired proposal preview could not be reversed; restoring the canonical scene.");
          }
          clearProposal("This staged proposal is no longer available.");
          updateStatus({ state: "ready", message: "The staged proposal was already unavailable; canonical state is unchanged." });
          return refreshPage();
        });
      }
      reviewInFlight = false;
      reviewButtons(false);
      renderProposalExpiry();
      setText("[data-webmcp-review-gate]", "Review still available · canonical revision unchanged");
      updateStatus({ state: "error", message: error && error.message ? error.message : "The staged proposal could not be revoked." });
    });
  }

  function copyDemoPrompt(button) {
    var prompt = one("[data-webmcp-demo-prompt]");
    var status = one("[data-webmcp-copy-status]");
    var value = prompt ? String(prompt.textContent || "").trim() : "";
    if (!value) return;
    if (!demoClean) {
      if (status) status.textContent = "Prepare a clean demo before copying this exact showcase prompt.";
      return;
    }
    var copied;
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      copied = navigator.clipboard.writeText(value);
    } else {
      copied = new Promise(function (resolve, reject) {
        var input = document.createElement("textarea");
        input.value = value;
        input.setAttribute("readonly", "");
        input.style.position = "fixed";
        input.style.opacity = "0";
        document.body.appendChild(input);
        input.select();
        try {
          if (!document.execCommand("copy")) throw new Error("Copy was not accepted by the browser.");
          resolve();
        } catch (error) {
          reject(error);
        } finally {
          input.remove();
        }
      });
    }
    button.disabled = true;
    Promise.resolve(copied).then(function () {
      if (status) status.textContent = "Copied. Paste it into the browser agent.";
      button.textContent = "Prompt copied";
    }).catch(function () {
      if (status) status.textContent = "Select the prompt text and copy it manually.";
    }).finally(function () {
      button.disabled = false;
    });
  }

  document.addEventListener("studio:webmcp:status", function (event) {
    updateStatus(event && event.detail);
  });

  document.addEventListener("studio:webmcp:proposal", function (event) {
    var proposal = event && event.detail ? event.detail : null;
    clearApprovalOutcome();
    pendingProposal = proposal;
    proposalHydration++;
    renderProposal();
    activateScenePreview(proposal);
  });

  document.addEventListener("studio:webmcp:trace", function (event) {
    appendTrace(event && event.detail);
  });

  document.addEventListener("studio:webmcp:focus", function (event) {
    focusEntity(event && event.detail);
  });

  document.addEventListener("gosx:navigate", function (event) {
    renderStatus();
    renderToolFlow();
    renderTrace();
    renderApprovalOutcome();
    if (pendingProposal) renderProposal();
    if (activeScenePreview) {
      var renderedRevision = renderedCanonicalRevision();
      var previewRevision = proposalBaseRevision(activeScenePreview);
      if (renderedRevision !== null && previewRevision !== null && renderedRevision !== previewRevision) {
        requireCanonicalReload("The canonical scene changed during review; restoring its current revision before restaging.");
        return;
      }
      markScenePreview(false);
      activateScenePreview(activeScenePreview);
    }
    if (focusedEntityId) {
      var intent = pendingFocusNavigation;
      var navigationURL = event && event.detail ? event.detail.url : window.location.href;
      window.setTimeout(function () {
        if (intent && pendingFocusNavigation === intent) {
          if (focusNavigationLanded(intent, navigationURL)) {
            highlightFocusedEntity(focusedEntityId);
            return;
          }
          // Server HTML from the older navigation may have restored its own
          // selection. Keep the viewport local-first while the bounded retry
          // reconciles the Inspector and URL to the newer WebMCP intent.
          if (window.__gosxStudioSelection && typeof window.__gosxStudioSelection.apply === "function") {
            window.__gosxStudioSelection.apply(intent.id);
          }
          highlightFocusedEntity(intent.id);
          scheduleFocusNavigation(intent);
          return;
        }
        highlightFocusedEntity(focusedEntityId);
      }, 0);
    }
    discoverDemoState();
    discoverPendingProposal();
  });

  document.addEventListener("gosx:scene3d:input", function (event) {
    var detail = event && event.detail;
    var input = detail && detail.kind === "pick" ? detail.input : null;
    if (!input || (input.type !== "select" && input.type !== "click") || !input.selectedID) return;
    clearAgentFocus();
  });

  document.addEventListener("click", function (event) {
    var hierarchySelection = event.target && event.target.closest
      ? event.target.closest("a[data-entity-id]")
      : null;
    if (hierarchySelection) clearAgentFocus();
    var reset = event.target && event.target.closest ? event.target.closest("[data-studio-demo-reset]") : null;
    if (reset) {
      event.preventDefault();
      resetDemo(reset);
      return;
    }
    var commit = event.target && event.target.closest ? event.target.closest("[data-webmcp-commit]") : null;
    if (commit) {
      event.preventDefault();
      commitProposal(commit);
      return;
    }
    var discard = event.target && event.target.closest ? event.target.closest("[data-webmcp-discard]") : null;
    if (discard) {
      event.preventDefault();
      discardProposal(discard);
      return;
    }
    var copy = event.target && event.target.closest ? event.target.closest("[data-webmcp-copy-prompt]") : null;
    if (!copy) return;
    event.preventDefault();
    copyDemoPrompt(copy);
  });

  // A narrow diagnostics surface keeps the asynchronous preview lifecycle
  // executable in browser-free tests and observable during demo rehearsals.
  // It does not expose commit: canonical authority remains server-owned and
  // reachable only through the explicit human review control above.
  window.__gosxStudioWebMCPPreview = Object.freeze({
    activate: activateScenePreview,
    revert: revertScenePreview,
    activeProposalID: function () {
      return activeScenePreview ? String(activeScenePreview.proposalId || "") : "";
    }
  });

  renderStatus();
  renderToolFlow();
  renderTrace();
  renderApprovalOutcome();
  discoverDemoState();
  discoverPendingProposal();
  window.setInterval(renderProposalExpiry, 10000);
  window.setInterval(discoverDemoState, 15000);
})();
