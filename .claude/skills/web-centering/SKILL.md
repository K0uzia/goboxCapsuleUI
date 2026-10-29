---
name: web-centering
description: Décide ce qui doit être centré sur une page (règle de web-design.md, section Centrage) et le vérifie par la mesure - marges gauche et droite de chaque bloc, à plusieurs largeurs, avec le script check-centering.mjs. À utiliser pour toute nouvelle page ou section (confirmation, formulaire, page légale, onglets, étapes, état vide) et avant de déclarer une mise en page terminée.
---

# Centrage

Source : `web-design.md` (section Centrage) ; logique de mesure reprise de `KouziaCRM/scripts/check-desktop.ts` (marges gauche et droite comparées au pixel près).

## 1. Décider, bloc par bloc

Pour chaque bloc de la page, répondre dans l'ordre :

1. Le bloc est-il plus étroit que son conteneur ? Non : rien à centrer.
2. Forme-t-il une colonne avec un voisin (barre latérale, sommaire, récapitulatif de commande) ? Oui : la paire est centrée, pas le bloc seul.
3. Est-il un en-tête de page isolée, un état (vide, réussite, erreur), une frise d'étapes, une colonne de formulaire seule, une barre d'onglets de colonne, un groupe d'actions sous un texte centré ? Oui : centré (bloc et contenu).
4. Est-il du texte courant, une liste, un tableau, un champ ? Aligné à gauche, dans une colonne elle-même centrée.

Écrire la décision dans le CSS : `margin-inline: auto` ou `justify-items: center` sur le parent, `text-align: center` seulement pour les titres, chapeaux, états et étapes.

## 2. Motifs prêts à l'emploi

- Colonne étroite centrée : parent `display: grid; justify-items: center`, enfants `width: 100%; max-width: <mesure>`.
- Paire colonne + voisin : `grid-template-columns: <voisin> minmax(0, <mesure>); justify-content: center`.
- Onglets d'une colonne : bande segmentée `display: flex; justify-content: center`, onglets `flex: 1 1 auto; justify-content: center`.
- Frise d'étapes : `grid-template-columns: repeat(n, minmax(0, 1fr))`, chaque étape `justify-items: center; text-align: center`, trait de liaison en pseudo-élément ; en colonne sous 640 px.
- Actions d'un formulaire centré : `display: flex; justify-content: center`.

## 3. Mesurer (obligatoire avant « terminé »)

```bash
node .claude/skills/web-centering/check-centering.mjs http://127.0.0.1:4321 .claude/skills/web-centering/pages.json
```

- `pages.json` : liste `{ "path", "check": [{ "selector", "within"? }] }`. Sans `within`, la référence est la fenêtre (barre de défilement exclue) ; avec `within`, la boîte de contenu du parent indiqué.
- Le script ouvre chaque page à 360, 800, 1280 et 1920 px et signale tout écart de marges supérieur à 3 px, ainsi que tout défilement horizontal.
- Ajouter chaque nouvelle page centrée au fichier ; une page qui échoue n'est pas livrée.
- Dépendance : `playwright-core` (déjà utilisé pour les tests du projet).
