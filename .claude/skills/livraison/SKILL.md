---
name: livraison
description: Checklist de preuve à cocher avant de remettre un travail à vérifier - répond à la demande, fait avec le modèle, chiffres vérifiés, contrôles lancés, prêt à envoyer, non vérifié déclaré, journal des erreurs respecté. À utiliser à la fin de chaque tâche, avant de dire « c'est fait ».
---

# Livraison avec preuve

Aucun travail n'est remis sans ce tableau rempli. Une preuve est un fait observable : sortie de commande, fichier et ligne, URL et code HTTP, capture. « Ça devrait marcher » n'est pas une preuve.

## Checklist

| # | Contrôle | Preuve exigée |
|---|----------|---------------|
| 1 | **Répond à la demande** | la demande reprise point par point, et pour chaque point où il est traité (fichier, page, section) ; tout point non traité est dit |
| 2 | **Fait avec le modèle** | (a) modèle du projet (mode d'emploi, « Modèle à respecter ») : ce qui a été comparé, ou `sans objet` ; (b) kit : rules et conventions applicables respectées |
| 3 | **Chiffres vérifiés** | chaque chiffre, date, prix, horaire, mesure livré : sa source ou la commande qui le produit ; aucun chiffre estimé ni recopié sans source |
| 4 | **Contrôles lancés** | commandes exécutées (`make verify` au minimum, plus celles du skill `web-audit` selon le changement) et leur résultat réel : erreurs, avertissements restants et leur justification |
| 5 | **Rendu observé** | pour une page touchée : rendue et regardée, puis jugée comme le public du site la verra (images nettes, texte lisible, mise en page conçue et non composants empilés ; sinon la ligne est rouge), console sans erreur ; sinon la ligne est rouge ou marquée `non vérifié`. Captures : pour une nouvelle mise en page ou la reprise d'un autre site, chaque page en bureau et 390 px, clair et sombre (contrôle visuel en plus des recherches de texte) ; pour une petite modification, une capture de la zone touchée suffit. Captures à 390 px avec Firefox sans interface (Chrome sans interface impose une largeur minimale) |
| 6 | **Prêt à envoyer** | aucun marqueur `à fournir`, TODO, texte de démo ou contenu inventé dans ce qui est livré ; liens testés ; orthographe relue ; nommage et format conformes |
| 7 | **Non vérifié déclaré** | liste explicite de ce qui n'a pas pu être vérifié, et pourquoi |
| 8 | **Journal respecté** | chaque règle du « Journal des erreurs » du mode d'emploi et de `.claude/journal-kit.md` concernée par ce travail : respectée |
| 9 | **Ajouts du projet** | chaque ligne de « Ajouts à la checklist de livraison » du mode d'emploi |
| 10 | **Mode d'emploi à jour** | étape passée à `livré`, nouvelles décisions et éléments à fournir inscrits (après validation) |

## Compte rendu remis

1. Une phrase : ce qui est livré.
2. Le tableau : `# | Contrôle | Statut (vert, rouge, sans objet) | Preuve`.
3. Ce qui reste à faire et ce qui attend l'humain (validation, éléments à fournir).

Une ligne rouge : le dire en tête. Le travail n'est pas présenté comme fini.

## Interdits

- Cocher une ligne sans avoir lancé le contrôle dans cette session.
- Affaiblir un contrôle pour qu'il passe (rule `web-method`).
- Résumer une sortie de commande de façon plus favorable qu'elle n'est.
