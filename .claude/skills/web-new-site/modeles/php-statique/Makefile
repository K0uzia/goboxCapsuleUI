# SPDX-License-Identifier: GPL-3.0-or-later
# Raccourcis de développement du site (détails : setup-local.sh, .claude/rules/projet.md)
LOCAL ?= bash setup-local.sh
PORT  ?= 8000
PREVIEW_PORT ?= 8001
HOST  ?= localhost
# Domaine de production du projet (canonical, sitemap, audit)
SITE  ?= https://example.org
AUDIT := .claude/skills/web-audit/scripts

.DEFAULT_GOAL := help
.PHONY: help info deps dev open build preview lint check verify repo clean

# ---------- Aide ----------
help: ## Affiche cette aide
	@awk 'BEGIN {FS = ":.*##"; printf "\nCibles disponibles :\n\n"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo

# ---------- Mise en place ----------
info:    ## État local : PHP, extensions, dist/
	@$(LOCAL) info

deps:    ## Installe PHP 8.2+ et mbstring (dnf ou apt)
	$(LOCAL) deps

# ---------- Développement ----------
dev:     ## Serveur de dev, pages rendues à la volée (Ctrl+C pour arrêter)
	PORT=$(PORT) HOST=$(HOST) $(LOCAL) dev

open:    ## Ouvre le site de dev dans le navigateur
	xdg-open http://$(HOST):$(PORT)

build:   ## Génère le site statique dans dist/
	$(LOCAL) build

preview: build ## Construit puis sert dist/ comme l'hébergeur
	PREVIEW_PORT=$(PREVIEW_PORT) HOST=$(HOST) $(LOCAL) preview

# ---------- Qualité ----------
lint:    ## Syntaxe de tous les fichiers PHP
	@for f in $$(find bin src -name '*.php'); do php -l "$$f" >/dev/null || exit 1; done; echo "PHP : syntaxe OK"

check: build ## Build + audit de dist/ (SEO, liens, titres, CSS, poids)
	node $(AUDIT)/check-dist.mjs dist $(SITE)

repo:    ## Aucun secret ni fichier sensible suivi par Git
	node $(AUDIT)/check-repo.mjs

verify: lint check repo ## Tout ce qui doit être vert avant de livrer

clean:   ## Supprime dist/
	rm -rf dist
