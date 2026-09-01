(function () {
  "use strict";

  if (typeof document === "undefined" || window.__gosxStudioProjectBridge) return;
  window.__gosxStudioProjectBridge = true;

  function request(url, options) {
    if (window.__gosx && typeof window.__gosx.request === "function") {
      return window.__gosx.request(url, options);
    }
    return fetch(url, options);
  }

  function csrfToken() {
    var field = document.querySelector('input[name="csrf_token"]');
    return field && typeof field.value === "string" ? field.value : "";
  }

  document.addEventListener("click", function (event) {
    var button = event.target && event.target.closest ? event.target.closest("#studio-open-project") : null;
    if (!button) return;
    event.preventDefault();
    if (!window.gosxDesktop || !window.gosxDesktop.dialog) {
      button.setAttribute("title", "Native project opening requires the GoSX Desktop host");
      return;
    }
    var dirty = button.getAttribute("data-dirty") === "true";
    var discard = !dirty || window.confirm("Discard unsaved changes and open another project?");
    if (!discard) return;
    window.gosxDesktop.dialog.openFile({
      title: "Open GoSX 3D Studio Project",
      filters: [{ name: "GoSX SceneDoc", pattern: "scene.scene3d" }],
      defaultExt: "scene3d",
      allowMultiple: false
    }).then(function (paths) {
      if (!Array.isArray(paths) || !paths.length) return null;
      var headers = {
        "Accept": "application/json",
        "Content-Type": "application/json",
        "X-GoSX-Desktop-Intent": "native-dialog"
      };
      var token = csrfToken();
      if (token) headers["X-CSRF-Token"] = token;
      return request("/api/studio/project/open-from-desktop", {
        method: "POST",
        headers: headers,
        body: JSON.stringify({
          path: String(paths[0]),
          expectedRevision: Number(button.getAttribute("data-revision") || 0),
          discardUnsaved: dirty
        })
      });
    }).then(function (response) {
      if (response && response.ok) window.location.assign("/");
    }).catch(function (error) {
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("open project", error, {
          scope: "studio", type: "project", fallback: "current-project"
        });
      }
    });
  });

  document.addEventListener("click", function (event) {
    var button = event.target && event.target.closest ? event.target.closest("#studio-choose-asset") : null;
    if (!button) return;
    event.preventDefault();
    var input = document.getElementById("studio-asset-path");
    var status = document.getElementById("studio-asset-dialog-status");
    if (!input) return;
    if (!window.gosxDesktop || !window.gosxDesktop.dialog) {
      if (status) status.textContent = "Native chooser unavailable; enter a trusted local path.";
      input.focus();
      return;
    }
    if (status) status.textContent = "Opening native asset chooser…";
    window.gosxDesktop.dialog.openFile({
      title: "Import GoSX 3D Studio Asset",
      filters: [
        { name: "Studio assets", pattern: "*.glb;*.gltf;*.png;*.jpg;*.jpeg;*.wav;*.ogg;*.mp4;*.obj" },
        { name: "All files", pattern: "*.*" }
      ],
      allowMultiple: false
    }).then(function (paths) {
      if (!Array.isArray(paths) || !paths.length) {
        if (status) status.textContent = "Asset selection cancelled.";
        return;
      }
      input.value = String(paths[0]);
      input.dispatchEvent(new Event("input", { bubbles: true }));
      if (status) status.textContent = "Selected " + String(paths[0]).split(/[\\/]/).pop() + "; confirm import.";
      input.focus();
    }).catch(function (error) {
      if (status) status.textContent = "Asset chooser failed; enter a trusted local path.";
      input.focus();
      if (window.__gosx && typeof window.__gosx.reportFailure === "function") {
        window.__gosx.reportFailure("choose asset", error, {
          scope: "studio", type: "asset", fallback: "manual-path"
        });
      }
    });
  });
})();
