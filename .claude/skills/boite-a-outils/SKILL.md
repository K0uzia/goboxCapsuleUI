---
name: boite-a-outils
description: Cherche, reprend ou ajoute une brique (composant, section, module, gabarit) dans la boîte à outils Kouzia (../kouzia-boite-a-outils). À utiliser avant de coder un élément d'interface ou une fonction qui a pu déjà être faite, et après en avoir créé un réutilisable.
---

# Boîte à outils

Contraintes : rule `boite-a-outils`. Format exact d'une brique : `../kouzia-boite-a-outils/README.md`.

## Chercher

1. Lire `../kouzia-boite-a-outils/INDEX.md` en entier (court par construction).
2. Retenir au plus trois candidates (catégorie, socle, mots-clés). Ouvrir leur `fiche.json` et `NOTES.md` seulement.
3. Dire à l'humain ce qui est repris, d'où (`elements/<categorie>/<id>`), et ce qui reste à faire.

## Reprendre

1. Copier les fichiers de la brique dans le projet, au bon endroit du socle (`src/components/`, `src/styles/components.css`, `templates/…`).
2. Remplacer les tokens de `base/tokens.css` par ceux du projet (mêmes noms : rien à faire si le projet suit le skill `web-design-system`).
3. Remplacer les textes neutres par le contenu réel du projet. Aucune donnée manquante inventée : la section se masque.
4. Contrôles habituels du projet (`make verify`, skill `livraison`).

## Ajouter (avec l'accord de l'humain)

Une brique mérite la boîte si elle a servi, fonctionne et peut servir à un autre site.

1. Proposer : nom, catégorie (`composants`, `sections`, `modules`, `gabarits`), socles, ce qui la rend réutilisable, licence du projet d'origine. Attendre l'accord. Licence GPL ou inconnue : réécrire entièrement, ne rien copier ; provenance dans `fiche.json` (`origine`).
2. Créer `../kouzia-boite-a-outils/elements/<categorie>/<id>/` avec `fiche.json`, `apercu.html`, `style.css`, `script.js` si besoin, `source/` (fichiers d'origine du socle) et `NOTES.md` (intégration, variantes, pièges).
3. Rendre la brique générique : textes neutres, tokens de `base/tokens.css`, aucune marque ni donnée du client.
4. Lancer `node ../kouzia-boite-a-outils/outils/indexer.mjs` : il vérifie les fiches, refuse les tirets interdits et regénère `INDEX.md` et `catalogue.html`.
5. `node ../kouzia-boite-a-outils/outils/captures.mjs <dossier temporaire> elements/<categorie>/<id>` : 1440 et 390 px, sombre et clair, sans erreur ni défilement horizontal. Regarder les quatre captures.
6. Commit dans le dépôt de la boîte (local, push sur accord).
