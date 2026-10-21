(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioCertificationBridge) return;
  window.__gosxStudioCertificationBridge = true;

  // The evidence suite runs in the background, so after an edit the card reads
  // "recomputing" and had no way back to "current" until something else caused
  // a render.
  //
  // One heartbeat drives this. Each tick reads the card's own state out of the
  // DOM, which costs nothing and needs no navigation event: the page morphs in
  // place after an edit, and the next tick simply sees the new class. A request
  // goes out only while the card is actually waiting for a run, so a settled
  // card is silent.
  var TICK_MS = 1500;
  var MAX_WAITING_TICKS = 120; // About three minutes, then leave the card as it is.

  var waitingTicks = 0;
  var inFlight = false;

  function cardState() {
    var element = document.querySelector(".certification-state");
    if (!element) return "";
    var match = /certification-state-([a-z]+)/.exec(element.className || "");
    return match ? match[1] : "";
  }

  function revision(root) {
    var field = root.querySelector('input[name="expectedRevision"]');
    return field ? String(field.value || field.getAttribute("value") || "") : "";
  }

  function refresh() {
    var expectedRevision = revision(document);
    return request("/", { headers: { Accept: "text/html" }, cache: "no-store", credentials: "same-origin" })
      .then(function (response) { return response && response.ok ? response.text() : null; })
      .then(function (html) {
        if (!html || !expectedRevision || revision(document) !== expectedRevision) return;
        var rendered = new DOMParser().parseFromString(html, "text/html");
        if (revision(rendered) !== expectedRevision) return;
        // Evidence completing is not a navigation. Reconcile only its
        // canonical markup; never race a selection with an older full page.
        [".certification-card", ".telemetry-proof.certification-state"].forEach(function (selector) {
          var current = document.querySelector(selector);
          var next = rendered.querySelector(selector);
          if (!current || !next) return;
          var details = current.querySelector("details");
          var open = details && details.open;
          current.className = next.className;
          current.replaceChildren.apply(current, Array.prototype.map.call(next.childNodes, function (child) {
            return document.importNode(child, true);
          }));
          if (details) {
            var replacement = current.querySelector("details");
            if (replacement) replacement.open = open;
          }
        });
      });
  }

  function request(url, options) {
    if (window.__gosx && typeof window.__gosx.request === "function") {
      return window.__gosx.request(url, options);
    }
    return fetch(url, options);
  }

  function tick() {
    var state = cardState();
    if (state !== "recomputing" && state !== "pending") {
      waitingTicks = 0;
      return;
    }
    if (inFlight || waitingTicks++ >= MAX_WAITING_TICKS) return;
    inFlight = true;
    request("/api/studio/certification/state", { headers: { Accept: "application/json" } })
      .then(function (response) {
        return response && response.ok ? response.json() : null;
      })
      .then(function (payload) {
        if (!payload) return;
        // The finished run has to describe the document on screen. A newer
        // edit started another run, and a later tick catches that one.
        if (payload.state === "current" && payload.revision === payload.documentRevision) {
          waitingTicks = 0;
          return refresh();
        }
      })
      .catch(function () {})
      .finally(function () {
        inFlight = false;
      });
  }

  window.setInterval(tick, TICK_MS);
})();
