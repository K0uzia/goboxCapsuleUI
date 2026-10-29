.PHONY: help deps deps-system deps-go deps-mod update run dev dev-user build fmt vet test check clean

# ---- Configuration ---------------------------------------------------------

APP        := gobox
MAIN_PKG   := ./cmd/gobox
BUILD_DIR  := build

# Paquets compilables (le main.go à la racine est un ancien prototype)
PKGS       := ./cmd/... ./internal/...

# Version de Go à installer (doit correspondre à go.mod)
GO_VERSION := 1.25.2

LDFLAGS := -s -w

# ---- Détection de l'OS -----------------------------------------------------
#
# Windows : les recettes passent par PowerShell (fonctionne depuis PowerShell,
# cmd ou Git Bash). Linux : shell POSIX habituel.

ifeq ($(OS),Windows_NT)
  PLATFORM    := windows
  SHELL       := powershell.exe
  .SHELLFLAGS := -NoProfile -ExecutionPolicy Bypass -Command
  EXE         := .exe

  # Utilise le go du PATH, sinon celui de l'installeur officiel
  GO_DEFAULT  := C:/Program Files/Go/bin/go.exe
  GO_BIN      := $(if $(shell (Get-Command go -ErrorAction SilentlyContinue).Source),go,& '$(GO_DEFAULT)')

  # Fyne (OpenGL) exige CGO, donc gcc. Si gcc n'est pas encore dans le PATH
  # (terminal ouvert avant `make deps`), on prend celui installé par winget.
  GCC_DIR     := $(if $(shell (Get-Command gcc -ErrorAction SilentlyContinue).Source),,$(patsubst %/gcc.exe,%,$(firstword $(wildcard $(subst \,/,$(LOCALAPPDATA))/Microsoft/WinGet/Packages/BrechtSanders.WinLibs*/mingw64/bin/gcc.exe))))
  GO_ENV      := $$env:CGO_ENABLED='1';$(if $(GCC_DIR), $$env:Path='$(GCC_DIR);'+$$env:Path;)
  GO          := $(GO_ENV) $(GO_BIN)

  # Fyne sous Windows : pas de tag X11
  GO_TAGS     ?=

  MKDIR_BUILD := New-Item -ItemType Directory -Force -Path '$(BUILD_DIR)' | Out-Null
  RM_BUILD    := if (Test-Path '$(BUILD_DIR)') { Remove-Item -Recurse -Force '$(BUILD_DIR)' }
else
  PLATFORM    := linux
  EXE         :=

  GO_ARCH     := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
  GO_TARBALL  := go$(GO_VERSION).linux-$(GO_ARCH).tar.gz

  # Utilise le go du PATH, sinon celui de /usr/local/go
  GO          := $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)

  # Fyne : X11 seul par défaut (Linux Mint). `make GO_TAGS=` compile X11 + Wayland
  # (nécessite libwayland-dev et libxkbcommon-dev, installés par `make deps`).
  GO_TAGS     ?= x11

  MKDIR_BUILD := mkdir -p $(BUILD_DIR)
  RM_BUILD    := rm -rf $(BUILD_DIR)
endif

BINARY := $(BUILD_DIR)/$(APP)$(EXE)
TAGS   := $(if $(GO_TAGS),-tags $(GO_TAGS))

# ---- Aide ------------------------------------------------------------------

help: ## Affiche cette aide
ifeq ($(PLATFORM),windows)
	@Select-String -Path '$(firstword $(MAKEFILE_LIST))' -Pattern '^([a-zA-Z_ -]+):.*?## (.*)$$' | ForEach-Object { '  {0,-12} {1}' -f $$_.Matches[0].Groups[1].Value, $$_.Matches[0].Groups[2].Value }
else
	@grep -E '^[a-zA-Z_ -]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
endif

# ---- Installation ----------------------------------------------------------

# Outils système appelés par les sondes (lspci, dmidecode, xrandr, stress-ng)
# + compilateur C et bibliothèques OpenGL/X11 pour l'interface graphique (Fyne)
APT_PACKAGES := curl pciutils dmidecode x11-xserver-utils stress-ng \
                gcc pkg-config libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev \
                xorg-dev libwayland-dev libxkbcommon-dev

# Windows : Fyne a seulement besoin d'un compilateur C (MinGW-w64 via winget)
WINGET_GCC := BrechtSanders.WinLibs.POSIX.UCRT

deps: deps-system deps-go deps-mod ## Installe TOUT ce dont le projet a besoin
ifeq ($(PLATFORM),windows)
	@Write-Host "`nOK : dépendances installées. Ouvre un nouveau terminal (PATH) puis lance 'make dev'."
else
	@echo "\nOK : dépendances installées. Lance 'make dev'."
	@command -v go >/dev/null || \
		echo "Astuce : ajoute 'export PATH=\$$PATH:/usr/local/go/bin' à ~/.bashrc pour utiliser go directement"
endif

deps-system: ## Installe les outils système (apt / winget)
ifeq ($(PLATFORM),windows)
	@if ((Get-Command gcc -ErrorAction SilentlyContinue) -or '$(GCC_DIR)') { 'gcc déjà installé' } else { winget install -e --id $(WINGET_GCC) --accept-source-agreements --accept-package-agreements; if ($$LASTEXITCODE) { exit $$LASTEXITCODE } }
else
	sudo apt-get update
	sudo apt-get install -y $(APT_PACKAGES)
endif

deps-go: ## Installe Go $(GO_VERSION) si absent ou mauvaise version
ifeq ($(PLATFORM),windows)
	@if ((Get-Command go -ErrorAction SilentlyContinue) -and ((go version) -match 'go$(GO_VERSION) ')) { 'Go $(GO_VERSION) déjà installé' } else { 'Installation de Go $(GO_VERSION)...'; winget install -e --id GoLang.Go --version $(GO_VERSION) --accept-source-agreements --accept-package-agreements; if ($$LASTEXITCODE) { exit $$LASTEXITCODE } }
else
	@if $(GO) version 2>/dev/null | grep -q "go$(GO_VERSION) "; then \
		echo "Go $(GO_VERSION) déjà installé"; \
	else \
		echo "Installation de Go $(GO_VERSION)..."; \
		curl -fsSL -o /tmp/$(GO_TARBALL) https://go.dev/dl/$(GO_TARBALL) && \
		sudo rm -rf /usr/local/go && \
		sudo tar -C /usr/local -xzf /tmp/$(GO_TARBALL) && \
		rm -f /tmp/$(GO_TARBALL); \
	fi
endif

deps-mod: ## Télécharge les modules Go (go.mod)
	$(GO) mod download
	$(GO) mod verify

update: ## Met à jour toutes les dépendances Go vers leur dernière version
	$(GO) get -u $(PKGS)
	$(GO) mod tidy

# ---- Développement ---------------------------------------------------------

# go run compile dans un dossier temporaire (pas de fichier dans build/) puis
# lance via sudo en gardant l'accès à l'écran (DISPLAY / XAUTHORITY)
XAUTH    := $(or $(XAUTHORITY),$(HOME)/.Xauthority)
SUDO_GUI := sudo env DISPLAY=$(DISPLAY) XAUTHORITY=$(XAUTH)

dev run: ## Lance l'application sans build (sudo sous Linux pour lire dmidecode...)
ifeq ($(PLATFORM),windows)
	@Write-Host 'Compilation... (la 1re fois, Fyne/OpenGL via gcc peut prendre 2-3 min)'
	$(GO) run $(TAGS) $(MAIN_PKG)
else
	$(GO) run $(TAGS) -exec "$(SUDO_GUI)" $(MAIN_PKG)
endif

dev-user: ## Lance l'application sans sudo (détails RAM indisponibles)
	$(GO) run $(TAGS) $(MAIN_PKG)

# ---- Build -----------------------------------------------------------------

build: ## Compile le binaire optimisé dans build/
	@$(MKDIR_BUILD)
	$(GO) build $(TAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN_PKG)
	@echo "Binaire : $(BINARY)"

# ---- Qualité ---------------------------------------------------------------

fmt: ## Formate le code
	$(GO) fmt $(PKGS)

vet: ## Analyse statique
	$(GO) vet $(TAGS) $(PKGS)

test: ## Lance les tests
	$(GO) test $(TAGS) $(PKGS)

check: fmt vet test ## fmt + vet + test

clean: ## Supprime les binaires
	$(RM_BUILD)
	$(GO) clean -cache
