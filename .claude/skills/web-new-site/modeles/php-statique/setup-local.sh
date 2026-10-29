#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# ------------------------------------------------------------------------------
# setup-local.sh : développement local du site (PHP sans dépendance).
#   PHP sert de moteur de gabarits : en dev il rend les pages à la volée,
#   au build il les écrit en HTML statique dans dist/ (seul dossier publié).
#
# USAGE :
#   ./setup-local.sh info      # version de PHP, extensions, état de dist/
#   ./setup-local.sh deps      # installe PHP CLI et mbstring (dnf ou apt)
#   ./setup-local.sh dev       # serveur de dev sur http://localhost:${PORT:-8000}
#   ./setup-local.sh build     # génère le site statique dans dist/
#   ./setup-local.sh preview   # sert dist/ comme l'hébergeur, sur ${PREVIEW_PORT:-8001}
#
# ENV : PORT=8000  PREVIEW_PORT=8001  HOST=localhost
# ------------------------------------------------------------------------------

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

PORT="${PORT:-8000}"
PREVIEW_PORT="${PREVIEW_PORT:-8001}"
HOST="${HOST:-localhost}"
EXT_REQUISES=(mbstring)

ok() { printf '✅ %s\n' "$*"; }
die() { printf '❌ %s\n' "$*" >&2; exit 1; }

check_php() {
    command -v php >/dev/null 2>&1 || die "PHP absent (lancez : make deps)"
    php -r 'exit(version_compare(PHP_VERSION, "8.2.0", ">=") ? 0 : 1);' \
        || die "PHP $(php -r 'echo PHP_VERSION;') : la version 8.2 minimum est requise."
    local ext
    for ext in "${EXT_REQUISES[@]}"; do
        php -r "exit(extension_loaded('$ext') ? 0 : 1);" || die "Extension PHP manquante : $ext (lancez : make deps)"
    done
}

install_deps() {
    local sudo=()
    [[ $EUID -ne 0 ]] && command -v sudo >/dev/null 2>&1 && sudo=(sudo)
    if command -v dnf >/dev/null 2>&1; then
        "${sudo[@]}" dnf install -y php-cli php-mbstring
    elif command -v apt-get >/dev/null 2>&1; then
        "${sudo[@]}" apt-get update
        "${sudo[@]}" apt-get install -y php-cli php-mbstring
    else
        die "Gestionnaire de paquets non reconnu. À installer : PHP 8.2+ avec mbstring."
    fi
    check_php
    ok "PHP prêt : $(php -r 'echo PHP_VERSION;')"
}

cmd_info() {
    php -v | head -n1
    local ext
    for ext in "${EXT_REQUISES[@]}"; do
        php -r "exit(extension_loaded('$ext') ? 0 : 1);" && echo "  ✓ $ext" || echo "  ✗ $ext"
    done
    command -v node >/dev/null && echo "node        : $(node -v) (contrôles make check)" || echo "node        : absent (nécessaire pour make check)"
    if [[ -d dist ]]; then
        echo "dist/       : $(find dist -name index.html | wc -l) pages, $(du -sh dist | cut -f1), construit le $(date -r dist '+%d/%m/%Y %H:%M')"
    else
        echo "dist/       : pas encore construit (make build)"
    fi
}

cmd_dev() {
    check_php
    echo "🔌 Site (dev) → http://${HOST}:${PORT}  (Ctrl+C pour arrêter)"
    # bin/dev.php rend chaque page à la demande et laisse php -S servir les fichiers de public/
    exec php -d display_errors=1 -d error_reporting=32767 -d zend.assertions=1 \
        -S "${HOST}:${PORT}" -t public bin/dev.php
}

cmd_build() {
    check_php
    php -d zend.assertions=1 bin/build.php
}

cmd_preview() {
    check_php
    [[ -f dist/index.html ]] || die "dist/ absent : lancez d'abord make build"
    echo "🔎 Site (build) → http://${HOST}:${PREVIEW_PORT}  (Ctrl+C pour arrêter)"
    exec php -S "${HOST}:${PREVIEW_PORT}" -t dist bin/preview.php
}

case "${1:-help}" in
    info) cmd_info ;;
    deps) install_deps ;;
    dev) cmd_dev ;;
    build) cmd_build ;;
    preview) cmd_preview ;;
    *)
        cat <<USAGE
Usage : $0 {info|deps|dev|build|preview}
ENV   : PORT=8000 PREVIEW_PORT=8001 HOST=localhost
USAGE
        exit 1
        ;;
esac
