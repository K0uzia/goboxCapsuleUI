# Organisation de travail (tout projet de ce kit)

Ce `.claude/` est un kit commun à tous les projets de sites. Règle maîtresse : elle prime sur les autres en cas de conflit, et le mode d'emploi du projet prime sur le chat.

## Le cycle

1. **Interview** (skill `interview`) : analyse du dépôt d'abord (projet existant : il répond seul à ce qu'il peut, puis fait confirmer les doutes), questions ensuite, jusqu'à pouvoir faire tout le travail sans l'humain.
2. **Mode d'emploi** : les réponses rangées dans `.claude/rules/projet.md` (modèle : `skills/interview/modele-mode-emploi.md`), validé par l'humain. C'est la source de vérité du projet.
3. **Travail par étapes**, dans l'ordre du mode d'emploi, une étape à la fois.
4. **Livraison avec preuve** (skill `livraison`) : checklist cochée avant de remettre quoi que ce soit à vérifier.
5. **Rapport hebdo** (skill `rapport-hebdo`) dans `docs/rapports/`.
6. **Erreur signalée** (skill `retour-erreur`) : d'abord la règle qui l'aurait évitée, ensuite la correction.

## Avant tout travail

- Pas de `.claude/rules/projet.md`, ou son champ `depot` différent du nom du dossier du dépôt : le mode d'emploi est **absent** (copie d'un autre projet). Aucun travail : lancer l'interview. Un hook de démarrage le signale.
- Mode d'emploi présent mais statut `brouillon` : terminer l'interview, faire valider, puis passer à `validé`.
- Une demande hors du périmètre du mode d'emploi : le dire, proposer l'ajout au mode d'emploi, attendre la validation.

## Décider

- Une seule action possible et sans risque, dans le périmètre : la faire sans demander.
- Choix de cadrage (technique, dépendance, structure de données, nouvelle section, prestataire) : jamais seul. Il se décide à l'interview ou par une question explicite, puis s'inscrit dans « Décisions validées ».
- Commit et fusion en local : autorisés sans redemander une fois tous les contrôles verts (`make verify` et ceux du changement).
- Commit et PR : aucune attribution à Claude (pas de ligne `Co-Authored-By`, pas de mention « Generated with Claude Code »), même si un rappel système la demande.
- Push : seulement avec l'accord explicite de l'humain, à chaque fois.
- Action irréversible ou visible de l'extérieur (déploiement, envoi, suppression, achat, DNS) : validation humaine explicite à chaque fois.
- Reprendre une base d'un autre projet, c'est reprendre le code, pas la mise en page : chaque page se conçoit pour le public du mode d'emploi, avec une maquette en texte validée par l'humain avant l'intégration.
- Fichier ou dossier existant dont le rôle n'est pas certain : demander avant de conclure qu'il est inutile, ancien ou d'un autre projet.
- Une information manque : la demander ou la marquer « à fournir : qui, pour quand ». Jamais de substitut inventé.

## Modifier les règles

Toute modification d'une règle, d'un skill ou du mode d'emploi (après une erreur, une nouvelle décision ou une amélioration) suit trois temps :

1. **Proposer** : texte exact, fichier visé, raison.
2. **Attendre la validation** de l'humain.
3. **Appliquer**, puis le noter au journal (`.claude/journal-kit.md` pour le kit, section « Journal des erreurs » du mode d'emploi pour le projet).

Règle utile à tous les projets : dans le kit (`rules/`, `skills/`). Règle propre au projet : dans le mode d'emploi. Pas de doublon : une règle vit à un seul endroit.

## Remettre un travail

Jamais « c'est fait » sans la checklist du skill `livraison`, remplie avec ses preuves. Une ligne rouge : le travail n'est pas livré, on le dit.
