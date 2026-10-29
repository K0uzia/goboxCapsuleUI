# Architecture des sites

Le socle se choisit à l'interview, jamais par défaut. Trois socles possibles :

| Socle | Quand | Production |
|-------|-------|------------|
| **Statique PHP, architecture Astro** (recommandé pour une vitrine, à valider à l'interview) | contenu sans compte ni paiement | HTML de `dist/` seul, aucun PHP côté serveur |
| **Astro (l'outil)** | seulement sur demande explicite | HTML de `dist/` |
| **Dynamique PHP** | e-commerce, comptes, back-office, formulaires traités sur place | PHP + base de données sur le serveur |

## Commun aux trois

- Rangement Astro : `src/pages/` (une page = un fichier = une URL), `src/layouts/`, `src/components/`, `src/content/<collection>/*.md`, `src/data/*.json`, `src/styles/`, `public/` (copié tel quel). Une source de données unique alimente HTML, méta, JSON-LD et `llms.txt`.
- Commandes par `make` : `help`, `info`, `deps`, `dev`, `build`, `preview`, `lint`, `check`, `repo`, `verify`, `clean`. `make verify` = tout ce qui doit être vert avant livraison.
- URL avec barre finale, minuscules ASCII, mots séparés par `-`.

## Statique PHP

- PHP ≥ 8.2 comme moteur de gabarits, sans framework ni Composer, `declare(strict_types=1)`, `e()` sur tout affichage.
- Dev : `php -S` + `bin/dev.php`, pages rendues à chaque requête, redirections et 404 actives. Build : `bin/build.php` → `dist/` (CSS au nom haché, `sitemap.xml`, `robots.txt`, `llms.txt`, `404.html`). Prévisualisation : `bin/preview.php` sert `dist/` comme l'hébergeur.
- **Le build échoue au moindre avertissement PHP** (gestionnaire d'erreurs qui lève une exception). Pas d'opérateur `@` pour masquer une erreur.
- Page dynamique `[slug].php` : renvoie `['chemins' => fn() => [...], 'page' => function (string $slug) {...}]`. Une page ne déclare pas de fonction nommée.
- Modèle prêt à copier : `skills/web-new-site/modeles/php-statique/`.

## Dynamique PHP

- Même rangement des vues et du contenu. Code serveur dans `src/` hors du dossier public, seul `public/index.php` exposé.
- Requêtes préparées uniquement, sessions sécurisées, CSRF sur tout POST, rôles déclarés par route, secrets dans `.env` non versionné. Skills `web-forms-security`, `web-security-headers`.
- Paiement par prestataire, jamais de carte stockée. Tests automatiques pour chaque règle métier et chaque droit (403).

## Astro

- Seulement si l'humain le demande. `package-lock.json` versionné, `npm ci`, `npm audit --omit=dev` avant production. Les cibles `make` appellent `npm`.
