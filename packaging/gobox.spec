# Paquet installable en double-clic. Pas de dépendances automatiques : les noms de
# bibliothèques varient selon les distributions, et tout bureau a celles de gobox.
%global debug_package %{nil}
%global __os_install_post %{nil}
Name:      gobox
Version:   %{gobox_version}
Release:   %{gobox_release}
Summary:   Toolbox de reconditionnement de PC de La Capsule
License:   Proprietary
BuildArch: x86_64
AutoReqProv: no
Recommends: dmidecode smartmontools pciutils stress-ng

%description
Toolbox de reconditionnement de PC de La Capsule : tests du materiel et fiche de reconditionnement.

%install
install -Dm755 %{gobox_src}/build/gobox %{buildroot}/usr/bin/gobox
install -Dm644 %{gobox_src}/packaging/gobox.desktop %{buildroot}/usr/share/applications/gobox.desktop
install -Dm644 %{gobox_src}/packaging/gobox.svg %{buildroot}/usr/share/icons/hicolor/scalable/apps/gobox.svg

%files
/usr/bin/gobox
/usr/share/applications/gobox.desktop
/usr/share/icons/hicolor/scalable/apps/gobox.svg
