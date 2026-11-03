#!/bin/sh
# Ajoute gobox à une image live en préparation : icône dans le menu des applications.
# Jamais sur le PC à reconditionner, qui part ensuite en vente.
# Usage : sudo sh install.sh [binaire]
set -eu
DIR=$(dirname "$0")
BIN=${1:-$DIR/gobox}
install -Dm755 "$BIN" /usr/bin/gobox
install -Dm644 "$DIR/gobox.desktop" /usr/share/applications/gobox.desktop
install -Dm644 "$DIR/gobox.svg" /usr/share/icons/hicolor/scalable/apps/gobox.svg
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database -q /usr/share/applications || true
echo "gobox installé : icône dans le menu des applications"
