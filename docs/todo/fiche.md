# Fiche technique / certificat

Code : `internal/ui/gui/fiche.go` (fiche à l'écran), `print.go` (fiche A4),
`collect.go` (inventaire).

## Fait

- [x] Fiche à l'écran : référence, date, technicien, avancement, machine, aspect, nettoyage, verdict
- [x] Contrôles reliés aux tests (pastilles), section stress test & maintenance (listes oui / non)
- [x] Observations du technicien
- [x] Détail des tests réalisés (case DÉTAILLÉ)
- [x] Inventaire : CPU, RAM, disques, GPU, réseau, batterie, marque / modèle / n° de série (DMI)
- [x] Verdict calculé comme la maquette (en cours / apte grade A ou B / non apte)
- [x] **PRINT** : fiche A4 HTML d'après la maquette, ouverte dans le navigateur (impression ou PDF)

## À faire

- [ ] **PDF natif** sans navigateur (`internal/export/pdf.go`, ex. `go-pdf/fpdf`) + impression directe (`lp`, imprimante des réglages)
- [ ] Enregistrer chaque fiche (`~/gobox/fiches/<ref>.pdf` + `.json`) pour la traçabilité
- [ ] Export JSON / CSV (`internal/export/json.go`, `csv.go` vides)
- [ ] Envoi de la fiche à la base de gestion (LACAPSULE ouvrait `gest.goupil-ere.org`) — **API à définir**
- [ ] Numéro de référence : format à valider (actuellement `RC-AAAA-MMJJ-HHMM`)
- [ ] Logo La Capsule sur la fiche (`~/assets/logo_lacapsule_*.svg`)
- [ ] Inventaire manquant par rapport à LACAPSULE : **taille d'écran en pouces** (EDID), modèle des cartes réseau (PCI), espace disque utilisé, version OpenGL, n° de série / tension batterie
- [ ] Batterie : afficher la note A/B/C/F et la barre de santé comme la maquette
- [ ] Section **Shred** remplie automatiquement depuis le rapport SecMet (méthode, n° de certificat)
- [ ] Fiche sur 2 pages si l'inventaire est long (la maquette annonce « page 1-2 »)
