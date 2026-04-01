#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
GUI_DIR="$ROOT_DIR/gui"
WAILS_BUILD_TAGS="${WAILS_BUILD_TAGS:-}"
RELEASE_REPO="${RELEASE_REPO:-weinne/dewdrop}"
RELEASE_TAG="${RELEASE_TAG:-latest}"
INSTALL_GUI_ALREADY_IN_TERMINAL="${INSTALL_GUI_ALREADY_IN_TERMINAL:-0}"

if [[ -t 1 ]]; then
  COLOR_BLUE='\033[1;34m'
  COLOR_GREEN='\033[1;32m'
  COLOR_YELLOW='\033[1;33m'
  COLOR_RED='\033[1;31m'
  COLOR_RESET='\033[0m'
else
  COLOR_BLUE=''
  COLOR_GREEN=''
  COLOR_YELLOW=''
  COLOR_RED=''
  COLOR_RESET=''
fi

set_root_dir() {
  ROOT_DIR="$1"
  GUI_DIR="$ROOT_DIR/gui"
}

log_step() {
  echo -e "${COLOR_BLUE}==>${COLOR_RESET} $*"
}

log_ok() {
  echo -e "${COLOR_GREEN}✓${COLOR_RESET} $*"
}

log_warn() {
  echo -e "${COLOR_YELLOW}!${COLOR_RESET} $*"
}

log_err() {
  echo -e "${COLOR_RED}Erro:${COLOR_RESET} $*" >&2
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log_err "comando '$1' não encontrado."
    exit 1
  fi
}

launch_in_terminal_if_needed() {
  if [[ "$INSTALL_GUI_ALREADY_IN_TERMINAL" == "1" ]]; then
    return 0
  fi

  if [[ -t 0 && -t 1 ]]; then
    return 0
  fi

  local script_path
  script_path="$(readlink -f "${BASH_SOURCE[0]}")"
  local -a term_cmds=(
    "x-terminal-emulator -e"
    "konsole -e"
    "gnome-terminal --"
    "xfce4-terminal -e"
    "xterm -e"
  )

  local cmd
  for cmd in "${term_cmds[@]}"; do
    local bin
    bin="${cmd%% *}"
    if command -v "$bin" >/dev/null 2>&1; then
      log_step "Sem TTY detectado; abrindo terminal automaticamente..."
      # shellcheck disable=SC2086
      eval "$cmd" env INSTALL_GUI_ALREADY_IN_TERMINAL=1 bash "$script_path" "$@"
      exit 0
    fi
  done

  log_warn "Sem TTY e nenhum terminal compatível encontrado. Continuando no contexto atual."
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
    rclone fuse3 systemd webkit2gtk-4.1 base-devel go
}

in_repo_layout() {
  [[ -d "$ROOT_DIR/gui" && -f "$ROOT_DIR/packaging/nfpm/rclone-auto-gui.desktop" ]]
}

resolve_release_tag() {
  if [[ "$RELEASE_TAG" != "latest" ]]; then
    echo "$RELEASE_TAG"
    return 0
  fi

  require_cmd curl
  local tag
  tag="$(curl -fsSL "https://api.github.com/repos/${RELEASE_REPO}/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

  if [[ -z "$tag" ]]; then
    log_err "não foi possível descobrir a última release de ${RELEASE_REPO}."
    exit 1
  fi

  echo "$tag"
}

download_release_asset() {
  local release_tag="$1"
  local file_name="$2"
  local out_file="$3"

  require_cmd curl
  local url="https://github.com/${RELEASE_REPO}/releases/download/${release_tag}/${file_name}"
  log_step "Baixando $file_name"
  curl -fL "$url" -o "$out_file"
}

install_from_release_package() {
  local manager="$1"
  local release_tag
  release_tag="$(resolve_release_tag)"
  local version="${release_tag#v}"

  local temp_dir
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "$temp_dir"' EXIT

  case "$manager" in
    apt)
      local deb_name="rclone-auto-gui_${version}_amd64.deb"
      download_release_asset "$release_tag" "$deb_name" "$temp_dir/$deb_name"
      sudo apt-get update
      sudo apt-get install -y "$temp_dir/$deb_name"
      ;;
    dnf)
      local rpm_name="rclone-auto-gui-${version}-1.x86_64.rpm"
      download_release_asset "$release_tag" "$rpm_name" "$temp_dir/$rpm_name"
      sudo dnf install -y "$temp_dir/$rpm_name"
      ;;
    *)
      log_err "instalação por pacote não suportada para o gerenciador '$manager'."
      exit 1
      ;;
  esac

  log_ok "Instalação concluída via pacote da release ${release_tag}. Rode: rclone-auto-gui"
}

build_from_release_source() {
  local release_tag
  release_tag="$(resolve_release_tag)"

  require_cmd curl
  require_cmd tar

  local temp_dir
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "$temp_dir"' EXIT

  local tarball_url="https://github.com/${RELEASE_REPO}/archive/refs/tags/${release_tag}.tar.gz"
  local tarball_path="$temp_dir/source.tar.gz"

  log_step "Baixando código-fonte da release ${release_tag}..."
  curl -fL "$tarball_url" -o "$tarball_path"
  tar -xzf "$tarball_path" -C "$temp_dir"

  local extracted
  extracted="$(find "$temp_dir" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  if [[ -z "$extracted" ]]; then
    log_err "não foi possível extrair o código-fonte da release ${release_tag}."
    exit 1
  fi

  set_root_dir "$extracted"

  install_system_deps
  install_go_if_missing

  log_step "Compilando aplicação GUI a partir da release ${release_tag}..."
  build_gui_binary

  log_step "Instalando binário em /usr/local/bin..."
  sudo install -Dm755 "$GUI_DIR/rclone-auto-gui" /usr/local/bin/rclone-auto-gui

  log_step "Instalando atalho desktop..."
  sudo install -Dm644 "$ROOT_DIR/packaging/nfpm/rclone-auto-gui.desktop" /usr/share/applications/rclone-auto-gui.desktop

  log_ok "Instalação concluída por build da release ${release_tag}. Rode: rclone-auto-gui"
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

  log_step "Go não encontrado. Instalando Go local em ~/.local/go..."
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
    log_err "gerenciador de pacotes não suportado automaticamente."
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
    log_step "Tentando compilar com tags Go: $go_tags"
    if go build -tags "$go_tags" -o rclone-auto-gui .; then
      log_ok "Build concluído com tags: $go_tags"
      return 0
    fi
    last_candidate="$go_tags"
  done

  log_err "não foi possível compilar com nenhuma tag Wails suportada."
  echo "Tentativas: ${tag_candidates[*]}" >&2
  echo "Dica: exporte WAILS_BUILD_TAGS manualmente e rode de novo." >&2
  echo "Exemplo: WAILS_BUILD_TAGS=production,webkit2_41 ./scripts/install.sh" >&2
  echo "Ultima tentativa: $last_candidate" >&2
  return 1
}

print_usage() {
  cat <<'EOF'
Uso:
  ./scripts/install.sh

Comportamento:
  - Dentro do repositório: instala dependências, compila e instala a GUI.
  - Fora do repositório:
    - apt/dnf: baixa pacote da release e instala.
    - pacman: baixa código-fonte da release, compila e instala.

Variáveis opcionais:
  RELEASE_REPO=weinne/dewdrop   # owner/repo da release
  RELEASE_TAG=latest            # ex: v0.1.0
  WAILS_BUILD_TAGS=production,webkit2_41
EOF
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    print_usage
    return 0
  fi

  launch_in_terminal_if_needed "$@"
  require_cmd bash

  if in_repo_layout; then
    install_system_deps
    install_go_if_missing

    if ! command -v go >/dev/null 2>&1; then
      log_err "Go não disponível após tentativa de instalação."
      exit 1
    fi

    log_step "Compilando aplicação GUI..."
    build_gui_binary

    log_step "Instalando binário em /usr/local/bin..."
    sudo install -Dm755 "$GUI_DIR/rclone-auto-gui" /usr/local/bin/rclone-auto-gui

    log_step "Instalando atalho desktop..."
    sudo install -Dm644 "$ROOT_DIR/packaging/nfpm/rclone-auto-gui.desktop" /usr/share/applications/rclone-auto-gui.desktop

    log_ok "Instalação concluída. Rode: rclone-auto-gui"
    return 0
  fi

  if command -v apt-get >/dev/null 2>&1; then
    install_from_release_package "apt"
    return 0
  fi

  if command -v dnf >/dev/null 2>&1; then
    install_from_release_package "dnf"
    return 0
  fi

  if command -v pacman >/dev/null 2>&1; then
    build_from_release_source
    return 0
  fi

  log_err "não foi possível determinar um método de instalação para esta distro."
  exit 1
}

main "$@"
