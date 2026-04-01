# GUI Backend Bootstrap

This directory contains the first implementation slice for the native Linux GUI project.

## Current scope

- Go module scaffold for the upcoming Wails app.
- Adapter to list remotes with `rclone listremotes`.
- Adapter to read and write `remote-paths.conf` using the same `REMOTE|PATH` format from the Bash script.
- Adapter for `systemctl --user` unit status and actions.
- Service layer to aggregate per-remote state and derive the global tray state:
  - `idle`
  - `syncing`
  - `error`
- Initial executable at `cmd/rclone-auto-gui` that prints state as JSON.
- App API wrapper in `internal/app` designed for direct Wails binding.
- Action-mode CLI for backend operations (`--action` + `--remote`).
- Poller component for periodic snapshot emission (tray/dashboard refresh).
- Binding facade with simple methods (`GetSnapshot`, `ExecuteAction`) ready for Wails registration.
- Wails desktop shell in [main.go](main.go) with periodic snapshot events.
- Minimal dashboard frontend in [frontend/dist/index.html](frontend/dist/index.html).
- Runtime dependency guard: app exits when required tools are missing.

## Why this comes first

The GUI and tray need a stable backend contract before rendering and interaction are added.
This module creates that contract while preserving compatibility with the existing script.

## Planned next steps

1. Add native tray indicator tied to global tray state.
2. Expand dashboard actions with autostart toggles and open local folder.
3. Add wizard flow for new remote creation.
4. Add notifications for sync state transitions.

## Local run

After installing Go:

```bash
cd gui
go test ./...
go run ./cmd/rclone-auto-gui
go run ./cmd/rclone-auto-gui --watch --interval 3s
go run ./cmd/rclone-auto-gui --action start-mount --remote drive-pessoal
go run ./cmd/rclone-auto-gui --action stop-sync --remote drive-pessoal

# Desktop shell (requires Wails CLI)
wails dev
wails build

# Optional explicit build tags for direct go build
WAILS_BUILD_TAGS=production,webkit2_41 go build -tags "$WAILS_BUILD_TAGS" -o build/bin/rclone-auto-gui .
```

The command emits JSON snapshots intended for GUI/tray consumers.

## Distribution files

- AUR: [../packaging/aur/PKGBUILD](../packaging/aur/PKGBUILD)
- DEB/RPM: [../packaging/nfpm/nfpm.yaml](../packaging/nfpm/nfpm.yaml)
- Installer script: [../scripts/install-gui.sh](../scripts/install-gui.sh)
- Package builder script: [../scripts/build-packages.sh](../scripts/build-packages.sh)
