(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioWebMCPUI) return;
  window.__gosxStudioWebMCPUI = true;

  var pendingProposal = null;
  var proposalHydration = 0;
  var focusedEntityId = "";
  var toolStates = {
    scene_get_state: "idle",
    scene_find_objects: "idle",
    scene_focus_object: "idle",
    scene_preview_actions: "idle"
  };
  var toolStateStorageKey = "gosx3d:webmcp-flow:v1";
  try {
    var storedToolStates = JSON.parse(window.sessionStorage.getItem(toolStateStorageKey) || "null");
    Object.keys(toolStates).forEach(function (tool) {
      if (storedToolStates && ["idle", "running", "complete", "error"].indexOf(storedToolStates[tool]) >= 0) {
        toolStates[tool] = storedToolStates[tool];
      }
    });
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
    if (element) element.textContent = value == null ? "" : String(value);
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
    setText("[data-webmcp-status-label]", latestStatus.label || statusLabel(latestStatus.state));
    setText("[data-webmcp-status-message]", latestStatus.message);
    setText("[data-webmcp-tool-count]", latestStatus.toolCount + (latestStatus.toolCount === 1 ? " tool" : " tools"));
  }

  function renderToolFlow() {
    Object.keys(toolStates).forEach(function (tool) {
      var item = one('[data-webmcp-flow-tool="' + tool + '"]');
      if (item) item.setAttribute("data-state", toolStates[tool]);
    });
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

  function vectorsEqual(left, right) {
    if (!left || !right) return left === right;
    return ["x", "y", "z"].every(function (axis) {
      return Number(left[axis]) === Number(right[axis]);
    });
  }

  function changeSummary(change) {
    change = change || {};
    var kind = String(change.kind || "scene edit");
    var target = String(change.target || "scene");
    var before = change.before || {};
    var after = change.after || {};
    if (kind === "rename-entity" && (before.name || after.name)) {
      return kind + " · " + target + " · " + String(before.name || "unnamed") + " → " + String(after.name || "unnamed");
    }
    if (kind === "assign-material") {
      var beforeMaterial = before.material || before.mesh && before.mesh.material;
      var afterMaterial = after.material || after.mesh && after.mesh.material;
      if (beforeMaterial || afterMaterial) {
        return kind + " · " + target + " · " + String(beforeMaterial || "none") + " → " + String(afterMaterial || "none");
      }
    }
    if (kind === "set-transform" && after.transform) {
      var beforeTransform = before.transform || {};
      var afterTransform = after.transform || {};
      var fields = ["position", "rotation", "scale"].filter(function (field) {
        return !vectorsEqual(beforeTransform[field], afterTransform[field]);
      }).map(function (field) {
        return field + " " + vectorSummary(beforeTransform[field]) + " → " + vectorSummary(afterTransform[field]);
      });
      return kind + " · " + target + (fields.length ? " · " + fields.join(" · ") : " · transform updated");
    }
    return kind + " · " + target;
  }

  function renderChanges(receipt) {
    var list = one("[data-webmcp-proposal-changes]");
    if (!list) return;
    while (list.firstChild) list.removeChild(list.firstChild);
    var changes = receipt && Array.isArray(receipt.changes) ? receipt.changes : [];
    changes.forEach(function (change) {
      var item = document.createElement("li");
      item.textContent = changeSummary(change);
      list.appendChild(item);
    });
    list.hidden = changes.length === 0;
  }

  function renderProposal() {
    if (!pendingProposal) return;
    var receipt = pendingProposal.receipt || {};
    var affected = Array.isArray(receipt.affected) ? receipt.affected : [];
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
    }
    var canonicalRevision = receipt.beforeRevision == null ? "?" : String(receipt.beforeRevision);
    var approvalRevision = pendingProposal.preview && pendingProposal.preview.revision != null
      ? String(pendingProposal.preview.revision)
      : (Number.isFinite(Number(receipt.beforeRevision)) ? String(Number(receipt.beforeRevision) + 1) : "?");
    setText("[data-webmcp-proposal-revision]", "canonical " + canonicalRevision + " unchanged · approval " + approvalRevision);
    setText(
      "[data-webmcp-proposal-affected]",
      affected.length ? affected.join(", ") : String(receipt.operations || 0) + " operations"
    );
    var fingerprint = one("[data-webmcp-proposal-fingerprint]");
    var fingerprintValue = receipt.afterFingerprint || "preview ready";
    if (fingerprint) {
      fingerprint.textContent = fingerprintValue;
      fingerprint.setAttribute("title", fingerprintValue);
    }
    renderChanges(receipt);
    var actions = one("[data-webmcp-review-actions]");
    if (actions) actions.hidden = false;
    updateStatus({
      state: "proposal",
      tool: "scene_preview_actions",
      message: "Canonical revision " + canonicalRevision + " is unchanged. Review the exact staged operations, then apply or discard."
    });
  }

  function clearProposal(message) {
    pendingProposal = null;
    var agentPanel = one(".agent-panel");
    if (agentPanel) agentPanel.classList.remove("has-pending-proposal");
    var rationale = one("[data-webmcp-proposal-rationale]");
    if (rationale) rationale.hidden = true;
    var actions = one("[data-webmcp-review-actions]");
    if (actions) actions.hidden = true;
    renderChanges(null);
    setText("[data-webmcp-proposal-summary]", message || "No staged WebMCP proposal is awaiting review.");
    setText("[data-webmcp-proposal-actor]", "none");
    setText("[data-webmcp-proposal-policy]", "Arbiter · awaiting proposal");
    setText("[data-webmcp-proposal-revision]", "current");
    setText("[data-webmcp-proposal-affected]", "0 entities");
    setText("[data-webmcp-proposal-fingerprint]", "not staged");
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
        if (pendingProposal) clearProposal("No staged WebMCP proposal is awaiting review in this browser session.");
        return;
      }
      pendingProposal = proposal;
      toolStates.scene_preview_actions = "complete";
      renderToolFlow();
      renderProposal();
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

  function focusEntity(detail) {
    var id = detail && detail.id ? String(detail.id) : "";
    if (!id) return;
    focusedEntityId = id;
    highlightFocusedEntity(id);

    var current = new URL(window.location.href).searchParams.get("selection");
    if (current === id) return;
    var targetURL = new URL(window.location.href);
    targetURL.searchParams.set("selection", id);
    var target = targetURL.pathname + targetURL.search;
    window.setTimeout(function () {
      if (window.__gosx_page_nav && typeof window.__gosx_page_nav.navigate === "function") {
        window.__gosx_page_nav.navigate(target, { preserveScroll: true });
        return;
      }
      window.location.assign(target);
    }, 0);
  }

  function refreshPage() {
    var navigation = window.__gosx && window.__gosx.navigation
      ? window.__gosx.navigation
      : window.__gosx_page_nav;
    if (navigation && typeof navigation.revalidate === "function") {
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
        return Promise.resolve(navigation.navigate(window.location.href, {
          replace: true, preserveScroll: true, force: true, revalidate: true
        })).catch(function () {
          window.location.reload();
        });
      } catch (_) {
        window.location.reload();
        return Promise.resolve();
      }
    }
    window.location.reload();
    return Promise.resolve();
  }

  function hideDemoPanel() {
    var panel = one("[data-studio-demo-panel]");
    if (panel) panel.hidden = true;
  }

  function discoverDemoState() {
    request("/api/studio/demo/status", {
      method: "GET",
      headers: { Accept: "application/json" }
    }).then(function (response) {
      return responsePayload(response).then(function (payload) {
        if (!response || !response.ok) throw responseError(response, payload);
        return payload;
      });
    }).then(function (state) {
      var panel = one("[data-studio-demo-panel]");
      var button = one("[data-studio-demo-reset]");
      if (!panel || !button || !state || state.enabled !== true) {
        hideDemoPanel();
        return;
      }
      panel.hidden = false;
      button.setAttribute("data-revision", String(state.revision));
      button.disabled = false;
    }).catch(hideDemoPanel);
  }

  function resetDemo(button) {
    var expectedRevision = Number(button.getAttribute("data-revision"));
    if (!Number.isSafeInteger(expectedRevision) || expectedRevision < 1) {
      updateStatus({ state: "error", message: "The shared demo revision is unavailable; reload before resetting." });
      return;
    }
    if (!window.confirm("Reset the shared public demo scene for everyone? Pending proposals and current scene edits will be discarded.")) return;
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
      clearToolFlow();
      clearProposal("The shared demo was restored; no staged proposal is awaiting review.");
      updateStatus({
        state: "ready",
        message: "Shared demo restored at scene revision " + String(payload && payload.revision || "current") + "."
      });
      return refreshPage();
    }).catch(function (error) {
      button.disabled = false;
      updateStatus({ state: "error", message: error && error.message ? error.message : "The shared demo could not be reset." });
    });
  }

  function commitProposal(button) {
    if (!pendingProposal || !pendingProposal.proposalId) return;
    var proposal = pendingProposal;
    button.disabled = true;
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
      clearProposal("The reviewed proposal was applied to the canonical scene.");
      updateStatus({
        state: "applied",
        message: "Applied by human review at scene revision " +
          String(payload && payload.receipt ? payload.receipt.afterRevision : "current") + "."
      });
      return refreshPage();
    }).catch(function (error) {
      if (error && (error.status === 404 || error.status === 409 || error.status === 410)) {
        proposalHydration++;
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
        return;
      }
      button.disabled = false;
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
    var proposalId = pendingProposal.proposalId;
    button.disabled = true;
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
      clearProposal("Proposal revoked; the canonical scene was never changed.");
      updateStatus({ state: "ready", message: "The staged proposal was revoked without changing the canonical scene." });
    }).catch(function (error) {
      if (error && (error.status === 404 || error.status === 410)) {
        proposalHydration++;
        clearProposal("This staged proposal is no longer available.");
        updateStatus({ state: "ready", message: "The staged proposal was already unavailable; canonical state is unchanged." });
        return;
      }
      button.disabled = false;
      updateStatus({ state: "error", message: error && error.message ? error.message : "The staged proposal could not be revoked." });
    });
  }

  function copyDemoPrompt(button) {
    var prompt = one("[data-webmcp-demo-prompt]");
    var status = one("[data-webmcp-copy-status]");
    var value = prompt ? String(prompt.textContent || "").trim() : "";
    if (!value) return;
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
    pendingProposal = event && event.detail ? event.detail : null;
    proposalHydration++;
    renderProposal();
  });

  document.addEventListener("studio:webmcp:focus", function (event) {
    focusEntity(event && event.detail);
  });

  document.addEventListener("gosx:navigate", function () {
    renderStatus();
    renderToolFlow();
    if (pendingProposal) renderProposal();
    if (focusedEntityId) window.setTimeout(function () { highlightFocusedEntity(focusedEntityId); }, 0);
    discoverDemoState();
    discoverPendingProposal();
  });

  document.addEventListener("click", function (event) {
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

  renderStatus();
  renderToolFlow();
  discoverDemoState();
  discoverPendingProposal();
})();
