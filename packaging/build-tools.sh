#!/bin/sh
# Prépare l'archive des outils embarqués dans le binaire (dans le conteneur de
# compilation) : smartctl, dmidecode et lspci, avec leurs bibliothèques et licences.
# gobox l'extrait dans un dossier temporaire, effacé à sa fermeture.
set -eu
OUT=$1
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$WORK/bin" "$WORK/libexec" "$WORK/lib" "$WORK/share" "$WORK/licences"

# Chaque outil avec ses bibliothèques, sauf la glibc : celle du PC est plus récente
# que celle de la compilation, et elle reste compatible.
embed() {
	name=$1
	options=$2
	install -m755 "/usr/sbin/$name" "$WORK/libexec/$name"
	ldd "/usr/sbin/$name" | awk '/=> \//{print $3}' | while read -r lib; do
		case $(basename "$lib") in
		libc.so* | libm.so* | libdl.so* | libpthread.so* | librt.so* | libresolv.so* | ld-linux*) ;;
		*) cp -Ln "$lib" "$WORK/lib/" ;;
		esac
	done
	cat >"$WORK/bin/$name" <<WRAPPER
#!/bin/sh
D=\$(dirname "\$0")/..
LD_LIBRARY_PATH="\$D/lib" exec "\$D/libexec/$name" $options "\$@"
WRAPPER
	chmod 755 "$WORK/bin/$name"
}
embed smartctl ''
embed dmidecode ''
embed lspci '-i "$D/share/pci.ids"'
cp /usr/share/hwdata/pci.ids "$WORK/share/pci.ids"
for pkg in smartmontools dmidecode pciutils; do
	mkdir -p "$WORK/licences/$pkg"
	find /usr/share/doc/"$pkg"* /usr/share/licenses/"$pkg"* \( -name 'COPYING*' -o -name 'LICENSE*' \) -exec cp {} "$WORK/licences/$pkg/" \; 2>/dev/null || true
	[ -n "$(ls "$WORK/licences/$pkg")" ] || { echo "Erreur : licence de $pkg introuvable" >&2; exit 1; }
done
tar --owner=0 --group=0 -czf "$OUT" -C "$WORK" .
