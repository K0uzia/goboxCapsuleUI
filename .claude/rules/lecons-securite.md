# Leçons de sécurité (défauts réellement trouvés dans les projets Kouzia)

Chaque ligne est un défaut constaté puis corrigé dans un projet réel, devenu règle. Elle s'applique aux types de site indiqués (rule `types-de-site`) : V vitrine, F vitrine-formulaire, A associatif, B boutique, E espace-client, O outil-interne. Une nouvelle faille trouvée s'ajoute ici (proposer, valider, appliquer, rule `00-organisation`).

## Entrées et requêtes

| Règle | Contrôle | Types |
|-------|----------|-------|
| L'adresse IP du client est `REMOTE_ADDR` (ou `request.ip` dérivé du proxy du tunnel). `X-Forwarded-For`, `Client-IP` et `X-Forwarded-Proto` ne sont lus que si la connexion vient d'un mandataire déclaré dans la configuration. Sinon une limite de débit se contourne en changeant un en-tête. | test : une requête avec `X-Forwarded-For` falsifié garde la même IP | F A B E O |
| Une redirection n'accepte qu'un chemin local : un seul `/` en tête, jamais `//`, `/\`, `/%5C`, ni caractère de contrôle (une tabulation redonne `//` dans le navigateur). Sinon : accueil. | test : `/\autre-site`, `//autre-site`, `/%09/autre` redirigent vers `/` | A B E O |
| Un lien envoyé par e-mail (invitation, réinitialisation, confirmation) se construit depuis l'URL configurée du site (`APP_URL`), jamais depuis l'en-tête `Host`. | test : POST avec `Host: autre.exemple`, le lien reçu pointe sur `APP_URL` | F A B E O |
| Motifs de routes : partie fixe échappée (`preg_quote`), sinon `/panierXjson` sert `/panier.json`. `HEAD` permis quand `GET` l'est. | test de routeur | A B E O |
| Un lien saisi par un éditeur (`href` de carte, bouton, menu) refuse `javascript:`, `data:` et `vbscript:`. | test de l'assainisseur | A B E O |
| CORS et garde d'origine par préfixe de route : l'origine du site public n'est autorisée que sur l'API publique, jamais sur l'administration (un cookie SameSite=Lax part depuis un sous-domaine du même site). | test : requête admin avec l'Origin du site public refusée | E O |
| Un identifiant venu de l'URL n'entre jamais tel quel dans un chemin de fichier (le routeur décode %2F) : format validé (cuid, UUID), ressource vérifiée avant écriture, chemin résolu contrôlé sous le dossier prévu. | test : ..%2F refusé | A B E O |
| Liens de paiement reçus d'une API (Stripe Checkout, portail) : liste blanche des hôtes du prestataire avant toute redirection. | test : hôte inconnu refusé | B E |

## Formulaires, sessions et comptes

| Règle | Contrôle | Types |
|-------|----------|-------|
| Un refus CSRF répond 403 générique par le pipeline normal, sans `die`/`exit` et sans jamais renvoyer jeton, identifiant de session ou clés du formulaire (même en « debug »). | test : POST sans jeton, réponse sans aucune valeur de session | F A B E O |
| Connexion, mot de passe oublié, inscription : même message et même travail (hachage factice, mêmes requêtes) que le compte existe, n'existe pas ou soit bloqué. « Compte verrouillé » ne s'affiche jamais pour un compte précis. | test des trois cas, même texte | A B E O |
| Jeton à usage unique (réinitialisation, invitation, confirmation) : seul son SHA-256 est stocké ; consommation par écriture conditionnelle (`UPDATE … WHERE used_at IS NULL`, une ligne modifiée) ; durée courte ; un nouveau jeton annule les précédents ; purge planifiée. | test : double soumission, une seule réussit | A B E O |
| Changement ou réinitialisation de mot de passe : toutes les autres sessions sont fermées (empreinte du hachage en session ou table de sessions), la 2FA reste demandée ensuite. | test avec deux sessions | A B E O |
| Un administrateur ne fixe jamais le mot de passe d'un autre compte : il envoie une invitation ou une réinitialisation. | relecture | A B E O |
| Seuil de blocage comparé en PHP ou lié en entier : un paramètre lié en texte peut rendre la comparaison SQL toujours fausse (vu sous SQLite). | test sous SQLite et MariaDB | A B E O |
| Limite de débit par IP ET par identifiant de compte. | test | A B E O |
| Essai de code (2FA, code par e-mail) réservé par une écriture atomique AVANT la vérification : sinon des essais en parallèle dépassent le plafond. | test : 12 essais simultanés, plafond respecté | A B E O |
| Code court envoyé par e-mail (6 chiffres) : plafond par compte et par jour en plus des limites par IP, blocage puis alerte. | test | E O |
| Le verrouillage d'un compte après échecs ne doit pas être déclenchable à volonté par un tiers : délai croissant par IP plutôt que blocage dur du compte, ou blocage court et alerte. | relecture | A B E O |
| Une session ouverte sans second facteur (réseau local) n'est jamais acceptée par l'accès Internet : l'état « second facteur vérifié » est porté par la session. | test | O |

## E-mails

| Règle | Contrôle | Types |
|-------|----------|-------|
| CR, LF et caractères de contrôle refusés dans destinataire, objet, nom, `Reply-To` et en-têtes ajoutés ; adresses validées ; en-têtes réservés (From, To, Cc, Bcc, Subject, Content-*…) interdits en ajout. Sinon injection d'en-tête (destinataires ajoutés). | test d'injection | F A B E O |
| File d'envoi : un message est pris par écriture conditionnelle (jamais « supprimer puis réinsérer »), essais plafonnés, échecs gardés et visibles dans le back-office, relance limitée par clic. | test de concurrence | A B E O |
| Recette et version de travail : jamais de vrai e-mail (capteur), capteur prioritaire sur le SMTP réglé. | test : SMTP réel réglé, message capturé | A B E O |

## Données et fichiers

| Règle | Contrôle | Types |
|-------|----------|-------|
| Export CSV : toute cellule qui commence par `=`, `+`, `-`, `@`, tabulation ou retour chariot est préfixée d'une apostrophe (sinon formule exécutée par le tableur). | test de l'export | A B E O |
| Import CSV : taille et nombre de lignes plafonnés. | test | A B E O |
| Import XLSX ou ODS : DTD refusée (XXE, entités en cascade), taille décompressée, nombre de lignes et de colonnes plafonnés avant lecture (bombe zip), format vérifié par le contenu et non par l'extension. | test : fichiers piégés refusés | A B E O |
| Service worker : jamais de page personnelle, d'authentification, de réponse POST, `private` ou `no-store` en cache ; caches, IndexedDB et service worker effacés à la déconnexion (`Clear-Site-Data`). Saisie renvoyée hors ligne : clé d'idempotence, enregistrée une seule fois. | test hors ligne puis déconnexion | E O |
| Sauvegarde : chiffrée et authentifiée (fichier tronqué ou modifié refusé), clé hors du dossier des sauvegardes, vérification régulière, restauration précédée d'une sauvegarde de sécurité, jamais téléchargeable par le web. | test : un octet modifié, restauration refusée | A B E O |
| Fichier servi par plages (vidéo, audio) : chemin résolu sous le dossier prévu (liens symboliques compris), lecture par blocs sans tout charger en mémoire, 416 pour une plage hors du fichier. | test : `..`, lien symbolique, `bytes=-500` | A B E O |
| Réservation d'une ressource limitée (créneau, dernier exemplaire) : contrôle et écriture dans une même transaction verrouillée, plus une contrainte d'unicité en dernier rempart. | test : deux processus simultanés, un seul réussit | A B E |
| Noms de tables et de colonnes venus d'une configuration : vérifiés contre une liste blanche avant d'entrer dans le SQL. | relecture | A B E O |
| Données ou coordonnées d'un client jamais en dur dans le code (JSON-LD, gabarits) : une source unique en configuration. | recherche dans le code | V F A B E O |

## Journaux, clés et secrets

| Règle | Contrôle | Types |
|-------|----------|-------|
| Les journaux de requêtes masquent tout segment de chemin et toute valeur de requête qui ressemble à un jeton, y compris pour une adresse mal tapée (sans route) : un lien contenant un jeton ne doit jamais arriver en clair dans un journal. | test : URL inconnue avec jeton, journal sans jeton | A B E O |
| Clé de chiffrement validée strictement au démarrage et à chaque lecture (`^[0-9a-fA-F]{64}$`) : `Buffer.from(…, "hex")` ignore en silence les caractères invalides. Une seule fonction de lecture de la clé. | test : clé avec un caractère invalide refusée | A B E O |
| Une clé sert à un seul usage : sous-clés dérivées (HKDF) pour le hachage des codes de secours, des e-mails ou les signatures ; ancien format accepté puis réécrit à l'usage. | relecture | A B E O |
| Aucune clé de production, aucun secret, aucune donnée réelle dans le dépôt, la boîte à outils, les tests ou les captures ; vecteurs de test signés par une clé de test. | `check-repo.mjs` | V F A B E O |
| Les secrets d'exploitation (jeton du tunnel, mot de passe initial) ne restent pas dans le .env de l'application ni dans ses sauvegardes : fichier root 600 ou variable du seul processus qui en a besoin. | relecture des scripts d'installation | O |
| CI : le jeton d'écriture du dépôt n'est jamais accessible au code npm du build (job de build en lecture seule avec persist-credentials: false, job de déploiement séparé qui ne lance pas npm) ; actions épinglées par SHA. | relecture du workflow | V F A B E O |
| Mises à jour d'un serveur : sources vérifiées (signature ou somme de contrôle), jamais « latest » téléchargé sans contrôle. | relecture | O |

## Interface

| Règle | Contrôle | Types |
|-------|----------|-------|
| Pas d'`innerHTML` avec une donnée : construction par `createElement`/`textContent` ou `<template>` cloné. | `boite-a-outils/outils/regles.mjs` | V F A B E O |
| Échap dans un menu ou une bulle est arrêté (capture, `stopPropagation`) pour ne pas fermer aussi la fenêtre qui les contient ; la fenêtre ignore un Échap déjà traité. | test clavier | A B E O |
| Une bibliothèque front se range hors de tout dossier `vendor/` : les serveurs PHP bloquent souvent `/vendor/` (QR code blanc constaté). | requête HTTP sur le fichier | A B E O |

## Hébergement

| Règle | Contrôle | Types |
|-------|----------|-------|
| Derrière un tunnel qui termine le HTTPS, le serveur doit savoir qu'il est en HTTPS (`fastcgi_param HTTPS on` ou mandataire déclaré), sinon cookies non sécurisés et pas de HSTS. | en-têtes réels du site | V F A B E O |
| Un processus de service ne doit jamais tomber sur une erreur de flux non rattrapée (EPIPE sur l'entrée d'une commande déjà terminée) : écouteurs d'erreur sur stdin, stdout, stderr. | test avec une commande qui sort aussitôt | O |
| Une opération de publication doit pouvoir être relancée après une interruption (pas de copie d'un lien sur lui-même). | test de republication | O |
| La détection « requête venue du tunnel » est restrictive par défaut : en-têtes du tunnel présents ou hôte public => traitée comme Internet, même si le réglage de proxy de confiance est faux ; erreur au démarrage si le tunnel tourne avec un réglage incohérent. | test : en-tête du tunnel sans proxy de confiance, 2FA exigée | O |
| Plusieurs sites sur une machine : le serveur web ne suit pas les liens symboliques dans les dossiers qu'un site contrôle (disable_symlinks), et les secrets d'un site ne sont lisibles que par son compte. | test : lien vers le .env d'un autre site refusé | O |
| Import ou restauration SQL : client lancé en bac à sable (mariadb --sandbox) et sous le compte du site, jamais en root. | relecture | O |
| Scripts shell : jamais de donnée insérée dans une chaîne passée à sh -c ou su -c (argument positionnel) ; remplacement de texte en JS par une fonction (replace(re, () => valeur)) pour que $ ne soit pas interprété. | test avec une apostrophe et un $ | O |
| Outils de développement (Prisma Studio, phpMyAdmin) jamais lancés en service permanent : profil à la demande. | relecture du compose | O |
