---
name: rapport-hebdo
description: Rédige le rapport hebdomadaire d'un projet dans docs/rapports/AAAA-Sxx.md à partir du mode d'emploi, de l'historique git et du journal des erreurs - fait avec preuves, en cours, en attente de l'humain, décisions à prendre, semaine suivante. À utiliser chaque semaine ou quand l'humain demande où en est le projet.
---

# Rapport hebdo

Fichier : `docs/rapports/AAAA-Sxx.md` (année et numéro de semaine ISO : `date +%G-S%V`). Destinataire : le porteur du projet seul. Un fichier par semaine, jamais réécrit après coup : une correction se fait dans le rapport suivant.

## Sources

- Mode d'emploi : étapes et leur état, « À fournir », décisions, journal des erreurs.
- `git log --since="7 days ago" --oneline` et les fichiers touchés.
- Résultats des contrôles de la semaine (`make verify`, audits).

Rien d'autre : pas de souvenir de conversation non tracé, pas d'estimation.

## Format

```markdown
# Rapport semaine AAAA-Sxx : <projet>

## En une phrase
…

## Fait cette semaine
| Étape | Livrable | Preuve |
|-------|----------|--------|

## En cours
| Étape | Où on en est | Prochain pas |
|-------|--------------|--------------|

## En attente de toi
| Élément ou validation | Depuis | Bloque |
|-----------------------|--------|--------|

## Décisions à prendre
- … (avec la recommandation)

## Erreurs et règles ajoutées
| Erreur | Règle (où) |
|--------|------------|

## Semaine prochaine
- …

## État des contrôles
`make verify` : … (date, résultat)
```

Section sans contenu : écrire « rien », ne pas la supprimer.
