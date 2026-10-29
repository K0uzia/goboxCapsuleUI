# Mode d'emploi : gobox (interface)

- depot: goboxCapsuleUI
- type: à confirmer (proposé : logiciel-bureau, ajouté au kit après l'étape 1)
- statut: validé le 2026-10-28
- mise à jour: 2026-10-28

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
| 2026-10-28 | Ni imprimante par défaut ni dossier des scripts dans les réglages | l'impression ouvre une page web où l'on télécharge ou imprime la fiche, l'imprimante se choisit là ; le technicien ne touche pas aux scripts |
| 2026-10-27 | Écarts avec la maquette : classés « voulu » ou « à corriger » par l'humain à l'étape 1, rien n'est corrigé avant | l'humain a déjà retouché l'interface après la maquette |
| 2026-10-27 | Règles d'interface logiciel du kit écrites après l'étape 1 | partir d'écarts réels |
| 2026-10-28 | Thème Kouzia (`../kouzia-boite-a-outils/base/tokens.css` : couleurs, polices, rayons) à la place des couleurs de la maquette ; la maquette garde la structure | demande de l'humain : « applique celui de mon EI kouzia » |
| 2026-10-28 | Écarts de l'étape 1 : classement délégué à Claude, objectif interface professionnelle, simple et intuitive sur la structure de la maquette | « fait comme tu le sens […] je te fais confiance » |
| 2026-10-28 | Données simulées automatiques hors Linux, badge « Données simulées » visible ; vraies mesures toujours sous Linux | travailler les interfaces des tests sous Windows |
| 2026-10-28 | Stress test : durée au choix 3, 10, 30 min ou 1 h | réponse d'interview (OCCT recommande 1 h) |
| 2026-10-28 | Verdict du stress test : erreur de calcul ou ≥ 90 °C = NUL ; throttling ou 85 à 89 °C = MOYEN | réponse d'interview |
| 2026-10-28 | Stress test refait test par test, OCCT comme référence fonctionnelle (pas visuelle) ; logique (vérification des calculs, throttling, charge variable) refaite avec l'interface | réponse d'interview |
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
| 9 | Boîte à outils | briques reprises de gobox, Tracebay-Autonome et Tracebay, proposées une par une | `node outils/indexer.mjs` sans erreur | à faire |

## À fournir

| Élément | Qui | Pour quand | Bloque |
|---------|-----|------------|--------|
| Liste des paramètres par défaut (en plus du thème sombre) | humain, avec Claude | étape 6 | étape 6 |

## Règles propres au projet

- Tout champ de saisie (Entry) placé dans une zone qui défile passe par `fichePanel.pass` (couche `scrollPass`) : le défilement interne d'un champ Fyne garde la molette et bloque la zone (`internal/widget/scroller.go:616` de Fyne). Test : `TestFicheScrollOverField`.
- Aucune fenêtre ni aucun navigateur ouvert au premier plan sur l'écran de l'humain, rien qui prenne le focus. Captures de l'application : rendu en mémoire, `GOBOX_CAPTURES=<dossier> go test -run TestCaptures ./internal/ui/gui` (`internal/ui/gui/capture_test.go`). Captures web : navigateur sans interface (demande de l'humain, 2026-10-28).

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
| 2026-10-28 | Défilement de la fiche bloqué quand le pointeur passe sur un champ de saisie (signalé par l'humain) | défilement interne des champs Fyne qui garde la molette ; aucun test de défilement | couche de renvoi de la molette et test automatique (« Règles propres au projet ») |
