# Paquet pour une image ou un poste d'atelier, jamais pour un PC à vendre. Pas de
# dépendances : les outils sont dans le binaire, tout bureau a ses bibliothèques.
%global debug_package %{nil}
%global __os_install_post %{nil}
%global _build_id_links none
Name:      gobox
Version:   %{gobox_version}
Release:   %{gobox_release}
Summary:   Toolbox de reconditionnement de PC de La Capsule
License:   Proprietary and GPLv2+ (outils embarques)
BuildArch: x86_64
AutoReqProv: no

%description
Toolbox de reconditionnement de PC de La Capsule : tests du materiel et fiche de reconditionnement.

%install
cp -a %{gobox_root}/. %{buildroot}/

%files
/usr/bin/gobox
/usr/share/applications/gobox.desktop
/usr/share/icons/hicolor/scalable/apps/gobox.svg
