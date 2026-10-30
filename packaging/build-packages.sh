#!/bin/sh
# Crée build/gobox.rpm et build/gobox.deb (dans le conteneur de compilation), avec
# smartctl, dmidecode et lspci embarqués : rien à installer sur le PC. Version
# datée : un paquet plus récent remplace le précédent.
set -eu
VERSION=$(date +%Y.%m.%d)
RELEASE=$(date +%H%M)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

ROOT=$WORK/root
TOOLS=$ROOT/usr/lib/gobox
install -Dm755 build/gobox "$ROOT/usr/bin/gobox"
install -Dm644 packaging/gobox.desktop "$ROOT/usr/share/applications/gobox.desktop"
install -Dm644 packaging/gobox.svg "$ROOT/usr/share/icons/hicolor/scalable/apps/gobox.svg"
mkdir -p "$TOOLS/bin" "$TOOLS/libexec" "$TOOLS/lib" "$TOOLS/share" "$TOOLS/licences"

# Chaque outil avec ses bibliothèques, sauf la glibc : celle du PC est plus récente
# que celle de la compilation, et elle reste compatible.
embed() {
	name=$1
	shift
	install -m755 "/usr/sbin/$name" "$TOOLS/libexec/$name"
	ldd "/usr/sbin/$name" | awk '/=> \//{print $3}' | while read -r lib; do
		case $(basename "$lib") in
		libc.so* | libm.so* | libdl.so* | libpthread.so* | librt.so* | libresolv.so* | ld-linux*) ;;
		*) cp -Ln "$lib" "$TOOLS/lib/" ;;
		esac
	done
	printf '#!/bin/sh\nLD_LIBRARY_PATH=/usr/lib/gobox/lib exec /usr/lib/gobox/libexec/%s %s "$@"\n' "$name" "$*" >"$TOOLS/bin/$name"
	chmod 755 "$TOOLS/bin/$name"
}
embed smartctl
embed dmidecode
embed lspci -i /usr/lib/gobox/share/pci.ids
cp /usr/share/hwdata/pci.ids "$TOOLS/share/pci.ids"
for pkg in smartmontools dmidecode pciutils; do
	mkdir -p "$TOOLS/licences/$pkg"
	find /usr/share/doc/"$pkg"* /usr/share/licenses/"$pkg"* \( -name 'COPYING*' -o -name 'LICENSE*' \) -exec cp {} "$TOOLS/licences/$pkg/" \; 2>/dev/null || true
	[ -n "$(ls "$TOOLS/licences/$pkg")" ] || { echo "Erreur : licence de $pkg introuvable" >&2; exit 1; }
done

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
