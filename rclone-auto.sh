#!/bin/bash

# ==========================================
# RClone Auto v48.0 (Navigation Fix)
# Autor: Weinne
# Feature: Botões de "Voltar" adicionados ao Wizard e melhoria no fluxo de cancelamento.
# ==========================================

# --- Configurações de Tema ---
export GUM_CHOOSE_CURSOR="👉 "
export GUM_CHOOSE_CURSOR_FOREGROUND="#00FFFF"
export GUM_CHOOSE_ITEM_FOREGROUND="#E0E0E0"
export GUM_CHOOSE_SELECTED_FOREGROUND="#00FFFF"
export GUM_INPUT_CURSOR_FOREGROUND="#FF00FF"
export GUM_CONFIRM_SELECTED_BACKGROUND="#6A0DAD"

# Diretórios
APP_NAME="rclone-auto"
USER_BIN_DIR="$HOME/.local/bin"
SYSTEMD_DIR="$HOME/.config/systemd/user"
SHORTCUT_DIR="$HOME/.local/share/applications"
CLOUD_DIR="$HOME/Nuvem"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/$APP_NAME"
RCLONE_SOURCE_FILE="$CONFIG_DIR/rclone-source.conf"
SYSTEM_ICON="folder-remote"

CURRENT_PATH=$(readlink -f "$0")
SCRIPT_DIR=$(dirname "$CURRENT_PATH")
TARGET_BIN="$USER_BIN_DIR/$APP_NAME"

mkdir -p "$USER_BIN_DIR" "$SYSTEMD_DIR" "$CLOUD_DIR" "$SHORTCUT_DIR"
mkdir -p "$CONFIG_DIR"
export PATH="$USER_BIN_DIR:$PATH"

# Binários
RCLONE_BIN=""
GUM_BIN=""
RCLONE_SOURCE="auto"

# --- 1. Inicialização ---

handle_cli_args() {
    case "$1" in
        --enable-boot)
            NAME="$2"
            if [ -z "$NAME" ]; then
                echo "Uso: $APP_NAME --enable-boot <nome-remoto>"
                exit 1
            fi
            systemctl --user enable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
            echo "Auto-start habilitado (quando existentes) para: $NAME"
            exit 0
            ;;
        --disable-boot)
            NAME="$2"
            if [ -z "$NAME" ]; then
                echo "Uso: $APP_NAME --disable-boot <nome-remoto>"
                exit 1
            fi
            systemctl --user disable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
            echo "Auto-start desabilitado (quando existentes) para: $NAME"
            exit 0
            ;;
        --open-path)
            TARGET_PATH="$2"
            if [ -z "$TARGET_PATH" ]; then
                echo "Uso: $APP_NAME --open-path <caminho-local>"
                exit 1
            fi

            # Resolve caminho absoluto
            if TARGET_ABS=$(readlink -f "$TARGET_PATH" 2>/dev/null); then
                TARGET_PATH="$TARGET_ABS"
            fi

            # Se não estiver dentro da pasta de nuvem, apenas delega para xdg-open
            case "$TARGET_PATH" in
                "$CLOUD_DIR"/*) ;;
                *)
                    xdg-open "$TARGET_PATH" >/dev/null 2>&1 &
                    exit 0
                    ;;
            esac

            # Descobre remoto e subcaminho: ~/Nuvem/<remote>/<subpath>
            REL="${TARGET_PATH#$CLOUD_DIR/}"
            REMOTE_NAME="${REL%%/*}"
            SUB_PATH="${REL#*/}"
            if [ -z "$REMOTE_NAME" ] || [ "$REMOTE_NAME" = "$REL" ]; then
                echo "rclone-auto: caminho não parece pertencer a um remoto em '$CLOUD_DIR'." >&2
                exit 1
            fi

            REMOTE_SPEC="${REMOTE_NAME}:${SUB_PATH}"

            # Localiza binário do rclone (sem depender do splash)
            if [ -f "$USER_BIN_DIR/rclone" ]; then
                RCLONE_BIN="$USER_BIN_DIR/rclone"
            elif command -v rclone &> /dev/null; then
                RCLONE_BIN=$(command -v rclone)
            fi

            if [ -z "$RCLONE_BIN" ]; then
                echo "rclone-auto: rclone não encontrado para resolver caminho remoto." >&2
                exit 1
            fi

            # Garante que REMOTE_NAME é de fato um remoto conhecido do rclone
            if ! "$RCLONE_BIN" listremotes 2>/dev/null | grep -q "^${REMOTE_NAME}:"; then
                echo "rclone-auto: '$REMOTE_NAME' não é um remoto rclone conhecido. Nada a fazer." >&2
                exit 1
            fi
            
            ITEM_NAME=$(basename "$TARGET_PATH")

            # Se tivermos (ou conseguirmos instalar) yad, usamos um menu gráfico; caso contrário,
            # fallback para TUI (Gum) se em terminal, ou abre localmente.
            if ensure_yad; then
                CHOICE=$(yad --list \
                    --title="Opções de Pasta na Nuvem" \
                    --width=420 --height=260 --center \
                    --window-icon="$SYSTEM_ICON" \
                    --text="Selecione uma ação para:\n<b>$ITEM_NAME</b>\n\nRemoto: <b>$REMOTE_NAME</b>" \
                    --column="Ação" \
                    "🌐 Abrir na Web" \
                    "🔗 Copiar link de compartilhamento" \
                    "📄 Abrir arquivo localmente" \
                    "📄⬇️ Copiar e abrir localmente" \
                    "🔙 Cancelar" \
                    --print-column=1)

                # Se usuário fechou a janela ou cancelou
                if [ $? -ne 0 ] || [ -z "$CHOICE" ]; then
                    exit 0
                fi
            else
                # Sem yad: se estivermos em um terminal interativo, usa Gum; senão, apenas abre localmente.
                if [ -t 0 ]; then
                    bootstrap_gum
                    ui_header
                    ui_talk "Opções para '$ITEM_NAME' em '$REMOTE_NAME'."
                    CHOICE=$(echo -e "🌐 Abrir na Web\n🔗 Copiar link de compartilhamento\n📄 Abrir arquivo localmente\n📄⬇️ Copiar e abrir localmente\n🔙 Cancelar" | \
                        $GUM_BIN choose --header "Opções de Pasta na Nuvem")
                else
                    xdg-open "$TARGET_PATH" >/dev/null 2>&1 &
                    exit 0
                fi
            fi

            case "$CHOICE" in
                "🌐 Abrir na Web"*)
                    URL=$("$RCLONE_BIN" link "$REMOTE_SPEC" 2>/dev/null)
                    if [ -z "$URL" ]; then
                        echo "rclone-auto: não foi possível obter link web para '$REMOTE_SPEC'." >&2
                        exit 1
                    fi
                    xdg-open "$URL" >/dev/null 2>&1 &
                    ;;
                "🔗 Copiar link de compartilhamento"*)
                    URL=$("$RCLONE_BIN" link "$REMOTE_SPEC" 2>/dev/null)
                    if [ -z "$URL" ]; then
                        echo "rclone-auto: não foi possível obter link web para '$REMOTE_SPEC'." >&2
                        exit 1
                    fi

                    if command -v yad &> /dev/null; then
                        # Apenas mostra o link em um campo editável para o usuário copiar manualmente.
                        yad --form \
                            --title="Link de compartilhamento" \
                            --width=520 --center \
                            --window-icon="$SYSTEM_ICON" \
                            --text="Copie o link abaixo (Ctrl+C) ou edite se desejar:" \
                            --field="Link:":"" "$URL" \
                            --button="Fechar":0 >/dev/null 2>&1
                    else
                        # Fallback em modo texto (sem YAD disponível)
                        echo "Copie o link abaixo manualmente:"
                        echo "$URL"
                    fi
                    ;;
                "📄 Abrir arquivo localmente"*)
                    xdg-open "$TARGET_PATH" >/dev/null 2>&1 &
                    ;;
                "📄⬇️ Copiar e abrir localmente"*)
                    TMP_DIR=$(mktemp -d /tmp/rclone-auto-open.XXXXXX 2>/dev/null)
                    if [ -z "$TMP_DIR" ]; then
                        echo "rclone-auto: não foi possível criar diretório temporário." >&2
                        exit 1
                    fi
                    "$RCLONE_BIN" copy "$REMOTE_SPEC" "$TMP_DIR" >/dev/null 2>&1
                    EXPORTED_FILE=$(find "$TMP_DIR" -maxdepth 1 -type f | head -n1)
                    if [ -z "$EXPORTED_FILE" ]; then
                        echo "rclone-auto: cópia para uso local falhou." >&2
                        exit 1
                    fi
                    xdg-open "$EXPORTED_FILE" >/dev/null 2>&1 &
                    ;;
                "🔙 Cancelar"*)
                    ;;
            esac

            exit 0
            ;;
        --help|-h)
            echo "Uso: $APP_NAME [opções]"
            echo ""
            echo "Sem opções: abre a interface interativa (TUI)."
            echo ""
            echo "Opções:"
            echo "  --enable-boot <nome-remoto>   Habilita auto-start (mount/sync) para o remoto"
            echo "  --disable-boot <nome-remoto>  Desabilita auto-start (mount/sync) para o remoto"
            echo "  --open-path <caminho-local>   Abre menu 'Opções de Pasta na Nuvem' para itens em ~/Nuvem"
            echo "  -h, --help                    Mostra esta ajuda"
            exit 0
            ;;
    esac

    # Se nenhuma opção especial foi tratada, segue fluxo normal (TUI)
}

ensure_yad() {
    # Já instalado
    if command -v yad &> /dev/null; then
        return 0
    fi

    # Sem terminal interativo: não tentamos instalar automaticamente
    if [ ! -t 0 ]; then
        return 1
    fi

    echo "O utilitário 'yad' não está instalado. Ele é usado para mostrar o menu gráfico 'Opções de Pasta na Nuvem'."
    printf "Tentar instalar automaticamente agora? [s/N] "
    read -r REPLY
    case "$REPLY" in
        s|S|sim|SIM|y|Y) ;;
        *) return 1 ;;
    esac

    INSTALL_CMD=""
    if command -v apt-get &> /dev/null; then
        INSTALL_CMD="sudo apt-get update && sudo apt-get install -y yad"
    elif command -v dnf &> /dev/null; then
        INSTALL_CMD="sudo dnf install -y yad"
    elif command -v pacman &> /dev/null; then
        INSTALL_CMD="sudo pacman -Sy --noconfirm yad"
    elif command -v zypper &> /dev/null; then
        INSTALL_CMD="sudo zypper install -y yad"
    fi

    if [ -z "$INSTALL_CMD" ]; then
        echo "Não foi possível detectar um gerenciador de pacotes suportado (apt, dnf, pacman, zypper)."
        return 1
    fi

    echo "Executando: $INSTALL_CMD"
    if sh -c "$INSTALL_CMD"; then
        if command -v yad &> /dev/null; then
            echo "'yad' instalado com sucesso."
            return 0
        fi
    fi

    echo "Falha ao instalar 'yad'."
    return 1
}

ensure_terminal() {
    if [ ! -t 0 ]; then
        for term in konsole gnome-terminal xfce4-terminal terminator xterm; do
            if command -v $term &> /dev/null; then $term -e "$CURRENT_PATH"; exit 0; fi
        done
        exit 1
    fi
}

load_rclone_source_preference() {
    if [ -f "$RCLONE_SOURCE_FILE" ]; then
        SRC=$(tr -d '[:space:]' < "$RCLONE_SOURCE_FILE")
        case "$SRC" in
            package|binary|auto) RCLONE_SOURCE="$SRC" ;;
            *) RCLONE_SOURCE="auto" ;;
        esac
    else
        RCLONE_SOURCE="auto"
    fi
}

save_rclone_source_preference() {
    printf '%s\n' "$1" > "$RCLONE_SOURCE_FILE"
    RCLONE_SOURCE="$1"
}

detect_package_installer() {
    if command -v apt-get &> /dev/null; then echo "apt"; return 0; fi
    if command -v dnf &> /dev/null; then echo "dnf"; return 0; fi
    if command -v yum &> /dev/null; then echo "yum"; return 0; fi
    if command -v zypper &> /dev/null; then echo "zypper"; return 0; fi
    echo "none"
}

find_system_rclone() {
    for p in /usr/bin/rclone /usr/local/bin/rclone /bin/rclone /snap/bin/rclone; do
        if [ -x "$p" ]; then
            echo "$p"
            return 0
        fi
    done

    if command -v rclone &> /dev/null; then
        p=$(command -v rclone)
        if [ "$p" != "$USER_BIN_DIR/rclone" ]; then
            echo "$p"
            return 0
        fi
    fi

    return 1
}

detect_effective_rclone_source() {
    INSTALLER=$(detect_package_installer)
    if [ "$RCLONE_SOURCE" = "package" ]; then
        echo "package"
        return 0
    fi
    if [ "$RCLONE_SOURCE" = "binary" ]; then
        echo "binary"
        return 0
    fi

    # auto: usa pacote em distros apt/rpm; em outras, binario local.
    if [ "$INSTALLER" = "apt" ] || [ "$INSTALLER" = "dnf" ] || [ "$INSTALLER" = "yum" ] || [ "$INSTALLER" = "zypper" ]; then
        echo "package"
    else
        echo "binary"
    fi
}

install_rclone_from_package() {
    INSTALLER=$(detect_package_installer)

    case "$INSTALLER" in
        apt)
            sudo apt-get update && sudo apt-get install -y rclone
            ;;
        dnf)
            sudo dnf install -y rclone
            ;;
        yum)
            sudo yum install -y rclone
            ;;
        zypper)
            sudo zypper --non-interactive install rclone
            ;;
        *)
            return 1
            ;;
    esac
}

install_rclone_binary() {
    rm -rf /tmp/rclone-auto-inst
    mkdir -p /tmp/rclone-auto-inst

    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) RCLONE_ARCH="amd64" ;;
        aarch64|arm64) RCLONE_ARCH="arm64" ;;
        *)
            echo "Arquitetura não suportada para download automático do binário: $ARCH" >&2
            return 1
            ;;
    esac

    if ! curl -fL "https://downloads.rclone.org/rclone-current-linux-${RCLONE_ARCH}.zip" -o /tmp/rclone-auto-inst/rclone.zip; then
        return 1
    fi

    unzip -q -o /tmp/rclone-auto-inst/rclone.zip -d /tmp/rclone-auto-inst
    NEW_BIN=$(find /tmp/rclone-auto-inst -type f -name rclone | head -n1)
    if [ -z "$NEW_BIN" ]; then
        return 1
    fi

    mv "$NEW_BIN" "$USER_BIN_DIR/rclone"
    chmod +x "$USER_BIN_DIR/rclone"
    RCLONE_BIN="$USER_BIN_DIR/rclone"
    return 0
}

ensure_rclone_bin() {
    load_rclone_source_preference
    EFFECTIVE_SOURCE=$(detect_effective_rclone_source)

    if [ "$EFFECTIVE_SOURCE" = "package" ]; then
        if SYS_RCLONE=$(find_system_rclone); then
            RCLONE_BIN="$SYS_RCLONE"
            return 0
        fi

        if install_rclone_from_package; then
            if SYS_RCLONE=$(find_system_rclone); then
                RCLONE_BIN="$SYS_RCLONE"
                return 0
            fi
        fi

        # Em modo auto, ainda permitimos fallback para binário quando o pacote falha.
        if [ "$RCLONE_SOURCE" = "auto" ]; then
            if install_rclone_binary; then
                return 0
            fi
        fi

        return 1
    fi

    if [ -x "$USER_BIN_DIR/rclone" ]; then
        RCLONE_BIN="$USER_BIN_DIR/rclone"
        return 0
    fi

    if install_rclone_binary; then
        return 0
    fi

    return 1
}

show_rclone_source_menu() {
    load_rclone_source_preference
    EFFECTIVE_SOURCE=$(detect_effective_rclone_source)
    INSTALLER=$(detect_package_installer)

    if [ "$EFFECTIVE_SOURCE" = "package" ]; then
        CURRENT_LABEL="Pacote do sistema"
    else
        CURRENT_LABEL="Binário local (~/.local/bin)"
    fi

    ui_talk "Fonte atual do Rclone: $CURRENT_LABEL"

    CHOICE=$(echo -e "📦 Usar pacote do sistema\n🧩 Usar binário local\n🔙 Voltar" | $GUM_BIN choose --header "Fonte do Rclone")

    case "$CHOICE" in
        "📦"*)
            if [ "$INSTALLER" = "none" ]; then
                ui_error "Não detectei apt/rpm (dnf/yum/zypper) neste sistema. Não dá para usar pacote aqui."
                return
            fi

            if install_rclone_from_package; then
                save_rclone_source_preference "package"
                if SYS_RCLONE=$(find_system_rclone); then
                    RCLONE_BIN="$SYS_RCLONE"
                fi
                ui_success "Rclone configurado para usar o pacote do sistema."
            else
                ui_error "Falha ao instalar rclone via pacote."
            fi
            ;;
        "🧩"*)
            if install_rclone_binary; then
                save_rclone_source_preference "binary"
                ui_success "Rclone configurado para usar binário local."
            else
                ui_error "Falha ao instalar/atualizar o binário local do rclone."
            fi
            ;;
    esac
}

run_fix_existing_services() {
    FIXER_BIN="$USER_BIN_DIR/rclone-auto-fix-services"
    FIXER_LOCAL="$SCRIPT_DIR/fix-existing-services.sh"

    if [ -x "$FIXER_BIN" ]; then
        bash "$FIXER_BIN"
    elif [ -x "$FIXER_LOCAL" ]; then
        bash "$FIXER_LOCAL"
    else
        ui_error "Script de correção não encontrado. Reinstale o rclone-auto."
        return
    fi

    if [ $? -eq 0 ]; then
        ui_success "Correção de serviços concluída."
    else
        ui_error "A correção encontrou erros."
    fi
}

bootstrap_gum() {
    if [ -f "$SCRIPT_DIR/gum" ] && [ -x "$SCRIPT_DIR/gum" ]; then GUM_BIN="$SCRIPT_DIR/gum"
    elif [ -f "$USER_BIN_DIR/gum" ]; then GUM_BIN="$USER_BIN_DIR/gum"
    elif command -v gum &> /dev/null; then GUM_BIN=$(command -v gum); fi

    if [ -z "$GUM_BIN" ]; then
        echo "⬇️  Instalando interface gráfica (Gum)..."
        rm -rf /tmp/gum*
        ARCH=$(uname -m); case $ARCH in x86_64) GUM_ARCH="x86_64";; aarch64|arm64) GUM_ARCH="arm64";; esac
        curl -L -o /tmp/gum.tar.gz "https://github.com/charmbracelet/gum/releases/download/v0.14.5/gum_0.14.5_Linux_${GUM_ARCH}.tar.gz"
        tar -xzf /tmp/gum.tar.gz -C /tmp/
        mv $(find /tmp -name gum -type f -executable | head -n 1) "$USER_BIN_DIR/"
        chmod +x "$USER_BIN_DIR/gum"
        GUM_BIN="$USER_BIN_DIR/gum"
        echo "✅ Interface instalada!"
    fi
}

check_deps_splash() {
    bootstrap_gum

    clear
    echo ""
    $GUM_BIN style --foreground 212 --border double --border-foreground 212 --padding "1 2" --align center "RCLONE AUTO v48" "System Initialization"
    echo ""

    check_step() {
        if $1; then
            $GUM_BIN style --foreground 46 "✓ $2.......... [OK]"
        else
            $GUM_BIN style --foreground 196 "✗ $2.......... [FAIL]"
            return 1
        fi
    }

    if command -v fusermount3 &> /dev/null; then
        check_step "true" "FUSE3 Filesystem"
    else
        check_step "false" "FUSE3 Filesystem"
        echo "⚠️  Erro Crítico: Instale o fuse3 (sudo apt install fuse3)"
        read -p "Enter para sair..."
        exit 1
    fi
    sleep 0.1

    if ensure_rclone_bin; then
        check_step "true" "Rclone Core"
    else
        check_step "false" "Rclone Core"
        echo "⚠️  Erro Crítico: não consegui preparar o rclone (pacote/binário)."
        read -p "Enter para sair..."
        exit 1
    fi
    sleep 0.1

    check_step "true" "Graphic UI (Gum)"
    sleep 0.5
}

install_system() {
    if [ -f "$CURRENT_PATH" ] && [ "$CURRENT_PATH" != "$TARGET_BIN" ]; then cp -f "$CURRENT_PATH" "$TARGET_BIN"; chmod +x "$TARGET_BIN"; fi
    if [ "$GUM_BIN" == "$SCRIPT_DIR/gum" ] && [ ! -f "$USER_BIN_DIR/gum" ]; then cp "$SCRIPT_DIR/gum" "$USER_BIN_DIR/"; chmod +x "$USER_BIN_DIR/gum"; fi
    if [ -f "$SCRIPT_DIR/fix-existing-services.sh" ]; then cp -f "$SCRIPT_DIR/fix-existing-services.sh" "$USER_BIN_DIR/rclone-auto-fix-services"; chmod +x "$USER_BIN_DIR/rclone-auto-fix-services"; fi

    # Helper: espera internet antes de iniciar mounts/syncs (usado via ExecStartPre nos units).
    cat <<'EOF' > "$USER_BIN_DIR/rclone-auto-wait-online"
#!/usr/bin/env bash
set -euo pipefail

URL="${1:-https://connectivitycheck.gstatic.com/generate_204}"
TIMEOUT="${RCLONE_AUTO_NET_TIMEOUT:-6}"
INTERVAL="${RCLONE_AUTO_NET_INTERVAL:-2}"

end=$((SECONDS + TIMEOUT))
while (( SECONDS < end )); do
    if command -v curl >/dev/null 2>&1; then
        if curl -fsS --max-time 5 "$URL" >/dev/null 2>&1; then
            exit 0
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -q --spider --timeout=5 "$URL" >/dev/null 2>&1; then
            exit 0
        fi
    fi

    # Fallback barato: pelo menos DNS funcional.
    if command -v getent >/dev/null 2>&1; then
        if getent hosts rclone.org >/dev/null 2>&1; then
            exit 0
        fi
    fi

    sleep "$INTERVAL"
done

echo "rclone-auto: internet não disponível após ${TIMEOUT}s" >&2
exit 1
EOF
    chmod +x "$USER_BIN_DIR/rclone-auto-wait-online"

    # Opcional: tenta garantir que 'yad' esteja disponível para a integração gráfica do resolvedor.
    # Se não conseguir (ou se não houver TTY), apenas segue sem falhar a instalação.
    ensure_yad >/dev/null 2>&1 || true

    DESKTOP_FILE="$SHORTCUT_DIR/$APP_NAME.desktop"
    echo -e "[Desktop Entry]\nName=RClone Auto\nComment=Gerenciador de Nuvens\nExec=\"$TARGET_BIN\"\nIcon=$SYSTEM_ICON\nTerminal=true\nType=Application\nCategories=Utility;Network;" > "$DESKTOP_FILE"
    chmod +x "$DESKTOP_FILE"

    # Resolvedor simples: abre qualquer caminho dentro da Nuvem no navegador via rclone link.
    RESOLVER_DESKTOP="$SHORTCUT_DIR/${APP_NAME}-resolver.desktop"
    cat <<EOF > "$RESOLVER_DESKTOP"
[Desktop Entry]
Name=Opções de Pasta na Nuvem
Comment=Menu de ações para itens em pastas montadas da Nuvem
Exec="$TARGET_BIN" --open-path "%f"
Icon=$SYSTEM_ICON
Terminal=false
Type=Application
NoDisplay=false
MimeType=application/octet-stream;
EOF
    chmod +x "$RESOLVER_DESKTOP"

    if command -v update-desktop-database &> /dev/null; then update-desktop-database "$SHORTCUT_DIR" 2>/dev/null; fi
    if [ -d "$CLOUD_DIR" ]; then echo -e "[Desktop Entry]\nIcon=$SYSTEM_ICON\nType=Directory" > "$CLOUD_DIR/.directory" 2>/dev/null; fi
}

# --- 2. Interface Helper ---

ui_header() {
    clear
    $GUM_BIN style --foreground 212 --border-foreground 212 --border double --align center --width 50 --margin "1 2" --padding "0 2" "☁️  RClone Auto"
}

ui_talk() {
    echo ""
    $GUM_BIN style --foreground 255 --padding "0 1" "🤖 Assistente:"
    $GUM_BIN style --foreground 212 --padding "0 2" "$1"
    echo ""
}

ui_success() { $GUM_BIN style --foreground 46 "✅ $1"; sleep 1.5; }
ui_error() { $GUM_BIN style --foreground 196 "❌ $1"; $GUM_BIN confirm "Ok" --affirmative "Entendi" --negative ""; }

diag_result() {
    OK="$1"
    MSG="$2"
    DETAIL="${3:-}"

    if [ "$OK" -eq 0 ]; then
        $GUM_BIN style --foreground 46 "✓ $MSG"
    else
        $GUM_BIN style --foreground 196 "✗ $MSG"
        if [ -n "$DETAIL" ]; then
            $GUM_BIN style --foreground 214 "  $DETAIL"
        fi
    fi
}

test_remote_health() {
    REMOTE="$1"
    MODE="$2"

    ui_talk "Executando teste rápido para '$REMOTE' ($MODE)..."

    FAIL=0

    if [ ! -x "$USER_BIN_DIR/rclone-auto-wait-online" ]; then
        diag_result 1 "Helper de rede ausente" "Esperado em: $USER_BIN_DIR/rclone-auto-wait-online"
        FAIL=1
    else
        diag_result 0 "Helper de rede presente"
    fi

    if [ "$MODE" = "mount" ]; then
        UNIT="rclone-mount-${REMOTE}.service"
        if systemctl --user is-active --quiet "$UNIT"; then
            diag_result 0 "Serviço mount ativo"
        else
            diag_result 1 "Serviço mount não está ativo" "Use: systemctl --user status $UNIT"
            FAIL=1
        fi
    else
        UNIT="rclone-sync-${REMOTE}.timer"
        if systemctl --user is-active --quiet "$UNIT"; then
            diag_result 0 "Timer de sync ativo"
        else
            diag_result 1 "Timer de sync não está ativo" "Use: systemctl --user status $UNIT"
            FAIL=1
        fi
    fi

    if "$RCLONE_BIN" lsd "${REMOTE}:" --max-depth 0 >/dev/null 2>&1; then
        diag_result 0 "Acesso ao remoto confirmado"
    else
        diag_result 1 "Falha ao acessar remoto" "Pode ser token expirado, rede ou configuração do remoto."
        FAIL=1
    fi

    if [ "$FAIL" -eq 0 ]; then
        ui_success "Teste rápido concluído sem falhas."
    else
        ui_error "Teste rápido detectou problemas em '$REMOTE'."
    fi
}

run_full_diagnostic() {
    ui_talk "Executando diagnóstico completo do ambiente e serviços..."

    FAILS=0
    MOUNT_UNITS=0
    SYNC_UNITS=0

    if command -v systemctl >/dev/null 2>&1; then
        diag_result 0 "systemctl disponível"
    else
        diag_result 1 "systemctl não encontrado"
        FAILS=$((FAILS+1))
    fi

    if systemctl --user show-environment >/dev/null 2>&1; then
        diag_result 0 "systemd --user acessível"
    else
        diag_result 1 "systemd --user inacessível" "Abra o script em uma sessão de usuário com systemd --user ativo."
        FAILS=$((FAILS+1))
    fi

    if [ -x "$RCLONE_BIN" ]; then
        diag_result 0 "rclone detectado" "$RCLONE_BIN"
    else
        diag_result 1 "rclone não encontrado" "Defina a fonte do rclone em Ferramentas."
        FAILS=$((FAILS+1))
    fi

    if [ -x "$GUM_BIN" ]; then
        diag_result 0 "gum detectado" "$GUM_BIN"
    else
        diag_result 1 "gum não encontrado"
        FAILS=$((FAILS+1))
    fi

    if command -v fusermount3 >/dev/null 2>&1 || command -v fusermount >/dev/null 2>&1; then
        diag_result 0 "FUSE disponível"
    else
        diag_result 1 "FUSE indisponível" "Instale fuse3 para mounts funcionarem."
        FAILS=$((FAILS+1))
    fi

    if [ -x "$USER_BIN_DIR/rclone-auto-wait-online" ]; then
        diag_result 0 "Helper de rede presente"
    else
        diag_result 1 "Helper de rede ausente"
        FAILS=$((FAILS+1))
    fi

    REMOTES=$("$RCLONE_BIN" listremotes 2>/dev/null | sed 's/:$//')
    if [ -n "$REMOTES" ]; then
        diag_result 0 "Remotos configurados detectados"
    else
        diag_result 1 "Nenhum remoto configurado"
    fi

    for unit_path in "$SYSTEMD_DIR"/rclone-mount-*.service; do
        [ -f "$unit_path" ] || continue
        MOUNT_UNITS=$((MOUNT_UNITS+1))
        unit_name=$(basename "$unit_path")
        remote="${unit_name#rclone-mount-}"
        remote="${remote%.service}"

        if grep -q "StartLimitIntervalSec" "$unit_path"; then
            diag_result 1 "$unit_name contém chave legada incompatível" "Rode: Ferramentas -> Corrigir Serviços Existentes"
            FAILS=$((FAILS+1))
        fi

        if grep -q "^ExecStop=/bin/fusermount" "$unit_path"; then
            diag_result 1 "$unit_name usa ExecStop legado (/bin/fusermount)" "Rode: Ferramentas -> Corrigir Serviços Existentes"
            FAILS=$((FAILS+1))
        fi

        if ! "$RCLONE_BIN" listremotes 2>/dev/null | grep -q "^${remote}:"; then
            diag_result 1 "$unit_name aponta para remoto inexistente" "$remote"
            FAILS=$((FAILS+1))
        fi

        if systemctl --user is-enabled --quiet "$unit_name" 2>/dev/null && ! systemctl --user is-active --quiet "$unit_name" 2>/dev/null; then
            diag_result 1 "$unit_name está enabled mas inativo" "Verifique: systemctl --user status $unit_name"
            FAILS=$((FAILS+1))
        fi
    done

    for unit_path in "$SYSTEMD_DIR"/rclone-sync-*.timer; do
        [ -f "$unit_path" ] || continue
        SYNC_UNITS=$((SYNC_UNITS+1))
        unit_name=$(basename "$unit_path")

        if systemctl --user is-enabled --quiet "$unit_name" 2>/dev/null && ! systemctl --user is-active --quiet "$unit_name" 2>/dev/null; then
            diag_result 1 "$unit_name está enabled mas inativo" "Verifique: systemctl --user status $unit_name"
            FAILS=$((FAILS+1))
        fi
    done

    $GUM_BIN style --foreground 212 ""
    $GUM_BIN style --foreground 212 "Resumo: mounts=$MOUNT_UNITS, timers=$SYNC_UNITS, falhas=$FAILS"

    if [ "$FAILS" -eq 0 ]; then
        ui_success "Diagnóstico concluído. Tudo certo."
    else
        ui_error "Diagnóstico concluído com falhas. Use Ferramentas -> Corrigir Serviços Existentes e revise os status."
    fi
}

# --- 3. Lógica Core ---

setup_sync() {
    REMOTE="$1"; LOCAL="$CLOUD_DIR/$REMOTE"; AUTO_START="$2"
    mkdir -p "$LOCAL"
    cat <<EOF > "$SYSTEMD_DIR/rclone-sync-${REMOTE}.service"
[Unit]
Description=Sync $REMOTE
[Service]
Type=oneshot
ExecStartPre=%h/.local/bin/rclone-auto-wait-online
ExecStart=$(readlink -f "$RCLONE_BIN") bisync "${REMOTE}:" "${LOCAL}" --create-empty-src-dirs --compare size,modtime,checksum --slow-hash-sync-only --resync --verbose
EOF
    cat <<EOF > "$SYSTEMD_DIR/rclone-sync-${REMOTE}.timer"
[Unit]
Description=Timer 15m $REMOTE
[Timer]
OnBootSec=5min
OnUnitActiveSec=15min
[Install]
WantedBy=timers.target
EOF
    $GUM_BIN spin --spinner dot --title "Configurando timer..." -- sleep 1
    systemctl --user daemon-reload
    if [ "$AUTO_START" = "auto" ]; then
        systemctl --user enable --now "rclone-sync-${REMOTE}.timer"
    else
        systemctl --user start "rclone-sync-${REMOTE}.timer"
    fi
    $GUM_BIN spin --title "Sincronizando arquivos..." -- systemctl --user start "rclone-sync-${REMOTE}.service"
    ui_success "Pronto! A pasta $REMOTE está sincronizada."
    test_remote_health "$REMOTE" "sync"
}

setup_mount() {
    REMOTE="$1"; LOCAL="$CLOUD_DIR/$REMOTE"; AUTO_START="$2"
    mkdir -p "$LOCAL"
    cat <<EOF > "$SYSTEMD_DIR/rclone-mount-${REMOTE}.service"
[Unit]
Description=Mount $REMOTE
After=graphical-session.target
PartOf=graphical-session.target
[Service]
Type=notify
ExecStartPre=%h/.local/bin/rclone-auto-wait-online
ExecStart=$(readlink -f "$RCLONE_BIN") mount ${REMOTE}: "${LOCAL}" --vfs-cache-mode full --no-modtime
ExecStop=fusermount3 -u "${LOCAL}"
Restart=on-failure
RestartSec=15
[Install]
WantedBy=graphical-session.target
EOF
    $GUM_BIN spin --spinner dot --title "Montando disco..." -- sleep 1
    systemctl --user daemon-reload
    if [ "$AUTO_START" = "auto" ]; then
        systemctl --user enable --now "rclone-mount-${REMOTE}.service"
    else
        systemctl --user start "rclone-mount-${REMOTE}.service"
    fi
    if systemctl --user is-active --quiet "rclone-mount-${REMOTE}.service"; then
        if [ -d "$CLOUD_DIR" ]; then echo -e "[Desktop Entry]\nIcon=$SYSTEM_ICON\nType=Directory" > "$CLOUD_DIR/.directory" 2>/dev/null; fi
        ui_success "Conectado! Acesso disponível na pasta Nuvem."
        test_remote_health "$REMOTE" "mount"
    else
        ui_error "Houve um erro ao montar o disco."
    fi
}

stop_all() {
    NAME="$1"
    $GUM_BIN spin --title "Desconectando..." -- sleep 1
    systemctl --user stop "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" "rclone-sync-${NAME}.service" 2>/dev/null
    systemctl --user disable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
    rm "$SYSTEMD_DIR/rclone-mount-${NAME}.service" "$SYSTEMD_DIR/rclone-sync-${NAME}.timer" "$SYSTEMD_DIR/rclone-sync-${NAME}.service" 2>/dev/null
    systemctl --user daemon-reload
    ui_success "Serviço parado."
}

open_remote_in_browser() {
    NAME="$1"
    CONF=$("$RCLONE_BIN" config file | grep ".conf" | tail -n1)
    TYPE=$(sed -n "/^\[$NAME\]/,/^\[/p" "$CONF" | grep "^type" | awk '{print $3}')

    case "$TYPE" in
        drive)
            ui_talk "Abrindo Google Drive no navegador..."
            xdg-open "https://drive.google.com/drive/u/0/my-drive" >/dev/null 2>&1 &
            ;;
        onedrive)
            ui_talk "Abrindo OneDrive no navegador..."
            xdg-open "https://onedrive.live.com/" >/dev/null 2>&1 &
            ;;
        *)
            ui_error "Abertura web automática ainda não está disponível para o tipo: $TYPE"
            ;;
    esac
}

# --- 4. Ferramentas Globais ---

update_binaries() {
    load_rclone_source_preference
    EFFECTIVE_SOURCE=$(detect_effective_rclone_source)

    if [ "$EFFECTIVE_SOURCE" = "package" ]; then
        if install_rclone_from_package; then
            if SYS_RCLONE=$(find_system_rclone); then RCLONE_BIN="$SYS_RCLONE"; fi
            ui_success "Rclone via pacote atualizado."
        else
            ui_error "Falha ao atualizar rclone via pacote."
        fi
    else
        if install_rclone_binary; then
            ui_success "Rclone binário atualizado."
        else
            ui_error "Falha ao atualizar rclone binário."
        fi
    fi
}

create_shortcuts() {
    $GUM_BIN spin --title "Gerando ícones..." -- sleep 1
    LIST=$(systemctl --user list-unit-files | grep "rclone-mount-" | grep "enabled" | awk '{print $1}')
    for s in $LIST; do
        NAME=$(echo "$s" | sed 's/rclone-mount-//;s/.service//')
        MOUNT_POINT="$CLOUD_DIR/$NAME"
        SHORTCUT="$HOME/Desktop/$NAME.desktop"
        echo -e "[Desktop Entry]\nName=$NAME\nExec=xdg-open \"$MOUNT_POINT\"\nIcon=$SYSTEM_ICON\nType=Application" > "$SHORTCUT"
        chmod +x "$SHORTCUT"
    done
    ui_success "Atalhos criados no Desktop!"
}

fix_icons() {
    if [ -d "$CLOUD_DIR" ]; then echo -e "[Desktop Entry]\nIcon=$SYSTEM_ICON\nType=Directory" > "$CLOUD_DIR/.directory" 2>/dev/null; fi
    ui_success "Ícones corrigidos."
}

do_global_tools() {
    ui_talk "Aqui estão algumas ferramentas para manter tudo em ordem."
    CHOICE=$(echo -e "🖥️  Criar Atalhos no Desktop\n🎨 Corrigir Ícones\n🧪 Testar Configuração\n🩹 Corrigir Serviços Existentes\n⚙️  Fonte do Rclone (pacote/binário)\n⬇️  Atualizar Tudo\n♻️  Reinstalar Script\n🔙 Voltar" | $GUM_BIN choose --header "Ferramentas")

    case "$CHOICE" in
        "🖥️"*) create_shortcuts ;;
        "🎨"*) fix_icons ;;
        "🧪"*) run_full_diagnostic ;;
        "🩹"*) run_fix_existing_services ;;
        "⚙️"*) show_rclone_source_menu ;;
        "⬇️"*) update_binaries ;;
        "♻️"*) install_system; ui_success "Script reinstalado com sucesso." ;;
    esac
}

# --- 5. O Mago (Wizard) ---

do_wizard() {
    ui_talk "Olá! Vamos conectar uma nova nuvem ao seu sistema. Qual serviço você gostaria de adicionar?"

    POPULAR="drive (Google Drive)
onedrive (Microsoft OneDrive)
dropbox (Dropbox)
box (Box)
pcloud (pCloud)
s3 (S3 / AWS / Minio)
webdav (WebDAV)
smb (Windows Share / SMB)
sftp (SSH / SFTP)
--------------------------------
ALL (Outros / Listar Todos)
🔙 Voltar"

    SEL=$(echo "$POPULAR" | $GUM_BIN choose --height 15)

    # 1. Verifica Voltar/Cancelar
    if [[ "$SEL" == *"Voltar"* ]] || [ -z "$SEL" ]; then return; fi

    # 2. Ignora separador
    if [[ "$SEL" == *"---"* ]]; then return; fi

    # 3. Busca Global
    if [[ "$SEL" == "ALL"* ]]; then
        ui_talk "Buscando lista completa de drivers..."
        FULL_LIST=$("$RCLONE_BIN" help backends 2>/dev/null | tail -n +2 | awk '{printf "%s (%s)\n", $1, substr($0, index($0,$2))}')
        # Adiciona Voltar na busca global
        FULL_LIST="${FULL_LIST}
🔙 Voltar"

        SEL=$(echo "$FULL_LIST" | $GUM_BIN choose --header "Busca Global" --height 15)
        if [[ "$SEL" == *"Voltar"* ]] || [ -z "$SEL" ]; then return; fi
    fi

    PROVIDER=$(echo "$SEL" | awk '{print $1}' | sed 's/\x1b\[[0-9;]*m//g')

    ui_talk "Ótima escolha ($PROVIDER). Como devemos chamar essa conexão? (Dica: use nomes curtos como 'pessoal' ou 'trabalho')."

    SUFFIX=$($GUM_BIN input --placeholder "ex: pessoal" | tr -cd '[:alnum:]_-')
    if [ -z "$SUFFIX" ]; then ui_talk "Operação cancelada."; return; fi

    NAME="${PROVIDER}-${SUFFIX}"
    if "$RCLONE_BIN" listremotes | grep -q "^${NAME}:"; then ui_error "Esse nome ($NAME) já está em uso."; return; fi

    ui_talk "Certo. Vou abrir seu navegador para você fazer o login no $PROVIDER. Preparado?"
    if $GUM_BIN confirm "Sim, abrir navegador" --negative "Cancelar"; then
        "$RCLONE_BIN" config create "$NAME" "$PROVIDER"
    else
        ui_talk "Operação cancelada."
        return
    fi

    if "$RCLONE_BIN" listremotes | grep -q "^${NAME}:"; then
        ui_talk "Sucesso! Agora, como você quer usar essa nuvem?"
        ACTION=$(echo -e "MOUNT (Disco Virtual - Economiza espaço)\nSYNC (Backup Offline - Cópia segura)" | $GUM_BIN choose)

        ui_talk "Você quer que essa conexão inicie automaticamente junto com a sua sessão?"
        if $GUM_BIN confirm "Sim, iniciar automaticamente" --negative "Não, iniciar só quando eu mandar"; then
            AUTO_MODE="auto"
        else
            AUTO_MODE="manual"
        fi

        if [[ "$ACTION" == MOUNT* ]]; then
            setup_mount "$NAME" "$AUTO_MODE"
        else
            setup_sync "$NAME" "$AUTO_MODE"
        fi
    else
        ui_error "Não consegui confirmar a criação. Tente novamente."
    fi
}

do_manage() {
    REMOTES=$("$RCLONE_BIN" listremotes 2>/dev/null)
    if [ -z "$REMOTES" ]; then ui_talk "Você ainda não tem nenhuma conexão configurada."; return; fi

    MENU_ITENS=""
    for r in $REMOTES; do
        clean="${r%:}"
        STATUS="⚪"; TYPE="Inativo"
        if systemctl --user is-active --quiet "rclone-mount-${clean}.service"; then STATUS="🟢"; TYPE="Montado";
        elif systemctl --user is-active --quiet "rclone-sync-${clean}.timer"; then STATUS="🔵"; TYPE="Sync"; fi

        LINE=$(printf "%s  %-20s  (%s)" "$STATUS" "$clean" "$TYPE")
        MENU_ITENS+="${LINE}\n"
    done
    MENU_ITENS+="🔙 Voltar"

    ui_talk "Aqui estão suas conexões. Selecione uma para ver opções."
    CHOICE=$(echo -e "$MENU_ITENS" | $GUM_BIN choose --height 10)

    if [[ "$CHOICE" == *"Voltar"* ]] || [ -z "$CHOICE" ]; then return; fi
    NAME=$(echo "$CHOICE" | awk '{print $2}')

    AUTO_ENABLED="não"
    if systemctl --user list-unit-files | grep -q "rclone-mount-${NAME}.service" && \
       systemctl --user is-enabled --quiet "rclone-mount-${NAME}.service"; then
        AUTO_ENABLED="sim"
    elif systemctl --user list-unit-files | grep -q "rclone-sync-${NAME}.timer" && \
         systemctl --user is-enabled --quiet "rclone-sync-${NAME}.timer"; then
        AUTO_ENABLED="sim"
    fi

    if [[ "$CHOICE" == *"Montado"* ]] || [[ "$CHOICE" == *"Sync"* ]]; then
        ACTION=$(echo -e "📂 Abrir Pasta\n🌐 Abrir na Web\n🔴 Desconectar\n⚙️  Alternar Auto-start (atual: $AUTO_ENABLED)\n🔙 Voltar" | $GUM_BIN choose --header "Opções para $NAME")
        case "$ACTION" in
            "📂 Abrir"*) xdg-open "$CLOUD_DIR/$NAME" ;;
            "🌐 Abrir na Web"*) open_remote_in_browser "$NAME" ;;
            "🔴 Desconectar"*) if $GUM_BIN confirm "Tem certeza que deseja parar $NAME?"; then stop_all "$NAME"; fi ;;
            "⚙️  Alternar Auto-start"*)
                if [ "$AUTO_ENABLED" = "sim" ]; then
                    systemctl --user disable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
                    ui_success "Auto-start desativado para $NAME."
                else
                    systemctl --user enable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
                    ui_success "Auto-start ativado para $NAME."
                fi ;;
        esac
    else
        ACTION=$(echo -e "🟢 Ativar (Mount)\n🔵 Ativar (Sync)\n🌐 Abrir na Web\n⚙️  Alternar Auto-start (atual: $AUTO_ENABLED)\n✏️  Renomear\n🗑️  Excluir\n🔙 Voltar" | $GUM_BIN choose --header "Opções para $NAME")
        case "$ACTION" in
            "🟢 Ativar"*) setup_mount "$NAME" ;;
            "🔵 Ativar"*) setup_sync "$NAME" ;;
            "🌐 Abrir na Web"*) open_remote_in_browser "$NAME" ;;
            "⚙️  Alternar Auto-start"*)
                if [ "$AUTO_ENABLED" = "sim" ]; then
                    systemctl --user disable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
                    ui_success "Auto-start desativado para $NAME."
                else
                    systemctl --user enable "rclone-mount-${NAME}.service" "rclone-sync-${NAME}.timer" 2>/dev/null
                    ui_success "Auto-start ativado para $NAME."
                fi ;;
            "🗑️  Excluir"*)
                if $GUM_BIN confirm "Excluir configurações de $NAME permanentemente?"; then
                    stop_all "$NAME"; "$RCLONE_BIN" config delete "$NAME"; ui_success "Removido.";
                fi ;;
            "✏️  Renomear"*)
                ui_talk "Qual será o novo sufixo para $NAME?"
                NEW_SUF=$($GUM_BIN input | tr -cd '[:alnum:]_-')
                if [ -n "$NEW_SUF" ]; then
                    TYPE=$(echo "$NAME" | cut -d- -f1); NEW_NAME="${TYPE}-${NEW_SUF}"; stop_all "$NAME"
                    CONF=$("$RCLONE_BIN" config file | grep ".conf" | tail -n1)
                    sed -i "s/^\[$NAME\]$/\[$NEW_NAME\]/" "$CONF"
                    if [ -d "$CLOUD_DIR/$NAME" ]; then mv "$CLOUD_DIR/$NAME" "$CLOUD_DIR/$NEW_NAME"; fi
                    ui_success "Renomeado para $NEW_NAME"
                fi ;;
        esac
    fi
}

# --- 6. Loop Principal ---

handle_cli_args "$@"

ensure_terminal
check_deps_splash
install_system

while true; do
    ui_header
    ACTIVE=$(systemctl --user list-unit-files | grep -E "rclone-(mount|sync)-" | grep "enabled" | wc -l)

    echo -e "   Conexões Ativas: \033[1;32m$ACTIVE\033[0m"
    echo ""

    CHOICE=$(echo -e "🚀 Nova Conexão\n📂 Gerenciar Conexões\n🛠️  Ferramentas\n🔧 Configuração Avançada\n🚪 Sair" | $GUM_BIN choose)

    case "$CHOICE" in
        "🚀 Nova"*) do_wizard ;;
        "📂 Gerenciar"*) do_manage ;;
        "🛠️  Ferramentas"*) do_global_tools ;;
        "🔧 Configuração"*) "$RCLONE_BIN" config ;;
        "🚪 Sair") clear; exit 0 ;;
    esac
done
