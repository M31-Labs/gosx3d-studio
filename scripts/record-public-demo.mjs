import { createInterface } from "node:readline/promises";
import { stdin, stdout } from "node:process";

// Cue-driven WebMCP recording helper for the public GoSX 3D Studio demo.
//
// Attach only: this file never imports Playwright and never launches or closes
// a browser. Start native Windows Chrome yourself with remote debugging and
// WebMCP enabled, then open https://gosx3d.m31labs.dev in exactly one tab.
// The one Page.navigate below is off-camera setup; there is no Page.reload.

const CDP_ENDPOINT = String(process.env.CDP_ENDPOINT || "http://172.29.240.1:9336").replace(/\/$/, "");
const STUDIO_URL = "https://gosx3d.m31labs.dev/";
const TARGET_PREFIX = "https://gosx3d.m31labs.dev";
const TARGET_ID = String(process.env.TARGET_ID || "").trim();
const WIDTH = Number(process.env.DEVICE_WIDTH || 1920);
const HEIGHT = Number(process.env.DEVICE_HEIGHT || 1080);
const EXPECTED_VERSION = "0.55.1";
const EXPECTED_TOOLS = [
  "scene_find_objects",
  "scene_focus_object",
  "scene_get_state",
  "scene_preview_actions",
];
const CORAL_ENTITY_ID = "piece-player-1-01";
const STEEL_MATERIAL_ID = "board-steel-material";

const proposalInput = (revision) => ({
  expectedRevision: revision,
  title: "Prepare Launch Board",
  rationale:
    "Resolve Board and Brushed Steel by stable ID, show the exact reversible viewport diff, and leave canonical authority with the human reviewer.",
  operations: [
    { kind: "rename-entity", target: "board", name: "Launch Board" },
    { kind: "assign-material", target: "board", material: STEEL_MATERIAL_ID },
  ],
});

if (!stdin.isTTY || !stdout.isTTY) {
  throw new Error("This recording driver requires an interactive terminal for operator cues.");
}
if (!Number.isInteger(WIDTH) || WIDTH < 1200 || !Number.isInteger(HEIGHT) || HEIGHT < 700) {
  throw new Error(`DEVICE_WIDTH/DEVICE_HEIGHT must describe a desktop viewport; got ${WIDTH}x${HEIGHT}`);
}

const endpointURL = new URL(CDP_ENDPOINT);
if (!/^https?:$/.test(endpointURL.protocol)) {
  throw new Error(`CDP_ENDPOINT must be HTTP(S), got ${endpointURL.protocol}`);
}

const rl = createInterface({ input: stdin, output: stdout });
const interrupt = new AbortController();
let interrupted = false;
process.once("SIGINT", () => {
  interrupted = true;
  interrupt.abort();
});

class OperatorAbort extends Error {
  constructor(message = "Operator aborted the take") {
    super(message);
    this.name = "OperatorAbort";
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function ask(prompt, { abortable = true } = {}) {
  try {
    return await rl.question(prompt, abortable ? { signal: interrupt.signal } : undefined);
  } catch (error) {
    if (interrupted || error?.name === "AbortError") throw new OperatorAbort("Interrupted by operator");
    throw error;
  }
}

async function cue(message) {
  const answer = String(await ask(`\n${message}\n[ENTER] continue  ·  type ABORT to stop: `)).trim();
  if (answer.toUpperCase() === "ABORT") throw new OperatorAbort();
  if (answer) throw new Error(`Unexpected cue response ${JSON.stringify(answer)}; press Enter or type ABORT.`);
}

async function requireResetConfirmation(message) {
  const answer = String(await ask(`\n${message}\nType RESET to continue, or ABORT: `)).trim().toUpperCase();
  if (answer === "ABORT") throw new OperatorAbort();
  if (answer !== "RESET") throw new Error("Shared reset was not confirmed; expected RESET.");
}

async function fetchJSON(url) {
  const response = await fetch(url, { signal: AbortSignal.timeout(5000) });
  if (!response.ok) throw new Error(`${url}: ${response.status} ${response.statusText}`);
  return response.json();
}

function reachableWebSocketURL(raw) {
  const url = new URL(raw);
  if (["localhost", "127.0.0.1", "::1", "[::1]"].includes(url.hostname)) {
    url.hostname = endpointURL.hostname;
    if (endpointURL.port) url.port = endpointURL.port;
  }
  return url.href;
}

const browserVersion = await fetchJSON(`${CDP_ENDPOINT}/json/version`);
const targets = await fetchJSON(`${CDP_ENDPOINT}/json/list`);
const candidates = targets.filter((candidate) =>
  candidate.type === "page" && String(candidate.url || "").startsWith(TARGET_PREFIX),
);
const target = TARGET_ID
  ? candidates.find((candidate) => candidate.id === TARGET_ID)
  : candidates.length === 1 ? candidates[0] : null;

if (!target) {
  const detail = candidates.map((candidate) => `${candidate.id} ${candidate.url}`).join("\n  ");
  throw new Error(
    candidates.length > 1
      ? `Multiple public Studio tabs are open. Re-run with TARGET_ID set to one of:\n  ${detail}`
      : `No native Windows Chrome page is open at ${TARGET_PREFIX}. Open it first; this script will not launch a browser.`,
  );
}
if (!String(browserVersion.Browser || "").startsWith("Chrome/") ||
    !String(browserVersion["User-Agent"] || "").includes("Windows NT")) {
  throw new Error(`Refusing a non-Windows/non-Chrome endpoint: ${browserVersion.Browser || "unknown browser"}`);
}

const socket = new WebSocket(reachableWebSocketURL(target.webSocketDebuggerUrl));
await new Promise((resolve, reject) => {
  socket.addEventListener("open", resolve, { once: true });
  socket.addEventListener("error", reject, { once: true });
});

let nextCommandID = 0;
const pendingCommands = new Map();
const events = [];
const activeTools = new Map();
let acceptResetDialog = false;
let studioReady = false;
const recordingGuard = {
  armed: false,
  frameId: "",
  documentRequests: [],
  frameNavigations: [],
  sameDocumentNavigations: [],
};

function command(method, params = {}, timeoutMS = 25000) {
  const id = ++nextCommandID;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      pendingCommands.delete(id);
      reject(new Error(`CDP timeout after ${timeoutMS}ms: ${method}`));
    }, timeoutMS);
    pendingCommands.set(id, {
      method,
      resolve: (value) => {
        clearTimeout(timer);
        resolve(value);
      },
      reject: (error) => {
        clearTimeout(timer);
        reject(error);
      },
    });
    socket.send(JSON.stringify({ id, method, params }));
  });
}

socket.addEventListener("message", ({ data }) => {
  const message = JSON.parse(String(data));
  if (message.id) {
    const waiter = pendingCommands.get(message.id);
    if (!waiter) return;
    pendingCommands.delete(message.id);
    if (message.error) waiter.reject(new Error(`${waiter.method}: ${JSON.stringify(message.error)}`));
    else waiter.resolve(message.result || {});
    return;
  }

  events.push(message);
  const params = message.params || {};
  if (message.method === "WebMCP.toolsAdded") {
    for (const tool of params.tools || []) activeTools.set(tool.name, tool);
    return;
  }
  if (message.method === "WebMCP.toolsRemoved") {
    for (const tool of params.tools || []) activeTools.delete(tool.name);
    return;
  }
  if (message.method === "Network.requestWillBeSent" && recordingGuard.armed &&
      params.type === "Document" && (!recordingGuard.frameId || params.frameId === recordingGuard.frameId)) {
    recordingGuard.documentRequests.push({ url: params.request?.url || "", loaderId: params.loaderId || "" });
    return;
  }
  if (message.method === "Page.frameNavigated" && recordingGuard.armed &&
      params.frame && !params.frame.parentId) {
    recordingGuard.frameNavigations.push({ url: params.frame.url || "", loaderId: params.frame.loaderId || "" });
    return;
  }
  if (message.method === "Page.navigatedWithinDocument" && recordingGuard.armed &&
      (!recordingGuard.frameId || params.frameId === recordingGuard.frameId)) {
    recordingGuard.sameDocumentNavigations.push(params.url || "");
    return;
  }
  if (message.method === "Page.javascriptDialogOpening") {
    const isReset = String(params.message || "").startsWith("Reset the shared public demo scene");
    const accept = acceptResetDialog && isReset;
    command("Page.handleJavaScriptDialog", { accept }).catch(() => {});
  }
});

async function evaluate(expression) {
  const response = await command("Runtime.evaluate", {
    expression,
    awaitPromise: true,
    returnByValue: true,
    userGesture: true,
  });
  if (response.exceptionDetails) {
    throw new Error(
      response.exceptionDetails.exception?.description ||
      response.exceptionDetails.text ||
      `Runtime.evaluate failed: ${JSON.stringify(response.exceptionDetails)}`,
    );
  }
  return response.result?.value;
}

async function waitUntil(label, probe, timeoutMS = 35000, intervalMS = 125) {
  const deadline = Date.now() + timeoutMS;
  let lastValue = null;
  let lastError = null;
  while (Date.now() < deadline) {
    try {
      lastValue = await probe();
      if (lastValue) return lastValue;
      lastError = null;
    } catch (error) {
      lastError = error;
    }
    await sleep(intervalMS);
  }
  throw new Error(
    `Timed out waiting for ${label}; lastValue=${JSON.stringify(lastValue)}` +
    (lastError ? `; lastError=${lastError.message}` : ""),
  );
}

async function waitForEvent(method, predicate, startIndex, timeoutMS = 35000) {
  return waitUntil(
    `${method} event`,
    async () => events.slice(startIndex).find((event) =>
      event.method === method && predicate(event.params || {})) || null,
    timeoutMS,
    25,
  );
}

async function uiSnapshot() {
  return evaluate(`(() => {
    const text = (selector) => {
      const element = document.querySelector(selector);
      return element ? String(element.textContent || "").replace(/\\s+/g, " ").trim() : null;
    };
    const footer = text(".status-bar");
    const revisionMatch = footer && footer.match(/Scene revision\\s+(\\d+)/i);
    const selectedMaterialForm = document.querySelector('form[data-gosx-form][action*="setMaterial"]');
    const details = selectedMaterialForm && selectedMaterialForm.closest("details");
    const rows = details ? Array.from(details.querySelectorAll("dl.properties > div")) : [];
    const property = (name) => {
      const row = rows.find((candidate) => String(candidate.querySelector("dt")?.textContent || "").trim() === name);
      return row ? String(row.querySelector("dd")?.textContent || "").trim() : null;
    };
    const mount = document.querySelector("[data-gosx-scene3d]") ||
      document.querySelector("[data-gosx-scene3d-mounted]");
    const mountGuard = window.__gosx3dPublicDemoMountGuard || null;
    const canvas = mount?.querySelector("canvas") || null;
    const proposal = document.querySelector("[data-webmcp-proposal]");
    const review = document.querySelector("[data-webmcp-review-actions]");
    const badge = document.querySelector("[data-webmcp-preview-badge]");
    const approvalOutcome = document.querySelector("[data-webmcp-approval-outcome]");
    const commit = document.querySelector("[data-webmcp-commit]");
    const commitRect = commit?.getBoundingClientRect();
    return {
      href: location.href,
      readyState: document.readyState,
      secureContext: window.isSecureContext,
      platform: navigator.platform,
      userAgent: navigator.userAgent,
      statusLabel: text("[data-webmcp-status-label]"),
      toolCount: text("[data-webmcp-tool-count]"),
      revision: revisionMatch ? Number(revisionMatch[1]) : null,
      selectedID: document.querySelector("[data-selection-id]")?.getAttribute("data-selection-id") || null,
      boardHierarchyText: text('[data-hierarchy-id="board"]'),
      backend: mount?.getAttribute("data-gosx-scene3d-backend") || null,
      renderer: mount?.getAttribute("data-gosx-scene3d-renderer") || null,
      mounted: mount?.getAttribute("data-gosx-scene3d-mounted") || null,
      ready: mount?.getAttribute("data-gosx-scene3d-ready") || null,
      meshObjects: Number(
        mount?.getAttribute("data-gosx-scene3d-webgpu-mesh-objects") ||
        mount?.getAttribute("data-gosx-scene3d-webgl-mesh-objects") ||
        mount?.getAttribute("data-gosx-scene3d-render-mesh-objects") ||
        0
      ),
      sceneBoundary: mountGuard ? {
        armed: true,
        sameMount: mountGuard.mount === mount,
        mountConnected: mountGuard.mount?.isConnected === true,
        sameCanvas: mountGuard.canvas === canvas,
        canvasConnected: mountGuard.canvas?.isConnected === true,
        canvasInsideMount: mountGuard.mount?.contains(mountGuard.canvas) === true,
      } : { armed: false },
      reset: {
        hidden: document.querySelector("[data-studio-demo-panel]")?.hidden ?? null,
        revision: Number(document.querySelector("[data-studio-demo-reset]")?.getAttribute("data-revision")),
        disabled: document.querySelector("[data-studio-demo-reset]")?.disabled ?? null,
      },
      flow: Object.fromEntries(${JSON.stringify(EXPECTED_TOOLS)}.map((name) => [
        name,
        document.querySelector('[data-webmcp-flow-tool="' + name + '"]')?.getAttribute("data-state") || null,
      ])),
      trace: Array.from(document.querySelectorAll("[data-webmcp-trace] li"))
        .map((item) => String(item.textContent || "").replace(/\\s+/g, " ").trim()),
      proposal: {
        pending: document.querySelector(".agent-panel")?.classList.contains("has-pending-proposal") === true,
        reviewHidden: review ? review.hidden : null,
        hidden: proposal ? getComputedStyle(proposal).display === "none" : null,
        summary: text("[data-webmcp-proposal-summary]"),
        actor: text("[data-webmcp-proposal-actor]"),
        policy: text("[data-webmcp-proposal-policy]"),
        revision: text("[data-webmcp-proposal-revision]"),
        changes: Array.from(document.querySelectorAll("[data-webmcp-proposal-changes] li"))
          .map((item) => String(item.textContent || "").replace(/\\s+/g, " ").trim()),
        commitDisabled: commit?.disabled ?? null,
        commitText: text("[data-webmcp-commit]"),
        commitVisible: commit ? !commit.hidden && getComputedStyle(commit).display !== "none" &&
          getComputedStyle(commit).visibility !== "hidden" && commitRect.width > 0 && commitRect.height > 0 : false,
      },
      preview: {
        active: document.querySelector(".scene-stage")?.getAttribute("data-webmcp-preview") === "true",
        badgeHidden: badge ? badge.hidden : null,
        badgeText: text("[data-webmcp-preview-badge]"),
      },
      approvalOutcome: {
        hidden: approvalOutcome ? approvalOutcome.hidden : null,
        text: text("[data-webmcp-approval-outcome]"),
      },
      evidence: text(".telemetry-proof.certification-state"),
      material: selectedMaterialForm ? {
        id: selectedMaterialForm.querySelector('input[name="materialId"]')?.value || null,
        name: property("Canonical material") || property("Material"),
      } : null,
      activity: Array.from(document.querySelectorAll("#agent-activity-panel .console-lines p")).map((row) => ({
        actor: String(row.querySelector(".author-label, .runtime-label")?.textContent || "").trim(),
        transactionID: row.querySelector("[data-transaction-id]")?.getAttribute("data-transaction-id") || "",
        summary: String(row.querySelector("[data-transaction-id]")?.textContent || "").replace(/\\s+/g, " ").trim(),
      })),
    };
  })()`);
}

async function canonicalSummary() {
  return evaluate(`Promise.all([
    fetch("/api/studio/document", { headers: { Accept: "application/json" }, cache: "no-store" }),
    fetch("/api/studio/demo/status", { headers: { Accept: "application/json" }, cache: "no-store" })
  ]).then(async ([documentResponse, demoResponse]) => {
    if (!documentResponse.ok || !demoResponse.ok) throw new Error("Canonical/demo status request failed");
    const scene = await documentResponse.json();
    const demo = await demoResponse.json();
    const board = scene.entities && scene.entities.board;
    return {
      revision: Number(scene.revision),
      entityCount: Object.keys(scene.entities || {}).length,
      board: board ? { id: board.id, name: board.name, materialID: board.mesh && board.mesh.material } : null,
      demo: { enabled: demo.enabled === true, clean: demo.clean === true, revision: Number(demo.revision) },
    };
  })`);
}

async function currentProposal() {
  return evaluate(`fetch("/api/studio/webmcp/proposals/current", {
    headers: { Accept: "application/json" }, cache: "no-store"
  }).then(async (response) => {
    if (!response.ok) throw new Error("Current proposal request failed: " + response.status);
    const payload = await response.json();
    return payload && payload.proposal && payload.proposal.proposalId
      ? { proposalId: String(payload.proposal.proposalId), title: String(payload.proposal.title || "") }
      : null;
  })`);
}

async function elementObject(selector) {
  const result = await command("Runtime.evaluate", {
    expression: `document.querySelector(${JSON.stringify(selector)})`,
    returnByValue: false,
  });
  if (!result.result?.objectId || result.result.subtype === "null") {
    throw new Error(`Element not found: ${selector}`);
  }
  return result.result.objectId;
}

async function clickSelector(selector, label) {
  const state = await evaluate(`(() => {
    const element = document.querySelector(${JSON.stringify(selector)});
    if (!element) return { missing: true };
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return {
      missing: false,
      disabled: !!element.disabled || element.getAttribute("aria-disabled") === "true",
      visible: style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0,
    };
  })()`);
  if (state.missing || state.disabled || !state.visible) {
    throw new Error(`${label}: click target is not visibly enabled (${JSON.stringify(state)})`);
  }
  const objectId = await elementObject(selector);
  await command("DOM.scrollIntoViewIfNeeded", { objectId });
  await sleep(125);
  const model = await command("DOM.getBoxModel", { objectId });
  const quad = model.model?.border || model.model?.content;
  if (!quad || quad.length !== 8) throw new Error(`${label}: no clickable box model`);
  const x = (quad[0] + quad[2] + quad[4] + quad[6]) / 4;
  const y = (quad[1] + quad[3] + quad[5] + quad[7]) / 4;
  await command("Input.dispatchMouseEvent", { type: "mouseMoved", x, y });
  await command("Input.dispatchMouseEvent", {
    type: "mousePressed", x, y, button: "left", buttons: 1, clickCount: 1,
  });
  await command("Input.dispatchMouseEvent", {
    type: "mouseReleased", x, y, button: "left", buttons: 0, clickCount: 1,
  });
  console.log(`  visible native click: ${label}`);
}

async function invokeTool(toolName, input) {
  const tool = await waitUntil(`active native WebMCP tool ${toolName}`, async () => activeTools.get(toolName) || null);
  const startIndex = events.length;
  const accepted = await command("WebMCP.invokeTool", {
    frameId: tool.frameId,
    toolName,
    input,
  });
  await waitForEvent(
    "WebMCP.toolInvoked",
    (params) => params.invocationId === accepted.invocationId,
    startIndex,
  );
  const response = await waitForEvent(
    "WebMCP.toolResponded",
    (params) => params.invocationId === accepted.invocationId,
    startIndex,
  );
  if (response.params?.status !== "Completed") {
    throw new Error(`${toolName} did not complete: ${JSON.stringify(response.params)}`);
  }
  let output = response.params?.output;
  if (typeof output === "string") output = JSON.parse(output);
  const result = output?.structuredContent?.result;
  if (!result) throw new Error(`${toolName} returned no structured result`);
  await waitUntil(`${toolName} visible flow completion`, async () => {
    const ui = await uiSnapshot();
    return ui.flow[toolName] === "complete" ? ui : null;
  });
  await assertRecordedSceneBoundary(`${toolName} completion`);
  console.log(`  completed: ${toolName}`);
  return result;
}

async function resetSharedDemo(label) {
  const pending = await currentProposal();
  if (pending) {
    throw new Error(
      `${label}: proposal ${pending.proposalId} is already pending. This take driver will not add a pre-take Discard; clean it first.`,
    );
  }
  const before = await canonicalSummary();
  await waitUntil("enabled shared demo reset", async () => {
    const ui = await uiSnapshot();
    return ui.reset.hidden === false && ui.reset.disabled === false && ui.reset.revision === before.revision ? ui : null;
  });
  const dialogIndex = events.length;
  acceptResetDialog = true;
  try {
    await clickSelector("[data-studio-demo-reset]", label);
    await waitForEvent(
      "Page.javascriptDialogOpening",
      (params) => String(params.message || "").startsWith("Reset the shared public demo scene"),
      dialogIndex,
    );
    const expectedRevision = before.revision + 1;
    const clean = await waitUntil("clean canonical reset", async () => {
      const canonical = await canonicalSummary();
      const ui = await uiSnapshot();
      return canonical.revision === expectedRevision && canonical.demo.clean &&
        canonical.board?.name === "Board" && canonical.board?.materialID === "board-material" &&
        ui.revision === expectedRevision && !ui.proposal.pending && ui.preview.badgeHidden === true &&
        new URL(ui.href).search === ""
        ? { canonical, ui }
        : null;
    });
    return clean;
  } finally {
    acceptResetDialog = false;
  }
}

async function waitForCurrentEvidence(revision) {
  return waitUntil(`Evidence 31/31 current for revision ${revision}`, async () => {
    const ui = await uiSnapshot();
    const pattern = new RegExp(`^Evidence\\s+31/31\\s+·\\s+current\\s+·\\s+rev\\s+0*${revision}(?:\\s+↗)?$`, "i");
    return pattern.test(ui.evidence || "") ? ui : null;
  }, 50000);
}

function pairedPlan(activity, expectedPlan) {
  const proposed = activity.find((entry) =>
    entry.actor === "PROPOSED" && entry.transactionID === `webmcp-proposal:${expectedPlan}`);
  const approved = activity.find((entry) =>
    entry.actor === "APPROVED" && entry.transactionID === `webmcp-commit:${expectedPlan}`);
  if (!proposed || !approved) return null;
  const proposedPlan = proposed.transactionID.slice("webmcp-proposal:".length);
  const approvedPlan = approved.transactionID.slice("webmcp-commit:".length);
  return proposedPlan && proposedPlan === approvedPlan ? { proposed, approved, plan: proposedPlan } : null;
}

function assertNoRecordedDocumentNavigation() {
  const violations = [
    ...recordingGuard.documentRequests.map((entry) => `Document request ${entry.url}`),
    ...recordingGuard.frameNavigations.map((entry) => `main-frame navigation ${entry.url}`),
  ];
  if (violations.length) {
    throw new Error(`Recorded take crossed the zero-reload boundary: ${violations.join("; ")}`);
  }
}

async function armRecordedSceneMount() {
  const state = await evaluate(`(() => {
    const mount = document.querySelector("[data-gosx-scene3d]") ||
      document.querySelector("[data-gosx-scene3d-mounted]");
    const canvas = mount?.querySelector("canvas") || null;
    if (!mount || !canvas || !mount.isConnected || !canvas.isConnected) {
      return { armed: false, hasMount: !!mount, hasCanvas: !!canvas };
    }
    window.__gosx3dPublicDemoMountGuard = { mount, canvas };
    return { armed: true };
  })()`);
  if (state?.armed !== true) {
    throw new Error(`Could not arm the mounted Scene3D canvas guard: ${JSON.stringify(state)}`);
  }
}

async function assertRecordedSceneBoundary(label) {
  assertNoRecordedDocumentNavigation();
  const ui = await uiSnapshot();
  const boundary = ui.sceneBoundary || {};
  const stable = boundary.armed === true && boundary.sameMount === true &&
    boundary.mountConnected === true && boundary.sameCanvas === true &&
    boundary.canvasConnected === true && boundary.canvasInsideMount === true;
  if (!stable || ui.mounted !== "true" || ui.ready !== "true" ||
      ui.backend !== "webgpu" || ui.renderer !== "webgpu" || ui.meshObjects !== 145) {
    throw new Error(
      `${label}: the recorded Scene3D mount/render boundary changed: ` +
      JSON.stringify({ boundary, mounted: ui.mounted, ready: ui.ready,
        backend: ui.backend, renderer: ui.renderer, meshObjects: ui.meshObjects }),
    );
  }
  return ui;
}

async function explicitPostTakeCleanup(reason) {
  console.log(`\nExplicit cleanup requested (${reason}). Recording should already be stopped.`);
  const pendingAtStart = await currentProposal();
  const ui = pendingAtStart
    ? await waitUntil("pending proposal UI before explicit cleanup", async () => {
        const snapshot = await uiSnapshot();
        return snapshot.proposal.pending ? snapshot : null;
      })
    : await uiSnapshot();
  if (ui.proposal.pending) {
    await clickSelector("[data-webmcp-discard]", "explicit aborted-take Discard");
    await waitUntil("proposal discarded", async () => {
      const snapshot = await uiSnapshot();
      return !snapshot.proposal.pending && snapshot.preview.badgeHidden === true ? snapshot : null;
    });
  }
  const pending = await currentProposal();
  if (pending) throw new Error(`Could not clear pending proposal ${pending.proposalId}`);
  return resetSharedDemo("explicit post-take shared reset");
}

console.log(`\nGoSX 3D Studio public recording driver`);
console.log(`Attach-only target: native Windows ${browserVersion.Browser}`);
console.log(`Target id: ${target.id}`);
console.log("No Linux browser will be launched or used. No repo, deployment, cluster, or Devpost state is touched.");
console.log("\nHONESTY BANNER");
console.log("Narrate: ‘Chrome invokes the four page tools.’");
console.log("Do NOT say ‘the agent chose these calls’ unless a visible agent actually receives the prompt and chooses them.");

let takeCompleted = false;
try {
  await Promise.all([
    command("Runtime.enable"),
    command("Page.enable"),
    command("DOM.enable"),
    command("Network.enable"),
  ]);
  await command("Emulation.setDeviceMetricsOverride", {
    width: WIDTH,
    height: HEIGHT,
    deviceScaleFactor: 1,
    mobile: false,
  });
  await command("WebMCP.enable");

  // One deliberate off-camera navigation guarantees the public build is the
  // document being recorded. No later top-level document request or reload is
  // allowed; GoSX managed same-document route transitions remain permitted.
  await command("Page.navigate", { url: STUDIO_URL });
  const nativeWindow = await command("Browser.getWindowForTarget", { targetId: target.id });
  await command("Browser.setWindowBounds", {
    windowId: nativeWindow.windowId,
    bounds: { windowState: "normal" },
  });
  await command("Page.bringToFront");
  await command("Emulation.setFocusEmulationEnabled", { enabled: true });

  await waitUntil("Studio producer registration", async () => {
    const ui = await uiSnapshot();
    return ui.readyState === "complete" && ui.toolCount === "4 tools" && ui.reset.hidden === false ? ui : null;
  });
  activeTools.clear();
  await command("WebMCP.enable");
  const ready = await waitUntil("WebGPU Studio and four native WebMCP tools", async () => {
    const ui = await uiSnapshot();
    const names = [...activeTools.keys()].sort();
    return ui.statusLabel === "Agent tools ready" && ui.toolCount === "4 tools" &&
      ui.secureContext && ui.platform === "Win32" && ui.userAgent.includes("Windows NT") &&
      ui.mounted === "true" && ui.ready === "true" && ui.backend === "webgpu" && ui.renderer === "webgpu" &&
      ui.meshObjects === 145 &&
      JSON.stringify(names) === JSON.stringify(EXPECTED_TOOLS)
      ? ui
      : null;
  });
  studioReady = true;

  const health = await evaluate(`fetch("/api/health", {
    headers: { Accept: "application/json" }, cache: "no-store"
  }).then(async (response) => {
    if (!response.ok) throw new Error("health " + response.status);
    return response.json();
  })`);
  if (health?.ok !== true || health?.version !== EXPECTED_VERSION) {
    throw new Error(`Public health mismatch: ${JSON.stringify(health)}`);
  }
  if (await currentProposal()) {
    throw new Error("A proposal is already pending. Clean the shared workspace before starting this driver.");
  }

  console.log(`\nPreflight passed: GoSX ${EXPECTED_VERSION} · WebGPU · 4 native page tools · ${ready.meshObjects} meshes.`);
  await requireResetConfirmation(
    "OFF CAMERA: reset the single shared public workspace for everyone. Make sure nobody else is testing it.",
  );
  const reset = await resetSharedDemo("off-camera Prepare clean demo");

  await clickSelector(`[data-entity-id="${CORAL_ENTITY_ID}"]`, "off-camera select a Coral Piece");
  await waitUntil("non-Board opening selection", async () => {
    const ui = await uiSnapshot();
    return ui.selectedID === CORAL_ENTITY_ID ? ui : null;
  });
  const baseline = reset.canonical.revision;
  await waitForCurrentEvidence(baseline);
  const baselineCanonical = await canonicalSummary();
  if (baselineCanonical.entityCount !== 150 || baselineCanonical.board?.name !== "Board") {
    throw new Error(`Unexpected clean scene: ${JSON.stringify(baselineCanonical)}`);
  }

  const frameTree = await command("Page.getFrameTree");
  recordingGuard.frameId = String(frameTree.frameTree?.frame?.id || "");
  if (!recordingGuard.frameId) throw new Error("Could not resolve the native Chrome main frame for the reload guard.");
  recordingGuard.armed = true;
  await armRecordedSceneMount();
  await assertRecordedSceneBoundary("recording start");

  console.log(`\nREADY TO RECORD · baseline revision ${baseline}`);
  console.log("Opening state: Coral Piece selected · Board canonical · Evidence 31/31 current.");
  await cue("START OBS. Hold the clean scene for two seconds, make the small opening orbit, then return to the Agent panel.");

  await cue("INSPECT beat ready. Narrate that Chrome is invoking the first typed page tool.");
  const state = await invokeTool("scene_get_state", {});
  if (state.scene?.revision !== baseline) {
    throw new Error(`Inspect saw revision ${state.scene?.revision}; expected ${baseline}. Another visitor may have edited.`);
  }

  await cue("FIND beat ready. Keep Board untouched and let the trace show the structured lookup.");
  const found = await invokeTool("scene_find_objects", { query: "board", visibleOnly: true, limit: 10 });
  const board = found.objects?.find((object) => object.id === "board");
  if (board?.name !== "Board" || board?.materialId !== "board-material") {
    throw new Error(`Find did not resolve the clean Board: ${JSON.stringify(board)}`);
  }

  await cue("FOCUS beat ready. Watch hierarchy and Inspector move to stable ID board without a reload.");
  const focus = await invokeTool("scene_focus_object", { objectId: "board" });
  if (focus.focusRequested !== true || focus.revision !== baseline) {
    throw new Error(`Focus result crossed the revision boundary: ${JSON.stringify(focus)}`);
  }
  await assertRecordedSceneBoundary("visible Board focus");
  await waitUntil("visible Board focus", async () => {
    const ui = await uiSnapshot();
    return ui.selectedID === "board" && ui.boardHierarchyText?.includes("Board") ? ui : null;
  });

  await cue("STAGE beat ready. This creates the only proposal in the take; it does not commit canonical state.");
  const preview = await invokeTool("scene_preview_actions", proposalInput(baseline));
  if (preview.canonicalSceneChanged !== false || preview.humanCommitRequired !== true) {
    throw new Error(`Preview did not preserve the human gate: ${JSON.stringify(preview)}`);
  }
  const staged = await waitUntil("visible reversible Launch Board proposal", async () => {
    const ui = await uiSnapshot();
    const canonical = await canonicalSummary();
    const exactChanges = ui.proposal.changes.some((change) => change.includes("Board") && change.includes("Launch Board")) &&
      ui.proposal.changes.some((change) => change.includes("board-material") && change.includes("board-steel-material"));
    return canonical.revision === baseline && canonical.board?.name === "Board" &&
      canonical.board?.materialID === "board-material" && ui.proposal.pending &&
      ui.proposal.reviewHidden === false && ui.proposal.commitDisabled === false &&
      ui.proposal.commitVisible === true && ui.proposal.commitText === "Apply 2 exact edits" &&
      ui.proposal.summary === "Prepare Launch Board" && ui.proposal.actor === "agent://webmcp" &&
      ui.proposal.policy === "2/2 checks passed" &&
      ui.proposal.revision?.includes(`canonical ${baseline} unchanged`) && exactChanges &&
      ui.preview.active && ui.preview.badgeHidden === false && ui.preview.badgeText?.includes("not committed")
      ? { ui, canonical }
      : null;
  });
  await assertRecordedSceneBoundary("staged preview");

  console.log(`\nPROPOSAL READY · canonical revision ${baseline} remains unchanged · proposal ${preview.proposalId}`);
  await cue(
    "REVIEW / ORBIT: show the four-call trace, exact two-edit proposal, review checks, and fingerprint. Orbit the live Brushed Steel preview once; close the check details before continuing.",
  );

  const beforeApply = await canonicalSummary();
  const beforeApplyUI = await uiSnapshot();
  if (beforeApply.revision !== baseline || beforeApply.board?.name !== "Board" ||
      !beforeApplyUI.proposal.pending || beforeApplyUI.preview.badgeHidden !== false) {
    throw new Error("The shared revision or proposal changed during review. Stop and reset for a clean take.");
  }
  await assertRecordedSceneBoundary("human review");

  await cue(
    "MANUAL HUMAN APPROVAL: click the visible ‘Apply 2 exact edits’ button yourself exactly once. Wait for Evidence 31/31 · current at the next revision, sweep Launch Board / Brushed Steel / paired activity, leave two seconds still, and STOP OBS. Press Enter here only after recording has stopped.",
  );

  const expectedRevision = baseline + 1;
  const finalEvidence = await waitForCurrentEvidence(expectedRevision);
  const final = await waitUntil("verified human-approved final scene", async () => {
    const canonical = await canonicalSummary();
    const ui = await uiSnapshot();
    const pair = pairedPlan(ui.activity, preview.proposalId);
    const flowComplete = EXPECTED_TOOLS.every((name) => ui.flow[name] === "complete");
    return canonical.revision === expectedRevision && canonical.demo.clean === false &&
      canonical.board?.name === "Launch Board" && canonical.board?.materialID === STEEL_MATERIAL_ID &&
      ui.revision === expectedRevision && ui.statusLabel === "Human-approved change applied" &&
      ui.selectedID === "board" && ui.boardHierarchyText?.includes("Launch Board") &&
      ui.material?.id === STEEL_MATERIAL_ID && ui.material?.name === "Brushed Steel" &&
      ui.approvalOutcome.hidden === false &&
      ui.approvalOutcome.text?.includes("Human approved") &&
      ui.approvalOutcome.text?.includes(`revision ${baseline} → ${expectedRevision}`) &&
      !ui.proposal.pending && ui.preview.badgeHidden === true &&
      flowComplete && ui.trace.length === EXPECTED_TOOLS.length && pair &&
      finalEvidence.evidence === ui.evidence
      ? { canonical, ui, pair }
      : null;
  }, 50000);
  if (await currentProposal()) throw new Error("A proposal remained pending after the recorded human approval.");
  await assertRecordedSceneBoundary("verified final scene");
  recordingGuard.armed = false;
  takeCompleted = true;
  console.log(
    `Recording verified: revision ${baseline} → ${expectedRevision} · Launch Board · Brushed Steel · ` +
    `paired plan ${final.pair.plan} · Evidence 31/31 · 0 document reloads · ` +
    `same mounted WebGPU canvas · ${recordingGuard.sameDocumentNavigations.length} managed same-document transition(s).`,
  );
  console.log("The driver observed—but did not perform—the human-only Apply action.");
  console.log("Native Chrome remains open. Leave the applied result for the demo, or use Prepare clean demo manually after recording.");
} catch (error) {
  recordingGuard.armed = false;
  console.error(`\n${error instanceof OperatorAbort ? "TAKE ABORTED" : "TAKE STOPPED"}: ${error.message}`);
  console.error("No automatic scene cleanup has run. Stop OBS before choosing cleanup.");
  if (studioReady) {
    try {
      const cleanupChoice = String(await ask(
        "Type RESET to explicitly discard any aborted proposal and reset the shared scene, or press Enter to leave it unchanged: ",
        { abortable: false },
      )).trim().toUpperCase();
      if (cleanupChoice === "RESET") {
        const clean = await explicitPostTakeCleanup("operator aborted/stopped take");
        console.log(`Clean public scene restored at revision ${clean.canonical.revision}.`);
      } else if (cleanupChoice) {
        console.error(`Unknown cleanup response ${JSON.stringify(cleanupChoice)}; scene left unchanged.`);
      }
    } catch (cleanupError) {
      console.error(`Explicit cleanup failed: ${cleanupError.message}`);
    }
  }
  process.exitCode = error instanceof OperatorAbort ? 130 : 1;
} finally {
  await command("Emulation.setFocusEmulationEnabled", { enabled: false }, 1500).catch(() => {});
  await command("Emulation.clearDeviceMetricsOverride", {}, 1500).catch(() => {});
  socket.close();
  rl.close();
}

if (!takeCompleted && process.exitCode == null) process.exitCode = 1;
