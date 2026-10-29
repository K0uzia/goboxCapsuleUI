# Interface (Fyne)

Code : `internal/ui/gui/` : `app.go` (fenêtre, en-tête, barre d'étapes, Réglages, À propos),
`fiche.go` (colonne gauche), `tools.go` (colonne droite et vue d'un test), `widgets.go`,
`theme.go` (thème Kouzia), `fonts/` (Exo 2, Share Tech Mono, licence OFL).

Captures sans ouvrir de fenêtre : `GOBOX_CAPTURES=<dossier> go test -run TestCaptures ./internal/ui/gui`.

## Fait

- [x] Fenêtre deux colonnes d'après la maquette : fiche à gauche, outils à droite
- [x] Thème Kouzia : sombre par défaut, clair au choix dans Réglages (valable pour la session)
- [x] Polices Exo 2 et Share Tech Mono intégrées au binaire
- [x] En-tête : marque, indicateurs Durée et CPU, bouton PRINT (confirmation si des tests restent à faire)
- [x] En-tête de fiche fixe (référence, avancement, technicien) ; champ technicien signalé tant qu'il est vide
- [x] 14 boutons de tests, 3 catégories colorées (périphériques / stockage / système)
- [x] État des boutons : à faire, en cours (contour animé), OK, moyen, nul, absent ; survol et focus clavier
- [x] Fiche compacte : synthèse sur une ligne de 4 cadres, contrôles et maintenance sur deux colonnes ; la molette fait défiler la fiche même au-dessus d'un champ
- [x] Vue « test en cours » : bandeau de catégorie, progression, panneau propre au test, journal, annuler
- [x] Vue « résultat » : encadré teinté, pastille, détail, durée, relancer / confirmer juste en dessous
- [x] Le résultat n'est inscrit dans la fiche qu'après **Confirmer**
- [x] Barre d'étapes automatique : suit la fiche (technicien, puis la première catégorie inachevée), met ses tests en avant, propose « Imprimer la fiche » à la fin
- [x] Boutons **À propos** (raccourcis) et **Réglages** (thème)
- [x] Raccourcis : Échap annule le test, Entrée confirme (inactifs pendant le test clavier) ; Tab parcourt les boutons
- [x] Déroulé en 4 étapes sous les boutons de tests
- [x] Icônes Font Awesome Free intégrées (tests, en-tête, boutons, cases du stress test) ; attribution dans « À propos »
- [x] Journal replié par défaut (barre avec le dernier message), dépliable : pastille d'importance, heure, texte qui passe à la ligne, nombre de lignes, bouton Copier
- [x] Pistes d'amélioration affichées avec le résultat d'un test (stress : pâte thermique, dépoussiérage, ventilateur, chargeur, mémoire…)
- [x] Icônes à la taille du texte voisin (environ 1,2 fois sa hauteur), 16 px dans les boutons
- [x] Choix en boutons qui passent à la ligne (stress test) au lieu de boutons radio coupés
- [x] Rendu vérifié en 1366×768, 1280×1000, 1920×1080 et 2560×1440
- [x] Avancement par catégorie (« 3 / 8 faits ») et icône de statut sur chaque bouton de test
- [x] « Détail des tests » dans Réglages (plus de pied de fiche)
- [x] Test automatisé du parcours bouton → résultat → confirmation (`tools_test.go`)

## À faire

- [ ] Autres paramètres par défaut des Réglages (à définir ensemble)
- [ ] Fiche A4 imprimée (`print.go`) : comparer avec la maquette `Fiche.dc.html` et l'aligner sur le thème
- [ ] Bouton **Shred** (voir [shred-secmet.md](shred-secmet.md))
- [ ] Icône d'application + `.desktop`
- [ ] Vérifier sur le poste Linux de l'atelier (rendu OpenGL réel, sondes branchées)
