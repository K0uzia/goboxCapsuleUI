# Suivi des tâches gobox

Tout ce qu'il reste à faire, par chantier. Une case cochée = fait et vérifié.
Le *pourquoi* (comparaison avec LACAPSULE v1.8) est dans [`../ROADMAP.md`](../ROADMAP.md).

| Fichier | Chantier |
|---|---|
| [ui.md](ui.md) | Interface Fyne (maquette « Toolbox ») |
| [tests.md](tests.md) | Tests matériel (onglet de droite) |
| [fiche.md](fiche.md) | Fiche technique, impression, export |
| [shred-secmet.md](shred-secmet.md) | Effacement des disques via SecMet |
| [systeme.md](systeme.md) | Clone, déploiement, MAJ, nettoyage |
| [build-ci.md](build-ci.md) | Makefile, CI GitHub, paquets, multi-OS |
| [dette.md](dette.md) | Bugs et code à reprendre dans l'existant |

## Questions ouvertes

À trancher avec l'équipe avant d'avancer sur le chantier concerné.

- [ ] **Shred : que veut dire « connecté au PC » ?** Faut-il une session ouverte directement sur le poste en reconditionnement, et donc refuser de lancer l'effacement depuis un bureau à distance (RustDesk) ou en SSH ? → [shred-secmet.md](shred-secmet.md)
- [ ] **Clonage** : quel outil / quelles images l'atelier utilise-t-il aujourd'hui ? → [systeme.md](systeme.md)
- [ ] **Déploiement post-install** : quelles étapes (pilotes, utilisateur atelier…) ? → [systeme.md](systeme.md)
- [ ] **Base de gestion** (`gest.goupil-ere.org`) : existe-t-il une API pour envoyer la fiche ? → [fiche.md](fiche.md)
- [ ] **Référence de fiche** : garder `RC-AAAA-MMJJ-HHMM` ou reprendre la numérotation de la base ? → [fiche.md](fiche.md)

## Repères

- **Point d'entrée GUI** : `cmd/gobox` → `internal/ui/gui`
- **Ancien point d'entrée terminal** : `cmd/goboxcli` (conservé en secours)
- **Sondes matériel** : `internal/probe` (lecture `/sys`, `/proc`, Linux uniquement)
- **Notation** : `internal/diagnostic/*` (batterie faite, disque en TODO)
- **Maquette de référence** : artefact Claude « Toolbox — fenêtre principale » + « Fiche A4 »
- **LACAPSULE** (outil actuel) : `/usr/share/LACAPSULE/MULTITOOL/*.sh`
- **SecMet** : `~/Bureau/team/#INTERNE/Programmes/SecMet` (Rust, GTK4)

## Lancer

```bash
make deps     # une fois : outils système, Go, modules
make dev      # lance la fenêtre (sudo, pour dmidecode / SMART)
make test     # tests Go
make build    # binaire optimisé dans build/
```
