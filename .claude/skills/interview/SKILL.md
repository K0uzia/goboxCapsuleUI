---
name: interview
description: Mène l'interview de démarrage d'un projet de site, neuf ou déjà avancé (le dépôt est d'abord analysé pour répondre seul, puis les doutes sont confirmés auprès de l'humain), jusqu'à pouvoir tout faire sans lui, et rédige le mode d'emploi .claude/rules/projet.md. À utiliser quand le mode d'emploi est absent, en brouillon, ou d'un autre dépôt, quand le .claude arrive dans un projet existant, et quand une demande sort du périmètre.
---

# Interview de projet

But : un mode d'emploi assez complet pour que tout le travail puisse être fait sans revenir vers l'humain, sauf pour les validations prévues.

## Deux cas

- **Projet neuf** : peu à observer, l'interview pose l'essentiel des questions.
- **Projet existant** (déjà construit ou avancé, le `.claude` vient d'être ajouté) : l'analyse du dépôt répond d'abord à l'interview ; les questions ne portent que sur ce qui est déduit ou inconnu.

## Déroulé

1. **Analyser sans rien modifier** (tous les cas, approfondi pour un projet existant) :
   - documentation : README, `CLAUDE.md`, `AGENTS.md`, docs, ancien mode d'emploi, commentaires d'en-tête ;
   - historique : `git log` (rythme, auteurs, dernières zones touchées), branches, tags ;
   - technique : `package.json`, `composer.json`, `Makefile`, scripts, structure de `src/`, routes, base de données, `.env.example` (noms seulement, jamais les valeurs) ;
   - hébergement et mise en ligne : `.htaccess`, scripts de déploiement, CI, domaine dans la config ;
   - contenu et site : pages, données, mentions légales, formulaires, traceurs, langues ; site en ligne ou copie `wget` s'il y en a une ;
   - état : lancer les contrôles existants (`make verify`, tests, build) et noter le résultat.
2. **Classer chaque information** avec sa source :
   - `constaté` : lu dans un fichier ou une sortie de commande (citer le chemin) ;
   - `déduit` : probable mais non écrit (dire de quoi c'est déduit) ;
   - `inconnu`.
3. **Créer le mode d'emploi en `brouillon`** depuis `modele-mode-emploi.md`, champ `depot` = nom du dossier du dépôt, rempli avec le `constaté` et le `déduit` (ce dernier marqué `à confirmer`). Projet existant : le socle en place est un fait (« Décisions validées » : `constaté AAAA-MM-JJ`), pas un choix à refaire ; les écarts relevés vont dans « Règles propres au projet » classés P0/P1/P2.
4. **Présenter ce qui a été compris**, par thème, en peu de lignes : « J'ai compris que… (source) ». Demander une confirmation groupée du `constaté`, puis poser une question par point `déduit` (« Est-ce bien… ? ») et par point `inconnu`.
5. **Questionner par thème** pour le reste, dans l'ordre ci-dessous, 3 à 5 questions par message, chacune avec une proposition par défaut quand il en existe une raisonnable. Ne jamais demander ce qui est `constaté` sans doute.
6. **Relancer** toute réponse floue (« on verra », « comme d'habitude », « normal ») par une question fermée ou un exemple concret.
7. **Remplir au fil de l'eau** : chaque réponse va dans sa section du mode d'emploi ; un `déduit` confirmé devient `constaté (confirmé le AAAA-MM-JJ)`.
8. **Arrêt** quand chaque champ du modèle est rempli, ou marqué `à fournir : <qui>, <pour quand>`, ou `sans objet`, et qu'il ne reste aucun `à confirmer`.
9. **Relecture** : présenter le mode d'emploi complet, demander sa validation, puis passer le statut à `validé` avec la date.

## Questions par thème

| Thème | À obtenir |
|-------|-----------|
| Demande | la demande dans les mots de l'humain ; ce qui prouvera que c'est réussi (critères vérifiables) ; échéance |
| Client et contexte | structure, activité, territoire, identité légale (raison sociale, SIRET, adresse, directeur de publication) |
| Publics | qui visite, pour quelle action principale, par ordre de priorité ; ce qu'il doit trouver en un écran |
| Contenu existant | textes, photos, logo, charte, documents, ancien site ; droits sur les images (personnes, enfants) |
| Contenu manquant | qui le fournit, pour quand ; ce qui est masqué en attendant |
| Type de site | vitrine, vitrine-formulaire, associatif, boutique, espace-client ou outil-interne (rule `types-de-site`) : il fixe les briques, les obligations et les exclusions ; rien d'un type plus lourd sans décision écrite |
| Socle technique | statique PHP (architecture Astro), Astro (l'outil), ou dynamique PHP (rule `web-architecture`) ; comptes, paiement, formulaires, données personnelles |
| Mise à jour | qui modifie le contenu, à quelle fréquence, avec quelles compétences |
| Hébergement | hébergeur imposé, accès (FTP, SSH), domaine, DNS, e-mails ; qui détient les accès |
| Modèle à respecter | charte graphique, maquette, page type, site de référence, document déjà validé ; sinon `sans objet` |
| Ton et vocabulaire | registre, mots imposés ou interdits, langue(s) |
| Contraintes | accessibilité exigée (secteur public, EAA), budget, délais, prestataires imposés |
| Validation | qui valide quoi, à quelles étapes ; ce qui part sans validation |
| Rapport | jour du rapport hebdo, points à suivre en particulier |

## Règles

- Jamais de choix de cadrage à la place de l'humain : proposer, recommander, faire trancher.
- Une information donnée en passant dans une autre réponse se range quand même dans sa section.
- Un chiffre ou un fait légal non sourcé reste `à confirmer`.
- L'interview reprend dès qu'une demande sort du périmètre : questions limitées au nouveau sujet, puis mise à jour du mode d'emploi (rule `00-organisation`, « Modifier les règles »).
