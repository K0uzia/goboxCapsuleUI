.PHONY: help deps deps-system deps-go deps-mod update run dev dev-user build fmt vet test check clean

# ---- Configuration ---------------------------------------------------------

APP        := gobox
MAIN_PKG   := ./cmd/gobox
BUILD_DIR  := build
BINARY     := $(BUILD_DIR)/$(APP)

# Paquets compilables (le main.go à la racine est un ancien prototype)
PKGS       := ./cmd/... ./internal/...

# Version de Go à installer (doit correspondre à go.mod)
GO_VERSION := 1.25.2
GO_ARCH    := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
GO_TARBALL := go$(GO_VERSION).linux-$(GO_ARCH).tar.gz

# Utilise le go du PATH, sinon celui de /usr/local/go
GO := $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)

LDFLAGS := -s -w

# Fyne : X11 seul par défaut (Linux Mint). `make GO_TAGS=` compile X11 + Wayland
# (nécessite libwayland-dev et libxkbcommon-dev, installés par `make deps`).
GO_TAGS ?= x11
TAGS    := $(if $(GO_TAGS),-tags $(GO_TAGS))

# ---- Aide ------------------------------------------------------------------

help: ## Affiche cette aide
	@grep -E '^[a-zA-Z_ -]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

# ---- Installation ----------------------------------------------------------

# Outils système appelés par les sondes (lspci, dmidecode, xrandr, stress-ng)
# + compilateur C et bibliothèques OpenGL/X11 pour l'interface graphique (Fyne)
APT_PACKAGES := curl pciutils dmidecode x11-xserver-utils stress-ng \
                gcc pkg-config libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev \
                xorg-dev libwayland-dev libxkbcommon-dev

deps: deps-system deps-go deps-mod ## Installe TOUT ce dont le projet a besoin
	@echo "\nOK : dépendances installées. Lance 'make dev'."
	@command -v go >/dev/null || \
		echo "Astuce : ajoute 'export PATH=\$$PATH:/usr/local/go/bin' à ~/.bashrc pour utiliser go directement"

deps-system: ## Installe les outils système (apt)
	sudo apt-get update
	sudo apt-get install -y $(APT_PACKAGES)

deps-go: ## Installe Go $(GO_VERSION) si absent ou mauvaise version
	@if $(GO) version 2>/dev/null | grep -q "go$(GO_VERSION) "; then \
		echo "Go $(GO_VERSION) déjà installé"; \
	else \
		echo "Installation de Go $(GO_VERSION)..."; \
		curl -fsSL -o /tmp/$(GO_TARBALL) https://go.dev/dl/$(GO_TARBALL) && \
		sudo rm -rf /usr/local/go && \
		sudo tar -C /usr/local -xzf /tmp/$(GO_TARBALL) && \
		rm -f /tmp/$(GO_TARBALL); \
	fi

deps-mod: ## Télécharge les modules Go (go.mod)
	$(GO) mod download
	$(GO) mod verify

update: ## Met à jour toutes les dépendances Go vers leur dernière version
	$(GO) get -u ./cmd/... ./internal/...
	$(GO) mod tidy

# ---- Développement ---------------------------------------------------------

# go run compile dans un dossier temporaire (pas de fichier dans build/) puis
# lance via sudo en gardant l'accès à l'écran (DISPLAY / XAUTHORITY)
XAUTH    := $(or $(XAUTHORITY),$(HOME)/.Xauthority)
SUDO_GUI := sudo env DISPLAY=$(DISPLAY) XAUTHORITY=$(XAUTH)

dev run: ## Lance l'application sans build (sudo pour lire dmidecode...)
	$(GO) run $(TAGS) -exec "$(SUDO_GUI)" $(MAIN_PKG)

dev-user: ## Lance l'application sans sudo (détails RAM indisponibles)
	$(GO) run $(TAGS) $(MAIN_PKG)

# ---- Build -----------------------------------------------------------------

build: ## Compile le binaire optimisé dans build/
	@mkdir -p $(BUILD_DIR)
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
	rm -rf $(BUILD_DIR)
	$(GO) clean -cache
