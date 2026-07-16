(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioCameraRig) return;
  window.__gosxStudioCameraRig = true;

  var INPUT = "studio.viewport.cameraIn";
  var OUTPUT = "studio.viewport.cameraOut";
  var lastPose = null;

  function setSignal(value) {
    if (typeof window.__gosx_set_shared_signal !== "function") return false;
    window.__gosx_set_shared_signal(INPUT, value);
    return true;
  }

  function readPose() {
    if (typeof window.__gosx_get_shared_signal === "function") {
      var value = window.__gosx_get_shared_signal(OUTPUT);
      if (value && typeof value === "object") return value;
    }
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
      front: { kind: "orthographic", position: { x: target.x, y: target.y, z: target.z + 30 }, rotation: { x: 0, y: 0, z: 0 } },
      right: { kind: "orthographic", position: { x: target.x + 30, y: target.y, z: target.z }, rotation: { x: 0, y: Math.PI / 2, z: 0 } },
      top:   { kind: "orthographic", position: { x: target.x, y: target.y + 30, z: target.z }, rotation: { x: -Math.PI / 2, y: 0, z: 0 } }
    };
    if (view === "perspective") {
      var stage = document.querySelector("[data-camera-home]");
      var home = stage ? stage.getAttribute("data-camera-home") : "";
      var parts = (home || "0,8.2,10.4").split(",").map(Number);
      var pose = { kind: "perspective", position: { x: parts[0] || 0, y: parts[1] || 8, z: parts[2] || 10 }, fov: 45 };
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
    if (!pose || !pose.position) {
      applyView("perspective");
      pose = lastPose;
    }
    var next = JSON.parse(JSON.stringify(pose));
    next.position.x += dx; next.position.y += dy; next.position.z += dz;
    lastPose = next;
    setSignal(next);
  }

  function bind() {
    document.querySelectorAll("[data-camera-view]").forEach(function (button) {
      button.addEventListener("click", function () {
        applyView(String(button.getAttribute("data-camera-view") || ""));
      });
    });
    document.addEventListener("keydown", function (event) {
      if (event.target && /INPUT|TEXTAREA|SELECT/.test(event.target.tagName)) return;
      var step = event.shiftKey ? 2 : 0.5;
      if (event.key === "f") { applyView("perspective"); var t = focusTarget(); nudge(t.x * 0, 0, 0); }
      else if (event.key === "ArrowLeft") nudge(-step, 0, 0);
      else if (event.key === "ArrowRight") nudge(step, 0, 0);
      else if (event.key === "ArrowUp") nudge(0, step, 0);
      else if (event.key === "ArrowDown") nudge(0, -step, 0);
      else if (event.key === "+" || event.key === "=") nudge(0, 0, -step);
      else if (event.key === "-") nudge(0, 0, step);
      else return;
      event.preventDefault();
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
