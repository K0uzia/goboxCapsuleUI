# Effacement des disques : SecMet

## Décision

On **ne réécrit pas** l'effacement dans gobox : on s'appuie sur **SecMet**
(`~/Bureau/team/#INTERNE/Programmes/SecMet`, Rust / GTK4, v0.3.0). SecMet fait
déjà tout ce que LACAPSULE faisait mal :

- effacement NIST SP 800-88 par commandes firmware (ATA Security Erase / Sanitize, NVMe Format / Sanitize), écriture hôte vérifiée en repli, sans `hdparm` / `shred` / `nvme-cli` ;
- disque système protégé, disques USB en écriture hôte uniquement ;
- SSD « gelés » (frozen) : dialogue dédié + mise en veille ;
- effacements en parallèle, arrêt possible ;
- vérification après effacement (relecture d'échantillons, TestDisk) ;
- **certificat PDF signé Ed25519** avec QR code vérifiable hors ligne ;
- droits via `pkexec` (une seule demande), paquet `.deb` et AppImage.

Les scripts `SHREDHDD.sh` / `WIPESSD.sh` de LACAPSULE ne sont **pas** repris
(saisie manuelle de la lettre du disque, aucune protection, erreurs de syntaxe).

## Contrainte d'usage

> L'effacement se lance **uniquement en étant connecté au PC que l'on
> reconditionne** (session sur ce poste, démarré sur la clé / le réseau), jamais
> depuis un autre poste ni sur un banc qui héberge d'autres disques.

À traduire dans gobox :

- [ ] N'afficher le bouton **Shred** que si gobox tourne **sur la machine en cours de reconditionnement**, démarrée en live (racine du système non portée par un disque interne : `/run/live/medium`, NFS, overlay…) → les disques internes ne sont jamais le système
- [ ] Refuser (message clair) si gobox tourne depuis un système installé sur un disque interne
- [ ] Afficher le n° de série DMI de la machine dans la confirmation, pour être sûr d'être sur le bon PC
- [ ] ❓ À confirmer avec l'équipe : « connecté au PC » = session locale sur le poste (pas de bureau à distance RustDesk / SSH) ? Si oui, refuser aussi quand la session est distante

## Intégration (étapes)

1. [ ] **Lanceur** : bouton *Shred* (catégorie Stockage) → lance `secmet` (paquet) ou l'AppImage (chemin dans les réglages), gobox attend sa fermeture
2. [ ] **Retour dans la fiche** : lire le dernier `~/.secmet_logs/session_report_<date>.json` → pour chaque disque : modèle, n° de série, méthode, catégorie NIST (Purge / Clear), résultat ; cocher « Shred des disques : oui » automatiquement
3. [ ] Joindre / référencer le certificat PDF (`~/.secmet_logs/certificates/`) dans la fiche
4. [ ] Installer SecMet dans `make deps` (ou embarquer l'AppImage dans le paquet gobox)
5. [ ] Plus tard (à discuter) : utiliser directement les crates Rust `secmet-core` / `secmet-helper` via le helper privilégié plutôt que l'interface GTK → une seule fenêtre

## À lire dans SecMet avant d'intégrer

- `docs/DEV_CONTEXT.md` : emplacement et format des journaux / rapports
- `docs/CAHIER_DES_CHARGES.md`
- `packaging/secmet.policy` : règle PolicyKit (gobox pourra s'en inspirer pour ses propres droits)
