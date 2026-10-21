"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");
const source = fs.readFileSync(path.join(__dirname, "..", "public", "studio-certification.js"), "utf8");

async function completeEvidence({ incomingRevision = "0001" } = {}) {
  let tick;
  let requested = [];
  const state = { className: "certification-state certification-state-pending" };
  const card = { className: "certification-card", querySelector() { return null; }, replaceChildren(...children) { this.children = children; } };
  const proof = { className: state.className, querySelector() { return null; }, replaceChildren(...children) { this.children = children; this.className = "telemetry-proof certification-state certification-state-current"; } };
  const document = {
    querySelector(selector) {
      if (selector === ".certification-state") return state;
      if (selector === 'input[name="expectedRevision"]') return { value: "0001" };
      if (selector === ".certification-card") return card;
      if (selector === ".telemetry-proof.certification-state") return proof;
      return null;
    },
    importNode(child) { return child; }
  };
  const window = {
    setInterval(callback) { tick = callback; },
    __gosx: { request(url) {
      requested.push(url);
      return Promise.resolve({ ok: true, json() { return Promise.resolve({ state: "current", revision: 1, documentRevision: 1 }); }, text() { return Promise.resolve("canonical HTML"); } });
    } }
  };
  class DOMParser {
    parseFromString() {
      return { querySelector(selector) {
        if (selector === 'input[name="expectedRevision"]') return { value: incomingRevision };
        return { className: selector.slice(1), childNodes: ["current evidence"] };
      } };
    }
  }
  vm.runInNewContext(source, { document, window, DOMParser }, { filename: "studio-certification.js" });
  tick();
  // Drain the bounded request/HTML reconciliation promise chain.
  await new Promise(resolve => setImmediate(resolve));
  return { card, proof, requested };
}

test("background evidence reconciles only canonical proof fragments without navigation", async () => {
  const h = await completeEvidence();
  assert.deepEqual(h.requested, ["/api/studio/certification/state", "/"]);
  assert.deepEqual(h.card.children, ["current evidence"]);
  assert.deepEqual(h.proof.children, ["current evidence"]);
  assert.doesNotMatch(source, /__gosx_page_nav|location\.(?:reload|assign)|navigation\.(?:navigate|revalidate)/);
});

test("evidence from a newer document cannot overwrite the revision on screen", async () => {
  const h = await completeEvidence({ incomingRevision: "0002" });
  assert.equal(h.card.children, undefined);
  assert.equal(h.proof.children, undefined);
});
