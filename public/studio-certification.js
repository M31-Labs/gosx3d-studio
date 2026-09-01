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

  function refresh() {
    var navigation = window.__gosx && window.__gosx.navigation
      ? window.__gosx.navigation
      : window.__gosx_page_nav;
    if (navigation && typeof navigation.revalidate === "function") {
      navigation.revalidate({ replace: true, preserveScroll: true }).catch(function () { window.location.reload(); });
      return;
    }
    if (navigation && typeof navigation.navigate === "function") {
      navigation.navigate(window.location.href, {
        replace: true, preserveScroll: true, force: true, revalidate: true
      }).catch(function () { window.location.reload(); });
      return;
    }
    window.location.reload();
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
        inFlight = false;
        if (!payload) return;
        // The finished run has to describe the document on screen. A newer
        // edit started another run, and a later tick catches that one.
        if (payload.state === "current" && payload.revision === payload.documentRevision) {
          waitingTicks = 0;
          refresh();
        }
      })
      .catch(function () {
        inFlight = false;
      });
  }

  window.setInterval(tick, TICK_MS);
})();
