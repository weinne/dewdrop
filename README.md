# ☁️ RClone Auto

> **The definitive Rclone manager for Linux.**  
> Manage cloud mounts and syncs with a modern, friendly TUI.

![Bash](https://img.shields.io/badge/Language-Bash-4EAA25?style=flat-square)
![Interface](https://img.shields.io/badge/Interface-Gum_(Charm)-ff69b4?style=flat-square)
![Platform](https://img.shields.io/badge/Platform-Linux-blue?style=flat-square)
![License](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)

**RClone Auto** is an advanced Bash script that automates configuration, mounting and synchronization of **Rclone** remotes. It hides CLI complexity behind a rich visual experience (menus, filters, colors) and ensures persistence via `systemd --user`.

## Native GUI Work In Progress

The native Linux GUI implementation has started in [gui/README.md](gui/README.md).

Current implementation status:

- Backend module in Go for remote discovery and unit status orchestration.
- JSON snapshot output for future Wails dashboard and tray integration.
- Compatibility maintained with existing config and unit naming conventions.
- Wails desktop shell with live snapshot events and a first dashboard view.

## Packaging and Installation

The GUI app can be distributed as:

- `.deb` and `.rpm` using `nfpm` config in [packaging/nfpm/nfpm.yaml](packaging/nfpm/nfpm.yaml)
- `AUR` package with [packaging/aur/PKGBUILD](packaging/aur/PKGBUILD)
- Cross-distro installer script: [scripts/install-gui.sh](scripts/install-gui.sh)

### Runtime dependency policy

At startup, the GUI checks required dependencies (`rclone`, `systemd --user`, `fuse3`).
If requirements are missing, the app exits early with a clear error.

### Build packages (.deb / .rpm)

Use the release helper script:

```bash
./scripts/build-packages.sh
```

If needed, override build tags/version:

```bash
WAILS_BUILD_TAGS=production,webkit2_40 VERSION=0.1.0 ./scripts/build-packages.sh
```

### Install script

```bash
./scripts/install-gui.sh
```

The script installs dependencies, builds the GUI and installs it under `/usr/local/bin/rclone-auto-gui`.
It automatically tries compatible Wails Linux build tags (`production,webkit2_41`, `production,webkit2_40`, `production,webkit2_36`) and uses the first one that works.
If needed, override build tags with `WAILS_BUILD_TAGS`, for example:

```bash
WAILS_BUILD_TAGS=production,webkit2_40 ./scripts/install-gui.sh
```

---

## ✨ Main Features

- **🎨 Modern TUI (Gum)**: Navigable menus, search filters, loading spinners and clear confirmations.
- **🚀 Smart Auto‑Install**: Automatically prepares dependencies (`rclone` and `gum`) with source selection support.
- **📦 Package‑First Rclone (Auto Mode)**: On `apt`/RPM-based systems (`dnf`, `yum`, `zypper`), `rclone` is installed from system packages first.
- **🧩 Binary Fallback**: If package installation is unavailable (or on non-`apt`/RPM systems), the script uses a local binary in `~/.local/bin`.
- **📦 Portable / Offline Friendly**: Supports bundled binaries so you can run it on machines without prior installation.
- **⚡ Two operation modes**:
  - **Mount**: Turns the cloud into a virtual drive (on‑demand access, minimal local space).
  - **Sync**: Creates a real offline copy with scheduled bi‑directional sync (every 15 minutes).
- **📁 Custom Local Directory**: Choose a custom local folder per remote, with a simple browser starting at `/home` and detected mounted disks.
- **🧠 Contextual Management Menu**: Manage connections intuitively: pick a connection → choose actions (Open Folder, Stop, Rename, Delete).
- **🏷️ Naming conventions**: Encourages organized names (e.g. `drive-work`, `s3-backup`) with a dynamic list of providers.
- **🛠️ System Tools**: Automatically creates app launchers and desktop shortcuts, fixes folder icons and updates binaries.

---

## 📦 Installation

You don’t need to pre‑install `rclone` or `gum`. The script bootstraps what is missing and can choose between package-managed or local-binary `rclone`.

### Quick Method (Online)

```bash
# 1. Download the script
wget https://raw.githubusercontent.com/weinne/rclone-auto/main/rclone-auto.sh

# 2. Make it executable
chmod +x rclone-auto.sh

# 3. Run
./rclone-auto.sh
```

### Portable / Bundled Use (Offline‑friendly)

To create a bundle that works on machines with limited internet or no admin rights:

1. Download the `gum` binary for the target architecture.
2. Place it in the same directory as the script (or in a small `bin/` next to it).
3. The script will automatically detect the local binary and skip the download.

> **Note**: Binary installs now detect architecture (`amd64` / `arm64`) automatically.

---

## 🎮 How It Works

Just run the script. If you are on a graphical desktop, it will try to open itself in your preferred terminal emulator.

```bash
./rclone-auto.sh      # from the cloned repo

# After installation, you can usually just call:
rclone-auto
```

### Main Menu

1. **🚀 New Connection**
   - Shows a curated list of popular providers (Google Drive, OneDrive, Dropbox, S3, WebDAV, etc.) and an **ALL** option with every backend supported by your `rclone`.
   - Guides you through browser‑based authentication using `rclone config create`.
   - Asks whether you want to use the remote as:
     - **MOUNT** (virtual drive) or
     - **SYNC** (offline backup using `rclone bisync` with a 15‑minute timer).
   - Creates and starts the corresponding `systemd --user` units automatically.

2. **📂 Manage Connections**
   - Lists all existing remotes with live status:
     - `🟢` Mounted
     - `🔵` Sync (timer active)
     - `⚪` Inactive
   - Selecting a connection shows context‑specific actions:
     - **Open Folder** (opens `~/Nuvem/<name>` with `xdg-open`)
     - **Disconnect** (stops and disables `mount` / `sync` units, cleans systemd files)
     - **Activate Mount / Sync**
     - **Rename** (renames the `rclone` config section and the local folder)
     - **Delete** (stops everything and removes the remote from `rclone config`)
     - **Change Local Directory** (updates service paths for mount/sync)

### Custom Local Path Behavior

- During new connection setup, you can choose:
  - default path: `~/Nuvem/<remote-name>`
  - custom path: selected via interactive folder picker
- The folder picker starts from:
  - your home path
  - `/home`
  - mounted disks under `/mnt`, `/media`, and `/run/media/$USER`
- When changing directory for an existing **sync** connection, the app automatically:
  - stops sync service/timer
  - moves local files to the new folder
  - rewrites units and restores previous active/enabled state

3. **🛠️ Tools**
   - **Create Desktop Shortcuts** for all active mounts (one `.desktop` file per remote).
   - **Fix Folder Icons** by regenerating a `.directory` file for the main cloud folder.
  - **Test Configuration** to run a full health check (dependencies, units, remotes, enabled/inactive services).
  - **Fix Existing Services** with an in-app shortcut that updates legacy `systemd --user` units.
  - **Rclone Source (package/binary)** to switch between package-managed `rclone` and local binary.
  - **Update Rclone** honoring your selected source:
    - package mode: updates via package manager.
    - binary mode: updates local binary in `~/.local/bin`.
   - **Reinstall Script** into `~/.local/bin/rclone-auto` and refresh the app launcher entry.

4. **🔧 Advanced Configuration**
   - Opens the native `rclone config` so you can edit remotes manually if needed.

5. **🚪 Exit**
   - Clears the screen and quits.

### CLI Diagnostics (No TUI)

You can run a full diagnostic directly in terminal without opening the interactive menu:

```bash
./rclone-auto.sh --test-all
```

- Exit code `0`: no failures found
- Exit code `1`: one or more problems detected

---

## 🔧 Technical Architecture

- **Persistence**  
  Uses per‑user `systemd` units:
  - `rclone-mount-<name>.service` for mounts (`rclone mount` with FUSE).
  - `rclone-sync-<name>.service` + `rclone-sync-<name>.timer` for periodic `rclone bisync`.
  - Runs entirely under `systemctl --user` – no `sudo` required for normal operation.

### Boot / Session behavior

- Mount auto-start is now tied to the **graphical session** (via `graphical-session.target`).
- Mounts and sync runs wait for **internet** before starting (via an `ExecStartPre` helper).
- Internet check is **fail-fast + retry** (short timeout), so it does not block boot/login for long when network is still coming up.
- A **quick automatic test** runs after each new service activation (mount/sync) to validate runtime health.

If you created services with an older version, run the fixer script once:

```bash
chmod +x fix-existing-services.sh
./fix-existing-services.sh
```

Or run it from the app menu:

- `Tools` → `Fix Existing Services`

- **Core script behavior**
  - Ensures it is running in a real terminal (`ensure_terminal`).
  - Bootstraps `gum` and `rclone` using selectable source logic (`package` or `binary`).
  - Persists rclone source preference in `~/.config/rclone-auto/rclone-source.conf`.
  - Installs itself into `~/.local/bin/rclone-auto` and creates a `.desktop` launcher.
  - Centralizes all cloud folders under:
    - `~/Nuvem/<remote-name>`

- **Directories**
  - Binaries: `~/.local/bin/`
  - Systemd units (user): `~/.config/systemd/user/`
  - Rclone config: `~/.config/rclone/`
  - Mounts / sync roots: `~/Nuvem/`
  - Desktop entry (launcher): `~/.local/share/applications/rclone-auto.desktop`

- **Icons / Desktop integration**
  - Writes a `.directory` file inside `~/Nuvem` so file managers (Dolphin, Nautilus, etc.) show a cloud‑style icon.
  - Creates `.desktop` shortcuts on `~/Desktop` for direct access to mounted folders.
  - `Open Cloud Folder Options` resolver (`--open-path`) now works with both default and custom local paths configured per remote.

---

## 📋 Requirements

- **Operating System**: Linux (Ubuntu, Debian, Fedora, Arch, etc.)
- **System tools**: `bash`, `curl`, `unzip`, `systemd --user` enabled.
- **Optional admin rights**: Needed only when choosing package-managed `rclone` (uses `sudo`).
- **FUSE**: `fuse3` / `fusermount3` must be available for mounts to work.
- **Internet access**: Required on first run to download `rclone` and `gum`, unless you bundle binaries locally.

---

## 🤝 Contributing

Pull requests are welcome!

1. Fork this repository.
2. Create your feature branch (`git checkout -b feature/MyFeature`).
3. Commit your changes (`git commit -m 'Add MyFeature'`).
4. Push to the branch (`git push origin feature/MyFeature`).
5. Open a Pull Request.

---

## 👏 Credits & Dependencies

This project is an automation wrapper built on top of amazing open‑source tools. All credit goes to the original authors of the underlying technologies:

- **[Gum](https://github.com/charmbracelet/gum)** – by [Charm](https://charm.sh/).  
  Used to build the modern, interactive TUI. Distributed under the MIT license.

- **[Rclone](https://rclone.org/)** – by Nick Craig‑Wood and contributors.  
  The robust engine responsible for all cloud connections and synchronization. Distributed under the MIT license.

> **Distribution note**  
> For a “batteries‑included” experience, this project may download or bundle binaries of the tools above. All intellectual property rights belong to their respective authors.

---

## 📜 License

This project (the `rclone-auto` script) is released under the **MIT License**.

You are free to use, modify and redistribute it, as long as you keep the original credits.
