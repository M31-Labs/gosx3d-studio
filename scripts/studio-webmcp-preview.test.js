"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "..", "public", "studio-webmcp-ui.js"), "utf8");

function element(tagName = "div") {
  const attributes = new Map();
  const node = {
    tagName: tagName.toUpperCase(),
    children: [],
    parentNode: null,
    className: "",
    textContent: "",
    isConnected: true,
    hidden: true,
    get firstChild() { return this.children[0] || null; },
    get firstElementChild() { return this.firstChild; },
    appendChild(child) {
      if (child.parentNode) child.parentNode.removeChild(child);
      this.children.push(child);
      child.parentNode = this;
      return child;
    },
    removeChild(child) {
      const index = this.children.indexOf(child);
      assert.notEqual(index, -1, "removing a detached mock child");
      this.children.splice(index, 1);
      child.parentNode = null;
      return child;
    },
    querySelector(selector) {
      for (const child of this.children) {
        if (child.matches(selector)) return child;
        const nested = child.querySelector(selector);
        if (nested) return nested;
      }
      return null;
    },
    matches(selector) {
      if (selector.startsWith(".")) return this.className.split(/\s+/).includes(selector.slice(1));
      const attribute = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/);
      if (attribute) return this.hasAttribute(attribute[1]) && (attribute[2] === undefined || this.getAttribute(attribute[1]) === attribute[2]);
      return this.tagName.toLowerCase() === selector;
    },
    setAttribute(name, value) { attributes.set(String(name), String(value)); },
    removeAttribute(name) { attributes.delete(String(name)); },
    getAttribute(name) { return attributes.has(String(name)) ? attributes.get(String(name)) : null; },
    hasAttribute(name) { return attributes.has(String(name)); }
  };
  node.classList = {
    add(value) { if (!node.matches("." + value)) node.className = (node.className + " " + value).trim(); },
    remove(value) { node.className = node.className.split(/\s+/).filter((name) => name !== value).join(" "); }
  };
  return node;
}

function response(payload) {
  return { ok: true, status: 200, json: () => Promise.resolve(payload) };
}

function previewHarness(options = {}) {
  const support = Object.fromEntries([
    "data-webmcp-registration-summary", "data-webmcp-mission-title",
    "data-webmcp-mission-heading", "data-webmcp-browser-help",
    "data-webmcp-copy-status", "data-webmcp-copy-prompt",
    "data-studio-demo-panel", "data-studio-demo-reset", "data-studio-demo-state"
  ].map((name) => [name, element()]));
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
  let mutationForm = null;
  let traceHost = options.trace ? element() : null;
  const agentPanel = options.review ? element() : null;
  const proposalCard = options.review ? element() : null;
  const scrollCalls = [];
  if (agentPanel) {
    agentPanel.scrollTop = 0;
    agentPanel.getBoundingClientRect = () => ({ top: 100 });
    agentPanel.scrollTo = (value) => { scrollCalls.push(value); agentPanel.scrollTop = value.top; };
    proposalCard.getBoundingClientRect = () => ({ top: 250 });
  }
  let fetchHandler = (url) => response(String(url).includes("/demo/status")
    ? options.support ? { enabled: true, clean: true, revision: 1 } : { enabled: false }
    : { proposal: null });
  let reloadCalls = 0;
  const listeners = new Map();
  const document = {
    activeElement: null,
    querySelector(selector) {
      if (options.support && /^\[[^\]]+\]$/.test(selector) && support[selector.slice(1, -1)]) return support[selector.slice(1, -1)];
      if (selector === ".scene-stage") return stage;
      if (selector === "[data-webmcp-preview-badge]") return badge;
      if (selector === "[data-gosx-scene3d]") return mount;
      if (selector === "[data-webmcp-trace]") return traceHost;
      if (selector === ".agent-panel") return agentPanel;
      if (selector === "[data-webmcp-proposal]") return proposalCard;
      return null;
    },
    querySelectorAll(selector) {
      return selector === 'form[data-gosx-form], [data-gizmo-mode]' && mutationForm ? [mutationForm] : [];
    },
    addEventListener(name, listener) {
      const values = listeners.get(name) || [];
      values.push(listener);
      listeners.set(name, values);
    },
    createElement(tagName) { return element(tagName); },
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
  if (options.boundedSelection) {
    window.__gosxStudioSelection.select = (id) => {
      selectionCalls.push(String(id));
      return Promise.resolve(true);
    };
  }
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
    fetch(url, requestOptions) { return Promise.resolve(fetchHandler(url, requestOptions)); }
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
    onFetch(handler) { fetchHandler = handler; },
    traceHost: () => traceHost,
    replaceTraceHost() { traceHost = element(); return traceHost; },
    agentPanel,
    support,
    scrollCalls,
    navigationCalls,
    selectionCalls,
    replaceMutationForm() {
      mutationForm = element("form");
      const button = element("button");
      button.disabled = false;
      mutationForm.querySelectorAll = () => [button];
      return { form: mutationForm, button };
    },
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

test("WebMCP focus uses bounded selection without scheduling whole-page navigation", async () => {
  const harness = previewHarness({ boundedSelection: true });
  harness.dispatch("studio:webmcp:focus", { id: "board" });
  await nextTask();
  assert.deepEqual(harness.selectionCalls, ["board"]);
  assert.deepEqual(harness.navigationCalls, []);
});

test("in-place Inspector selection preserves the active preview's human-review lock", async () => {
  const harness = previewHarness();
  await harness.api.activate(proposal("selection-lock"));
  const { form, button } = harness.replaceMutationForm();
  harness.dispatch("studio:selection", { selected: "board" });
  assert.equal(button.disabled, true);
  assert.equal(form.getAttribute("data-webmcp-review-locked"), "true");
  assert.equal(harness.commandCalls.length, 1, "restoring form locks must not replay the preview");
  assert.equal(harness.navigationCalls.length, 0);
});

test("a clean scene does not claim unavailable browser tools are ready", async () => {
  const harness = previewHarness({ support: true });
  harness.dispatch("studio:webmcp:status", { state: "unavailable", toolCount: 0 });
  await nextTask();
  assert.equal(harness.support["data-webmcp-registration-summary"].textContent, "4 tool definitions");
  assert.equal(harness.support["data-webmcp-mission-heading"].textContent, "Human editing works in this browser.");
  assert.match(harness.support["data-webmcp-copy-status"].textContent, /agent tools are unavailable here/);
  assert.match(harness.support["data-webmcp-browser-help"].textContent, /confirm that it reports 4 tools/);
  harness.dispatch("gosx:navigate");
  await nextTask();
  assert.doesNotMatch(harness.support["data-webmcp-copy-status"].textContent, /Ready for the browser agent/);
});

test("tool registration, not Chrome version or clean baseline, controls ready copy", async () => {
  const harness = previewHarness({ support: true });
  await nextTask();
  assert.equal(harness.support["data-webmcp-registration-summary"].textContent, "4 tool definitions");
  assert.match(harness.support["data-webmcp-copy-status"].textContent, /still being checked/);
  harness.dispatch("studio:webmcp:status", { state: "ready", toolCount: 4 });
  assert.equal(harness.support["data-webmcp-registration-summary"].textContent, "4 registered tools");
  assert.equal(harness.support["data-webmcp-copy-status"].textContent, "Ready for the browser agent.");
  harness.dispatch("studio:webmcp:status", { state: "unavailable", toolCount: 0 });
  assert.equal(harness.support["data-webmcp-registration-summary"].textContent, "4 tool definitions");
  assert.match(harness.support["data-webmcp-copy-status"].textContent, /unavailable here/);
});

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

test("unchanged native-call receipts retain list and item nodes across managed navigation", async () => {
  const harness = previewHarness({ trace: true });
  await nextTask();
  harness.dispatch("studio:webmcp:trace", { callId: "call-a", tool: "scene_get_state", message: "Inspected scene revision 1" });
  harness.dispatch("studio:webmcp:trace", { callId: "call-b", tool: "scene_find_objects", message: "Found Board" });
  const host = harness.traceHost();
  const list = host.querySelector(".webmcp-trace-list");
  const items = [...list.children];
  host.scrollTop = 17;

  harness.dispatch("gosx:navigate", { url: "https://studio.test/?selection=board" });
  await nextTask();
  await nextTask();

  assert.equal(host.querySelector(".webmcp-trace-list"), list, "unchanged receipts must not rebuild the trace list");
  assert.deepEqual(list.children, items, "receipt nodes should preserve identity and accessibility state");
  assert.equal(host.scrollTop, 17, "navigation must not reset a reader's trace scroll position");
});

test("native-call receipts hydrate a replacement host without dropping retained history", async () => {
  const harness = previewHarness({ trace: true });
  await nextTask();
  harness.dispatch("studio:webmcp:trace", { callId: "call-a", tool: "scene_get_state", message: "Inspected scene revision 1" });
  const oldList = harness.traceHost().querySelector(".webmcp-trace-list");
  const replacement = harness.replaceTraceHost();

  harness.dispatch("gosx:navigate", { url: "https://studio.test/?selection=board" });
  await nextTask();
  const list = replacement.querySelector(".webmcp-trace-list");
  assert.ok(list, "a newly reconciled host still needs its retained receipts");
  assert.notEqual(list, oldList);
  assert.equal(list.children.length, 1);
  assert.equal(list.children[0].textContent, "Inspected scene revision 1");
});

test("review reveals a newly staged proposal once without repeated hydration scrolls", async () => {
  const harness = previewHarness({ review: true });
  await nextTask();
  let currentProposal = proposal("A");
  harness.onFetch((url) => response(String(url).includes("/demo/status")
    ? { enabled: false }
    : { proposal: currentProposal }));

  harness.dispatch("studio:webmcp:proposal", currentProposal);
  await nextTask();
  assert.equal(harness.scrollCalls.length, 1, "a new proposal should reveal its review card");
  harness.agentPanel.scrollTop = 37;

  harness.dispatch("gosx:navigate", { url: "https://studio.test/?selection=board" });
  await nextTask();
  await nextTask();
  harness.dispatch("studio:webmcp:proposal", currentProposal);
  await nextTask();
  assert.equal(harness.scrollCalls.length, 1, "same-id navigation, hydration, and redelivery must not scroll again");
  assert.equal(harness.agentPanel.scrollTop, 37, "reviewer scroll position should survive rehydration");

  currentProposal = proposal("B");
  harness.dispatch("studio:webmcp:proposal", currentProposal);
  await nextTask();
  assert.equal(harness.scrollCalls.length, 2, "a different newly staged proposal should reveal its review card");
});

test("a queued reveal for a superseded proposal cannot scroll the current review", async () => {
  const harness = previewHarness({ review: true });
  await nextTask();
  harness.dispatch("studio:webmcp:proposal", proposal("A"));
  harness.dispatch("studio:webmcp:proposal", proposal("B"));
  await nextTask();

  assert.equal(harness.scrollCalls.length, 1, "only the still-current proposal may perform its queued reveal");
  assert.equal(harness.api.activeProposalID(), "B");
});
