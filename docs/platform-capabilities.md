# Desktop platform capability matrix

This matrix is evidence, not aspiration. Interactive host claims remain partial
until exercised on the named operating system.

| Capability | Windows amd64 | macOS | Linux |
|---|---|---|---|
| Application/server cross-build | available: PE32+ `app.exe` produced | buildable unsupported stub | buildable server |
| Native application window | implemented in GoSX WebView2; runtime verification pending | unsupported | unsupported |
| Native bridge | wired by the Studio Windows entrypoint; runtime verification pending | unsupported | unsupported |
| Native Open Project dialog | wired to `gosxDesktop.dialog.openFile`; runtime verification pending | unsupported | unsupported |
| Native Import Asset dialog | wired to `gosxDesktop.dialog.openFile`, typed format filters, and the shared inspected import form; runtime verification pending | unsupported | unsupported |
| Revision-safe project switch | available and host-neutral | available | available |
| Explicit save and journal recovery | available and host-neutral | available | available |
| Native menu | wired for Open Project, Import Asset, Save, Exit, Minimize, Maximize, Restore; runtime verification pending | unsupported | unsupported |
| Offline bundle | cross-built and inventoried | buildable | buildable |
| MSIX staging | manifest and payload staged | N/A | N/A |
| MSIX packing | blocked here: Windows SDK `MakeAppx` absent | N/A | N/A |
| Code signing | pending certificate and `SignTool` | pending | pending |
| Installed launch/recovery/update | unverified | unsupported | unsupported |
| Direct native GPU surface | unverified; packaged Scene3D canvas path is current | unsupported | unsupported |

## Continuous Windows evidence

`.github/workflows/windows.yml` runs the full browser-free evidence suite
(tests, vet, smoke, certify) natively on `windows-latest`, stages the offline
bundle and MSIX payload, best-effort packs an unsigned `.msix` when MakeAppx
is present, and uploads the certification JSON as the attached evidence for
this matrix. Released dependencies resolve from the public Go module proxy, so
the evidence job does not require a cross-repository credential.

## Windows verification commands

```powershell
go test ./...
gosx check app/page.gsx
gosx desktop --native-bridge --app-id m31labs.gosx3d-studio dev .
gosx build --prod --offline --msix --sign .
```

The live verification must open a project using the native picker, import a GLB
using the native asset picker and shared confirmation form, edit and journal a
command, terminate the process, recover the command, explicitly save, relaunch
cleanly, and run `studio-certify` without opening an external browser.

## Current cross-build evidence

- `GOOS=windows GOARCH=amd64 gosx build --prod --offline .` succeeds.
- Windows CI installs the official TinyGo 0.41.1 archive after verifying its
  published SHA-256 digest before production WASM and MSIX staging.
- The produced server entry is a PE32+ x86-64 executable and Windows defaults
  to the native desktop path unless `STUDIO_SERVER_ONLY=1` is set.
- MSIX staging emits a Windows Desktop/full-trust manifest with
  `broadFileSystemAccess`, but packing stops honestly when `MakeAppx` is absent.
