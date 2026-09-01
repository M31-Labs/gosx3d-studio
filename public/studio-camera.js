(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioCameraRig) return;
  window.__gosxStudioCameraRig = true;

  var INPUT = "studio.viewport.cameraIn";
  var OUTPUT = "studio.viewport.cameraOut";
  var lastPose = null;
  var activeView = "perspective";

  function setSignal(value) {
    var runtime = window.__gosx_runtime_api;
    if (runtime && typeof runtime.setSharedSignalValue === "function") {
      try {
        runtime.setSharedSignalValue(INPUT, value);
        return true;
      } catch (_) {}
    }
    if (typeof window.__gosx_set_shared_signal === "function") {
      try {
        var result = window.__gosx_set_shared_signal(INPUT, JSON.stringify(value));
        if (typeof result !== "string" || result === "") return true;
      } catch (_) {}
    }
    if (typeof window.__gosx_notify_shared_signal === "function") {
      try {
        window.__gosx_notify_shared_signal(INPUT, JSON.stringify(value));
        return true;
      } catch (_) {
        return false;
      }
    }
    return false;
  }

  function rememberPose(value) {
    if (value && typeof value === "object") lastPose = value;
  }

  function subscribePose() {
    if (typeof window.__gosx_subscribe_shared_signal !== "function") return;
    try {
      window.__gosx_subscribe_shared_signal(OUTPUT, rememberPose);
    } catch (_) {}
  }

  function readPose() {
    return lastPose;
  }

  function focusTarget() {
    var stage = document.querySelector("[data-camera-focus-x]");
    if (!stage) return { x: 0, y: 0, z: 0 };
    return {
      x: Number(stage.getAttribute("data-camera-focus-x")) || 0,
      y: Number(stage.getAttribute("data-camera-focus-y")) || 0,
      z: Number(stage.getAttribute("data-camera-focus-z")) || 0
    };
  }

  // View presets. Perspective restores the authored camera; front/top/right
  // are orthographic authoring views framed on the focus target.
  function applyView(view) {
    var target = focusTarget();
    var span = 12;
    var presets = {
      front: { kind: "orthographic", x: target.x, y: target.y, z: target.z + 30, rotationX: 0, rotationY: 0, rotationZ: 0 },
      right: { kind: "orthographic", x: target.x + 30, y: target.y, z: target.z, rotationX: 0, rotationY: Math.PI / 2, rotationZ: 0 },
      top:   { kind: "orthographic", x: target.x, y: target.y + 30, z: target.z, rotationX: -Math.PI / 2, rotationY: 0, rotationZ: 0 }
    };
    if (view === "perspective") {
      var stage = document.querySelector("[data-camera-home]");
      var home = stage ? stage.getAttribute("data-camera-home") : "";
      var parts = (home || "0,8.2,10.4").split(",").map(Number);
      var pose = { kind: "perspective", x: parts[0] || 0, y: parts[1] || 8, z: parts[2] || 10, fov: 45 };
      lastPose = pose;
      return setSignal(pose);
    }
    var preset = presets[view];
    if (!preset) return false;
    preset.left = -span; preset.right = span; preset.top = span; preset.bottom = -span; preset.zoom = 1;
    lastPose = preset;
    return setSignal(preset);
  }

  function nudge(dx, dy, dz) {
    var pose = readPose();
    if (!pose || !Number.isFinite(Number(pose.x)) || !Number.isFinite(Number(pose.y)) || !Number.isFinite(Number(pose.z))) {
      applyView("perspective");
      pose = lastPose;
    }
    var next = JSON.parse(JSON.stringify(pose));
    next.x = Number(next.x) + dx;
    next.y = Number(next.y) + dy;
    next.z = Number(next.z) + dz;
    lastPose = next;
    setSignal(next);
  }

  function syncButtons() {
    document.querySelectorAll("[data-camera-view]").forEach(function (button) {
      var active = button.getAttribute("data-camera-view") === activeView;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", active ? "true" : "false");
    });
  }

  function chooseView(view) {
    if (!applyView(view)) {
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("camera view", new Error("shared signal runtime unavailable"), {
          scope: "studio", type: "camera", fallback: "authored-camera"
        });
      }
      return false;
    }
    activeView = view;
    syncButtons();
    return true;
  }

  document.addEventListener("click", function (event) {
    var button = event.target && typeof event.target.closest === "function"
      ? event.target.closest("[data-camera-view]")
      : null;
    if (!button) return;
    chooseView(String(button.getAttribute("data-camera-view") || ""));
  });

  document.addEventListener("keydown", function (event) {
    if (event.target && /INPUT|TEXTAREA|SELECT/.test(event.target.tagName)) return;
    var step = event.shiftKey ? 2 : 0.5;
    if (String(event.key || "").toLowerCase() === "f") chooseView("perspective");
    else if (event.key === "ArrowLeft") nudge(-step, 0, 0);
    else if (event.key === "ArrowRight") nudge(step, 0, 0);
    else if (event.key === "ArrowUp") nudge(0, step, 0);
    else if (event.key === "ArrowDown") nudge(0, -step, 0);
    else if (event.key === "+" || event.key === "=") nudge(0, 0, -step);
    else if (event.key === "-") nudge(0, 0, step);
    else return;
    event.preventDefault();
  });

  document.addEventListener("gosx:navigate", function () {
    window.setTimeout(syncButtons, 0);
  });

  subscribePose();
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", syncButtons);
  else syncButtons();
})();
