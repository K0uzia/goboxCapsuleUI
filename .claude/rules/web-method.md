# Méthode technique (tout projet web)

Source : logique formelle, conventions clean code et validation de CapsuleOS, généralisées. Décider, modifier les règles et remettre un travail : rule `00-organisation`.

## Observer avant d'agir

- Avant d'agir sur l'existant : observer (fichiers, rendu, mesures), lister les écarts réels, ne corriger que ceux-là. Ne jamais deviner une mesure qu'on peut relever.
- Une hypothèse est fausse si le contrôle qui la teste échoue. On corrige le code, jamais le contrôle pour qu'il passe.

## Écrire peu, au bon endroit

- Avant de copier une base ou un modèle dans un dépôt existant, lister les fichiers en collision et les traiter un par un, jamais de `cp -a` à l'aveugle.
- Avant d'ajouter un fichier, un module ou une dépendance : un besoin réel l'exige-t-il ? un existant peut-il être étendu ? est-ce de la donnée plutôt que de la logique ? Trois non : ne pas l'ajouter.
- Une source de vérité par information (identité, contenu, prix, FAQ, tokens). Le reste est généré à partir d'elle. Un fichier généré ne se modifie jamais à la main.
- Diff minimal : une intention, une zone, un contrôle vert. Pas de refactor hors demande.
- Commentaires seulement pour un invariant non évident. Pas de commentaire narratif, pas de TODO non classé.
- Pas de nouvelle doc parallèle (roadmap, plan) : le mode d'emploi du projet couvre le sujet.

## Tirets

- Tiret cadratin (U+2014) et demi-cadratin (U+2013) interdits partout : code, commentaires, textes, données, messages de commit. À la place : deux-points, virgule, parenthèses ou trait d'union selon le sens.
- Tiret déguisé interdit aussi : trait en `::before` ou `::after` devant un sur-titre ou une puce, `content` en tiret (même échappé : `\2014`, `\2013`, `&mdash;`, `&ndash;`), image ou SVG de tiret. À la place : rien, une petite forme pleine (rond, carré) ou une icône.
- Permis : les filets utiles (séparateur de sections, fil d'étapes).

## Écarts et dette

- Tout écart connu est noté et classé : P0 bloquant (cassé, illégal, inaccessible, faux), P1 visible mais contournable, P2 finition.
- Absence d'une donnée réelle : échec visible ou section masquée, jamais un substitut inventé ni emprunté ailleurs.

## Vérifier

- Pendant le travail : contrôle minimal du périmètre touché. Corriger les erreurs de cette zone, pas toute la dette du dépôt.
- Avant de remettre : `make verify`, contrôles du skill `web-audit` adaptés au changement, puis skill `livraison`.
- Lire un fichier TSV ou CSV avec Python (`csv`), jamais avec `while read` : les tabulations vides y sont fusionnées et décalent les colonnes.
- Un serveur ou processus lancé pour un test s'arrête par son PID (`cmd & pid=$!; …; kill $pid`), jamais par motif ou par nom (`pkill`, `killall`, `taskkill /IM`, `Stop-Process -Name`) ; un hook le refuse (`hooks/refuser-arret-par-motif.mjs`).
