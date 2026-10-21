"use strict";

// Call with a Playwright Page against a disposable LOCAL Studio fixture.
// This exercises real inputs; it never commits a scene edit or resets a demo.
module.exports = async function selectionQA(page, { baseURL, proofDir, compact = false } = {}) {
  const assert = require("node:assert/strict");
  const fs = require("node:fs");
  const target = new URL(baseURL || "http://127.0.0.1:3136/");
  assert.ok(["127.0.0.1", "localhost", "[::1]"].includes(target.hostname), "local disposable fixture only");
  if (proofDir) fs.mkdirSync(proofDir, { recursive: true });
  const shot = async (name) => {
    if (proofDir) await page.screenshot({ path: proofDir + "/" + name + ".png", scale: "css" });
  };
  await page.setViewportSize(compact ? { width: 390, height: 844 } : { width: 1440, height: 900 });
  await page.goto(target.href, { waitUntil: "domcontentloaded" });
  await page.bringToFront();
  await page.waitForSelector('.scene-stage [data-gosx-scene3d-ready="true"] canvas');
  await page.waitForFunction(() => typeof window.__gosxStudioSelection?.select === "function");
  const canvas = await page.locator(".scene-stage canvas").elementHandle();
  await page.getByRole("button", { name: "Top", exact: true }).click();
  await page.evaluate(() => {
    window.__studioSelectionQA = { navigation: 0, headChanges: 0, camera: null };
    // Emulate only a stale on-screen evidence badge. The real read-only poll
    // and canonical HTML must settle it without navigating over a selection.
    document.querySelectorAll(".certification-state").forEach(node => {
      node.className = node.className.replace(/certification-state-[a-z]+/g, "certification-state-pending");
    });
    document.addEventListener("gosx:navigate", () => { window.__studioSelectionQA.navigation++; });
    new MutationObserver((records) => {
      for (const record of records) for (const node of [...record.addedNodes, ...record.removedNodes]) {
        if (node.nodeType === 1 && node.matches('link[rel="stylesheet"],style')) window.__studioSelectionQA.headChanges++;
      }
    }).observe(document.head, { childList: true });
    if (typeof window.__gosx_subscribe_shared_signal === "function") {
      window.__gosx_subscribe_shared_signal("studio.viewport.cameraOut", value => { window.__studioSelectionQA.camera = value; });
    }
  });
  await page.waitForTimeout(100);
  const beforeCamera = await page.evaluate(() => JSON.stringify(window.__studioSelectionQA.camera));
  if (compact) await page.locator('[data-studio-panel-choice="objects"]').click();
  const choose = async (id) => {
    await page.locator("[data-hierarchy-filter]").fill(id);
    await page.locator('[data-entity-id="' + id + '"]').click();
    await page.waitForFunction(id => {
      const form = document.querySelector('#inspector-panel form[action*="setTransform"]');
      return form && form.querySelector('[name="target"]').value === id && !document.querySelector("form[data-selection-pending]");
    }, id);
    assert.equal(await page.locator('[data-entity-id="' + id + '"]').getAttribute("aria-selected"), "true");
    assert.equal(await page.locator('[data-entity-id="' + id + '"]').getAttribute("aria-current"), "true");
  };
  const selections = ["board", "piece-player-1-01", "piece-player-4-01", "board"];
  for (const id of selections) await choose(id);
  await page.goBack();
  await page.waitForFunction(() => document.querySelector('#inspector-panel [name="target"]')?.value === "piece-player-4-01");
  await page.goForward();
  await page.waitForFunction(() => document.querySelector('#inspector-panel [name="target"]')?.value === "board");
  const routePattern = "**/?selection=piece-player-1-01";
  await page.route(routePattern, async route => {
    await new Promise(resolve => setTimeout(resolve, 400));
    await route.continue().catch(() => {});
  });
  await page.locator("[data-hierarchy-filter]").fill("piece-player-1-01");
  await page.locator('[data-entity-id="piece-player-1-01"]').click();
  await choose("board");
  await page.waitForTimeout(550);
  assert.equal(await page.locator('#inspector-panel [name="target"]').first().inputValue(), "board");
  await page.unroute(routePattern);
  const errorPattern = "**/?selection=piece-player-4-01";
  await page.route(errorPattern, route => route.fulfill({ status: 503, contentType: "text/plain", body: "Local test failure" }));
  await page.locator("[data-hierarchy-filter]").fill("piece-player-4-01");
  await page.locator('[data-entity-id="piece-player-4-01"]').click();
  await page.getByRole("button", { name: "Retry selection", exact: true }).waitFor();
  assert.ok(await page.locator("form[data-selection-pending]").count() > 0, "old forms remain locked after error");
  await shot(compact ? "mobile-selection-error" : "desktop-selection-error");
  await page.unroute(errorPattern);
  await page.getByRole("button", { name: "Retry selection", exact: true }).click();
  await page.waitForFunction(() => document.querySelector('#inspector-panel [name="target"]')?.value === "piece-player-4-01" && !document.querySelector("form[data-selection-pending]"));
  await page.waitForSelector(".telemetry-proof.certification-state-current");
  if (compact) {
    assert.equal(await page.locator(".studio-shell").getAttribute("data-studio-panel"), "objects");
    await page.locator('[data-studio-panel-choice="inspector"]').click();
  }
  await shot(compact ? "mobile-selection-inspector" : "desktop-selection-inspector");
  const stats = await page.evaluate(() => ({
    navigation: window.__studioSelectionQA.navigation,
    headChanges: window.__studioSelectionQA.headChanges,
    camera: JSON.stringify(window.__studioSelectionQA.camera),
    width: innerWidth, scrollWidth: document.documentElement.scrollWidth,
    inspector: document.getElementById("inspector-panel").getBoundingClientRect().toJSON(),
    selected: document.querySelector(".scene-stage").dataset.selectionId,
    urlSelected: new URL(location.href).searchParams.get("selection")
  }));
  assert.equal(stats.navigation, 0, "selection must not invoke whole-page managed navigation");
  assert.equal(stats.headChanges, 0, "selection must leave stylesheets mounted");
  assert.equal(stats.scrollWidth, stats.width);
  assert.equal(stats.selected, stats.urlSelected);
  assert.equal(await page.locator('[data-camera-view="top"]').getAttribute("aria-pressed"), "true");
  if (beforeCamera !== "null" && stats.camera !== "null") assert.equal(stats.camera, beforeCamera, "camera pose must remain unchanged");
  assert.equal(await canvas.evaluate(node => node.isConnected && node === document.querySelector(".scene-stage canvas")), true);
  await page.locator("a.brand").click();
  await page.waitForFunction(() => window.__studioSelectionQA.navigation > 0);
  assert.equal(new URL(page.url()).pathname, "/", "unrelated navigation still works");
  return { ...stats, compact, selections, staleResponseRejected: true, errorRetryRecovered: true, popstateCoherent: true, canvasPreserved: true, evidenceSettledWithoutNavigation: true, unrelatedNavigationWorks: true };
};
