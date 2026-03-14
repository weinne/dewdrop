#!/usr/bin/env bash
set -euo pipefail

SYSTEMD_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
USER_BIN_DIR="$HOME/.local/bin"
HELPER="$USER_BIN_DIR/rclone-auto-wait-online"
UNMOUNT_CMD=""

log() { printf '%s\n' "$*"; }

ensure_helper() {
  mkdir -p "$USER_BIN_DIR"

  cat <<'EOF' > "$HELPER"
#!/usr/bin/env bash
set -euo pipefail

URL="${1:-https://connectivitycheck.gstatic.com/generate_204}"
TIMEOUT="${RCLONE_AUTO_NET_TIMEOUT:-6}"
INTERVAL="${RCLONE_AUTO_NET_INTERVAL:-2}"

end=$((SECONDS + TIMEOUT))
while (( SECONDS < end )); do
  if command -v curl >/dev/null 2>&1; then
    if curl -fsS --max-time 5 "$URL" >/dev/null 2>&1; then exit 0; fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -q --spider --timeout=5 "$URL" >/dev/null 2>&1; then exit 0; fi
  fi

  if command -v getent >/dev/null 2>&1; then
    if getent hosts rclone.org >/dev/null 2>&1; then exit 0; fi
  fi

  sleep "$INTERVAL"
done

echo "rclone-auto: internet não disponível após ${TIMEOUT}s" >&2
exit 1
EOF

  chmod +x "$HELPER"
}

require_systemctl_user() {
  if ! command -v systemctl >/dev/null 2>&1; then
    log "Erro: 'systemctl' não encontrado."
    exit 1
  fi

  # Garante que o user manager está disponível
  if ! systemctl --user show-environment >/dev/null 2>&1; then
    log "Erro: 'systemctl --user' não está acessível nesta sessão."
    log "Dica: rode isso dentro de uma sessão do usuário (login normal) com systemd --user ativo."
    exit 1
  fi
}

detect_unmount_cmd() {
  if command -v fusermount3 >/dev/null 2>&1; then
    UNMOUNT_CMD="fusermount3"
    return 0
  fi
  if command -v fusermount >/dev/null 2>&1; then
    UNMOUNT_CMD="fusermount"
    return 0
  fi

  log "Aviso: fusermount/fusermount3 não encontrado. O ExecStop será preservado quando possível."
  UNMOUNT_CMD=""
}

backup_file() {
  local file="$1"
  local ts
  ts="$(date +%s)"
  cp -f "$file" "${file}.bak.${ts}"
}

fix_mount_unit() {
  local unit_path="$1"
  local unit_name remote execstart execstop

  unit_name="$(basename "$unit_path")"
  remote="${unit_name#rclone-mount-}"
  remote="${remote%.service}"

  execstart="$(grep -E '^ExecStart=' "$unit_path" | head -n1 | cut -d= -f2- || true)"
  execstop="$(grep -E '^ExecStop=' "$unit_path" | head -n1 | cut -d= -f2- || true)"

  if [[ -z "$execstart" ]]; then
    log "- Pulando $unit_name (sem ExecStart=)"
    return 0
  fi

  if [[ -z "$execstop" ]]; then
    if [[ -n "$UNMOUNT_CMD" ]]; then
      execstop="${UNMOUNT_CMD} -u \"%h/Nuvem/${remote}\""
    fi
  elif [[ -n "$UNMOUNT_CMD" ]]; then
    # Normaliza ExecStop antigo (/bin/fusermount) para o comando detectado no sistema.
    execstop="${UNMOUNT_CMD} -u \"%h/Nuvem/${remote}\""
  fi

  if [[ -z "$execstop" ]]; then
    log "- Pulando $unit_name (sem ExecStop= e sem fusermount disponível)"
    return 0
  fi

  backup_file "$unit_path"

  cat <<EOF > "$unit_path"
[Unit]
Description=Mount ${remote}
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=notify
ExecStartPre=%h/.local/bin/rclone-auto-wait-online
ExecStart=${execstart}
ExecStop=${execstop}
Restart=on-failure
RestartSec=15

[Install]
WantedBy=graphical-session.target
EOF

  log "- Corrigido: $unit_name"
}

fix_sync_unit() {
  local unit_path="$1"
  local unit_name remote execstart

  unit_name="$(basename "$unit_path")"
  remote="${unit_name#rclone-sync-}"
  remote="${remote%.service}"

  execstart="$(grep -E '^ExecStart=' "$unit_path" | head -n1 | cut -d= -f2- || true)"
  if [[ -z "$execstart" ]]; then
    log "- Pulando $unit_name (sem ExecStart=)"
    return 0
  fi

  backup_file "$unit_path"

  cat <<EOF > "$unit_path"
[Unit]
Description=Sync ${remote}

[Service]
Type=oneshot
ExecStartPre=%h/.local/bin/rclone-auto-wait-online
ExecStart=${execstart}
EOF

  log "- Corrigido: $unit_name"
}

reenable_if_enabled() {
  local unit="$1"
  if systemctl --user is-enabled --quiet "$unit" 2>/dev/null; then
    systemctl --user disable "$unit" >/dev/null 2>&1 || true
    systemctl --user enable "$unit" >/dev/null 2>&1 || true
    log "  re-habilitado: $unit"
  fi
}

restart_if_active() {
  local unit="$1"
  if systemctl --user is-active --quiet "$unit" 2>/dev/null; then
    systemctl --user restart "$unit" >/dev/null 2>&1 || true
    log "  reiniciado: $unit"
  fi
}

main() {
  require_systemctl_user

  if [[ ! -d "$SYSTEMD_DIR" ]]; then
    log "Nada a fazer: $SYSTEMD_DIR não existe."
    exit 0
  fi

  ensure_helper
  detect_unmount_cmd

  log "Corrigindo units em: $SYSTEMD_DIR"

  shopt -s nullglob

  local changed_units=()

  for unit_path in "$SYSTEMD_DIR"/rclone-mount-*.service; do
    fix_mount_unit "$unit_path"
    changed_units+=("$(basename "$unit_path")")
  done

  for unit_path in "$SYSTEMD_DIR"/rclone-sync-*.service; do
    fix_sync_unit "$unit_path"
    changed_units+=("$(basename "$unit_path")")
  done

  systemctl --user daemon-reload

  # Re-habilita e reinicia apenas mounts (onde o [Install] mudou)
  for unit in "${changed_units[@]}"; do
    case "$unit" in
      rclone-mount-*.service)
        reenable_if_enabled "$unit"
        restart_if_active "$unit"
        ;;
    esac
  done

  log "Concluído."
  log "Obs: backups foram criados como '*.bak.<timestamp>' no mesmo diretório."
}

main "$@"
