(function () {
  "use strict";

  if (typeof window === "undefined" || typeof document === "undefined") return;
  if (window.__gosxStudioWebMCPAdapter) {
    window.__gosxStudioWebMCPAdapter.register();
    return;
  }

  var EVENTS = {
    status: "studio:webmcp:status",
    focus: "studio:webmcp:focus",
    proposal: "studio:webmcp:proposal",
    trace: "studio:webmcp:trace"
  };
  var TOOL_NAMES = ["scene_get_state", "scene_find_objects", "scene_focus_object", "scene_preview_actions"];
  var REQUEST_TIMEOUT_MS = 15000;
  var MAX_OPERATIONS = 12;
  var registrationController = new AbortController();
  var registrationState = "idle";
  var registrationAttempts = 0;
  var retryTimer = 0;
  var callSequence = 0;

  function AdapterError(code, message, status) {
    this.name = "AdapterError";
    this.code = code;
    this.message = message;
    this.status = status || 0;
    if (Error.captureStackTrace) Error.captureStackTrace(this, AdapterError);
  }
  AdapterError.prototype = Object.create(Error.prototype);
  AdapterError.prototype.constructor = AdapterError;

  function dispatch(name, detail) {
    var event;
    if (typeof window.CustomEvent === "function") {
      event = new window.CustomEvent(name, { detail: detail });
    } else {
      event = document.createEvent("CustomEvent");
      event.initCustomEvent(name, false, false, detail);
    }
    document.dispatchEvent(event);
  }

  function emitStatus(state, message, fields) {
    var detail = {
      source: "webmcp",
      state: state,
      message: message || "",
      toolCount: registrationState === "ready" ? TOOL_NAMES.length : 0
    };
    Object.keys(fields || {}).forEach(function (key) { detail[key] = fields[key]; });
    dispatch(EVENTS.status, detail);
  }

  function fail(code, message, status) {
    throw new AdapterError(code, message, status);
  }

  function boundedMessage(value, fallback) {
    var message = typeof value === "string" ? value.trim() : "";
    if (!message) message = fallback;
    return message.length > 400 ? message.slice(0, 397) + "..." : message;
  }

  function normalizedError(error) {
    if (error instanceof AdapterError) return error;
    if (error && error.name === "AbortError") {
      return new AdapterError("CANCELLED", "The WebMCP tool call was cancelled.");
    }
    if (error instanceof TypeError) {
      return new AdapterError("NETWORK_ERROR", "The Studio API could not be reached.");
    }
    return new AdapterError("UNEXPECTED_ERROR", boundedMessage(error && error.message, "The tool call failed."));
  }

  function successResult(tool, message, data) {
    return {
      content: [{ type: "text", text: message }],
      structuredContent: { ok: true, tool: tool, result: data }
    };
  }

  function errorResult(tool, error) {
    var detail = {
      code: error.code || "UNEXPECTED_ERROR",
      message: boundedMessage(error.message, "The tool call failed.")
    };
    if (error.status) detail.status = error.status;
    return {
      isError: true,
      content: [{ type: "text", text: detail.message }],
      structuredContent: { ok: false, tool: tool, error: detail }
    };
  }

  function traceMessage(tool, result) {
    var data = result && result.data || {};
    if (tool === "scene_get_state") {
      var scene = data.scene || {};
      var counts = data.counts || {};
      return "Inspect · revision " + String(scene.revision == null ? "?" : scene.revision) + " · " + String(counts.objects == null ? "?" : counts.objects) + " objects";
    }
    if (tool === "scene_find_objects") {
      var objects = Array.isArray(data.objects) ? data.objects : [];
      var first = objects[0] || {};
      var kinds = Array.isArray(first.components) ? first.components : [];
      return "Find · " + String(data.totalMatches == null ? objects.length : data.totalMatches) + " match" + (Number(data.totalMatches) === 1 ? "" : "es") + (first.id ? " · " + String(first.id) : "") + (kinds[0] ? " · " + String(kinds[0]) : "");
    }
    if (tool === "scene_focus_object") {
      return "Focus · " + String(data.object && data.object.id || "object") + " · UI only";
    }
    if (tool === "scene_preview_actions") {
      var receipt = data.receipt || {};
      return "Stage · " + String(receipt.operations == null ? "?" : receipt.operations) + " operations · canonical " + String(receipt.beforeRevision == null ? "?" : receipt.beforeRevision) + " unchanged";
    }
    return tool + " completed";
  }

  function emitTrace(callId, tool, state, message, code) {
    dispatch(EVENTS.trace, {
      source: "webmcp",
      callId: callId,
      tool: tool,
      state: state,
      message: message,
      code: code || "",
      timestamp: new Date().toISOString()
    });
  }

  function execute(tool, handler) {
    return async function (input, options) {
      var signal = options && options.signal;
      var callId = "webmcp-" + String(Date.now()) + "-" + String(++callSequence);
      emitStatus("ready", "Running " + tool + "…", { tool: tool, toolCount: TOOL_NAMES.length });
      try {
        var result = await handler(typeof input === "undefined" ? {} : input, signal);
        emitTrace(callId, tool, "complete", traceMessage(tool, result));
        // The proposal event moves the UI into an explicit human-review state;
        // do not immediately overwrite that state with a generic completion.
        if (tool !== "scene_preview_actions") {
          emitStatus("ready", tool + " completed.", { tool: tool, toolCount: TOOL_NAMES.length });
        }
        return successResult(tool, result.message, result.data);
      } catch (caught) {
        var error = normalizedError(caught);
        emitTrace(callId, tool, "error", tool + " · " + String(error.code || "UNEXPECTED_ERROR"), error.code);
        emitStatus("error", error.message, { tool: tool, code: error.code, toolCount: TOOL_NAMES.length });
        return errorResult(tool, error);
      }
    };
  }

  function csrfToken() {
    var field = document.querySelector('input[name="csrf_token"]');
    return field && typeof field.value === "string" ? field.value : "";
  }

  async function requestJSON(path, init, executionSignal) {
    var url = new URL(path, window.location.origin);
    if (url.origin !== window.location.origin) {
      fail("CROSS_ORIGIN_REQUEST", "Studio WebMCP tools only call same-origin APIs.");
    }

    var method = String((init && init.method) || "GET").toUpperCase();
    var headers = { Accept: "application/json" };
    var body;
    if (method !== "GET" && method !== "HEAD") {
      var token = csrfToken();
      if (!token) fail("CSRF_UNAVAILABLE", "The Studio session token is unavailable; reload the page and try again.");
      headers["Content-Type"] = "application/json";
      headers["X-CSRF-Token"] = token;
      body = JSON.stringify(init && init.body);
    }

    var requestController = new AbortController();
    var relays = [];
    var timedOut = false;
    function link(signal) {
      if (!signal) return;
      if (signal.aborted) {
        requestController.abort(signal.reason);
        return;
      }
      var relay = function () { requestController.abort(signal.reason); };
      signal.addEventListener("abort", relay, { once: true });
      relays.push([signal, relay]);
    }
    link(executionSignal);
    link(registrationController.signal);
    var timeout = window.setTimeout(function () {
      timedOut = true;
      requestController.abort();
    }, REQUEST_TIMEOUT_MS);

    try {
      var response = await window.fetch(url.href, {
        method: method,
        headers: headers,
        body: body,
        cache: "no-store",
        credentials: "same-origin",
        mode: "same-origin",
        redirect: "error",
        referrerPolicy: "same-origin",
        signal: requestController.signal
      });
      var text = await response.text();
      var payload = null;
      if (text) {
        try {
          payload = JSON.parse(text);
        } catch (_) {
          fail("INVALID_JSON_RESPONSE", "The Studio API returned an invalid JSON response.", response.status);
        }
      }
      if (!response.ok) {
        var serverMessage = payload && payload.error;
        if (serverMessage && typeof serverMessage === "object") serverMessage = serverMessage.message;
        if (!serverMessage && payload && typeof payload.message === "string") serverMessage = payload.message;
        fail("HTTP_" + response.status, boundedMessage(serverMessage, "Studio API request failed with HTTP " + response.status + "."), response.status);
      }
      return payload;
    } catch (error) {
      if (timedOut) fail("REQUEST_TIMEOUT", "The Studio API did not respond within 15 seconds.");
      if (executionSignal && executionSignal.aborted) fail("CANCELLED", "The WebMCP tool call was cancelled.");
      if (registrationController.signal.aborted) fail("CANCELLED", "The page stopped the WebMCP tool call.");
      throw error;
    } finally {
      window.clearTimeout(timeout);
      relays.forEach(function (entry) { entry[0].removeEventListener("abort", entry[1]); });
    }
  }

  function expectObject(value, path) {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      fail("INVALID_INPUT", path + " must be an object.");
    }
    return value;
  }

  function onlyKeys(value, allowed, path) {
    Object.keys(value).forEach(function (key) {
      if (allowed.indexOf(key) === -1) fail("INVALID_INPUT", path + " contains unsupported property " + JSON.stringify(key) + ".");
    });
  }

  function readString(value, key, path, options) {
    options = options || {};
    if (!Object.prototype.hasOwnProperty.call(value, key)) {
      if (options.required) fail("INVALID_INPUT", path + "." + key + " is required.");
      return options.fallback || "";
    }
    if (typeof value[key] !== "string") fail("INVALID_INPUT", path + "." + key + " must be a string.");
    var result = value[key].trim();
    if (!result && !options.allowEmpty) fail("INVALID_INPUT", path + "." + key + " must not be empty.");
    if (options.max && result.length > options.max) fail("INVALID_INPUT", path + "." + key + " must be at most " + options.max + " characters.");
    return result;
  }

  function readID(value, key, path) {
    var result = readString(value, key, path, { required: true, max: 160 });
    if (value[key] !== result) fail("INVALID_INPUT", path + "." + key + " must not have leading or trailing whitespace.");
    return result;
  }

  function readInteger(value, key, path, minimum, maximum, fallback) {
    if (!Object.prototype.hasOwnProperty.call(value, key)) return fallback;
    var result = value[key];
    if (!Number.isInteger(result) || result < minimum || result > maximum) {
      fail("INVALID_INPUT", path + "." + key + " must be an integer from " + minimum + " to " + maximum + ".");
    }
    return result;
  }

  function readBoolean(value, key, path, fallback) {
    if (!Object.prototype.hasOwnProperty.call(value, key)) return fallback;
    if (typeof value[key] !== "boolean") fail("INVALID_INPUT", path + "." + key + " must be a boolean.");
    return value[key];
  }

  function readVec3(value, path) {
    expectObject(value, path);
    onlyKeys(value, ["x", "y", "z"], path);
    var out = {};
    ["x", "y", "z"].forEach(function (axis) {
      var number = value[axis];
      if (typeof number !== "number" || !Number.isFinite(number) || Math.abs(number) > 1000000) {
        fail("INVALID_INPUT", path + "." + axis + " must be a finite number between -1000000 and 1000000.");
      }
      out[axis] = number;
    });
    return out;
  }

  function copyVec3(value, fallback) {
    value = value || fallback || {};
    return {
      x: Number.isFinite(value.x) ? value.x : 0,
      y: Number.isFinite(value.y) ? value.y : 0,
      z: Number.isFinite(value.z) ? value.z : 0
    };
  }

  function multiplyQuaternion(a, b) {
    return {
      x: a.w * b.x + a.x * b.w + a.y * b.z - a.z * b.y,
      y: a.w * b.y - a.x * b.z + a.y * b.w + a.z * b.x,
      z: a.w * b.z + a.x * b.y - a.y * b.x + a.z * b.w,
      w: a.w * b.w - a.x * b.x - a.y * b.y - a.z * b.z
    };
  }

  function normalizedQuaternion(value) {
    value = value || {};
    var x = Number.isFinite(value.x) ? value.x : 0;
    var y = Number.isFinite(value.y) ? value.y : 0;
    var z = Number.isFinite(value.z) ? value.z : 0;
    var w = Number.isFinite(value.w) ? value.w : 0;
    var length = Math.sqrt(x * x + y * y + z * z + w * w);
    if (!length) return { x: 0, y: 0, z: 0, w: 1 };
    return { x: x / length, y: y / length, z: z / length, w: w / length };
  }

  function quaternionFromEuler(rotation) {
    function axisQuaternion(axis, angle) {
      var half = angle / 2;
      var sine = Math.sin(half);
      return { x: axis === "x" ? sine : 0, y: axis === "y" ? sine : 0, z: axis === "z" ? sine : 0, w: Math.cos(half) };
    }
    var qx = axisQuaternion("x", rotation.x);
    var qy = axisQuaternion("y", rotation.y);
    var qz = axisQuaternion("z", rotation.z);
    return normalizedQuaternion(multiplyQuaternion(multiplyQuaternion(qz, qy), qx));
  }

  function componentKinds(entity) {
    var kinds = [];
    if (entity.mesh) kinds.push("mesh");
    if (entity.light) kinds.push("light");
    if (entity.model) kinds.push("model");
    if (entity.prefab) kinds.push("prefab");
    if (entity.physics) kinds.push("physics");
    if (!kinds.length) kinds.push("group");
    return kinds;
  }

  function objectSummary(entity) {
    var summary = {
      id: entity.id,
      name: entity.name,
      parentId: entity.parent || null,
      components: componentKinds(entity),
      visible: entity.visible !== false,
      locked: entity.locked === true,
      childCount: Array.isArray(entity.children) ? entity.children.length : 0,
      transform: entity.transform || null
    };
    if (entity.mesh) {
      summary.materialId = entity.mesh.material || null;
      summary.geometryKind = entity.mesh.geometry && entity.mesh.geometry.kind || null;
    }
    if (entity.light) summary.lightKind = entity.light.kind || null;
    if (entity.model) summary.assetId = entity.model.asset || null;
    return summary;
  }

  function validateDocument(documentValue) {
    expectObject(documentValue, "Studio document response");
    if (!Number.isInteger(documentValue.revision) || documentValue.revision < 0) {
      fail("INVALID_SERVER_RESPONSE", "The Studio document response has no valid revision.");
    }
    expectObject(documentValue.entities, "Studio document response.entities");
    return documentValue;
  }

  async function getDocument(signal) {
    return validateDocument(await requestJSON("/api/studio/document", { method: "GET" }, signal));
  }

  function sceneCounts(documentValue) {
    var counts = { objects: 0, meshes: 0, lights: 0, models: 0, prefabs: 0, physicsBodies: 0, groups: 0, hidden: 0, locked: 0 };
    Object.keys(documentValue.entities).forEach(function (id) {
      var entity = documentValue.entities[id];
      counts.objects++;
      if (entity.mesh) counts.meshes++;
      if (entity.light) counts.lights++;
      if (entity.model) counts.models++;
      if (entity.prefab) counts.prefabs++;
      if (entity.physics) counts.physicsBodies++;
      if (!entity.mesh && !entity.light && !entity.model && !entity.prefab && !entity.physics) counts.groups++;
      if (entity.visible === false) counts.hidden++;
      if (entity.locked === true) counts.locked++;
    });
    return counts;
  }

  async function handleGetState(input, signal) {
    expectObject(input, "input");
    onlyKeys(input, [], "input");
    var responses = await Promise.all([
      getDocument(signal),
      requestJSON("/api/studio/selection", { method: "GET" }, signal)
    ]);
    var documentValue = responses[0];
    var selection = responses[1] || {};
    var rootObjects = (documentValue.rootIds || []).map(function (id) {
      return documentValue.entities[id] ? objectSummary(documentValue.entities[id]) : { id: id, missing: true };
    });
    var materials = Object.keys(documentValue.materials || {}).sort().map(function (id) {
      var material = documentValue.materials[id];
      return { id: id, name: material.name, color: material.color, roughness: material.roughness, metalness: material.metalness };
    });
    var selectionRevision = selection.state && selection.state.revision;
    var data = {
      scene: { schema: documentValue.schema, id: documentValue.id, name: documentValue.name, revision: documentValue.revision },
      counts: sceneCounts(documentValue),
      rootObjects: rootObjects,
      materials: materials,
      camera: documentValue.camera || null,
      environment: documentValue.environment || null,
      selection: selection.state || null,
      selectionMatchesSceneRevision: selectionRevision === documentValue.revision
    };
    return { message: "Scene " + JSON.stringify(documentValue.name) + " is at revision " + documentValue.revision + " with " + data.counts.objects + " objects.", data: data };
  }

  function validateFindInput(input) {
    expectObject(input, "input");
    onlyKeys(input, ["query", "types", "visibleOnly", "limit"], "input");
    var query = readString(input, "query", "input", { allowEmpty: true, max: 120, fallback: "" });
    var types = [];
    if (Object.prototype.hasOwnProperty.call(input, "types")) {
      if (!Array.isArray(input.types) || input.types.length > 6) fail("INVALID_INPUT", "input.types must be an array with at most 6 values.");
      input.types.forEach(function (type) {
        if (["group", "mesh", "light", "model", "prefab", "physics"].indexOf(type) === -1) {
          fail("INVALID_INPUT", "input.types contains unsupported object type " + JSON.stringify(type) + ".");
        }
        if (types.indexOf(type) === -1) types.push(type);
      });
    }
    return {
      query: query,
      types: types,
      visibleOnly: readBoolean(input, "visibleOnly", "input", false),
      limit: readInteger(input, "limit", "input", 1, 50, 20)
    };
  }

  async function handleFindObjects(input, signal) {
    var filter = validateFindInput(input);
    var documentValue = await getDocument(signal);
    var needle = filter.query.toLocaleLowerCase();
    var matches = Object.keys(documentValue.entities).map(function (id) {
      return documentValue.entities[id];
    }).filter(function (entity) {
      if (filter.visibleOnly && entity.visible === false) return false;
      var kinds = componentKinds(entity);
      if (filter.types.length && !filter.types.some(function (type) { return kinds.indexOf(type) !== -1; })) return false;
      if (!needle) return true;
      return String(entity.id).toLocaleLowerCase().indexOf(needle) !== -1 || String(entity.name || "").toLocaleLowerCase().indexOf(needle) !== -1;
    });
    matches.sort(function (a, b) {
      function score(entity) {
        if (!needle) return 3;
        var id = String(entity.id).toLocaleLowerCase();
        var name = String(entity.name || "").toLocaleLowerCase();
        if (id === needle || name === needle) return 0;
        if (id.indexOf(needle) === 0 || name.indexOf(needle) === 0) return 1;
        return 2;
      }
      var difference = score(a) - score(b);
      if (difference) return difference;
      return String(a.name || a.id).localeCompare(String(b.name || b.id)) || String(a.id).localeCompare(String(b.id));
    });
    var objects = matches.slice(0, filter.limit).map(objectSummary);
    var names = objects.slice(0, 5).map(function (object) { return object.name + " (" + object.id + ")"; });
    return {
      message: "Found " + matches.length + " matching object(s)" + (names.length ? ": " + names.join(", ") + (matches.length > names.length ? "." : ".") : "."),
      data: { revision: documentValue.revision, totalMatches: matches.length, truncated: matches.length > objects.length, objects: objects }
    };
  }

  async function handleFocusObject(input, signal) {
    expectObject(input, "input");
    onlyKeys(input, ["objectId"], "input");
    var objectId = readID(input, "objectId", "input");
    var documentValue = await getDocument(signal);
    var entity = documentValue.entities[objectId];
    if (!entity) fail("OBJECT_NOT_FOUND", "Scene object " + JSON.stringify(objectId) + " does not exist at revision " + documentValue.revision + ".");
    var object = objectSummary(entity);
    dispatch(EVENTS.focus, { source: "webmcp", id: objectId, objectId: objectId, revision: documentValue.revision, object: object });
    return { message: "Requested viewport focus on " + JSON.stringify(object.name) + " (" + objectId + ").", data: { revision: documentValue.revision, focusRequested: true, object: object } };
  }

  function clone(value) {
    return JSON.parse(JSON.stringify(value));
  }

  function requireEntity(entities, id, path) {
    var entity = entities[id];
    if (!entity) fail("INVALID_INPUT", path + " targets missing scene object " + JSON.stringify(id) + ".");
    return entity;
  }

  function ensureUnlocked(entity, path) {
    if (entity.locked === true) fail("INVALID_INPUT", path + " targets locked scene object " + JSON.stringify(entity.id) + ".");
  }

  function vec3Equal(left, right, fallback) {
    left = copyVec3(left, fallback);
    right = copyVec3(right, fallback);
    return left.x === right.x && left.y === right.y && left.z === right.z;
  }

  function editableState(entity) {
    var transform = entity && entity.transform || {};
    return {
      name: entity && entity.name || "",
      material: entity && entity.mesh && entity.mesh.material || "",
      position: copyVec3(transform.position),
      rotation: copyVec3(transform.rotation),
      scale: copyVec3(transform.scale, { x: 1, y: 1, z: 1 })
    };
  }

  function normalizeOperations(operations, documentValue) {
    if (!Array.isArray(operations) || !operations.length || operations.length > MAX_OPERATIONS) {
      fail("INVALID_INPUT", "input.operations must contain 1 to " + MAX_OPERATIONS + " actions.");
    }
    var entities = Object.assign({}, documentValue.entities);
    var materials = documentValue.materials || {};
    var touched = {};
    var normalized = operations.map(function (operation, index) {
      var path = "input.operations[" + index + "]";
      expectObject(operation, path);
      var kind = readString(operation, "kind", path, { required: true, max: 40 });
      var target;
      var entity;
      if (kind === "rename-entity") {
        onlyKeys(operation, ["kind", "target", "name"], path);
        target = readID(operation, "target", path);
        entity = requireEntity(entities, target, path);
        ensureUnlocked(entity, path);
        var name = readString(operation, "name", path, { required: true, max: 120 });
        if (entity.name === name) {
          fail("ALREADY_SATISFIED", path + " already names " + JSON.stringify(target) + " " + JSON.stringify(name) + "; inspect the current scene or choose a different edit.");
        }
        entity = clone(entity);
        entity.name = name;
        entities[target] = entity;
        touched[target] = true;
        return { kind: kind, target: target, name: name };
      }
      if (kind === "assign-material") {
        onlyKeys(operation, ["kind", "target", "material"], path);
        target = readID(operation, "target", path);
        entity = requireEntity(entities, target, path);
        ensureUnlocked(entity, path);
        var material = readID(operation, "material", path);
        if (!entity.mesh) fail("INVALID_INPUT", path + " can only assign material to a mesh object.");
        if (!materials[material]) fail("INVALID_INPUT", path + " references missing material " + JSON.stringify(material) + ".");
        if (entity.mesh.material === material) {
          fail("ALREADY_SATISFIED", path + " already assigns material " + JSON.stringify(material) + " to " + JSON.stringify(target) + "; inspect the current scene or choose a different material.");
        }
        entity = clone(entity);
        entity.mesh.material = material;
        entities[target] = entity;
        touched[target] = true;
        return { kind: kind, target: target, material: material };
      }
      if (kind === "set-transform") {
        onlyKeys(operation, ["kind", "target", "transform"], path);
        target = readID(operation, "target", path);
        entity = requireEntity(entities, target, path);
        ensureUnlocked(entity, path);
        var patch = expectObject(operation.transform, path + ".transform");
        onlyKeys(patch, ["position", "rotation", "scale"], path + ".transform");
        if (!Object.keys(patch).length) fail("INVALID_INPUT", path + ".transform must change position, rotation, or scale.");
        var current = entity.transform || {};
        var position = Object.prototype.hasOwnProperty.call(patch, "position") ? readVec3(patch.position, path + ".transform.position") : copyVec3(current.position);
        var rotation = Object.prototype.hasOwnProperty.call(patch, "rotation") ? readVec3(patch.rotation, path + ".transform.rotation") : copyVec3(current.rotation);
        var scale = Object.prototype.hasOwnProperty.call(patch, "scale") ? readVec3(patch.scale, path + ".transform.scale") : copyVec3(current.scale, { x: 1, y: 1, z: 1 });
        if (entity.light && (scale.x !== 1 || scale.y !== 1 || scale.z !== 1)) {
          fail("INVALID_INPUT", path + " cannot scale a light; light scale has no render meaning.");
        }
        if (vec3Equal(position, current.position) && vec3Equal(rotation, current.rotation) && vec3Equal(scale, current.scale, { x: 1, y: 1, z: 1 })) {
          fail("ALREADY_SATISFIED", path + " already matches the current transform for " + JSON.stringify(target) + "; inspect the current scene or choose a different transform.");
        }
        var transform = {
          position: position,
          quaternion: Object.prototype.hasOwnProperty.call(patch, "rotation") ? quaternionFromEuler(rotation) : normalizedQuaternion(current.quaternion),
          rotation: rotation,
          scale: scale
        };
        entity = clone(entity);
        entity.transform = transform;
        entities[target] = entity;
        touched[target] = true;
        return { kind: kind, target: target, transform: transform };
      }
      fail("INVALID_INPUT", path + ".kind must be rename-entity, set-transform, or assign-material.");
    });
    var changed = Object.keys(touched).some(function (id) {
      return JSON.stringify(editableState(entities[id])) !== JSON.stringify(editableState(documentValue.entities[id]));
    });
    if (!changed) {
      fail("ALREADY_SATISFIED", "The proposed operations cancel out or already match the canonical scene; inspect the current scene and stage a meaningful change.");
    }
    return normalized;
  }

  function receiptSummary(receipt) {
    return {
      transactionId: receipt.transactionId,
      mode: receipt.mode,
      applied: receipt.applied,
      beforeRevision: receipt.beforeRevision,
      afterRevision: receipt.afterRevision,
      operations: receipt.operations,
      actor: receipt.actor,
      affected: Array.isArray(receipt.affected) ? receipt.affected : [],
      beforeFingerprint: receipt.beforeFingerprint,
      afterFingerprint: receipt.afterFingerprint,
      telemetryCorrelationId: receipt.telemetryCorrelationId
    };
  }

  async function handlePreviewActions(input, signal) {
    expectObject(input, "input");
    onlyKeys(input, ["expectedRevision", "title", "rationale", "operations"], "input");
    var expectedRevision = readInteger(input, "expectedRevision", "input", 0, Number.MAX_SAFE_INTEGER);
    if (typeof expectedRevision === "undefined") fail("INVALID_INPUT", "input.expectedRevision is required.");
    var title = readString(input, "title", "input", { required: true, max: 80 });
    var rationale = readString(input, "rationale", "input", { allowEmpty: true, max: 400, fallback: "" });
    var documentValue = await getDocument(signal);
    if (expectedRevision !== documentValue.revision) {
      fail("REVISION_CONFLICT", "Scene revision changed from " + expectedRevision + " to " + documentValue.revision + "; inspect state and prepare a fresh proposal.", 409);
    }
    var operations = normalizeOperations(input.operations, documentValue);
    var response = await requestJSON("/api/studio/webmcp/proposals", {
      method: "POST",
      body: { expectedRevision: expectedRevision, title: title, rationale: rationale, operations: operations }
    }, signal);
    expectObject(response, "Studio proposal response");
    var proposalId = readString(response, "proposalId", "Studio proposal response", { required: true, max: 256 });
    var receipt = expectObject(response.receipt, "Studio proposal response.receipt");
    if (receipt.mode !== "propose" || receipt.applied !== false || receipt.beforeRevision !== expectedRevision) {
      fail("INVALID_SERVER_RESPONSE", "The Studio proposal endpoint did not return a non-applied proposal receipt for the requested revision.");
    }
    var preview = expectObject(response.preview, "Studio proposal response.preview");
    var conciseReceipt = receiptSummary(receipt);
    var governance = Array.isArray(response.governance) ? response.governance : [];
    var eventDetail = {
      source: "webmcp",
      proposalId: proposalId,
      title: title,
      rationale: rationale,
      operations: operations,
      receipt: receipt,
      governance: governance,
      preview: preview,
      materials: response.materials || {},
      sceneCommands: Array.isArray(response.sceneCommands) ? response.sceneCommands : [],
      reverseSceneCommands: Array.isArray(response.reverseSceneCommands) ? response.reverseSceneCommands : [],
      expiresAt: response.expiresAt || null
    };
    dispatch(EVENTS.proposal, eventDetail);
    return {
      message: "Staged proposal " + JSON.stringify(title) + " for human review; canonical scene revision " + expectedRevision + " was not changed.",
      data: {
        proposalId: proposalId,
        title: title,
        preview: preview,
        expiresAt: response.expiresAt || null,
        receipt: conciseReceipt,
        governance: governance.map(function (decision) {
          return {
            kind: decision && decision.kind,
            allowed: decision && decision.allowed === true,
            selected: decision && decision.selected,
            reason: decision && decision.reason
          };
        }),
        materials: response.materials || {},
        humanCommitRequired: true,
        canonicalSceneChanged: false
      }
    };
  }

  var ID_SCHEMA = { type: "string", minLength: 1, maxLength: 160, description: "Stable SceneDoc object or material ID." };
  var VEC3_SCHEMA = {
    type: "object",
    additionalProperties: false,
    required: ["x", "y", "z"],
    properties: {
      x: { type: "number", minimum: -1000000, maximum: 1000000 },
      y: { type: "number", minimum: -1000000, maximum: 1000000 },
      z: { type: "number", minimum: -1000000, maximum: 1000000 }
    }
  };
  var TRANSFORM_PATCH_SCHEMA = {
    type: "object",
    additionalProperties: false,
    minProperties: 1,
    description: "Only include transform fields to change. Rotation is XYZ Euler radians.",
    properties: { position: VEC3_SCHEMA, rotation: VEC3_SCHEMA, scale: VEC3_SCHEMA }
  };
  var OPERATION_SCHEMAS = [
    {
      type: "object", additionalProperties: false, required: ["kind", "target", "name"],
      properties: { kind: { const: "rename-entity" }, target: ID_SCHEMA, name: { type: "string", minLength: 1, maxLength: 120 } }
    },
    {
      type: "object", additionalProperties: false, required: ["kind", "target", "transform"],
      properties: { kind: { const: "set-transform" }, target: ID_SCHEMA, transform: TRANSFORM_PATCH_SCHEMA }
    },
    {
      type: "object", additionalProperties: false, required: ["kind", "target", "material"],
      properties: { kind: { const: "assign-material" }, target: ID_SCHEMA, material: ID_SCHEMA }
    }
  ];

  function register() {
    if (registrationState === "registering" || registrationState === "ready" || registrationController.signal.aborted) return registrationState === "ready";
    if (!document.modelContext || typeof document.modelContext.registerTool !== "function") {
      registrationAttempts++;
      if (registrationAttempts === 1) emitStatus("detecting", "Waiting for WebMCP browser support.");
      if (registrationAttempts <= 5) {
        var delay = [100, 250, 750, 1500, 3000][registrationAttempts - 1];
        retryTimer = window.setTimeout(register, delay);
      } else {
        registrationState = "unsupported";
        emitStatus("unavailable", "document.modelContext.registerTool is unavailable.");
      }
      return false;
    }

    registrationState = "registering";
    emitStatus("detecting", "Registering four structured Studio tools…", { tools: TOOL_NAMES.slice() });
    var registrations;
    try {
      registrations = [
        Promise.resolve(document.modelContext.registerTool({
          name: "scene_get_state",
          title: "Inspect scene state",
          description: "Read a concise canonical SceneDoc overview: revision, object/component counts, roots, materials, camera, environment, and current selection. Call this before preparing revision-safe actions.",
          inputSchema: { type: "object", additionalProperties: false, properties: {} },
          annotations: { readOnlyHint: true, untrustedContentHint: true },
          execute: execute("scene_get_state", handleGetState)
        }, { signal: registrationController.signal })),
        Promise.resolve(document.modelContext.registerTool({
          name: "scene_find_objects",
          title: "Find scene objects",
          description: "Search canonical SceneDoc objects by stable ID or human name, optionally filtering by component type and visibility. Returns concise object records with transforms for planning actions.",
          inputSchema: {
            type: "object",
            additionalProperties: false,
            properties: {
              query: { type: "string", maxLength: 120, description: "Case-insensitive substring of object name or stable ID. Empty returns all objects up to limit." },
              types: { type: "array", maxItems: 6, uniqueItems: true, items: { type: "string", enum: ["group", "mesh", "light", "model", "prefab", "physics"] } },
              visibleOnly: { type: "boolean", default: false },
              limit: { type: "integer", minimum: 1, maximum: 50, default: 20 }
            }
          },
          annotations: { readOnlyHint: true, untrustedContentHint: true },
          execute: execute("scene_find_objects", handleFindObjects)
        }, { signal: registrationController.signal })),
        Promise.resolve(document.modelContext.registerTool({
          name: "scene_focus_object",
          title: "Focus a scene object",
          description: "Request that the visible Studio UI focus and select one canonical object by stable ID so the viewport and Inspector converge. This only changes ephemeral UI state; it never mutates the canonical SceneDoc or commits an edit.",
          inputSchema: {
            type: "object",
            additionalProperties: false,
            required: ["objectId"],
            properties: { objectId: ID_SCHEMA }
          },
          annotations: { readOnlyHint: false, untrustedContentHint: true },
          execute: execute("scene_focus_object", handleFocusObject)
        }, { signal: registrationController.signal })),
        Promise.resolve(document.modelContext.registerTool({
          name: "scene_preview_actions",
          title: "Preview scene actions",
          description: "Validate and visibly stage 1-12 reversible scene actions against an exact canonical revision. This never commits: a human must review and explicitly commit the opaque proposal in the Studio UI. Use scene_get_state and scene_find_objects first.",
          inputSchema: {
            type: "object",
            additionalProperties: false,
            required: ["expectedRevision", "title", "operations"],
            properties: {
              expectedRevision: { type: "integer", minimum: 0, description: "Exact canonical revision returned by scene_get_state." },
              title: { type: "string", minLength: 1, maxLength: 80, description: "Short human-facing proposal title." },
              rationale: { type: "string", maxLength: 400, description: "Why these actions help the user's stated goal." },
              operations: { type: "array", minItems: 1, maxItems: MAX_OPERATIONS, items: { oneOf: OPERATION_SCHEMAS } }
            }
          },
          annotations: { readOnlyHint: false, untrustedContentHint: true },
          execute: execute("scene_preview_actions", handlePreviewActions)
        }, { signal: registrationController.signal }))
      ];
    } catch (caught) {
      registrationController.abort();
      registrationState = "failed";
      var synchronousError = normalizedError(caught);
      emitStatus("error", synchronousError.message, { code: "REGISTRATION_FAILED" });
      return false;
    }

    Promise.all(registrations).then(function () {
      if (registrationController.signal.aborted) return;
      registrationState = "ready";
      emitStatus("ready", "Four WebMCP tools are ready for human-agent scene collaboration.", { tools: TOOL_NAMES.slice() });
    }).catch(function (caught) {
      registrationController.abort();
      registrationState = "failed";
      var error = normalizedError(caught);
      emitStatus("error", error.message, { code: "REGISTRATION_FAILED" });
    });
    return true;
  }

  function dispose() {
    if (retryTimer) window.clearTimeout(retryTimer);
    retryTimer = 0;
    if (!registrationController.signal.aborted) registrationController.abort();
    registrationState = "disposed";
  }

  function retryRegistration() {
    if (registrationState === "ready" || registrationState === "registering" || registrationState === "disposed") return;
    if (!document.modelContext || typeof document.modelContext.registerTool !== "function") return;
    if (registrationController.signal.aborted) registrationController = new AbortController();
    registrationAttempts = 0;
    registrationState = "idle";
    register();
  }

  window.__gosxStudioWebMCPAdapter = {
    events: Object.assign({}, EVENTS),
    register: register,
    dispose: dispose
  };
  window.addEventListener("pagehide", function (event) {
    if (!event.persisted) dispose();
  });
  window.addEventListener("pageshow", function (event) {
    if (event.persisted) retryRegistration();
  });
  window.addEventListener("focus", retryRegistration);
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") retryRegistration();
  });
  register();
})();
