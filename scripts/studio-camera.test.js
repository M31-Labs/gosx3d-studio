"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");
const source = fs.readFileSync(path.join(__dirname, "..", "public", "studio-camera.js"), "utf8");

function cameraHarness() {
  const listeners = new Map();
  const signals = [];
  let remember;
  let resized;
  const stage = { clientWidth: 600, clientHeight: 400, getAttribute() { return null; } };
  const document = {
    readyState: "complete",
    querySelector() { return stage; },
    querySelectorAll() { return []; },
    addEventListener(name, listener) { listeners.set(name, listener); }
  };
  const window = {
    __gosx_runtime_api: { setSharedSignalValue(name, value) { signals.push({ ...value }); } },
    __gosx_subscribe_shared_signal(name, callback) { remember = callback; },
    addEventListener() {},
    setTimeout(callback) { callback(); }
  };
  class ResizeObserver {
    constructor(callback) { resized = callback; }
    observe() {}
    disconnect() {}
  }
  vm.runInNewContext(source, { window, document, ResizeObserver }, { filename: "studio-camera.js" });
  return {
    choose(view) { listeners.get("click")({ target: { closest() { return { getAttribute() { return view; } }; } } }); },
    resize(width, height) { stage.clientWidth = width; stage.clientHeight = height; resized(); },
    remember(value) { remember(value); },
    signals
  };
}

test("orthographic presets use the actual stage aspect, not square projection bounds", () => {
  const h = cameraHarness();
  for (const view of ["top", "front", "right"]) {
    h.choose(view);
    const pose = h.signals.at(-1);
    assert.equal((pose.right - pose.left) / (pose.top - pose.bottom), 1.5);
    assert.equal(pose.zoom, 1);
  }
});

test("panel resizing adjusts only projection bounds and skips hidden or unchanged stages", () => {
  const h = cameraHarness();
  const pose = { kind: "orthographic", x: 3, y: 4, z: 5, rotationX: -0.7, rotationY: 0.8, rotationZ: 0.2, left: -18, right: 18, top: 12, bottom: -12, zoom: 2 };
  h.remember(pose);
  h.resize(300, 600);
  assert.deepEqual(h.signals.at(-1), { ...pose, left: -6, right: 6 });
  h.resize(300, 600);
  h.resize(0, 600);
  assert.equal(h.signals.length, 1, "no resize feedback or hidden-pane reset");
  h.resize(600, 300);
  assert.deepEqual(h.signals.at(-1), { ...pose, left: -24, right: 24 });
  h.remember({ ...pose, kind: "perspective" });
  h.resize(700, 300);
  assert.equal(h.signals.length, 2, "perspective camera remains runtime-owned");
});
