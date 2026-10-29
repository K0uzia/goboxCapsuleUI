# Feuille de route gobox — refonte de LACAPSULE

Comparaison avec **LACAPSULE v1.8** (`/usr/share/LACAPSULE`, paquet `lacapsule`) :
une fenêtre JavaFX (jar de 137 Mo, dépend d'OpenJDK 17) qui lance 12 scripts
shell du dossier `MULTITOOL/` et affiche `~/resultat.pdf` dans un lecteur PDF
intégré, avec un chronomètre.

Objectif gobox : un seul binaire Go natif, sans Java ni dépendances externes
quand c'est possible (lecture directe de `/sys`, `/proc`, ioctl), plus rapide et
plus sûr.

Légende : ✅ fait · 🟡 partiel · ❌ manquant

---

## 1. Fiche technique / certificat (`Gathering/PRINT.sh`)

C'est la fonction centrale : un PDF par machine reconditionnée.

| Élément | LACAPSULE | gobox | À faire |
|---|---|---|---|
| Marque, modèle, **n° de série** machine | `dmidecode -s system-*` | 🟡 marque + modèle (titre) | Lire `/sys/class/dmi/id/{sys_vendor,product_name,product_serial}` (série = root) dans une sonde `probe/system.go` |
| CPU (architecture, modèle) | `/proc/cpuinfo` | ✅ | — |
| RAM par slot (taille, fréquence) | `inxi -m` | ✅ (dmidecode) | Repli `/proc/meminfo` déjà fait côté GUI |
| Disques : marque, modèle, taille, type | `inxi -D`, `lsblk` | ✅ | — |
| Disques : **état SMART** (BON / MAUVAIS) | `skdump --overall` | ❌ | Lire SMART : `smartctl -H -j` (JSON) ou ioctl natif ; heures de fonctionnement, secteurs réalloués |
| Disques : espace utilisé | `inxi -D` | ❌ | `statfs` sur les partitions montées |
| Batterie : charge, capacité d'origine / réelle, usure | `acpi -i` | ✅ + **note A/B/C/F** | — |
| Batterie : fabricant, modèle, série, tension, puissance | `upower` | 🟡 (pas affichés : série, tension) | Ajouter les champs dans la fiche |
| Cartes réseau (**modèle** WiFi / Ethernet) | `inxi -N` | 🟡 interfaces seulement | Nom de la carte PCI/USB via `/sys/class/net/*/device` + base `pci.ids` |
| GPU : modèle, pilote | `inxi -G` | ✅ | Ajouter rendu / version OpenGL |
| Écrans : ports connectés (HDMI, VGA, DP, eDP) | `xrandr` | 🟡 (`GPU.Outputs`) | Lister tous les connecteurs `/sys/class/drm/card*-*/status` |
| Écran : **taille en pouces** | `dimension.py` (GTK) | ❌ | Calcul depuis l'EDID (`/sys/class/drm/*/edid`, octets 21–22) |
| **Checklist manuelle** (voir liste ci-dessous) | cases `[ ]` imprimées | ❌ | Cases à cocher dans l'onglet *Fiche technique* |
| En-tête : date + nom du technicien | `enscript --header` | ❌ | Champ « Technicien » mémorisé dans les préférences |
| **Export PDF** | `enscript` + `ps2pdf` | ❌ (`export/pdf.go` vide) | Génération PDF native en Go (ex. `go-pdf/fpdf`), mise en page soignée, logo La Capsule |
| Export JSON / CSV | — | ❌ (fichiers vides) | Pour l'archivage et la synchronisation avec la base |
| Aperçu du PDF dans l'application | lecteur pdf.js intégré | ❌ | Aperçu dans l'application, ou ouverture avec le lecteur système |

**Checklist manuelle à reproduire** (cochée par le technicien, reportée dans le PDF) :
nettoyage intérieur / extérieur · aspect (neuf / occasion / nul) · sorties vidéo
HDMI / DP / VGA / DVI · webcam · lecteur de carte SD · ports USB défectueux (nombre) ·
clavier · pavé tactile / souris · carte son · haut-parleurs · sortie audio ·
entrée micro · micro interne · température CPU avant / après le stress test (< 90 °C) ·
changement de pâte thermique · effacement des disques · état du chargeur · pile
CMOS changée · lecteur DVD · commentaire libre.

> Les tests de la section 2 doivent **cocher automatiquement** ces cases quand
> c'est possible (USB, son, micro, clavier, température…).

---

## 2. Tests matériel (`Test/DIALOG.sh`, `Maintenance/maintenance.sh`)

Dans LACAPSULE, chaque test ouvre un outil externe différent (`dialog`, navigateur,
`cheese`, `mplayer`…). Dans gobox, tout se fait dans la fenêtre, onglet *Tests*.

| Test | LACAPSULE | gobox | À faire |
|---|---|---|---|
| **Stress CPU + température en direct** | `stress-ng` 190 s + `inxi -s` | ❌ (`cpu/stress_test.go` non compilé, mal nommé) | Charge CPU native (goroutines) ou `stress-ng` ; température via `/sys/class/hwmon` ; **graphique en direct** ; relevé avant / après → case < 90 °C |
| Test de l'écran (pixels morts, couleurs) | `screentest` | ❌ | Fenêtre plein écran avec couleurs unies, dégradés, grille |
| Test du clavier | page web `test-clavier.fr` ou `text_clavier.html` | ❌ | Clavier AZERTY dessiné, chaque touche s'allume à l'appui |
| Test des ports USB (brancher un périphérique) | différence `lsusb` avant / après | 🟡 (`probe/usb.go` liste les périphériques) | Détection en direct (scrutation de `/sys/bus/usb/devices`), compteur de ports testés |
| Test du son (haut-parleurs) | `mplayer test.wav` | ❌ | Lecture d'un son intégré au binaire (`embed`), gauche / droite séparés |
| Test du micro (enregistrer 10 s puis réécouter) | `arecord` + `mplayer` | ❌ | Enregistrer puis réécouter, avec vu-mètre |
| Test de la webcam | `cheese` / `ffplay` | ❌ | Aperçu V4L2 dans la fenêtre (ou lancer `ffplay` en repli) |
| Test de la connexion internet | `ping` + `curl` google.fr | ❌ (`network/connectivity.go` vide) | Résolution DNS + requête HTTP en Go, latence |
| Test de la carte graphique | `glxgears`, `glmark2` | ❌ | Lancer `glmark2` et récupérer le score, ou rendu 3D intégré |
| Santé du disque (SMART) | `skdump`, `smartctl -i` | ❌ | Voir section 1 ; + test court SMART (`smartctl -t short`) |
| Benchmark disque | — | ❌ (`disk/benchmark.go` vide) | Lecture séquentielle (non destructive) |
| Stress RAM | — | ❌ (`ram/stress.go` vide) | Test d'écriture / lecture mémoire (type memtester) |
| Monitoring complet (CPU, réseau) | `nmon`, `iptraf-ng`, `s-tui` | ❌ | Graphiques en direct dans l'onglet *Tests* |
| Notation globale de la machine | — | 🟡 (batterie seulement) | Compléter `disk/grading.go` (TODO) puis CPU, RAM ; note globale = `common.WorseGrade` |
| Chronomètre de reconditionnement | chrono dans la fenêtre | ❌ | Chrono démarré à l'ouverture, durée reportée dans la fiche |

---

## 3. Effacement et clonage des disques (`Shred/`, `Operations/CLONE.sh`)

Ces opérations détruisent des données : elles demandent le plus de soin.

> **Décision : l'effacement passe par SecMet** (outil interne en Rust, NIST SP 800-88,
> certificats signés), lancé depuis gobox **uniquement sur le PC en cours de
> reconditionnement**. Le tableau ci-dessous compare avec LACAPSULE ; le plan
> d'intégration est dans [`todo/shred-secmet.md`](todo/shred-secmet.md).

| Fonction | LACAPSULE | gobox | À faire |
|---|---|---|---|
| Effacement HDD (3 passes + zéros) | `shred -n 3 -z` dans un `gnome-terminal` par disque | ❌ | Écriture native multi-disques en parallèle, barre de progression et débit par disque |
| Effacement sécurisé SSD SATA (ATA Secure Erase) | `hdparm --security-erase`, mise en veille si le disque est « gelé » | ❌ | Même logique (déblocage par mise en veille), mot de passe aléatoire, vérification finale |
| Effacement NVMe | annoncé mais **absent** | ❌ | `nvme format --ses=1` / `nvme sanitize` |
| **Certificat d'effacement** | — | ❌ | PDF par disque (n° de série, méthode, date, résultat) ; utile pour la traçabilité et le RGPD |
| Clonage de disque | `dd` sur des noms tapés à la main | ❌ | Choix source / destination dans une liste, vérification des tailles, progression |
| Garde-fous | quasi aucun | — | Disque système et disques montés exclus d'office, double confirmation avec le n° de série, rien ne démarre sans action explicite |

---

## 4. Maintenance du poste (`Maintenance/MAJ.sh`, `CLEAN.sh`, `Operations/`)

| Fonction | LACAPSULE | gobox | À faire |
|---|---|---|---|
| Mise à jour du système (bouton bleu « MAJ ») | `apt` fix-broken, update, upgrade, autoremove, vidage des caches | ❌ | Lancer `apt` avec sortie en direct dans une fenêtre de journal |
| Nettoyage après usage (bouton mauve « CLEAN ») | désinstalle les outils, vide `/tmp`, cache, corbeille | ❌ | Sans objet en grande partie (binaire unique) ; garder la purge cache / corbeille / historique |
| Installer les prérequis | `install.sh` (30 paquets) | 🟡 `make deps` pour le dev | Contrôle au démarrage des outils facultatifs (`smartctl`, `hdparm`, `nvme`…) et bouton « Installer » |
| Réparation du démarrage | `boot-repair` (PPA) | ❌ | Bouton lanceur (installe si absent) |
| Création d'une ISO du système | `penguins-eggs` (`.deb` embarqué) | ❌ | Bouton lanceur |
| Accès à la base de gestion | ouvre `gest.goupil-ere.org` | ❌ | Bouton lien, puis **envoi direct de la fiche JSON à la base** |

---

## 5. Distribution

| Élément | LACAPSULE | gobox | À faire |
|---|---|---|---|
| Paquet | `.deb` 137 Mo + OpenJDK 17 | ❌ | `.deb` de quelques Mo (nfpm / goreleaser) ; cible `make package` |
| Lanceur dans le menu | `LACAPSULE.desktop` | ❌ | `.desktop` + icône |
| Droits administrateur | `sudo` dans chaque script, mot de passe demandé plusieurs fois | 🟡 `make dev` avec sudo | Lancement via `pkexec` (une seule demande, avec une règle polkit) |
| Fonctionnement en live USB / mode texte | scripts `dialog` | ❌ | Garder un mode terminal (`cmd/goboxcli`) en secours |

---

## 6. Défauts de LACAPSULE à ne pas reproduire

- `MAJ.sh` et `CLEAN.sh` font `chown -R $USER /var/lib/dpkg/* /var/cache/apt/*` : ils cassent les droits d'apt.
- `SHREDHDD.sh` : `shred -u` sur un périphérique, lettre de disque tapée à la main 16 fois, aucune confirmation, aucune protection du disque système.
- `WIPESSD.sh` contient des erreurs de syntaxe (`[[! $1 ]]`, `$1!=`) qui le font échouer ; mot de passe ATA en dur.
- `CLONE.sh` : `dd` sans aucune vérification de la source ni de la destination.
- `install.sh` / `DIALOG.sh` : noms de paquets faux (`skdump`, `smartctl`, `mlayer`, `x-www-brower`, `alsautils`…).
- `PRINT.sh` : choix du disque par `sed` fragiles sur `blkid` ; ne gère qu'une batterie `BAT0` et 2 disques.
- `CLEAN.sh` désinstalle les dépendances de l'outil lui-même.
- `postinst` : faute de chemin (`/urs/share`), et `$USER` vaut root pendant l'installation.

---

## Priorités proposées

1. **Fiche technique complète + export PDF** (section 1) : c'est l'usage quotidien. Il faut ajouter le n° de série, l'état SMART, la taille d'écran, la checklist et la génération du PDF.
2. **Tests interactifs** (section 2) : clavier, écran, USB, son, micro, webcam, internet, stress CPU avec température ; chaque test coche sa case dans la fiche.
3. **Effacement sécurisé + certificat d'effacement** (section 3), avec tous les garde-fous.
4. **Paquet `.deb` + lancement `pkexec`** (section 5), pour remplacer LACAPSULE sur les postes.
5. **Maintenance et lanceurs** (section 4), puis **synchronisation avec la base de gestion**.
