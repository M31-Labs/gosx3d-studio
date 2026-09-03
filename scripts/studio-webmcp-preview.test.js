"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "..", "public", "studio-webmcp-ui.js"), "utf8");

function element() {
  const attributes = new Map();
  return {
    hidden: true,
    setAttribute(name, value) { attributes.set(String(name), String(value)); },
    removeAttribute(name) { attributes.delete(String(name)); },
    getAttribute(name) { return attributes.has(String(name)) ? attributes.get(String(name)) : null; },
    hasAttribute(name) { return attributes.has(String(name)); }
  };
}

function response(payload) {
  return { ok: true, status: 200, json: () => Promise.resolve(payload) };
}

function previewHarness() {
  const stage = element();
  const badge = element();
  let mount = element();
  let commandHandler = () => Promise.resolve();
  const commandCalls = [];
  let navigationHandler = (target) => {
    window.location.href = new URL(target, window.location.href).href;
    return Promise.resolve(true);
  };
  const navigationCalls = [];
  const selectionCalls = [];
  let reloadCalls = 0;
  const listeners = new Map();
  const document = {
    activeElement: null,
    querySelector(selector) {
      if (selector === ".scene-stage") return stage;
      if (selector === "[data-webmcp-preview-badge]") return badge;
      if (selector === "[data-gosx-scene3d]") return mount;
      return null;
    },
    querySelectorAll() { return []; },
    addEventListener(name, listener) {
      const values = listeners.get(name) || [];
      values.push(listener);
      listeners.set(name, values);
    },
    createElement() { return element(); },
    body: element()
  };
  const window = {
    document,
    location: {
      href: "https://studio.test/?selection=board",
      pathname: "/",
      assign() {},
      reload() { reloadCalls += 1; }
    },
    sessionStorage: { getItem() { return null; }, setItem() {} },
    setTimeout,
    setInterval() { return 1; },
    matchMedia() { return { matches: true }; },
    __gosx: {
      scene3d: {
        dispatchCommands(target, commands) {
          commandCalls.push({ target, commands });
          return commandHandler(target, commands);
        }
      }
    },
    __gosx_page_nav: {
      navigate(target, options) {
        navigationCalls.push({ target, options });
        return navigationHandler(target, options);
      }
    },
    __gosxStudioSelection: {
      apply(id) { selectionCalls.push(String(id)); }
    }
  };
  const sandbox = {
    window,
    document,
    URL,
    Promise,
    Object,
    Array,
    String,
    Number,
    Boolean,
    Date,
    Error,
    JSON,
    Math,
    isFinite,
    fetch(url) {
      return Promise.resolve(response(String(url).includes("/demo/status")
        ? { enabled: false }
        : { proposal: null }));
    }
  };
  vm.runInNewContext(source, sandbox, { filename: "studio-webmcp-ui.js" });
  return {
    api: window.__gosxStudioWebMCPPreview,
    badge,
    commandCalls,
    mount: () => mount,
    replaceMount() { mount = element(); return mount; },
    onCommands(handler) { commandHandler = handler; },
    onNavigate(handler) { navigationHandler = handler; },
    navigationCalls,
    selectionCalls,
    setHref(value) { window.location.href = String(value); },
    dispatch(name, detail, fields) {
      const event = Object.assign({ detail: detail || null }, fields || {});
      (listeners.get(name) || []).forEach((listener) => listener(event));
    },
    reloadCalls: () => reloadCalls
  };
}

function nextTask() {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

function proposal(id) {
  return {
    proposalId: id,
    sceneCommands: [{ label: id + "+" }],
    reverseSceneCommands: [{ label: id + "-" }]
  };
}

test("activation followed immediately by discard serializes forward then reverse", async () => {
  const harness = previewHarness();
  let releaseForward;
  harness.onCommands((_mount, commands) => {
    if (commands[0].label === "A+") {
      return new Promise((resolve) => { releaseForward = resolve; });
    }
    return Promise.resolve();
  });

  const activation = harness.api.activate(proposal("A"));
  await Promise.resolve();
  await Promise.resolve();
  const rollback = harness.api.revert("A");
  assert.equal(typeof releaseForward, "function");
  releaseForward();
  assert.equal(await activation, true);
  assert.equal(await rollback, true);
  assert.deepEqual(harness.commandCalls.map((call) => call.commands[0].label), ["A+", "A-"]);
  assert.equal(harness.api.activeProposalID(), "");
  assert.equal(harness.badge.hidden, true);
});

test("a superseding proposal reverses the first preview before applying the next", async () => {
  const harness = previewHarness();
  await Promise.all([
    harness.api.activate(proposal("A")),
    harness.api.activate(proposal("B"))
  ]);
  assert.deepEqual(harness.commandCalls.map((call) => call.commands[0].label), ["A+", "A-", "B+"]);
  assert.equal(harness.api.activeProposalID(), "B");
  assert.equal(harness.badge.hidden, false);
});

test("rollback rejection keeps the proposal and preview disclosure active", async () => {
  const harness = previewHarness();
  await harness.api.activate(proposal("A"));
  harness.onCommands((_mount, commands) => {
    return commands[0].label === "A-" ? Promise.reject(new Error("command rejected")) : Promise.resolve();
  });
  assert.equal(await harness.api.revert("A"), false);
  assert.equal(harness.api.activeProposalID(), "A");
  assert.equal(harness.badge.hidden, false);
});

test("the same proposal reapplies after a Scene3D remount without touching the detached engine", async () => {
  const harness = previewHarness();
  const staged = proposal("A");
  await harness.api.activate(staged);
  const oldMount = harness.mount();
  const newMount = harness.replaceMount();
  harness.commandCalls.length = 0;

  await harness.api.activate(staged);
  assert.deepEqual(harness.commandCalls.map((call) => call.commands[0].label), ["A+"]);
  assert.equal(harness.commandCalls[0].target, newMount);
  assert.notEqual(harness.commandCalls[0].target, oldMount);
  assert.equal(harness.api.activeProposalID(), "A");
});

test("the same proposal reasserts preview disclosure after in-place navigation", async () => {
  const harness = previewHarness();
  const staged = proposal("A");
  await harness.api.activate(staged);
  harness.commandCalls.length = 0;

  // A managed navigation reconciles the badge from canonical server HTML,
  // where it is hidden, while preserving the already-previewed Scene3D mount.
  harness.badge.hidden = true;
  assert.equal(await harness.api.activate(staged), false);
  assert.equal(harness.badge.hidden, false);
  assert.equal(harness.api.activeProposalID(), "A");
  assert.deepEqual(harness.commandCalls, [], "an unchanged live mount must not receive duplicate commands");
});

test("a rejected forward preview retains its reverse path and disclosure", async () => {
  const harness = previewHarness();
  harness.onCommands((_mount, commands) => {
    return commands[0].label === "A+" ? Promise.reject(new Error("partial apply")) : Promise.resolve();
  });

  assert.equal(await harness.api.activate(proposal("A")), false);
  assert.equal(harness.api.activeProposalID(), "A");
  assert.equal(harness.badge.hidden, false);

  harness.onCommands(() => Promise.resolve());
  assert.equal(await harness.api.revert("A"), true);
  assert.deepEqual(harness.commandCalls.map((call) => call.commands[0].label), ["A+", "A-"]);
  assert.equal(harness.api.activeProposalID(), "");
  assert.equal(harness.badge.hidden, true);
});

test("a rejected superseding rollback keeps the prior preview disclosed until canonical remount", async () => {
  const harness = previewHarness();
  await harness.api.activate(proposal("A"));
  harness.onCommands((_mount, commands) => {
    return commands[0].label === "A-" ? Promise.reject(new Error("rollback rejected")) : Promise.resolve();
  });

  assert.equal(await harness.api.activate(proposal("B")), false);
  assert.equal(harness.api.activeProposalID(), "A");
  assert.equal(harness.badge.hidden, false);
  assert.deepEqual(harness.commandCalls.map((call) => call.commands[0].label), ["A+", "A-"]);
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(harness.reloadCalls(), 1);
});

test("a stale material navigation cannot overwrite a newer WebMCP focus", async () => {
  const harness = previewHarness();
  const staleURL = "https://studio.test/?selection=piece-player-1-01&applied=human-set-material-1";
  harness.setHref(staleURL);
  let finishFirstFocus;
  harness.onNavigate((target) => {
    if (harness.navigationCalls.length === 1) {
      return new Promise((resolve) => { finishFirstFocus = resolve; });
    }
    harness.setHref(new URL(target, staleURL).href);
    return Promise.resolve(true);
  });

  harness.dispatch("studio:webmcp:focus", { id: "board" });
  await nextTask();
  assert.equal(harness.navigationCalls.length, 1);
  assert.match(harness.navigationCalls[0].target, /selection=board/);

  // The older form redirect lands after focus and supersedes the first Board
  // request. GoSX reports that request as not applied, then emits its stale
  // same-document navigation.
  harness.setHref(staleURL);
  harness.dispatch("gosx:navigate", { url: staleURL });
  finishFirstFocus(false);
  await nextTask();
  await nextTask();

  assert.equal(harness.navigationCalls.length, 2, "focus should retry exactly once after the stale redirect wins");
  assert.match(harness.navigationCalls[1].target, /selection=board/);
  assert.match(harness.navigationCalls[1].target, /applied=human-set-material-1/);
  assert.equal(harness.selectionCalls.at(-1), "board", "stale server HTML should be corrected local-first");

  await nextTask();
  assert.equal(harness.navigationCalls.length, 2, "a landed focus must not schedule another retry");
});

test("a newer human hierarchy selection cancels a pending focus retry", async () => {
  const harness = previewHarness();
  harness.setHref("https://studio.test/?selection=piece-player-1-01");
  let finishFocus;
  harness.onNavigate(() => new Promise((resolve) => { finishFocus = resolve; }));

  harness.dispatch("studio:webmcp:focus", { id: "board" });
  await nextTask();
  assert.equal(harness.navigationCalls.length, 1);

  const humanLink = {
    getAttribute(name) { return name === "data-entity-id" ? "piece-player-1-02" : null; }
  };
  harness.dispatch("click", null, {
    target: {
      closest(selector) { return selector === "a[data-entity-id]" ? humanLink : null; }
    },
    preventDefault() {}
  });
  finishFocus(false);
  await nextTask();
  await nextTask();

  assert.equal(harness.navigationCalls.length, 1, "human selection must supersede the pending agent retry");
});
