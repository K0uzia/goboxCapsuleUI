---
name: retour-erreur
description: Transforme une erreur signalée par l'humain (ou découverte en travaillant) en règle durable dans le mode d'emploi ou le kit, avant de corriger le travail. À utiliser dès qu'une erreur, un oubli, une mauvaise supposition ou un contrôle raté est relevé.
---

# Retour d'erreur

Principe : une erreur ne se corrige pas seulement dans le chat. La règle qui l'aurait évitée est écrite d'abord, la correction vient ensuite.

## Étapes

1. **Nommer** l'erreur en une phrase factuelle : ce qui était attendu, ce qui a été fait.
2. **Trouver la cause** : règle absente, règle ignorée, question non posée, contrôle non lancé, supposition non vérifiée.
3. **Choisir l'endroit**, un seul :
   - utile à tous les projets → le fichier du kit concerné (`rules/…` ou `skills/…`) ;
   - propre au projet → mode d'emploi, « Règles propres au projet » ou « Ajouts à la checklist de livraison » ;
   - règle déjà présente mais ignorée → la rendre vérifiable (ligne de checklist, contrôle automatique) plutôt que la répéter.
4. **Proposer** à l'humain : texte exact, fichier et section, cause. Attendre la validation.
5. **Appliquer** la règle, puis la tracer : ligne au « Journal des erreurs » du mode d'emploi, et dans `.claude/journal-kit.md` si le kit a changé.
6. **Corriger** le travail en appliquant la nouvelle règle.
7. **Livrer** avec la checklist du skill `livraison` (ligne 8 : journal respecté).

## Une bonne règle

- Formulée comme une action vérifiable (« le build échoue au moindre avertissement »), pas comme une intention (« faire attention aux avertissements »).
- Courte, au bon endroit, sans doublon avec une règle existante.
- Quand c'est possible, rendue automatique : contrôle dans `make verify`, hook, ligne de checklist.
