#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Hook SessionStart : vérifie que le mode d'emploi (.claude/rules/projet.md) existe,
# appartient bien à ce dépôt (champ « depot ») et est validé. Le texte affiché
# est ajouté au contexte de la session (rule 00-organisation).
set -uo pipefail

racine="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
depot="$(basename "$racine")"
fichier="$racine/.claude/rules/projet.md"

if [[ ! -f "$fichier" ]]; then
    echo "MODE D'EMPLOI ABSENT pour le dépôt « $depot » : aucun travail avant l'interview (skill interview, rule 00-organisation)."
    exit 0
fi

declare_depot="$(grep -m1 -E '^- depot:' "$fichier" | sed -E 's/^- depot:[[:space:]]*//; s/[[:space:]]*$//')"
statut="$(grep -m1 -E '^- statut:' "$fichier" | sed -E 's/^- statut:[[:space:]]*//')"

if [[ "$declare_depot" != "$depot" ]]; then
    echo "MODE D'EMPLOI D'UN AUTRE PROJET : .claude/rules/projet.md déclare « ${declare_depot:-rien} », le dépôt est « $depot ». Le traiter comme absent : lancer l'interview (skill interview) et ne rien appliquer de ce fichier."
elif [[ "$statut" != validé* ]]; then
    echo "MODE D'EMPLOI EN BROUILLON (statut : ${statut:-inconnu}) : terminer l'interview et le faire valider avant tout travail."
else
    echo "Mode d'emploi du dépôt « $depot » : $statut. Il fait foi ; livraison avec le skill livraison."
fi

type_site="$(grep -m1 -E '^- type:' "$fichier" | sed -E 's/^- type:[[:space:]]*//; s/[[:space:]]*$//')"
case "$type_site" in
    vitrine|vitrine-formulaire|associatif|boutique|espace-client|outil-interne)
        echo "Type de site : $type_site. Briques, obligations et exclusions : rules types-de-site et lecons-securite." ;;
    *)
        echo "TYPE DE SITE ABSENT ou invalide dans le mode d'emploi (« ${type_site:-rien} ») : le fixer à l'interview (rule types-de-site) avant de coder." ;;
esac
exit 0
