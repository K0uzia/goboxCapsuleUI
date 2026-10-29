# Opérations système

Actuellement les boutons **Clone**, **Déploiement** et **MAJ** demandent
seulement au technicien de saisir le résultat (`runManual` dans `tests.go`).

| Opération | LACAPSULE | À faire dans gobox |
|---|---|---|
| MAJ du système | `MAJ.sh` : apt fix-broken, update, upgrade, autoremove, purge caches | [ ] Lancer `apt-get` avec la sortie en direct dans le journal du test, nombre de paquets mis à jour dans la fiche. **Ne pas** reprendre le `chown -R $USER /var/lib/dpkg` |
| Nettoyage après usage | `CLEAN.sh` : désinstalle les outils, vide /tmp, cache, corbeille | [ ] Purge cache / corbeille / historique uniquement (gobox est un binaire, rien à désinstaller) |
| Clonage | `CLONE.sh` : `dd` sur des noms tapés à la main | [ ] Choix de l'image et du disque dans une liste, vérification des tailles, progression, SHA-256 ; ou lanceur vers l'outil de clonage de l'atelier (**lequel ?**) |
| Déploiement post-install | — (maquette seulement) | [ ] À définir : pilotes, utilisateur atelier, nettoyage des journaux |
| Réparation du démarrage | `BOOT.sh` (boot-repair) | [ ] Bouton lanceur |
| Image ISO du système | `ISO.sh` (penguins-eggs) | [ ] Bouton lanceur |
| Installer les prérequis | `install.sh` (30 paquets, noms faux) | [ ] Au démarrage : vérifier les outils facultatifs (`smartctl`, `ffplay`, `secmet`…) et proposer de les installer |
| Lien base de gestion | `BASE.sh` → `gest.goupil-ere.org` | [ ] Bouton lien, puis envoi de la fiche (voir [fiche.md](fiche.md)) |
