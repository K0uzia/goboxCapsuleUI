# Tests matériel

Chaque test est une fonction `func(t *testCtx) result` déclarée dans
`internal/ui/gui/session.go` (`testDefs`). Le `testCtx` permet de journaliser,
d'afficher un panneau, de poser une question (`choose` / `ask`) et d'être annulé.

| Test | Fichier | État | Reste à faire |
|---|---|---|---|
| Stress CPU | `test_stress.go` | ✅ charge native sur tous les cœurs, 3 min, temp / charge / fréquence en direct, histogramme, seuil 90 °C | Durée réglable ; détecter le throttling (fréquence qui chute) |
| Ports | `test_devices.go` | ✅ détection en direct des branchements USB et des écrans (`/sys/class/drm`) | Débit USB (lecture d'un fichier test sur la clé) ; détail par type de sortie (HDMI / DP / VGA) dans la fiche |
| Son | `test_devices.go` | ✅ tonalité gauche puis droite (WAV généré), rejouer, verdict du technicien | **Micro** : enregistrer 10 s puis réécouter (LACAPSULE le fait) ; jack séparé |
| Réseau | `test_devices.go` | ✅ interfaces, lien, DNS, accès internet, latence | Scan Wi-Fi ; débit |
| Clavier | `test_keyboard.go` | ✅ clavier AZERTY / QWERTY, codes physiques X11, compteur, OK / moyen / nul selon le nombre de touches | Pavé numérique et touches F1–F12 ; claviers de portables (Fn) ; **à vérifier sur un vrai clavier** (Tab et AltGr peuvent être interceptés) |
| Souris | `test_devices.go` | ✅ zone de test : déplacement, clic gauche / droit, défilement | Multi-touch, clic milieu |
| Webcam | `test_devices.go` | 🟡 ouvre ffplay / mpv / cheese dans une autre fenêtre | Aperçu **dans** gobox (V4L2) |
| Carte SD | `test_devices.go` | 🟡 attend un `mmcblk*` | Lecteurs SD sur USB (`sd*` amovibles) |
| SMART HDD / SSD / NVMe | `test_smart.go` | 🟡 via `smartctl --json` : état, heures, secteurs réalloués, usure NVMe | Lecture SMART **native** (comme SecMet, sans smartctl) ; test court SMART |
| Écran (pixels morts) | — | ❌ | Plein écran couleurs unies / dégradés / grille (LACAPSULE : `screentest`) |
| GPU | — | ❌ | `glmark2` ou rendu intégré, score dans la fiche |
| Batterie | — | ❌ (la note A/B/C/F existe dans `diagnostic/battery`) | Bouton de test + mesure de décharge sur quelques minutes |
| RAM | — | ❌ (`diagnostic/ram/stress.go` vide) | Test mémoire type memtester |
| Benchmark disque | — | ❌ (`diagnostic/disk/benchmark.go` vide) | Lecture séquentielle non destructive |

## Points communs

- [ ] Quand gobox tourne en root, les tests son / webcam passent par `userCommand` (sudo -u) : **à valider sur un poste en live USB**
- [ ] Cocher automatiquement les champs manuels de la fiche quand un test le permet (USB défectueux…)
- [ ] Tests unitaires : WAV ✅, parcours ✅ ; ajouter SMART (JSON d'exemple) et clavier (codes)
