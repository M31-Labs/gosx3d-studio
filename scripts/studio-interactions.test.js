"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "..", "public", "studio-interactions.js"), "utf8");

function interactionsHarness({ demo = true, narrow = false, observe = true } = {}) {
  const listeners = new Map();
  const observers = [];
  let timerCalls = 0;
  let selectionRevealCollapsed;

  function changed(target, type, attributeName) {
    observers.forEach((observer) => {
      const options = observer.targets.get(target);
      if (!options || !options[type]) return;
      if (type === "attributes" && options.attributeFilter && !options.attributeFilter.includes(attributeName)) return;
      observer.pending = true;
    });
  }

  function element() {
    const attributes = new Map();
    const node = {
      getAttribute(name) { return attributes.has(name) ? attributes.get(name) : null; },
      setAttribute(name, value) {
        value = String(value);
        if (attributes.get(name) === value) return;
        attributes.set(name, value);
        changed(node, "attributes", name);
      },
      classList: {
        contains(name) { return String(node.getAttribute("class") || "").split(/\s+/).includes(name); },
        toggle(name, force) {
          const names = new Set(String(node.getAttribute("class") || "").split(/\s+/).filter(Boolean));
          if (force) names.add(name);
          else names.delete(name);
          node.setAttribute("class", Array.from(names).join(" "));
        }
      }
    };
    return node;
  }

  const body = element();
  let shell;
  let button;
  let panel;
  const choices = ["scene", "objects", "inspector", "agent"].map((name) => {
    const choice = element();
    choice.setAttribute("data-studio-panel-choice", name);
    choice.closest = (selector) => selector === "[data-studio-panel-choice]" ? choice : null;
    return choice;
  });
  function serverShell() {
    shell = element();
    shell.setAttribute("class", "studio-shell");
    button = element();
    button.setAttribute("aria-expanded", "true");
    button.closest = (selector) => selector === "[data-project-panel-toggle]" ? button : null;
    panel = element();
    panel.setAttribute("aria-hidden", "false");
  }
  serverShell();

  class MutationObserver {
    constructor(callback) {
      this.callback = callback;
      this.targets = new Map();
      this.pending = false;
      observers.push(this);
    }
    observe(target, options) { this.targets.set(target, options); }
    disconnect() { this.targets.clear(); this.pending = false; }
  }

  const selected = {
    scrollIntoView() { selectionRevealCollapsed = shell.classList.contains("project-panel-collapsed"); }
  };
  const document = {
    readyState: "complete",
    body,
    querySelector(selector) {
      if (selector === '.studio-shell[data-studio-demo="true"]') return demo ? shell : null;
      if (selector === ".studio-shell") return shell;
      if (selector === "[data-project-panel-toggle]") return button;
      if (selector === "#studio-hierarchy-tree [aria-selected='true']") return selected;
      return null;
    },
    querySelectorAll(selector) { return selector === "[data-studio-panel-choice]" ? choices : []; },
    getElementById(id) { return id === "project-panel" ? panel : null; },
    addEventListener(name, listener) {
      const entries = listeners.get(name) || [];
      entries.push(listener);
      listeners.set(name, entries);
    }
  };
  const window = {
    matchMedia() { return { matches: narrow }; },
    setTimeout() { timerCalls += 1; }
  };
  vm.runInNewContext(source, { document, window, MutationObserver: observe ? MutationObserver : undefined }, { filename: "studio-interactions.js" });

  function dispatch(name, fields = {}) {
    (listeners.get(name) || []).forEach((listener) => listener(fields));
  }

  return {
    collapsed() { return shell.classList.contains("project-panel-collapsed"); },
    expanded() { return button.getAttribute("aria-expanded"); },
    hidden() { return panel.getAttribute("aria-hidden"); },
    timerCalls() { return timerCalls; },
    selectionRevealCollapsed() { return selectionRevealCollapsed; },
    toggle() { dispatch("click", { target: button }); },
    reconcileServerShell() {
      shell.setAttribute("class", "studio-shell");
      button.setAttribute("aria-expanded", "true");
      panel.setAttribute("aria-hidden", "false");
    },
    reconcileCompactPanel() { shell.setAttribute("data-studio-panel", "scene"); },
    replaceServerShell() { serverShell(); changed(body, "childList"); },
    flushMutations() {
      // MutationObserver callbacks run at a microtask checkpoint before paint.
      for (let round = 0; observers.some((observer) => observer.pending); round += 1) {
        assert.ok(round < 10, "layout restoration must converge without an observer loop");
        observers.slice().forEach((observer) => {
          if (!observer.pending) return;
          observer.pending = false;
          observer.callback();
        });
      }
    },
    navigate() { dispatch("gosx:navigate"); },
    compactPanel() { return shell.getAttribute("data-studio-panel"); },
    choosePanel(name) { dispatch("click", { target: choices.find((choice) => choice.getAttribute("data-studio-panel-choice") === name) }); },
    pressedPanels() { return choices.filter((choice) => choice.getAttribute("aria-pressed") === "true").map((choice) => choice.getAttribute("data-studio-panel-choice")); }
  };
}

test("demo layout is collapsed before hierarchy scrolling and restored before navigation bootstrap finishes", () => {
  const harness = interactionsHarness();
  assert.equal(harness.collapsed(), true);
  assert.equal(harness.selectionRevealCollapsed(), true);
  harness.flushMutations();
  harness.reconcileServerShell();
  assert.equal(harness.collapsed(), false, "fixture emulates the SSR expanded class");
  harness.flushMutations();
  assert.equal(harness.collapsed(), true, "restored without waiting for gosx:navigate");
  assert.equal(harness.expanded(), "false");
  assert.equal(harness.hidden(), "true");
  assert.equal(harness.timerCalls(), 0);
});

test("explicit Assets choices survive shell replacement and repeated navigation", () => {
  const harness = interactionsHarness();
  harness.toggle();
  assert.equal(harness.collapsed(), false);
  harness.replaceServerShell();
  harness.flushMutations();
  harness.navigate();
  assert.equal(harness.collapsed(), false, "user-selected expanded state is retained");
  assert.equal(harness.expanded(), "true");
  assert.equal(harness.hidden(), "false");

  harness.toggle();
  harness.replaceServerShell();
  harness.flushMutations();
  assert.equal(harness.collapsed(), true, "replacement shell is restored before navigation completion");
  harness.navigate();
  assert.equal(harness.collapsed(), true);
  assert.equal(harness.expanded(), "false");
  assert.equal(harness.hidden(), "true");
  assert.equal(harness.timerCalls(), 0);
});

test("navigation synchronizes immediately without MutationObserver support", () => {
  const harness = interactionsHarness({ demo: false, narrow: true, observe: false });
  harness.reconcileServerShell();
  harness.navigate();
  assert.equal(harness.collapsed(), true);
  assert.equal(harness.selectionRevealCollapsed(), true);
  assert.equal(harness.timerCalls(), 0);
});

test("ordinary wide workspaces retain the expanded Assets default", () => {
  const harness = interactionsHarness({ demo: false, narrow: false });
  assert.equal(harness.collapsed(), false);
  assert.equal(harness.expanded(), "true");
  assert.equal(harness.hidden(), "false");
});

test("compact panes retain the explicit choice across managed navigation without remounting", () => {
  const harness = interactionsHarness({ narrow: true });
  assert.equal(harness.compactPanel(), "scene");
  assert.deepEqual(harness.pressedPanels(), ["scene"]);
  for (const name of ["objects", "inspector", "agent", "scene"]) {
    harness.choosePanel(name);
    harness.replaceServerShell();
    harness.flushMutations();
    harness.navigate();
    assert.equal(harness.compactPanel(), name);
    assert.deepEqual(harness.pressedPanels(), [name]);
  }
  assert.equal(harness.timerCalls(), 0);
});

test("a server panel attribute is restored before the next paint without waiting for navigation", () => {
  const harness = interactionsHarness({ narrow: true });
  harness.choosePanel("inspector");
  harness.flushMutations();
  harness.reconcileCompactPanel();
  assert.equal(harness.compactPanel(), "scene", "server output resets the shell default");
  harness.flushMutations();
  assert.equal(harness.compactPanel(), "inspector");
  assert.deepEqual(harness.pressedPanels(), ["inspector"]);
  assert.equal(harness.timerCalls(), 0);
});
