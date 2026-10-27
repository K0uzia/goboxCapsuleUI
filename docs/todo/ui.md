# Interface (Fyne)

Code : `internal/ui/gui/` — `app.go` (mise en page), `fiche.go` (colonne gauche),
`tools.go` (colonne droite + vue d'un test), `widgets.go`, `theme.go`.

## Fait

- [x] Fenêtre deux colonnes d'après la maquette : fiche à gauche, outils à droite
- [x] Thème clair aux couleurs de la maquette (`theme.go`)
- [x] 14 boutons de tests, 3 catégories colorées (périphériques / stockage / système)
- [x] État des boutons : à faire, en cours, OK, moyen, nul, absent
- [x] Vue « test en cours » : progression, panneau propre au test, journal horodaté, annuler
- [x] Vue « résultat » : pastille, titre, détail, durée, relancer / confirmer
- [x] Le résultat n'est inscrit dans la fiche qu'après **Confirmer**
- [x] Barre d'étapes en bas (‹ étape n / 5 ›)
- [x] Température CPU en direct en pied de fiche
- [x] Test automatisé du parcours bouton → résultat → confirmation (`tools_test.go`)

## À faire

- [ ] Barre d'étapes **utile** : l'étape filtre / met en avant la catégorie de boutons concernée, et « Imprimer la fiche » ouvre l'impression
- [ ] Boutons **Info** et **Réglages** de la maquette (imprimante par défaut, dossier de sortie des fiches, durée du stress test, nom de l'atelier)
- [ ] Mémoriser le nom du technicien (`fyne.Preferences`)
- [ ] Chronomètre de reconditionnement visible (la durée est déjà calculée pour la fiche)
- [ ] Hachures animées sur le bouton « en cours » (maquette : `tb-running`)
- [ ] Raccourcis clavier (Échap = annuler le test, Entrée = confirmer) — hors test clavier
- [ ] Bouton **Shred** (voir [shred-secmet.md](shred-secmet.md))
- [ ] Icône d'application + `.desktop`
- [ ] Vérifier le rendu en 1366×768 (écrans de portables reconditionnés)
- [ ] Mode sombre ? (à décider)
