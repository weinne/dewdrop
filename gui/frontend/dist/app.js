const stateEl = document.getElementById("trayState");
const listEl = document.getElementById("remoteList");
const toastEl = document.getElementById("toast");
const connectAccountBtn = document.getElementById("connectAccountBtn");
const openSettingsBtn = document.getElementById("openSettingsBtn");
const connectModal = document.getElementById("connectModal");
const providerCards = document.getElementById("providerCards");
const providerInput = document.getElementById("providerInput");
const remoteNameInput = document.getElementById("remoteNameInput");
const confirmConnectBtn = document.getElementById("confirmConnectBtn");
const cancelConnectBtn = document.getElementById("cancelConnectBtn");
const setupModal = document.getElementById("setupModal");
const setupPathInput = document.getElementById("setupPathInput");
const setupAutoStartInput = document.getElementById("setupAutoStartInput");
const confirmSetupBtn = document.getElementById("confirmSetupBtn");
const cancelSetupBtn = document.getElementById("cancelSetupBtn");
const settingsModal = document.getElementById("settingsModal");
const settingsDefaultAutoStart = document.getElementById("settingsDefaultAutoStart");
const settingsPollingInput = document.getElementById("settingsPollingInput");
const saveSettingsBtn = document.getElementById("saveSettingsBtn");
const cancelSettingsBtn = document.getElementById("cancelSettingsBtn");
const remoteSettingsModal = document.getElementById("remoteSettingsModal");
const remoteSettingsSubtitle = document.getElementById("remoteSettingsSubtitle");
const advancedActionsContainer = document.getElementById("advancedActionsContainer");
const closeRemoteSettingsBtn = document.getElementById("closeRemoteSettingsBtn");
const refreshBtn = document.getElementById("refreshBtn");
const remoteStatePanel = document.getElementById("remoteStatePanel");
const remoteStateTitle = document.getElementById("remoteStateTitle");
const remoteStateDescription = document.getElementById("remoteStateDescription");
const remoteDiagnosticsBox = document.getElementById("remoteDiagnosticsBox");
const remoteDiagnosticsText = document.getElementById("remoteDiagnosticsText");

const providers = [
  {
    id: "drive",
    label: "Google Drive",
    description: "Arquivos e colaboração Google",
    icon: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#0F9D58" d="M7.4 3h4.2l5 8.7h-4.2z"/><path fill="#DB4437" d="M6 14.7 8.1 11h8.6l-2.1 3.7z"/><path fill="#F4B400" d="M6.9 16.2 9 12.5l2.8 4.8H7.6z"/></svg>',
  },
  {
    id: "onedrive",
    label: "OneDrive",
    description: "Nuvem Microsoft para arquivos",
    icon: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#1678D0" d="M9.4 10.1a4.3 4.3 0 0 1 8 1.6 3.2 3.2 0 0 1 1.5 6H7.4a4 4 0 0 1 2-7.6z"/></svg>',
  },
  {
    id: "dropbox",
    label: "Dropbox",
    description: "Sincronização de pastas e projetos",
    icon: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#1F6FFF" d="m7 4 5 3.3-5 3.3-5-3.3zm10 0 5 3.3-5 3.3-5-3.3zm-10 7 5 3.3-5 3.3-5-3.3zm10 0 5 3.3-5 3.3-5-3.3z"/><path fill="#0F4FCC" d="m12 18.1 5-3.3 2 1.3-7 4.6-7-4.6 2-1.3z"/></svg>',
  },
  {
    id: "s3",
    label: "Amazon S3",
    description: "Buckets para backup e mídia",
    icon: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#C46B1A" d="M12 3 4.5 6.5v11L12 21l7.5-3.5v-11z"/><path fill="#F1A44F" d="m12 7 4.8 2.2v5.6L12 17l-4.8-2.2V9.2z"/></svg>',
  },
  {
    id: "webdav",
    label: "WebDAV",
    description: "Servidores próprios e NAS",
    icon: '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="8" fill="#17877A"/><path fill="#C9F3EE" d="M8 12h8M12 8c-1.5 1.1-2.3 2.4-2.3 4s.8 2.9 2.3 4m2.3-8c-1.5 1.1-2.3 2.4-2.3 4s.8 2.9 2.3 4" stroke="#C9F3EE" stroke-width="1.2" stroke-linecap="round"/></svg>',
  },
  {
    id: "generic",
    label: "Conta de nuvem",
    description: "Provedor personalizado",
    icon: '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#55738A" d="M8.3 10.2A4 4 0 0 1 16 11.7h.1a3.2 3.2 0 0 1 0 6.4H8a3.5 3.5 0 0 1 .3-7z"/></svg>',
  },
];

const providerById = Object.fromEntries(providers.map((provider) => [provider.id, provider]));

const SETTINGS_KEY = "dewdropSettings";
const LEGACY_SETTINGS_KEY = "rcloneAutoSettings";
const REMOTE_PROVIDERS_KEY = "dewdropRemoteProviders";

let toastTimer;
let setupRemoteName = "";
let currentSnapshot = null;
let currentRemoteSettings = "";

const defaultSettings = {
  defaultAutoStart: true,
  pollingMs: 3000,
};

function loadSettings() {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY) || localStorage.getItem(LEGACY_SETTINGS_KEY);
    if (!raw) {
      return { ...defaultSettings };
    }
    const parsed = JSON.parse(raw);
    return {
      defaultAutoStart: parsed.defaultAutoStart !== false,
      pollingMs: Math.max(1000, Number(parsed.pollingMs || 3000)),
    };
  } catch (err) {
    return { ...defaultSettings };
  }
}

function saveSettings(settings) {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
}

function loadRemoteProviders() {
  try {
    const raw = localStorage.getItem(REMOTE_PROVIDERS_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return {};
    }
    return parsed;
  } catch (err) {
    return {};
  }
}

function saveRemoteProviders(map) {
  localStorage.setItem(REMOTE_PROVIDERS_KEY, JSON.stringify(map));
}

function rememberRemoteProvider(remoteName, providerId) {
  if (!remoteName || !providerId || !providerById[providerId]) {
    return;
  }
  remoteProviders[remoteName] = providerId;
  saveRemoteProviders(remoteProviders);
}

function inferProviderFromName(remoteName) {
  const value = String(remoteName || "").toLowerCase();
  if (value.includes("onedrive")) {
    return "onedrive";
  }
  if (value.includes("dropbox")) {
    return "dropbox";
  }
  if (value.includes("webdav")) {
    return "webdav";
  }
  if (value.includes("s3") || value.includes("aws")) {
    return "s3";
  }
  if (value.includes("drive")) {
    return "drive";
  }
  return "generic";
}

function getProviderForRemote(remoteName) {
  const known = remoteProviders[remoteName];
  if (known && providerById[known]) {
    return providerById[known];
  }

  const guessedId = inferProviderFromName(remoteName);
  if (guessedId !== "generic") {
    rememberRemoteProvider(remoteName, guessedId);
  }
  return providerById[guessedId] || providerById.generic;
}

function syncProviderMapWithSnapshot(remotes) {
  const activeNames = new Set((remotes || []).map((item) => item.name));
  const previousNames = Object.keys(remoteProviders);
  let changed = false;
  previousNames.forEach((name) => {
    if (!activeNames.has(name)) {
      delete remoteProviders[name];
      changed = true;
    }
  });
  if (changed) {
    saveRemoteProviders(remoteProviders);
  }
}

let appSettings = loadSettings();
let remoteProviders = loadRemoteProviders();

function setSelectedProvider(providerId) {
  if (!providerInput || !providerCards) {
    return;
  }

  const normalized = providerById[providerId] ? providerId : "drive";
  providerInput.value = normalized;

  providerCards.querySelectorAll(".provider-card").forEach((button) => {
    const active = button.dataset.provider === normalized;
    button.classList.toggle("selected", active);
    button.setAttribute("aria-selected", active ? "true" : "false");
  });
}

function renderProviderCards() {
  if (!providerCards || !providerInput) {
    return;
  }

  providerCards.innerHTML = "";

  providers
    .filter((provider) => provider.id !== "generic")
    .forEach((provider) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "provider-card";
      button.dataset.provider = provider.id;
      button.setAttribute("role", "option");
      button.setAttribute("aria-selected", "false");
      button.innerHTML = `
        <span class="provider-icon" aria-hidden="true">${provider.icon}</span>
        <span class="provider-copy">
          <strong>${provider.label}</strong>
          <small>${provider.description}</small>
        </span>
      `;
      button.addEventListener("click", () => setSelectedProvider(provider.id));
      providerCards.appendChild(button);
    });

  setSelectedProvider(providerInput.value || "drive");
}

function showToast(message, isError = false) {
  if (!toastEl) {
    return;
  }

  clearTimeout(toastTimer);
  toastEl.textContent = message;
  toastEl.className = isError ? "toast visible error" : "toast visible";
  toastTimer = setTimeout(() => {
    toastEl.className = "toast";
  }, 3500);
}

function extractErrorMessage(err, fallback) {
  if (!err) {
    return fallback;
  }

  if (typeof err === "string" && err.trim()) {
    return err.trim();
  }

  if (typeof err.message === "string" && err.message.trim()) {
    return err.message.trim();
  }

  try {
    const raw = JSON.stringify(err);
    if (raw && raw !== "{}") {
      return raw;
    }
  } catch (_) {
    // ignore stringify errors
  }

  return fallback;
}

function paintTrayState(state) {
  if (state === "syncing") {
    stateEl.textContent = "sincronizando";
  } else if (state === "error") {
    stateEl.textContent = "revisar";
  } else {
    stateEl.textContent = "operacional";
  }

  stateEl.style.color = state === "syncing" ? "#2b70c9" : state === "error" ? "#bc4b2f" : "#59707b";
}

function summarizeIssue(message) {
  const value = String(message || "").trim();
  if (!value) {
    return "sem detalhes adicionais";
  }

  const normalized = value.toLowerCase();
  if (normalized.includes("not loaded") || normalized.includes("not found") || normalized.includes("could not be found")) {
    return "unidade de serviço ausente ou não carregada";
  }
  if (normalized.includes("permission denied") || normalized.includes("access denied")) {
    return "falta de permissão para operar com o systemd do usuário";
  }
  if (normalized.includes("fusermount") || normalized.includes("fuse")) {
    return "dependência FUSE indisponível";
  }
  if (normalized.includes("rclone")) {
    return "falha ao executar o rclone ou conta com configuração incompleta";
  }

  const compact = value.split(";")[0].split("\n")[0].trim();
  if (compact.length > 130) {
    return `${compact.slice(0, 127)}...`;
  }
  return compact;
}

function setDiagnosticsOutput(text, isError = false) {
  if (!remoteDiagnosticsBox || !remoteDiagnosticsText) {
    return;
  }

  remoteDiagnosticsText.textContent = text || "Sem diagnóstico disponível.";
  remoteDiagnosticsBox.classList.remove("hidden");
  remoteDiagnosticsBox.classList.toggle("error", Boolean(isError));
}

function clearDiagnosticsOutput() {
  if (!remoteDiagnosticsBox || !remoteDiagnosticsText) {
    return;
  }

  remoteDiagnosticsText.textContent = "";
  remoteDiagnosticsBox.classList.add("hidden");
  remoteDiagnosticsBox.classList.remove("error");
}

async function runRemoteDiagnosis(remoteName, options = {}) {
  if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
    return;
  }

  try {
    const result = await window.go.desktop.Application.ExecuteAction("diagnose-remote", remoteName);
    const details = (result?.message || "").trim();
    setDiagnosticsOutput(details || "Nenhum detalhe retornado pelo diagnóstico.");
    if (!options.silent) {
      showToast("Diagnóstico atualizado.");
    }
  } catch (err) {
    console.error(err);
    const message = extractErrorMessage(err, "Não foi possível gerar diagnóstico para esta conta.");
    setDiagnosticsOutput(message, true);
    if (!options.silent) {
      showToast(message, true);
    }
  }
}

function actionButton(label, actionType, remote, options = {}) {
  const btn = document.createElement("button");
  btn.textContent = label;
  btn.className = options.buttonClass || "action-btn";
  btn.addEventListener("click", async () => {
    if (options.confirmMessage && !window.confirm(options.confirmMessage)) {
      return;
    }

    if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
      return;
    }
    try {
      if (options.extended) {
        await window.go.desktop.Application.ExecuteActionWithOptions(
          actionType,
          remote,
          options.localPath || "",
          Boolean(options.autoStart),
        );
      } else {
        await window.go.desktop.Application.ExecuteAction(actionType, remote);
      }

      if (options.successMessage) {
        showToast(options.successMessage);
      }

      const snapshot = await refreshSnapshot();
      if (
        options.keepRemoteSettingsSynced &&
        currentRemoteSettings === remote &&
        !remoteSettingsModal.classList.contains("hidden")
      ) {
        openRemoteSettings(remote, snapshot || currentSnapshot);
      }

      if (typeof options.onSuccess === "function") {
        await options.onSuccess(snapshot || currentSnapshot);
      }
    } catch (err) {
      console.error(err);
      showToast(extractErrorMessage(err, "Não foi possível concluir essa ação."), true);
    }
  });
  return btn;
}

function setupActionButton(label, mode, remote, options = {}) {
  return actionButton(label, mode, remote.name, {
    buttonClass: options.buttonClass || "action-btn accent",
    extended: true,
    autoStart: Boolean(options.autoStart),
    localPath: options.localPath || remote.localPath || "",
    successMessage: options.successMessage || "Configuração aplicada com sucesso.",
    keepRemoteSettingsSynced: options.keepRemoteSettingsSynced,
  });
}

function openFolderButton(remote, className = "action-btn") {
  const btn = document.createElement("button");
  btn.textContent = "Abrir pasta";
  btn.className = className;
  btn.addEventListener("click", async () => {
    if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
      return;
    }

    try {
      await window.go.desktop.Application.OpenLocalFolder(remote);
    } catch (err) {
      console.error(err);
      showToast(extractErrorMessage(err, "Não foi possível abrir a pasta local."), true);
    }
  });

  return btn;
}

function getMode(remote) {
  if (remote.mountActive) {
    return "mount";
  }
  if (remote.syncActive || remote.syncTimerEnabled) {
    return "sync";
  }
  return "disabled";
}

function getModeBadge(remote) {
  const mode = getMode(remote);
  if (mode === "mount") {
    return { value: "Montagem ativa", variant: "mode-mount" };
  }
  if (mode === "sync") {
    return {
      value: remote.syncActive ? "Sincronização em execução" : "Sincronização agendada",
      variant: "mode-sync",
    };
  }
  return { value: "Serviço desativado", variant: "mode-disabled" };
}

function getAutoStartBadge(remote) {
  const mode = getMode(remote);
  if (mode === "sync") {
    return remote.syncTimerEnabled
      ? { value: "Sync inicia com a sessão", variant: "ok" }
      : { value: "Sync manual", variant: "muted" };
  }

  return remote.mountEnabled
    ? { value: "Montagem inicia com a sessão", variant: "ok" }
    : { value: "Montagem manual", variant: "muted" };
}

function getHealthBadge(remote) {
  if (remote.lastError) {
    return { value: "Requer diagnóstico", variant: "warning", title: remote.lastError };
  }
  return { value: "Saudável", variant: "ok", title: "Sem erros recentes" };
}

function createStatusGlyph(icon, title, variant = "neutral") {
  const item = document.createElement("span");
  item.className = `status-glyph ${variant}`;
  item.title = title;
  item.setAttribute("aria-label", title);
  item.innerHTML = icon;
  return item;
}

function modeGlyph(remote) {
  const mode = getMode(remote);
  if (mode === "mount") {
    return createStatusGlyph(
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7.5a2 2 0 0 1 2-2h4l1.6 1.8H19a2 2 0 0 1 2 2V17a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>',
      "Modo: montagem ativa",
      "ok",
    );
  }
  if (mode === "sync") {
    return createStatusGlyph(
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7.5 7A6.5 6.5 0 0 1 19 10h1.8l-2.8 3-2.8-3H17a5 5 0 0 0-8.7-2.8zM6 14a5 5 0 0 0 8.7 2.8l1.8 1.2A6.5 6.5 0 0 1 5 14H3.2l2.8-3 2.8 3z"/></svg>',
      remote.syncActive ? "Modo: sincronização em execução" : "Modo: sincronização agendada",
      "sync",
    );
  }

  return createStatusGlyph(
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 12h12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>',
    "Modo: serviço desativado",
    "muted",
  );
}

function autoStartGlyph(remote) {
  const mode = getMode(remote);
  const enabled = mode === "sync" ? remote.syncTimerEnabled : remote.mountEnabled;
  return createStatusGlyph(
    enabled
      ? '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/></svg>'
      : '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 7l10 10M17 7 7 17" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/></svg>',
    enabled ? "Inicialização automática ativa" : "Inicialização automática desativada",
    enabled ? "ok" : "muted",
  );
}

function healthGlyph(remote) {
  if (remote.lastError) {
    return createStatusGlyph(
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 2.8 19h18.4z"/><path d="M12 9v4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" fill="none"/><circle cx="12" cy="16" r="1"/></svg>',
      `Saúde: requer diagnóstico. ${summarizeIssue(remote.lastError)}`,
      "warn",
    );
  }
  return createStatusGlyph(
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4 4L19 6" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    "Saúde: sem falhas detectadas",
    "ok",
  );
}

function iconActionButton(icon, title, onClick, variant = "") {
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = `icon-action ${variant}`.trim();
  btn.title = title;
  btn.setAttribute("aria-label", title);
  btn.innerHTML = icon;
  btn.addEventListener("click", onClick);
  return btn;
}

function createStatusItem(label, info) {
  const item = document.createElement("div");
  item.className = `status-item ${info.variant || "muted"}`;
  if (info.title) {
    item.title = info.title;
  }

  const key = document.createElement("small");
  key.textContent = label;

  const value = document.createElement("strong");
  value.textContent = info.value;

  item.appendChild(key);
  item.appendChild(value);
  return item;
}

function buildActionSection(title, description) {
  const section = document.createElement("section");
  section.className = "advanced-section";

  const heading = document.createElement("h4");
  heading.textContent = title;

  const text = document.createElement("p");
  text.textContent = description;

  const actions = document.createElement("div");
  actions.className = "advanced-actions";

  section.appendChild(heading);
  section.appendChild(text);
  section.appendChild(actions);

  return { section, actions };
}

function applyRemoteStatePanel(remote) {
  const mode = getMode(remote);

  if (remote.lastError) {
    remoteStatePanel.className = "remote-state-panel state-warning";
    remoteStateTitle.textContent = "Requer revisão";
    remoteStateDescription.textContent = `Motivo provável: ${summarizeIssue(remote.lastError)}. Rode o diagnóstico para detalhes e use a correção automática se necessário.`;
    return;
  }

  if (mode === "mount") {
    remoteStatePanel.className = "remote-state-panel state-mount";
    remoteStateTitle.textContent = "Montagem ativa";
    remoteStateDescription.textContent = "Os arquivos estão acessíveis sob demanda. Ideal para economizar espaço local e navegar rapidamente.";
    return;
  }

  if (mode === "sync") {
    remoteStatePanel.className = "remote-state-panel state-sync";
    remoteStateTitle.textContent = remote.syncActive ? "Sincronização em execução" : "Sincronização agendada";
    remoteStateDescription.textContent = "A conta mantém cópia local e sincronização automática. Bom para trabalho offline e backup contínuo.";
    return;
  }

  remoteStatePanel.className = "remote-state-panel state-disabled";
  remoteStateTitle.textContent = "Serviço desativado";
  remoteStateDescription.textContent = "Escolha um modo para ativar esta conta: montagem sob demanda ou sincronização contínua.";
}

function paintRemotes(remotes) {
  listEl.innerHTML = "";
  if (!Array.isArray(remotes) || remotes.length === 0) {
    const empty = document.createElement("p");
    empty.className = "empty";
    empty.textContent = "Nenhuma conta conectada ainda. Use o botão Conectar conta na nuvem para começar.";
    listEl.appendChild(empty);
    return;
  }

  remotes.forEach((remote) => {
    const provider = getProviderForRemote(remote.name);
    const card = document.createElement("article");
    card.className = "remote-card";

    const head = document.createElement("div");
    head.className = "remote-head";

    const icon = document.createElement("div");
    icon.className = "remote-icon";
    icon.innerHTML = provider.icon;

    const identity = document.createElement("div");
    identity.className = "remote-identity";

    const h3 = document.createElement("h3");
    h3.textContent = remote.name;

    const providerLine = document.createElement("small");
    providerLine.className = "provider-line";
    providerLine.textContent = provider.label;

    identity.appendChild(h3);
    identity.appendChild(providerLine);

    const settingsBtn = iconActionButton(
      '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 8.3A3.7 3.7 0 1 1 8.3 12 3.7 3.7 0 0 1 12 8.3m9 3.7-2.1-.6a7.6 7.6 0 0 0-.7-1.7l1.2-1.8-1.8-1.8-1.8 1.2a7.6 7.6 0 0 0-1.7-.7L12 3h-2l-.6 2.1a7.6 7.6 0 0 0-1.7.7L5.9 4.6 4.1 6.4l1.2 1.8a7.6 7.6 0 0 0-.7 1.7L2.5 12v2l2.1.6a7.6 7.6 0 0 0 .7 1.7l-1.2 1.8 1.8 1.8 1.8-1.2a7.6 7.6 0 0 0 1.7.7l.6 2.1h2l.6-2.1a7.6 7.6 0 0 0 1.7-.7l1.8 1.2 1.8-1.8-1.2-1.8a7.6 7.6 0 0 0 .7-1.7L21 14z"/></svg>',
      "Configurar conta",
      () => openRemoteSettings(remote.name),
      "ghost",
    );

    head.appendChild(icon);
    head.appendChild(identity);
    head.appendChild(settingsBtn);

    const stateStrip = document.createElement("div");
    stateStrip.className = "status-strip";
    stateStrip.appendChild(modeGlyph(remote));
    stateStrip.appendChild(autoStartGlyph(remote));
    stateStrip.appendChild(healthGlyph(remote));

    const actions = document.createElement("div");
    actions.className = "card-actions";

    const mode = getMode(remote);
    if (mode !== "disabled") {
      actions.appendChild(
        iconActionButton(
          '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7.5a2 2 0 0 1 2-2h4l1.6 1.8H19a2 2 0 0 1 2 2V17a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>',
          "Abrir pasta",
          async () => {
            try {
              await window.go.desktop.Application.OpenLocalFolder(remote.name);
            } catch (err) {
              console.error(err);
              showToast(extractErrorMessage(err, "Não foi possível abrir a pasta local."), true);
            }
          },
          "primary",
        ),
      );
    } else {
      actions.appendChild(
        iconActionButton(
          '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 5v14l11-7z"/></svg>',
          "Ativar montagem recomendada",
          async () => {
            try {
              await window.go.desktop.Application.ExecuteActionWithOptions(
                "setup-mount",
                remote.name,
                remote.localPath || "",
                appSettings.defaultAutoStart,
              );
              await refreshSnapshot();
              showToast("Conta ativada em modo montagem.");
            } catch (err) {
              console.error(err);
              showToast(extractErrorMessage(err, "Não foi possível ativar a conta."), true);
            }
          },
          "primary",
        ),
      );
    }

    if (remote.lastError) {
      actions.appendChild(
        iconActionButton(
          '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 2.8 19h18.4z"/><path d="M12 9v4" stroke="currentColor" stroke-width="1.8" fill="none" stroke-linecap="round"/><circle cx="12" cy="16" r="1"/></svg>',
          "Ver diagnóstico",
          () => openRemoteSettings(remote.name),
        ),
      );
    }

    card.appendChild(head);
    card.appendChild(stateStrip);
    card.appendChild(actions);
    listEl.appendChild(card);
  });
}

async function refreshSnapshot() {
  if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
    paintTrayState("offline");
    return null;
  }

  try {
    const snapshot = await window.go.desktop.Application.GetSnapshot();
    currentSnapshot = snapshot;
    syncProviderMapWithSnapshot(snapshot.remotes);
    paintTrayState(snapshot.trayState);
    paintRemotes(snapshot.remotes);
    return snapshot;
  } catch (err) {
    console.error(err);
    paintTrayState("error");
    showToast(extractErrorMessage(err, "Não foi possível ler o status atual."), true);
    return null;
  }
}

function openSetupWizard(remoteName, suggestedPath) {
  setupRemoteName = remoteName;
  setupPathInput.value = suggestedPath || "";
  setupAutoStartInput.checked = appSettings.defaultAutoStart;

  document.querySelectorAll('input[name="setupMode"]').forEach((input) => {
    input.checked = input.value === "setup-mount";
  });

  updateModeSelectionUI();

  setupModal.classList.remove("hidden");
}

function closeSetupWizard() {
  setupModal.classList.add("hidden");
}

refreshBtn.addEventListener("click", refreshSnapshot);

function updateModeSelectionUI() {
  document.querySelectorAll(".mode-option").forEach((option) => {
    const input = option.querySelector('input[name="setupMode"]');
    option.classList.toggle("selected", Boolean(input?.checked));
  });
}

document.querySelectorAll('input[name="setupMode"]').forEach((input) => {
  input.addEventListener("change", updateModeSelectionUI);
});

openSettingsBtn.addEventListener("click", () => {
  settingsDefaultAutoStart.checked = appSettings.defaultAutoStart;
  settingsPollingInput.value = String(appSettings.pollingMs);
  settingsModal.classList.remove("hidden");
});

cancelSettingsBtn.addEventListener("click", () => {
  settingsModal.classList.add("hidden");
});

saveSettingsBtn.addEventListener("click", async () => {
  const polling = Math.max(1000, Number(settingsPollingInput.value || 3000));
  appSettings = {
    defaultAutoStart: Boolean(settingsDefaultAutoStart.checked),
    pollingMs: polling,
  };
  saveSettings(appSettings);
  settingsModal.classList.add("hidden");

  if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
    return;
  }

  try {
    await window.go.desktop.Application.SetPollingInterval(appSettings.pollingMs);
    showToast("Atualização automática ajustada.");
  } catch (err) {
    console.error(err);
    showToast(extractErrorMessage(err, "Não foi possível alterar o intervalo."), true);
  }
});

connectAccountBtn.addEventListener("click", async () => {
  renderProviderCards();
  connectModal.classList.remove("hidden");
  remoteNameInput.focus();
});

cancelConnectBtn.addEventListener("click", () => {
  connectModal.classList.add("hidden");
});

confirmConnectBtn.addEventListener("click", async () => {
  if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
    return;
  }

  const provider = (providerInput?.value || "").trim();
  const remoteName = (remoteNameInput.value || "").trim();

  if (!remoteName) {
    showToast("Digite um nome para a conta.", true);
    return;
  }

  try {
    confirmConnectBtn.disabled = true;
    confirmConnectBtn.textContent = "Conectando...";

    await window.go.desktop.Application.CreateCloudRemote(remoteName, provider);
    rememberRemoteProvider(remoteName, provider);
    connectModal.classList.add("hidden");
    remoteNameInput.value = "";

    const snapshot = await refreshSnapshot();
    const newRemote = snapshot?.remotes?.find((item) => item.name === remoteName);
    const suggestedPath = newRemote?.localPath || "";

    showToast("Conta conectada. Vamos escolher como usar.");
    openSetupWizard(remoteName, suggestedPath);
  } catch (err) {
    console.error(err);
    showToast(extractErrorMessage(err, "Falha ao conectar conta. Verifique permissões/login no navegador."), true);
  } finally {
    confirmConnectBtn.disabled = false;
    confirmConnectBtn.textContent = "Conectar agora";
  }
});

cancelSetupBtn.addEventListener("click", () => {
  closeSetupWizard();
});

confirmSetupBtn.addEventListener("click", async () => {
  if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
    return;
  }

  const selectedMode = document.querySelector('input[name="setupMode"]:checked')?.value || "setup-mount";
  const localPath = (setupPathInput.value || "").trim();
  const autoStart = Boolean(setupAutoStartInput.checked);

  if (!setupRemoteName) {
    showToast("Conta inválida para configurar.", true);
    return;
  }

  try {
    confirmSetupBtn.disabled = true;
    confirmSetupBtn.textContent = "Aplicando...";

    await window.go.desktop.Application.ExecuteActionWithOptions(selectedMode, setupRemoteName, localPath, autoStart);
    closeSetupWizard();
    await refreshSnapshot();
    showToast("Tudo pronto. Sua conta já está configurada.");
  } catch (err) {
    console.error(err);
    showToast(extractErrorMessage(err, "Não foi possível finalizar a configuração."), true);
  } finally {
    confirmSetupBtn.disabled = false;
    confirmSetupBtn.textContent = "Aplicar e começar";
  }
});

if (window.runtime && typeof window.runtime.EventsOn === "function") {
  window.runtime.EventsOn("snapshot", (snapshot) => {
    currentSnapshot = snapshot;
    syncProviderMapWithSnapshot(snapshot.remotes);
    paintTrayState(snapshot.trayState);
    paintRemotes(snapshot.remotes);
  });
}

closeRemoteSettingsBtn.addEventListener("click", () => {
  remoteSettingsModal.classList.add("hidden");
  clearDiagnosticsOutput();
});

function openRemoteSettings(remoteName, snapshot = currentSnapshot) {
  currentRemoteSettings = remoteName;
  remoteSettingsSubtitle.textContent = `Ações avançadas para ${remoteName}.`;
  advancedActionsContainer.innerHTML = "";
  clearDiagnosticsOutput();

  const remote = snapshot?.remotes?.find((item) => item.name === remoteName);
  if (!remote) {
    showToast("Conta não encontrada.", true);
    return;
  }

  applyRemoteStatePanel(remote);

  const operation = buildActionSection("Operação", "Defina como esta conta deve funcionar no dia a dia.");
  const mode = getMode(remote);

  if (mode === "mount") {
    operation.actions.appendChild(
      actionButton("Parar montagem", "stop-mount", remoteName, {
        buttonClass: "action-btn danger",
        successMessage: "Montagem pausada.",
        keepRemoteSettingsSynced: true,
      }),
    );
    operation.actions.appendChild(
      setupActionButton("Trocar para sincronização", "setup-sync", remote, {
        buttonClass: "action-btn accent",
        autoStart: remote.syncTimerEnabled || appSettings.defaultAutoStart,
        localPath: remote.localPath,
        keepRemoteSettingsSynced: true,
      }),
    );
  } else if (mode === "sync") {
    const shouldPause = remote.syncActive || remote.syncTimerEnabled;
    operation.actions.appendChild(
      actionButton(shouldPause ? "Pausar sincronização" : "Iniciar sincronização", shouldPause ? "stop-sync" : "start-sync", remoteName, {
        buttonClass: shouldPause ? "action-btn danger" : "action-btn",
        successMessage: shouldPause ? "Sincronização pausada." : "Sincronização iniciada.",
        keepRemoteSettingsSynced: true,
      }),
    );
    operation.actions.appendChild(
      setupActionButton("Trocar para montagem", "setup-mount", remote, {
        buttonClass: "action-btn accent",
        autoStart: remote.mountEnabled || appSettings.defaultAutoStart,
        localPath: remote.localPath,
        keepRemoteSettingsSynced: true,
      }),
    );
  } else {
    operation.actions.appendChild(
      setupActionButton("Ativar montagem", "setup-mount", remote, {
        buttonClass: "action-btn accent",
        autoStart: appSettings.defaultAutoStart,
        localPath: remote.localPath,
        keepRemoteSettingsSynced: true,
      }),
    );
    operation.actions.appendChild(
      setupActionButton("Ativar sincronização", "setup-sync", remote, {
        buttonClass: "action-btn accent alt",
        autoStart: appSettings.defaultAutoStart,
        localPath: remote.localPath,
        keepRemoteSettingsSynced: true,
      }),
    );
  }

  const automation = buildActionSection("Inicialização", "Controle a ativação automática durante o login da sessão.");
  if (mode === "sync") {
    const syncAutoAction = remote.syncTimerEnabled ? "disable-sync-autostart" : "enable-sync-autostart";
    automation.actions.appendChild(
      actionButton(
        remote.syncTimerEnabled ? "Desativar início automático de sync" : "Ativar início automático de sync",
        syncAutoAction,
        remoteName,
        {
          buttonClass: "action-btn",
          keepRemoteSettingsSynced: true,
          successMessage: "Preferência de início automático atualizada.",
        },
      ),
    );
  } else {
    const mountAutoAction = remote.mountEnabled ? "disable-mount-autostart" : "enable-mount-autostart";
    automation.actions.appendChild(
      actionButton(
        remote.mountEnabled ? "Desativar início automático de montagem" : "Ativar início automático de montagem",
        mountAutoAction,
        remoteName,
        {
          buttonClass: "action-btn",
          keepRemoteSettingsSynced: true,
          successMessage: "Preferência de início automático atualizada.",
        },
      ),
    );
  }

  const utilities = buildActionSection("Utilitários", "Acesso rápido para inspeção e verificação local.");
  utilities.actions.appendChild(openFolderButton(remoteName, "action-btn subtle"));

  const recovery = buildActionSection("Diagnóstico e recuperação", "Entenda o motivo da falha, tente corrigir automaticamente ou remova os serviços desta conta.");

  const diagnoseBtn = document.createElement("button");
  diagnoseBtn.className = "action-btn";
  diagnoseBtn.textContent = "Entender o problema";
  diagnoseBtn.addEventListener("click", () => runRemoteDiagnosis(remoteName, { silent: false }));
  recovery.actions.appendChild(diagnoseBtn);

  recovery.actions.appendChild(
    actionButton("Tentar correção automática", "repair-remote", remoteName, {
      buttonClass: "action-btn accent",
      keepRemoteSettingsSynced: true,
      successMessage: "Tentativa de correção concluída.",
      onSuccess: async () => {
        await runRemoteDiagnosis(remoteName, { silent: true });
      },
    }),
  );

  recovery.actions.appendChild(
    actionButton("Remover serviço desta conta", "remove-service", remoteName, {
      buttonClass: "action-btn danger",
      keepRemoteSettingsSynced: true,
      successMessage: "Serviços removidos. A conta permanece conectada para nova configuração.",
      confirmMessage: "Isso vai parar e remover as unidades de serviço desta conta. Deseja continuar?",
    }),
  );

  const deleteAccountBtn = document.createElement("button");
  deleteAccountBtn.className = "action-btn danger";
  deleteAccountBtn.textContent = "Deletar conta (remover do rclone)";
  deleteAccountBtn.addEventListener("click", async () => {
    if (!window.go || !window.go.desktop || !window.go.desktop.Application) {
      return;
    }

    const confirmed = window.confirm(
      "Isso vai remover os serviços e deletar a conta do rclone. Você precisará conectar novamente depois. Deseja continuar?",
    );
    if (!confirmed) {
      return;
    }

    try {
      await window.go.desktop.Application.DeleteCloudRemote(remoteName);
      currentRemoteSettings = "";
      remoteSettingsModal.classList.add("hidden");
      clearDiagnosticsOutput();
      await refreshSnapshot();
      showToast("Conta deletada. Você pode conectar novamente.");
    } catch (err) {
      console.error(err);
      showToast(extractErrorMessage(err, "Não foi possível deletar a conta."), true);
    }
  });
  recovery.actions.appendChild(deleteAccountBtn);

  advancedActionsContainer.appendChild(operation.section);
  advancedActionsContainer.appendChild(automation.section);
  advancedActionsContainer.appendChild(utilities.section);
  advancedActionsContainer.appendChild(recovery.section);

  remoteSettingsModal.classList.remove("hidden");

  if (remote.lastError) {
    runRemoteDiagnosis(remoteName, { silent: true });
  }
}

renderProviderCards();
updateModeSelectionUI();

if (window.go && window.go.desktop && window.go.desktop.Application) {
  window.go.desktop.Application.SetPollingInterval(appSettings.pollingMs).catch(() => {});
}

refreshSnapshot();
