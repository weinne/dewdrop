#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GUI_DIR="$ROOT_DIR/gui"
DIST_DIR="$ROOT_DIR/dist"

VERSION="${VERSION:-0.1.0}"
ARCH="${ARCH:-amd64}"
WAILS_BUILD_TAGS="${WAILS_BUILD_TAGS:-}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Erro: comando '$1' não encontrado." >&2
    exit 1
  fi
}

ensure_nfpm() {
  if command -v nfpm >/dev/null 2>&1; then
    return
  fi

  if ! command -v go >/dev/null 2>&1; then
    echo "Erro: nfpm não encontrado e Go não está disponível para instalar nfpm." >&2
    exit 1
  fi

  echo "nfpm não encontrado. Instalando via go install..."
  go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
  export PATH="$HOME/go/bin:$PATH"

  if ! command -v nfpm >/dev/null 2>&1; then
    echo "Erro: não foi possível instalar nfpm automaticamente." >&2
    exit 1
  fi
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

  if [[ ",$tags," != *",production,"* && ",$tags," != *",dev,"* && ",$tags," != *",bindings,"* ]]; then
    tags="production,${tags}"
  fi

  echo "$tags"
}

build_gui_binary() {
  (
    cd "$GUI_DIR"
    export CGO_ENABLED=1

    local detected_webkit_tag
    detected_webkit_tag="$(detect_webkit_tag)"

    local -a webkit_candidates=()
    if [[ -n "$WAILS_BUILD_TAGS" ]]; then
      # Allow passing the full tag list (eg: "production,webkit2_41") or only the webkit tag.
      webkit_candidates+=("$WAILS_BUILD_TAGS")
    fi
    if [[ -n "$detected_webkit_tag" ]]; then
      webkit_candidates+=("$detected_webkit_tag")
    fi
    webkit_candidates+=("webkit2_41" "webkit2_40" "webkit2_36")

    local candidate
    for candidate in "${webkit_candidates[@]}"; do
      local go_tags
      go_tags="$(normalize_go_tags "$candidate")"
      echo "Tentando compilar com tags Go: $go_tags"
      if go build -tags "$go_tags" -o build/bin/rclone-auto-gui .; then
        echo "Build concluido com tags: $go_tags"
        return 0
      fi
    done

    echo "Erro: nao foi possivel compilar o binario da GUI com tags suportadas." >&2
    echo "Dica: defina WAILS_BUILD_TAGS manualmente." >&2
    echo "Exemplo: WAILS_BUILD_TAGS=production,webkit2_41 ./scripts/build-packages.sh" >&2
    return 1
  )
}

main() {
  require_cmd go
  ensure_nfpm

  mkdir -p "$GUI_DIR/build/bin" "$DIST_DIR"

  echo "Compilando binario da GUI (Wails)..."
  build_gui_binary

  echo "Gerando pacote .deb"
  ( cd "$ROOT_DIR" && nfpm package \
      --packager deb \
      --config "packaging/nfpm/nfpm.yaml" \
      --target "$DIST_DIR/rclone-auto-gui_${VERSION}_${ARCH}.deb" )

  echo "Gerando pacote .rpm"
  ( cd "$ROOT_DIR" && nfpm package \
      --packager rpm \
      --config "packaging/nfpm/nfpm.yaml" \
      --target "$DIST_DIR/rclone-auto-gui-${VERSION}-1.x86_64.rpm" )

  echo "Gerando checksums"
  sha256sum "$DIST_DIR"/*.deb "$DIST_DIR"/*.rpm > "$DIST_DIR/SHA256SUMS"

  echo "Pacotes gerados em: $DIST_DIR"
  ls -lh "$DIST_DIR"
}

main "$@"
