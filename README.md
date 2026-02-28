# ☁️ RClone Auto

> **The definitive Rclone manager for Linux.**  
> Manage cloud mounts and syncs with a modern, friendly TUI.

![Bash](https://img.shields.io/badge/Language-Bash-4EAA25?style=flat-square)
![Interface](https://img.shields.io/badge/Interface-Gum_(Charm)-ff69b4?style=flat-square)
![Platform](https://img.shields.io/badge/Platform-Linux-blue?style=flat-square)
![License](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)

**RClone Auto** is an advanced Bash script that automates configuration, mounting and synchronization of **Rclone** remotes. It hides CLI complexity behind a rich visual experience (menus, filters, colors) and ensures persistence via `systemd --user`.

---

## ✨ Main Features

- **🎨 Modern TUI (Gum)**: Navigable menus, search filters, loading spinners and clear confirmations.
- **🚀 Smart Auto‑Install**: Automatically detects and downloads dependencies (`rclone` and `gum`) if they are not installed.
- **📦 Portable / Offline Friendly**: Supports bundled binaries so you can run it on machines without prior installation.
- **⚡ Two operation modes**:
  - **Mount**: Turns the cloud into a virtual drive (on‑demand access, minimal local space).
  - **Sync**: Creates a real offline copy with scheduled bi‑directional sync (every 15 minutes).
- **🧠 Contextual Management Menu**: Manage connections intuitively: pick a connection → choose actions (Open Folder, Stop, Rename, Delete).
- **🏷️ Naming conventions**: Encourages organized names (e.g. `drive-work`, `s3-backup`) with a dynamic list of providers.
- **🛠️ System Tools**: Automatically creates app launchers and desktop shortcuts, fixes folder icons and updates binaries.

---

## 📦 Installation

You don’t need to pre‑install `rclone` or `gum`. The script will bootstrap what is missing inside your home directory.

### Quick Method (Online)

```bash
# 1. Download the script
wget https://raw.githubusercontent.com/SEU_USUARIO/SEU_REPO/main/rclone-auto.sh

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

> **Note**: By default, `rclone` is downloaded for Linux `amd64`. For other architectures you may need to adjust the download URL and logic in the script.

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

3. **🛠️ Tools**
   - **Create Desktop Shortcuts** for all active mounts (one `.desktop` file per remote).
   - **Fix Folder Icons** by regenerating a `.directory` file for the main cloud folder.
   - **Update Rclone** by downloading the latest Linux `amd64` build into `~/.local/bin`.
   - **Reinstall Script** into `~/.local/bin/rclone-auto` and refresh the app launcher entry.

4. **🔧 Advanced Configuration**
   - Opens the native `rclone config` so you can edit remotes manually if needed.

5. **🚪 Exit**
   - Clears the screen and quits.

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

If you created services with an older version, run the fixer script once:

```bash
chmod +x fix-existing-services.sh
./fix-existing-services.sh
```

- **Core script behavior**
  - Ensures it is running in a real terminal (`ensure_terminal`).
  - Bootstraps `gum` and `rclone` if binaries are missing.
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

---

## 📋 Requirements

- **Operating System**: Linux (Ubuntu, Debian, Fedora, Arch, etc.)
- **System tools**: `bash`, `curl`, `unzip`, `systemd --user` enabled.
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
