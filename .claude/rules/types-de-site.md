# Types de site : ce qui s'applique, ce qui est exclu

Chaque projet a un type, choisi à l'interview et écrit dans le mode d'emploi (`- type:`). Le type décide du socle, des briques de la boîte à outils, des obligations et des interdits. Rien d'un type plus lourd n'entre dans un type plus léger sans décision écrite dans « Décisions validées » : pas de panier sur une vitrine, pas de comptes ni de back-office sur un site sans besoin d'édition, pas de logique d'ERP dans un site client.

Les leçons de sécurité qui s'appliquent à chaque type sont dans la rule `lecons-securite` (colonne « Types »).

## Les types

| Type | Quand | Socle (rule `web-architecture`) |
|------|-------|------------------|
| `vitrine` | présenter une activité, aucune donnée saisie traitée sur place | statique PHP (ou Astro sur demande) |
| `vitrine-formulaire` | vitrine avec contact ou demande de devis | statique + service de formulaire, ou point d'envoi PHP minimal |
| `associatif` | contenus mis à jour par le client (actualités, agenda, documents, newsletter) | dynamique PHP avec back-office |
| `boutique` | vente en ligne, panier, paiement | dynamique PHP, paiement par prestataire |
| `espace-client` | comptes, suivi, documents privés, application web | dynamique (PHP ou Node), pages publiques séparées |
| `outil-interne` | logiciel de gestion (ERP, hub, tableau de bord métier) pour un seul utilisateur ou une équipe | application (Fastify, React ou PHP), jamais indexée |

## Ce que chaque type reçoit

### `vitrine`
- **Briques** : `gabarits/socle-css`, `gabarits/page-base-seo`, `sections/*` (héros, services, réalisations, témoignages, pied de page), `gabarits/pages-legales-*`, `gabarits/page-404`, `modules/durcissement-apache`. Sur demande : `modules/multilingue` (site en plusieurs langues), `modules/export-statique` (site écrit en PHP livré en HTML), `composants/panneau-accessibilite`, `sections/carrousel` (jamais automatique en héros).
- **Obligations** : SEO complet (`web-seo`), données structurées de l'activité, pages légales, déclaration d'accessibilité, page 404 au gabarit du site, poids < 500 Ko par page.
- **Exclus** : session, cookie, base de données, back-office, comptes, panier, JavaScript d'application, script tiers (carte, chat, pixel). Aucun traitement de données personnelles sur place. Une carte interactive (`composants/carte-leaflet`, tuiles d'un service tiers) seulement par décision écrite, chargée au clic ; sinon adresse et lien d'itinéraire.

### `vitrine-formulaire`
- Tout `vitrine`, plus `modules/formulaire-contact` (ou `composants/envoi-sans-rechargement` + `modules/securite-formulaires` si l'envoi est traité en PHP).
- **Obligations** : anti-robots (pot de miel, délai minimal), limite de débit côté serveur, réponse identique quel que soit le cas, politique de confidentialité liée au formulaire, collecte minimale, pas de case pré-cochée.
- **Exclus** : comptes, back-office, stockage durable des messages sans durée de conservation écrite.

### `associatif`
- Tout `vitrine-formulaire`, plus `gabarits/socle-php`, `modules/authentification-admin` (2FA), `modules/comptes-admin-invitation` ou `modules/roles-et-journal` si plusieurs personnes éditent, `gabarits/tableau-de-bord-php` ou `gabarits/admin-coquille`, et seulement les modules demandés (actualités, agenda, documents, médiathèque, newsletter, sondage, pages en sections, prise de rendez-vous, réservation de matériel, recherche sur le site, vidéo locale, import tableur). Hébergement mutualisé sans SSH : `gabarits/installation-mutualise`.
- **Obligations** : toutes les leçons « back-office » de `lecons-securite`, sauvegarde de la base (`modules/sauvegarde-chiffree`), journal des actions, export CSV protégé (`modules/export-tableur`), tests automatiques (`modules/tests-et-ci-php`), durées de conservation, déclaration d'accessibilité (souvent obligatoire pour le secteur public et associatif subventionné : à trancher à l'interview), documents PDF accessibles ou version web (`modules/documents-version-web`).
- **Exclus** : paiement en ligne (on renvoie vers HelloAsso ou équivalent), comptes publics (`modules/espace-membre`, exigé par `modules/reservation-materiel`), sauf décision écrite.

### `boutique`
- Tout `associatif` côté administration, plus panier en session, paiement par prestataire (Stripe : webhook signé, idempotence atomique, mode test sur la recette), suivi de commande (`modules/acces-invite-lien-signe`), e-mails transactionnels (`modules/envoi-email` avec file d'envoi).
- **Obligations** : CGV, prix TTC, droit de rétractation, médiateur de la consommation, mentions de livraison, aucune carte stockée, clés de paiement réelles refusées hors production, `Product` et `Offer` en JSON-LD.
- **Exclus** : stockage de moyen de paiement, prix calculé côté navigateur (toujours relu côté serveur).

### `espace-client`
- Socle dynamique, `modules/espace-membre` (comptes publics ; `modules/authentification-admin` pour l'équipe), sessions courtes avec avertissement (`composants/minuteur-session`), pages publiques statiques séparées de l'application (`noindex` sur l'application). Application de terrain utilisée sans réseau : `modules/hors-ligne`.
- **Obligations** : contrôle d'accès vérifié sur chaque ressource (jamais par identifiant devinable), réponses identiques qui ne révèlent pas l'existence d'un compte, jetons à usage unique, export et suppression des données (RGPD).

### `outil-interne`
- Socle applicatif, `modules/securite-api-fastify` (Node) ou les modules PHP de sécurité, composants d'administration (`gabarits/admin-coquille`, `sections/admin-*`, composants React si le socle est React), `modules/sauvegarde-chiffree`, `modules/tests-et-ci-php`, `composants/minuteur-session`, `modules/import-tableur` et `modules/export-tableur` pour la reprise et la sortie des données.
- **Obligations** : mot de passe et 2FA obligatoires, accès par tunnel ou réseau privé, hôtes séparés (administration et API publique), `noindex` partout, journaux sans secrets, sauvegardes chiffrées, tests automatiques sur chaque règle métier et chaque droit.
- **Exclus** : référencement, pages publiques mêlées à l'administration, données d'un client dans un autre.

## Commun à tous les types

- Traceur non exempté (publicité, réseau social, vidéo tierce) : `composants/consentement-cookies`. Mesure d'audience exemptée seule : aucune bannière (rule `web-legal`).

## Contrôle

- Le mode d'emploi sans `- type:` est incomplet : l'interview le demande.
- Avant d'ajouter une brique, vérifier qu'elle appartient au type du projet. Sinon, le dire et proposer la décision (rule `00-organisation`, « Décider »).
