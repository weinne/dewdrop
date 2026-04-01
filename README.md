# Dewdrop

Gerenciador Linux para nuvem baseado em rclone, com foco em simplicidade para uso diário.

## O que é

Dewdrop é a evolução GUI do projeto rclone-auto:

1. Conecta contas de nuvem com menos atrito.
2. Ativa modo montagem (on-demand) ou sincronização (cópia local + timer).
3. Orquestra tudo via `systemd --user`.
4. Mantém compatibilidade com convenções já usadas no projeto original.

## Status atual

O projeto já tem aplicação desktop funcional com backend em Go + shell Wails.

Componentes principais:

1. Dashboard para listar contas e estado.
2. Ações por conta (ativar/parar/diagnosticar/reparar/remover serviço).
3. Criação de remote, incluindo fluxo OneDrive automatizado no backend.
4. Empacotamento para `.deb` e `.rpm`.

Detalhes técnicos da GUI: [gui/README.md](gui/README.md)

## Instalação

### Opção A: instalar pelo script (recomendado)

Script cross-distro: [scripts/install.sh](scripts/install.sh)

Execução local:

```bash
./scripts/install.sh
```

Comportamento do script:

1. Dentro do repositório: instala dependências, compila e instala a GUI.
2. Fora do repositório:
   1. `apt`/`dnf`: baixa pacote da release e instala.
   2. `pacman` (Arch-based): baixa source da release, compila e instala.

Variáveis úteis:

```bash
RELEASE_REPO=weinne/dewdrop RELEASE_TAG=v0.1.0 ./scripts/install.sh
WAILS_BUILD_TAGS=production,webkit2_40 ./scripts/install.sh
```

### Opção B: instalar por pacote

Baixe os assets na página de release e instale manualmente:

1. `.deb` para Debian/Ubuntu e derivados.
2. `.rpm` para Fedora/RHEL e derivados.

Releases: https://github.com/weinne/dewdrop/releases

## Build de pacotes

Gerar `.deb` e `.rpm` localmente:

```bash
./scripts/build-packages.sh
```

Override de versão/tags:

```bash
VERSION=0.1.0 WAILS_BUILD_TAGS=production,webkit2_41 ./scripts/build-packages.sh
```

Arquivos de saída:

1. [dist/rclone-auto-gui_0.1.0_amd64.deb](dist/rclone-auto-gui_0.1.0_amd64.deb)
2. [dist/rclone-auto-gui-0.1.0-1.x86_64.rpm](dist/rclone-auto-gui-0.1.0-1.x86_64.rpm)
3. [dist/SHA256SUMS](dist/SHA256SUMS)

## Execução em desenvolvimento

Backend/GUI:

```bash
cd gui
go test ./...
wails dev
```

Build local da aplicação:

```bash
cd gui
WAILS_BUILD_TAGS=production,webkit2_41 go build -tags "$WAILS_BUILD_TAGS" -o build/bin/rclone-auto-gui .
```

## Dependências em runtime

1. `rclone`
2. `systemd --user`
3. `fuse3`

Se algo essencial faltar, o app interrompe a inicialização com mensagem clara.

## Estrutura do repositório

1. [gui](gui): aplicação desktop e backend.
2. [packaging](packaging): nfpm, AUR e artefatos de distribuição.
3. [scripts](scripts): utilitários de build e instalação.
4. [rclone-auto.sh](rclone-auto.sh): fluxo TUI legado (mantido por compatibilidade).

## Fluxo git recomendado (dewdrop)

1. Trabalhar em branch de feature.
2. Push para `dewdrop` (repositório novo).
3. Publicar release com assets de pacote e `install.sh`.

## Licença

MIT. Veja [LICENSE.md](LICENSE.md).

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
