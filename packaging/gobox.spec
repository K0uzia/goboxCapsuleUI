# Paquet installable en double-clic. Pas de dépendances automatiques : les outils
# sont embarqués, et tout bureau a les bibliothèques graphiques de gobox.
%global debug_package %{nil}
%global __os_install_post %{nil}
Name:      gobox
Version:   %{gobox_version}
Release:   %{gobox_release}
Summary:   Toolbox de reconditionnement de PC de La Capsule
License:   Proprietary and GPLv2+ (outils embarques)
BuildArch: x86_64
AutoReqProv: no

%description
Toolbox de reconditionnement de PC de La Capsule : tests du materiel et fiche de reconditionnement. Embarque smartctl, dmidecode et lspci.

%install
cp -a %{gobox_root}/. %{buildroot}/

%files
/usr/bin/gobox
/usr/lib/gobox
/usr/share/applications/gobox.desktop
/usr/share/icons/hicolor/scalable/apps/gobox.svg
