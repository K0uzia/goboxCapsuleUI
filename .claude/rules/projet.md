# Mode d'emploi : gobox (interface)

- depot: goboxCapsuleUI
- type: à confirmer (proposé : logiciel-bureau, ajouté au kit après l'étape 1)
- statut: validé le 2026-10-28
- mise à jour: 2026-11-03

Les règles `00-organisation` et `web-method` s'appliquent. Les autres règles `web-*` ne s'appliquent que pour leurs principes d'interface (voir « Modèle à respecter »). Ce fichier ne contient que ce qui est propre au projet.

## Objectif

- **Demande** (mots de l'humain, 2026-10-27) : « on va pas toucher au script mais uniquement travailler sur de l'ui »
- **Pour qui** : les techniciens de l'atelier La Capsule qui reconditionnent des PC (`docs/ROADMAP.md`, fonctionnement décrit le 2026-10-27).
- **Réussi quand** (validé le 2026-10-27, pour commencer) :
  - l'application suit la maquette, sauf les écarts déclarés voulus à l'étape 1 ;
  - elle reste utilisable de 1366×768 à 2560×1440 ;
  - chaque étape livrée a sa preuve (captures, tests) ;
  - `make check` est vert.
- **Échéance** : aucune donnée.

## Contexte

- Application de bureau en Go 1.25.2 avec Fyne v2 : un seul binaire, point d'entrée `cmd/gobox` vers `internal/ui/gui` (constaté : `go.mod`, `cmd/gobox/main.go`, `docs/todo/README.md`).
- Remplace LACAPSULE v1.8 (JavaFX qui lance 12 scripts shell) (constaté : `docs/ROADMAP.md`).
- **Fonctionnement à l'atelier** : le PC à reconditionner est branché au réseau de l'atelier et démarre sur une image live servie par ce réseau, qui donne accès à la toolbox. Le même logiciel est lancé sur chaque PC (constaté : réponse d'interview du 2026-10-27).
- Sondes matériel Linux uniquement (`/sys`, `/proc`, `lspci`, `dmidecode`, `xrandr`, `smartctl`, `stress-ng`) (constaté : `internal/probe`, `internal/ui/gui/test_*.go`).
- Poste cible : Linux sous X11, depuis l'image live (constaté : commentaire `GO_TAGS` du `Makefile`, réponse d'interview du 2026-10-27). Linux d'abord, Windows dans un second temps une fois Linux terminé.
- Le 2026-10-27 et le 2026-10-28, le développement se fait sur un poste Windows : les sondes et les tests matériel ne peuvent pas y tourner (constaté : réponse d'interview du 2026-10-27).
- Interface actuelle : 14 fichiers, 2637 lignes dans `internal/ui/gui/` ; fiche à gauche, 14 tests à droite en 3 catégories, barre d'étapes en bas, thème clair « atelier » (constaté : `internal/ui/gui/*.go`). L'humain a déjà retouché l'interface après la maquette : une partie des écarts est voulue (constaté : réponse d'interview du 2026-10-27).
- Suivi des tâches déjà tenu dans `docs/todo/*.md` et `docs/ROADMAP.md` (constaté).
- Historique : prototype terminal (Bubble Tea) par un autre contributeur en 2025, interface Fyne ajoutée le 2026-10-27 (constaté : `git log`).

## Publics et action principale

1. Technicien d'atelier : voir en un écran l'état de la machine (fiche) et les tests restants, lancer un test, confirmer son résultat, imprimer la fiche.

## Périmètre

- **Dans** : `internal/ui/gui/` (mise en page, thème, widgets, vues des tests, fiche à l'écran).
- **Hors, le 2026-10-27 et le 2026-10-28** (poste Windows, rien de Linux ne peut tourner) : sondes `internal/probe`, notation `internal/diagnostic`, scripts et commandes système, export PDF côté logique (constaté : réponse d'interview du 2026-10-27). Ce n'est pas une exclusion définitive.
- `Makefile` : les modifications Windows restent non commitées, en l'état (constaté : réponse d'interview du 2026-10-27).
- **Aussi dans** : mettre à jour la boîte à outils (`../kouzia-boite-a-outils`) et le kit `.claude` avec ce qui existe dans gobox, ce qui y sera créé, et ce qui est intéressant dans Tracebay (constaté : réponse d'interview du 2026-10-27). Chaque ajout est proposé et validé (rule `boite-a-outils`).

## Décisions validées

| Date | Décision | Raison |
|------|----------|--------|
| 2026-10-27 | Socle : Go + Fyne v2, binaire unique (constaté) | en place |
| 2026-10-27 | Travail limité à l'interface le 2026-10-27 et le 2026-10-28 | poste Windows : les scripts Linux ne peuvent pas tourner |
| 2026-10-27 | Linux d'abord, Windows ensuite | réponse d'interview |
| 2026-10-27 | Aucune mémorisation du technicien ni de préférence propre à un PC | même logiciel lancé depuis une image live sur chaque PC : une valeur gardée serait la même partout |
| 2026-10-27 | Thème sombre par défaut, bouton pour passer en clair | réponse d'interview |
| 2026-10-28 | Réglages valables le temps de la session seulement : chaque lancement repart des paramètres par défaut | chaque PC est différent, plusieurs techniciens, plusieurs PC en même temps |
| 2026-10-28 | Paramètres par défaut définis avec l'humain (le thème sombre en est un) | réponse d'interview |
| 2026-10-28 | Ni imprimante par défaut ni dossier des scripts dans les réglages | l'impression ouvre la fiche dans le navigateur, l'imprimante se choisit là ; le technicien ne touche pas aux scripts |
| 2026-10-27 | Écarts avec la maquette : classés « voulu » ou « à corriger » par l'humain à l'étape 1, rien n'est corrigé avant | l'humain a déjà retouché l'interface après la maquette |
| 2026-10-27 | Règles d'interface logiciel du kit écrites après l'étape 1 | partir d'écarts réels |
| 2026-10-28 | Thème Kouzia (`../kouzia-boite-a-outils/base/tokens.css` : couleurs, polices, rayons) à la place des couleurs de la maquette ; la maquette garde la structure | demande de l'humain : « applique celui de mon EI kouzia » |
| 2026-10-28 | Écarts de l'étape 1 : classement délégué à Claude, objectif interface professionnelle, simple et intuitive sur la structure de la maquette | « fait comme tu le sens […] je te fais confiance » |
| 2026-10-28 | Données simulées automatiques hors Linux, badge « Données simulées » visible ; vraies mesures toujours sous Linux | travailler les interfaces des tests sous Windows |
| 2026-10-28 | Stress test : durée au choix 3, 10, 30 min ou 1 h | réponse d'interview (OCCT recommande 1 h) |
| 2026-10-30 | Verdict du stress test : erreur de calcul ou ≥ 90 °C = NUL ; 85 à 89 °C = MOYEN ; une fréquence réduite ne change pas le verdict (remplace la décision du 2026-10-28) | sur batterie, la limite de puissance baisse la fréquence alors que la machine chauffe moins |
| 2026-10-28 | Stress test refait test par test, OCCT comme référence fonctionnelle (pas visuelle) ; logique (vérification des calculs, throttling, charge variable) refaite avec l'interface | réponse d'interview |
| 2026-10-30 | Fiche finale en PDF direct (go-pdf/fpdf), ouverte dans le navigateur par défaut de la session | demande de l'humain : sauter l'étape web |
| 2026-10-30 | Fiche présentée comme une fiche produit de boutique : noir sur blanc, icônes vectorielles, seul le mot du résultat en couleur, résultats avec leur composant, pas de section « Tests gobox » ni « À signaler » | demande de l'humain : la couleur consomme, pas de doublon |
| 2026-10-30 | Fiche sans référence RC, chemin, date ni pagination dans les coins | demande de l'humain |
| 2026-10-30 | Résultat inscrit dans la fiche sans confirmation ; retour direct au menu après un verdict du technicien | demande de l'humain : validation trop longue |
| 2026-10-30 | Étapes centrées en haut, À propos et Réglages après PRINT | demande de l'humain |
| 2026-10-30 | Prises vidéo et USB : seulement celles présentes, avec leur quantité et leur type réel (USB 2.0, 3.x, USB-C ; VGA reconnu par l'EDID) | demande de l'humain |
| 2026-10-30 | `make build` : `build/gobox`, fichier unique portable (conteneur glibc 2.28), outils inclus (smartctl, dmidecode, lspci) extraits en temporaire, plus `gobox.rpm` et `gobox.deb` | le PC vendu ne reçoit que le fichier unique ; les paquets servent à l'image ou à un poste d'atelier |
| 2026-10-30 | Tests Clone et Déploiement retirés ; MAJ réelle (apt-get ou dnf : update puis upgrade) | demande de l'humain |
| 2026-10-30 | Tests distincts Charge et Chargeur ; sans chargeur : « SANS CHARGEUR », sans effet sur le verdict | un PC arrive souvent sans chargeur |
| 2026-10-30 | USB défectueux détectés automatiquement (journal du noyau, surintensité) au test des ports, plus de saisie | demande de l'humain |
| 2026-10-30 | Tests Wi-Fi, Bluetooth, SIM et lecteur de puce : matériel reconnu, lancés à l'ouverture, « DÉTECTÉ » quand aucune carte n'est lue, absents de la fiche sans le matériel | l'atelier n'a pas toujours de carte à insérer |
| 2026-10-30 | Puce qui pulse sur le technicien et le nettoyage tant qu'ils ne sont pas remplis (nettoyage : au moins une case) | demande de l'humain |
| 2026-10-30 | Lecteur de puce et SIM non vus par le système : le technicien indique Présent, En panne ou Absent ; prise USB 3 sans moitié USB 2 comptée USB-C | un emplacement peut exister sans être visible (constaté sur HP EliteBook 850 G3) |
| 2026-10-30 | Prises vidéo : liste du BIOS, puis table Intel (VBT), puis cartes AMD et NVIDIA ; une sortie DP et HDMI compte en « Sortie vidéo », son type se lit au branchement | constaté sur HP EliteBook 850 G3 : BIOS sans prise vidéo, table Intel à 2 sorties |
| 2026-10-30 | Ping dans le test réseau (1.1.1.1, latence et pertes), sans bip : le PC signale déjà les branchements | confirmé par l'humain |
| 2026-10-30 | gobox fonctionne sur l'image live et directement sur le PC ; la MAJ est définitive dans les deux cas : depuis l'image live, elle vise le système installé sur le disque (montage et chroot), après confirmation, sans annulation une fois lancée | demande de l'humain : pas de mise à jour temporaire |
| 2026-10-30 | Diagnostic matériel à copier depuis À propos, sans commande à taper | la commande `sudo build/gobox --verifier-outils` n'a pas marché chez l'humain |
| 2026-11-03 | Test clavier : plan complet (Échap, F1 à F12, navigation, flèches, pavé numérique). Portable reconnu par le type de châssis du BIOS : pavé et touches de navigation rares grisés au départ, un appui les fait apparaître, un clic retire une touche absente | demande de l'humain : les claviers n'ont pas tous les mêmes touches |
| 2026-11-03 | Nombre de prises USB lu seulement dans l'ACPI ; la liste du BIOS ne sert plus de compte. Un port non déclaré devient une prise dès qu'on y branche quelque chose | le BIOS liste plus de prises qu'il n'y en a |
| 2026-11-03 | Prise USB morte : bouton « Sans réaction » sous son type, comptée testée et défectueuse (complète la détection automatique du 2026-10-30) | une prise sans aucune réaction ne laisse pas de trace dans le système |
| 2026-11-03 | Fiche : rien n'est rempli d'avance (aspect, pâte, shred, pile, DVD) ; chaque champ vide est entouré et pulse, la liste « À remplir » reste dans l'en-tête, PRINT la rappelle. DVD « absent » seulement si aucun lecteur n'est détecté | demande de l'humain : les informations à remplir n'attirent pas l'œil |
| 2026-11-03 | Écran de résultat : « Valider ce résultat » à la place de « Retour au menu » | les techniciens cliquaient sur « Retour au menu » en croyant ne pas valider |
| 2026-11-03 | MAJ : barre de progression arrêtée et consigne « En attente de votre accord » jusqu'au clic sur « Lancer la mise à jour » | le technicien attendait en croyant la mise à jour lancée |
| 2026-10-27 | Tracebay-Autonome (tout sur un poste, système de licence) : source principale pour la logique ; Tracebay (serveur séparé sur le réseau) : source pour le fonctionnement en réseau | réponse d'interview |

## Modèle à respecter

- **Modèle du projet** : toile « Toolbox, maquette interactive » (https://claude.ai/artifact/Qq5iirvHvWyZ1ZD4qkUuxs), plan `Main.dc.html` (fenêtre principale, 1280 px, interactive) et plan `Fiche.dc.html` (fiche de reconditionnement A4) (constaté : toile lue le 2026-10-27). Les retouches déjà faites par l'humain priment quand il les déclare voulues.
- **Couleurs** : `internal/ui/gui/theme.go`, seule source des couleurs.
- **Kit** : `00-organisation`, `web-method`, principes d'interface de `web-design` (tokens, états, mouvement, interdits) et `web-a11y` (contraste, clavier, focus), transposés au bureau.

## Ton et vocabulaire

- Français, vocabulaire d'atelier (fiche, test, confirmer, technicien) (constaté dans l'interface).

## Données et contenus

- État de la session : `internal/ui/gui/session.go`. Inventaire : `internal/ui/gui/collect.go` (lit les sondes, ne les modifie pas).
- Rien n'est gardé d'un lancement à l'autre sur le PC (image live) : nom du technicien saisi à chaque fiche.

## Étapes

| # | Étape | Livrable | Preuve attendue | État |
|---|-------|----------|-----------------|------|
| 1 | Comparer la maquette et l'application | liste des écarts P0/P1/P2, chacun classé « voulu » ou « à corriger » par l'humain | captures côte à côte | livré (classement délégué à Claude le 2026-10-28) |
| 2 | Rendu de 1366×768 à 2560×1440 | fenêtre lisible et sans coupure à chaque taille | captures 1366×768, 1920×1080, 2560×1440 | livré, à valider |
| 3 | Thème sombre par défaut et bouton clair/sombre | deux thèmes depuis `theme.go`, contraste 4,5:1 | captures des deux thèmes, test d'interface | livré (thème Kouzia), à valider |
| 4 | Corriger les écarts « à corriger » de l'étape 1 | interface conforme | captures, `make check` vert | livré, à valider |
| 5 | Barre d'étapes utile | l'étape met en avant ses tests, la dernière lance l'impression | test d'interface, capture | livré, à valider |
| 6 | Boutons Réglages et Info | réglages de session (thème et autres paramètres définis avec l'humain), retour aux valeurs par défaut au lancement | test d'interface, capture | en partie : thème seul, autres paramètres à définir |
| 7 | Finitions | chronomètre visible, hachures du test en cours, raccourcis Échap et Entrée | test d'interface, capture | livré (contour animé à la place des hachures), à valider |
| 8 | Kit : projets logiciels | type `logiciel-bureau`, rule d'interface logiciel, interview, hook, livraison | textes validés, `journal-kit.md` | à faire |
| 9 | Boîte à outils | briques reprises de gobox, Tracebay-Autonome et Tracebay | `node outils/indexer.mjs` sans erreur | en partie : 5 briques de gobox ajoutées le 2026-10-30, Tracebay à faire |

## À fournir

| Élément | Qui | Pour quand | Bloque |
|---------|-----|------------|--------|
| Liste des paramètres par défaut (en plus du thème sombre) | humain, avec Claude | étape 6 | étape 6 |

## Règles propres au projet

- Tout champ de saisie (Entry) placé dans une zone qui défile passe par `fichePanel.pass` (couche `scrollPass`) : le défilement interne d'un champ Fyne garde la molette et bloque la zone (`internal/widget/scroller.go:616` de Fyne). Test : `TestFicheScrollOverField`.
- Aucune fenêtre ni aucun navigateur ouvert au premier plan sur l'écran de l'humain, rien qui prenne le focus. Captures de l'application : rendu en mémoire, `GOBOX_CAPTURES=<dossier> go test -run TestCaptures ./internal/ui/gui` (`internal/ui/gui/capture_test.go`). Captures web : navigateur sans interface (demande de l'humain, 2026-10-28).

- Le PC reconditionné part en vente : gobox n'y installe rien et n'y laisse rien. Ni paquet, ni installation, ni fichier à la fermeture (test `TestRemoveTraces`). Tout outil utilisé est inclus dans le binaire ou signalé comme manquant à l'humain. Audio, webcam et réglages du bureau restent ceux du système.
- Avant de coder : lire `../kouzia-boite-a-outils/INDEX.md`, et dire tout de suite à l'humain si le dossier manque. Toute brique réutilisable créée ici alimente la boîte (socle `go-fyne`) (demande de l'humain, 2026-10-30).
- Claude tient ce mode d'emploi à jour seul, à chaque décision ou erreur, sans demander de validation ; il lit le mode d'emploi avant de proposer une solution (demande de l'humain, 2026-10-30).
- Aucune donnée propre à un PC ou à un technicien n'est enregistrée localement (`fyne.Preferences` exclu) : image live partagée.
- P1 : `docs/todo/*.md` utilise des emoji et des tirets cadratin (contraire à `web-method` « Tirets ») : à traiter seulement si l'humain le demande.
- `Tracebay-neuf` : autre clone du même dépôt GitHub que `Tracebay` (`K0uzia/Tracebay`), avec des branches plus récentes (3.7.2 à 3.8.3) (constaté : `git remote`, `git branch` le 2026-10-27). Consulté seulement pour comparer une fonction avec `Tracebay`.

## Ajouts à la checklist de livraison

- Captures de la fenêtre en 1366×768 et 1920×1080, thèmes sombre et clair, pour toute zone touchée.
- `make check` vert (dont `tools_test.go`).

## Rapport hebdo

- Jour : sans objet pour le 2026-10-27 et le 2026-10-28 ; à fixer ensuite.
- Points suivis en particulier : écarts avec la maquette, rendu en 1366×768.

## Journal des erreurs

| Date | Erreur | Cause | Règle ajoutée (où) |
|------|--------|-------|--------------------|
| 2026-10-28 | Fenêtres de l'application ouvertes sur l'écran principal pendant les tests (demande de changement) | captures faites en lançant le vrai binaire | captures en mémoire, aucune fenêtre au premier plan (« Règles propres au projet ») |
| 2026-10-30 | Test clavier muet après le correctif de Tab | `SetOnKeyDown` n'est appelé que sans widget focalisé | le widget qui garde le focus reçoit les touches (`desktop.Keyable`), test `TestKeyboardKeepsFocus` |
| 2026-10-30 | Brave ouvert au lieu du navigateur par défaut | `xdg-settings` sans bureau dans l'environnement (pkexec, sudo) | navigateur lu dans `mimeapps.list`, test `TestFindDefaultBrowser` |
| 2026-10-30 | Outils laissés en dépendances alors que le PC démarre sur une image live | contexte du mode d'emploi non appliqué | outils embarqués (« Règles propres au projet ») |
| 2026-10-30 | Paquets `.rpm` et `.deb` proposés, qui installent sur un PC destiné à la vente | contrainte de vente absente du mode d'emploi, et pas vérifiée avant de choisir | règle « gobox n'y installe rien et n'y laisse rien » (« Règles propres au projet ») |
| 2026-10-30 | Boîte à outils ni lue ni alimentée, et son absence jamais signalée | dossier absent au premier passage, non signalé ; rule `boite-a-outils` non appliquée ensuite | lecture et alimentation obligatoires (« Règles propres au projet ») |
| 2026-10-30 | Caméra restée ouverte après son test | SIGKILL envoyé à sudo, qui ne le relaie pas | SIGTERM au groupe (`stopProcess`), test `TestStopProcessStopsChildren` |
| 2026-10-30 | Mode d'emploi non tenu à jour, validations redemandées | règle du kit suivie à la lettre | mise à jour autonome (« Règles propres au projet ») |
| 2026-11-03 | Santé batterie toujours à 0 % et alerte « trop basse » | capacités lues seulement en énergie (energy_*), alors que beaucoup de batteries les donnent en charge (charge_*) ; santé inconnue affichée comme 0 | lecture des deux unités et santé « non communiquée » plutôt que 0 (tests `TestReadBatteryChargeOnly`, `TestEvaluateUnknownHealth`) |
| 2026-11-03 | Clé USB comptée comme disque dur dans la fiche | deux listes de disques, une seule filtrait l'USB | une seule fonction `probe.IsInternalDisk` (test `TestIsInternalDisk`) |
| 2026-11-03 | Deux caméras pour une seule | chaque nœud /dev/video compté (image, métadonnées, infrarouge) | caméras regroupées par appareil (test `TestCamerasGroupNodes`) |
| 2026-11-03 | Fiche à « 0 testé, 0 défectueux » après un port USB essayé | prise morte invisible pour le système, et clé sur un port non déclaré ignorée | « Sans réaction » et adoption du port au branchement (test `TestUSBAdoptUnlistedPort`) |
| 2026-10-28 | Défilement de la fiche bloqué quand le pointeur passe sur un champ de saisie (signalé par l'humain) | défilement interne des champs Fyne qui garde la molette ; aucun test de défilement | couche de renvoi de la molette et test automatique (« Règles propres au projet ») |
