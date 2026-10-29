# Boîte à outils Kouzia (briques réutilisables)

Les composants, sections, modules et gabarits déjà réalisés vivent dans un dossier voisin du dépôt : `../kouzia-boite-a-outils/`. Le but : ne jamais recoder ce qui existe (temps, code et jetons économisés).

## Avant de coder

- Avant de créer un composant (N1), une section (N2), un module (fonction complète : formulaire, newsletter, sondage, back-office…) ou un gabarit : lire `../kouzia-boite-a-outils/INDEX.md` (une ligne par brique, fait pour être lu en entier à peu de frais). Ne pas parcourir les dossiers des briques sans raison.
- Une brique correspond : ouvrir seulement sa fiche (`fiche.json`, `NOTES.md`) et ses fichiers, la copier dans le projet, puis l'adapter aux tokens et au contenu du projet. On reprend le code, pas la mise en page (rule `00-organisation`) : la place et l'apparence se décident pour le public du projet.
- Le socle doit correspondre (champ `socles` de la fiche). Sinon, l'adapter et le signaler.
- Une brique proche mais pas identique : l'étendre dans le projet (variante, propriété personnalisée) plutôt que de repartir de zéro.
- Aucune brique : coder dans le projet, puis proposer son ajout (skill `boite-a-outils`, « Ajouter »).

## La boîte reste générique

- Jamais de contenu, de nom, de logo, de photo, de couleur de marque ni de donnée d'un client dans la boîte : textes neutres, valeurs prises dans `base/tokens.css`.
- Une brique ne s'ajoute ou ne se modifie dans la boîte qu'avec l'accord de l'humain (rule `00-organisation`, « Modifier les règles »). Depuis un projet client, on propose, on n'écrit pas seul.
- `INDEX.md` et `catalogue.html` sont générés par `node ../kouzia-boite-a-outils/outils/indexer.mjs` : jamais modifiés à la main.
