#!/bin/sh
# Compile un binaire Linux portable dans un conteneur à glibc ancienne, puis
# refuse le résultat s'il demande une glibc plus récente que MAX_GLIBC.
# Usage : build-portable.sh <go> <tags> <ldflags> <sortie> <paquet>
set -eu

GO_BIN=$1 TAGS=$2 LDFLAGS=$3 OUTPUT=$4 MAIN_PKG=$5
IMAGE=gobox-build:glibc2.28-outils2
MAX_GLIBC=2.28

CACHE_DIR=${XDG_CACHE_HOME:-$HOME/.cache}/gobox-build
mkdir -p "$CACHE_DIR/go"

if command -v podman >/dev/null 2>&1; then
	# Stockage propre et pilote vfs : le pilote par défaut refuse de travailler
	# sur le système de fichiers d'une image live (overlay sur overlay)
	ENGINE="podman --root $CACHE_DIR/podman --storage-driver vfs"
	USER_ARGS="--userns=keep-id"
	[ "$(id -u)" = 0 ] && USER_ARGS="" # keep-id n'existe pas en root (sudo make build)
elif command -v docker >/dev/null 2>&1; then
	ENGINE=docker
	USER_ARGS="--user $(id -u):$(id -g)"
else
	echo "Erreur : podman ou docker est nécessaire pour un binaire compatible avec toutes les distributions." >&2
	echo "Installez podman, ou utilisez make build-local (binaire limité aux distributions aussi récentes que celle-ci)." >&2
	exit 1
fi

# shellcheck disable=SC2086
if ! $ENGINE image inspect "$IMAGE" >/dev/null 2>&1; then
	echo "Préparation de l'image de compilation (une seule fois)..."
	$ENGINE build -t "$IMAGE" -f scripts/Containerfile.build scripts
fi

GOROOT_DIR=$("$GO_BIN" env GOROOT)
MODCACHE=$("$GO_BIN" env GOMODCACHE)
mkdir -p "$(dirname "$OUTPUT")"
"$GO_BIN" mod download

# label=disable : sans lui, SELinux refuse l'accès aux dossiers partagés
# shellcheck disable=SC2086
$ENGINE run --rm $USER_ARGS --security-opt label=disable \
	-v "$PWD:/src" -v "$GOROOT_DIR:/opt/go:ro" -v "$MODCACHE:/opt/gomod:ro" -v "$CACHE_DIR/go:/opt/gocache" \
	-w /src \
	-e PATH=/opt/go/bin:/usr/bin:/bin -e GOROOT=/opt/go -e GOMODCACHE=/opt/gomod -e GOCACHE=/opt/gocache \
	-e GOPROXY=off -e GOFLAGS=-buildvcs=false -e GOTOOLCHAIN=local -e CGO_ENABLED=1 -e HOME=/tmp \
	-e TAGS="$TAGS" -e LDFLAGS="$LDFLAGS" -e OUTPUT="$OUTPUT" -e MAIN_PKG="$MAIN_PKG" -e MAX_GLIBC="$MAX_GLIBC" \
	"$IMAGE" sh -c '
		set -eu
		sh packaging/build-tools.sh internal/ui/gui/outils.tar.gz
		go build -tags "$TAGS embedtools" -trimpath -ldflags "$LDFLAGS" -o "$OUTPUT" "$MAIN_PKG"
		needed=$(objdump -T "$OUTPUT" | grep -oE "GLIBC_[0-9.]+" | sed s/GLIBC_// | sort -Vu | tail -1)
		highest=$(printf "%s\n%s\n" "$needed" "$MAX_GLIBC" | sort -V | tail -1)
		if [ "$highest" != "$MAX_GLIBC" ]; then
			echo "Erreur : le binaire demande glibc $needed, maximum permis $MAX_GLIBC" >&2
			rm -f "$OUTPUT"
			exit 1
		fi
		echo "Contrôle réussi : compatible avec les distributions à glibc $needed ou plus récente"
		sh packaging/build-packages.sh
	'
