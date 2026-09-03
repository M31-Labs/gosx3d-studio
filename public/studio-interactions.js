(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioInteractions) return;
  window.__gosxStudioInteractions = true;

  var hierarchyQuery = "";
  var publicDemo = document.querySelector('.studio-shell[data-studio-demo="true"]') !== null;
  var projectPanelCollapsed = publicDemo || (typeof window.matchMedia === "function" &&
    window.matchMedia("(max-width: 88rem)").matches);
  var sceneRuntimeObserver = null;

  function closest(target, selector) {
    if (!target || typeof target.closest !== "function") return null;
    return target.closest(selector);
  }

  function normalized(value) {
    return String(value || "").trim().toLowerCase();
  }

  function filterHierarchy(query) {
    var tree = document.getElementById("studio-hierarchy-tree");
    if (!tree) return;

    var normalizedQuery = normalized(query);
    var rows = tree.querySelectorAll("[data-hierarchy-row]");
    var visible = 0;

    rows.forEach(function (row) {
      var searchable = [
        row.getAttribute("data-entity-name"),
        row.getAttribute("data-hierarchy-id"),
        row.getAttribute("data-entity-type")
      ].map(normalized).join("\n");
      var matches = normalizedQuery === "" || searchable.indexOf(normalizedQuery) !== -1;
      row.hidden = !matches;
      if (matches) visible += 1;
    });

    var empty = tree.querySelector("[data-hierarchy-empty]");
    if (empty) {
      empty.hidden = visible !== 0;
      empty.textContent = normalizedQuery === ""
        ? "No scene entities are available."
        : "No scene entities match this search.";
    }

    var count = document.querySelector("[data-hierarchy-filter-count]");
    if (count) {
      count.textContent = normalizedQuery === ""
        ? String(rows.length)
        : visible + " / " + rows.length;
    }

    var status = document.getElementById("studio-hierarchy-search-status");
    if (status) {
      status.textContent = visible === 0
        ? (normalizedQuery === "" ? "No scene entities are available." : "No scene entities match this search.")
        : visible + (visible === 1 ? " scene entity shown" : " scene entities shown");
    }
    syncHierarchyRoving();
  }

  function visibleHierarchyLinks() {
    return Array.prototype.filter.call(
      document.querySelectorAll("#studio-hierarchy-tree [role='treeitem']"),
      function (link) {
        var row = closest(link, "[data-hierarchy-row]");
        return row && !row.hidden;
      }
    );
  }

  function syncHierarchyRoving(preferred) {
    var links = visibleHierarchyLinks();
    if (!links.length) return;
    var active = preferred && links.indexOf(preferred) !== -1 ? preferred : null;
    if (!active) {
      active = links.find(function (link) { return link.getAttribute("tabindex") === "0"; }) ||
        links.find(function (link) { return link.getAttribute("aria-selected") === "true"; }) || links[0];
    }
    links.forEach(function (link) { link.setAttribute("tabindex", link === active ? "0" : "-1"); });
  }

  function revealHierarchySelection() {
    var selected = document.querySelector("#studio-hierarchy-tree [aria-selected='true']");
    if (selected && typeof selected.scrollIntoView === "function") {
      selected.scrollIntoView({ block: "nearest" });
    }
  }

  function syncProjectPanel() {
    var shell = document.querySelector(".studio-shell");
    var button = document.querySelector("[data-project-panel-toggle]");
    var panel = document.getElementById("project-panel");
    if (!shell || !button || !panel) return;

    shell.classList.toggle("project-panel-collapsed", projectPanelCollapsed);
    button.setAttribute("aria-expanded", projectPanelCollapsed ? "false" : "true");
    button.setAttribute("title", projectPanelCollapsed ? "Show Project / Assets panel" : "Hide Project / Assets panel");
    panel.setAttribute("aria-hidden", projectPanelCollapsed ? "true" : "false");
  }

  function syncSceneRuntimeStatus() {
    var status = document.querySelector("[data-scene-runtime-status]");
    var mount = document.querySelector('[data-gosx-engine="GoSXScene3D"]');
    if (!status) return;
    if (!mount) {
      status.textContent = "Scene3D fallback";
      status.setAttribute("title", "The interactive Scene3D mount is unavailable; the authored fallback remains visible.");
      return;
    }
    var ready = mount.getAttribute("data-gosx-scene3d-ready") === "true" && !!mount.querySelector("canvas");
    if (ready) {
      var backend = String(mount.getAttribute("data-gosx-scene3d-backend") || "runtime").toUpperCase();
      status.textContent = "Scene3D · " + backend;
      status.setAttribute("title", "Interactive Scene3D viewport mounted successfully.");
      var backendReadout = document.querySelector("[data-scene-runtime-backend]");
      if (backendReadout) backendReadout.textContent = backend;
      return;
    }
    var failed = mount.getAttribute("data-gosx-runtime-state") === "error" ||
      !!mount.getAttribute("data-gosx-fallback-active");
    status.textContent = failed ? "Scene3D fallback" : "Scene3D initializing";
    status.setAttribute(
      "title",
      failed
        ? "The interactive runtime could not mount; the authored fallback remains visible."
        : "The interactive Scene3D viewport is initializing."
    );
  }

  function observeSceneRuntime() {
    if (sceneRuntimeObserver) sceneRuntimeObserver.disconnect();
    var mount = document.querySelector('[data-gosx-engine="GoSXScene3D"]');
    if (!mount || typeof MutationObserver !== "function") return;
    sceneRuntimeObserver = new MutationObserver(syncSceneRuntimeStatus);
    sceneRuntimeObserver.observe(mount, { attributes: true, childList: true });
  }

  function syncPage() {
    var search = document.querySelector("[data-hierarchy-filter]");
    if (search) search.value = hierarchyQuery;
    filterHierarchy(hierarchyQuery);
    revealHierarchySelection();
    syncProjectPanel();
    syncSceneRuntimeStatus();
    observeSceneRuntime();
  }

  document.addEventListener("input", function (event) {
    var search = closest(event.target, "[data-hierarchy-filter]");
    if (!search) return;
    hierarchyQuery = String(search.value || "");
    filterHierarchy(hierarchyQuery);
  });

  document.addEventListener("click", function (event) {
    var projectToggle = closest(event.target, "[data-project-panel-toggle]");
    if (projectToggle) {
      projectPanelCollapsed = !projectPanelCollapsed;
      syncProjectPanel();
    }
  });

  document.addEventListener("keydown", function (event) {
    var current = closest(event.target, "#studio-hierarchy-tree [role='treeitem']");
    if (!current) return;
    var links = visibleHierarchyLinks();
    var index = links.indexOf(current);
    if (index < 0) return;
    var next = null;
    if (event.key === "ArrowDown") next = links[Math.min(index + 1, links.length - 1)];
    else if (event.key === "ArrowUp") next = links[Math.max(index - 1, 0)];
    else if (event.key === "Home") next = links[0];
    else if (event.key === "End") next = links[links.length - 1];
    else if (event.key === " " || event.key === "Spacebar") {
      event.preventDefault();
      current.click();
      return;
    }
    else return;
    event.preventDefault();
    syncHierarchyRoving(next);
    next.focus();
    next.scrollIntoView({ block: "nearest" });
  });

  document.addEventListener("gosx:navigate", function () {
    window.setTimeout(syncPage, 0);
  });
  document.addEventListener("gosx:ready", syncSceneRuntimeStatus);
  document.addEventListener("gosx:error", syncSceneRuntimeStatus);

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", syncPage);
  } else {
    syncPage();
  }
})();
