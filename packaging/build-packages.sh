#!/bin/sh
# Crée build/gobox.rpm et build/gobox.deb depuis build/gobox (dans le conteneur de
# compilation). Version datée : un paquet plus récent remplace le précédent.
set -eu
VERSION=$(date +%Y.%m.%d)
RELEASE=$(date +%H%M)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

ROOT=$WORK/root
install -Dm755 build/gobox "$ROOT/usr/bin/gobox"
install -Dm644 packaging/gobox.desktop "$ROOT/usr/share/applications/gobox.desktop"
install -Dm644 packaging/gobox.svg "$ROOT/usr/share/icons/hicolor/scalable/apps/gobox.svg"

rpmbuild -bb --quiet --define "_topdir $WORK/rpm" --define "gobox_version $VERSION" \
	--define "gobox_release $RELEASE" --define "gobox_root $ROOT" packaging/gobox.spec
cp "$WORK"/rpm/RPMS/x86_64/gobox-*.rpm build/gobox.rpm

mkdir -p "$WORK/deb"
cat >"$WORK/deb/control" <<CONTROL
Package: gobox
Version: $VERSION-$RELEASE
Architecture: amd64
Maintainer: La Capsule <info@lacapsule.org>
Section: admin
Priority: optional
Description: Toolbox de reconditionnement de PC de La Capsule
CONTROL
OUT=$PWD/build/gobox.deb
cd "$WORK/deb"
echo 2.0 >debian-binary
tar --owner=0 --group=0 -czf control.tar.gz ./control
tar --owner=0 --group=0 -czf data.tar.gz -C "$ROOT" .
rm -f "$OUT"
ar rc "$OUT" debian-binary control.tar.gz data.tar.gz
