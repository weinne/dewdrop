#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GUI_DIR="$ROOT_DIR/gui"
WAILS_BUILD_TAGS="${WAILS_BUILD_TAGS:-}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Erro: comando '$1' não encontrado." >&2
    exit 1
  fi
}

install_deps_apt() {
  sudo apt-get update
  sudo apt-get install -y \
    rclone fuse3 systemd \
    libwebkit2gtk-4.1-0 libwebkit2gtk-4.1-dev \
    libgtk-3-dev \
    gcc pkg-config
}

install_deps_dnf() {
  sudo dnf install -y \
    rclone fuse3 systemd \
    webkit2gtk4.1-devel gtk3-devel \
    gcc pkgconfig
}

install_deps_pacman() {
  sudo pacman -Syu --noconfirm
  sudo pacman -S --needed --noconfirm \
    rclone fuse3 systemd webkit2gtk-4.1 base-devel
}

detect_webkit_tag() {
  if command -v pkg-config >/dev/null 2>&1; then
    if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
      echo "webkit2_41"
      return 0
    fi
    if pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
      echo "webkit2_40"
      return 0
    fi
  fi
  echo ""
}

normalize_go_tags() {
  local tags="$1"
  if [[ -z "$tags" ]]; then
    echo "production"
    return 0
  fi

  # Wails requires one of: dev | production | bindings.
  if [[ ",$tags," != *",production,"* && ",$tags," != *",dev,"* && ",$tags," != *",bindings,"* ]]; then
    tags="production,${tags}"
  fi

  echo "$tags"
}

install_go_if_missing() {
  if command -v go >/dev/null 2>&1; then
    return
  fi

  echo "Go não encontrado. Instalando Go local em ~/.local/go..."
  require_cmd curl
  require_cmd tar

  local go_version
  go_version="$(curl -fsSL https://go.dev/VERSION?m=text | head -n1)"

  mkdir -p "$HOME/.local" "$HOME/.local/bin"
  rm -rf "$HOME/.local/go"
  curl -fsSL "https://go.dev/dl/${go_version}.linux-amd64.tar.gz" -o "/tmp/${go_version}.linux-amd64.tar.gz"
  tar -C "$HOME/.local" -xzf "/tmp/${go_version}.linux-amd64.tar.gz"
  export PATH="$HOME/.local/go/bin:$HOME/.local/bin:$PATH"
}

install_system_deps() {
  if command -v apt-get >/dev/null 2>&1; then
    install_deps_apt
  elif command -v dnf >/dev/null 2>&1; then
    install_deps_dnf
  elif command -v pacman >/dev/null 2>&1; then
    install_deps_pacman
  else
    echo "Erro: gerenciador de pacotes não suportado automaticamente." >&2
    exit 1
  fi
}

build_gui_binary() {
  cd "$GUI_DIR"
  export CGO_ENABLED=1

  local detected_webkit_tag
  detected_webkit_tag="$(detect_webkit_tag)"

  local -a tag_candidates=()
  if [[ -n "$WAILS_BUILD_TAGS" ]]; then
    # Allow full tag list (eg: "production,webkit2_41") or only the webkit tag.
    tag_candidates+=("$WAILS_BUILD_TAGS")
  fi
  if [[ -n "$detected_webkit_tag" ]]; then
    tag_candidates+=("$detected_webkit_tag")
  fi
  # Wails v2.12 supports: webkit2_41 (webkit2gtk-4.1), webkit2_40 (webkit2gtk-4.0), webkit2_36 (legacy)
  tag_candidates+=("webkit2_41" "webkit2_40" "webkit2_36")

  local candidate
  local last_candidate=""
  for candidate in "${tag_candidates[@]}"; do
    local go_tags
    go_tags="$(normalize_go_tags "$candidate")"
    echo "Tentando compilar com tags Go: $go_tags"
    if go build -tags "$go_tags" -o rclone-auto-gui .; then
      echo "Build concluido com tags: $go_tags"
      return 0
    fi
    last_candidate="$go_tags"
  done

  echo "Erro: não foi possível compilar com nenhuma tag Wails suportada." >&2
  echo "Tentativas: ${tag_candidates[*]}" >&2
  echo "Dica: exporte WAILS_BUILD_TAGS manualmente e rode de novo." >&2
  echo "Exemplo: WAILS_BUILD_TAGS=production,webkit2_41 ./scripts/install-gui.sh" >&2
  echo "Ultima tentativa: $last_candidate" >&2
  return 1
}

main() {
  require_cmd bash
  install_system_deps
  install_go_if_missing

  if ! command -v go >/dev/null 2>&1; then
    echo "Erro: Go não disponível após tentativa de instalação." >&2
    exit 1
  fi

  echo "Compilando aplicação GUI..."
  build_gui_binary

  echo "Instalando binário em /usr/local/bin..."
  sudo install -Dm755 rclone-auto-gui /usr/local/bin/rclone-auto-gui

  echo "Instalando atalho desktop..."
  sudo install -Dm644 "$ROOT_DIR/packaging/nfpm/rclone-auto-gui.desktop" /usr/share/applications/rclone-auto-gui.desktop

  echo "Instalação concluída. Rode: rclone-auto-gui"
}

main "$@"
