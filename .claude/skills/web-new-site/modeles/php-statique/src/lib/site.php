<?php
// SPDX-License-Identifier: GPL-3.0-or-later
// Fonctions communes aux pages : données, contenus Markdown, gabarits, dates.
// Organisation inspirée d'Astro : src/pages (routes), src/layouts, src/components,
// src/content (Markdown), src/data (JSON), public (copié tel quel).
declare(strict_types=1);

const ROOT = __DIR__ . '/../..';

/** Échappe une valeur pour l'afficher dans le HTML. À utiliser sur TOUT affichage de donnée. */
function e(?string $texte): string
{
    return htmlspecialchars($texte ?? '', ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

/** Lit src/data/<nom>.json (mis en cache pour la durée de la requête ou du build). */
function data(string $nom): array
{
    static $cache = [];
    return $cache[$nom] ??= json_decode(
        file_get_contents(ROOT . "/src/data/$nom.json"), true, 512, JSON_THROW_ON_ERROR
    );
}

/** Adresse absolue d'un chemin du site (canonical, sitemap, Open Graph). */
function url_absolue(string $chemin): string
{
    return rtrim(data('site')['url'], '/') . $chemin;
}

/** Chemin de la page en cours de rendu (ex. « /actions/ »), fixé par render(). */
function chemin_courant(): string
{
    return $GLOBALS['SITE_CHEMIN'] ?? '/';
}

// ---------- Contenus Markdown (src/content/<collection>/*.md) ----------

/** Toutes les entrées d'une collection, triées par nom de fichier. Chaque entrée : en-tête + slug + html. */
function collection(string $nom): array
{
    static $cache = [];
    if (isset($cache[$nom])) {
        return $cache[$nom];
    }
    $entrees = [];
    foreach (glob(ROOT . "/src/content/$nom/*.md") as $fichier) {
        [$entete, $corps] = lire_markdown(file_get_contents($fichier));
        $entete['slug'] = basename($fichier, '.md');
        $entete['html'] = markdown($corps);
        $entrees[] = $entete;
    }
    return $cache[$nom] = $entrees;
}

/** Une entrée par son slug, ou null si elle n'existe pas. */
function entree(string $collection, string $slug): ?array
{
    foreach (collection($collection) as $e) {
        if ($e['slug'] === $slug) {
            return $e;
        }
    }
    return null;
}

/**
 * Sépare l'en-tête (entre deux lignes « --- ») du corps.
 * En-tête volontairement simple : « cle: valeur », listes « [a, b] », null, nombres.
 */
function lire_markdown(string $source): array
{
    if (!preg_match('/^---\R(.*?)\R---\R?(.*)$/s', $source, $m)) {
        return [[], $source];
    }
    $entete = [];
    foreach (preg_split('/\R/', $m[1]) as $ligne) {
        if (!preg_match('/^(\w+):\s*(.*)$/', $ligne, $l)) {
            continue;
        }
        $entete[$l[1]] = valeur_entete(trim($l[2]));
    }
    return [$entete, $m[2]];
}

function valeur_entete(string $v): mixed
{
    if ($v === '' || $v === 'null') {
        return null;
    }
    if (str_starts_with($v, '[') && str_ends_with($v, ']')) {
        $interieur = trim(substr($v, 1, -1));
        return $interieur === '' ? [] : array_map(fn($x) => valeur_entete(trim($x)), explode(',', $interieur));
    }
    if (preg_match('/^"(.*)"$/', $v, $m)) {
        return $m[1];
    }
    return ctype_digit($v) ? (int) $v : $v;
}

/**
 * Markdown minimal : titres « ## », paragraphes, listes « - », **gras**, [lien](url).
 * Le texte est échappé AVANT la mise en forme : aucun HTML brut ne passe.
 */
function markdown(string $texte): string
{
    $html = '';
    foreach (preg_split('/\R{2,}/', trim($texte)) as $bloc) {
        $bloc = trim($bloc);
        if ($bloc === '') {
            continue;
        }
        if (preg_match('/^(#{2,4})\s+(.*)$/', $bloc, $m)) {
            $n = strlen($m[1]);
            $html .= "<h$n>" . en_ligne($m[2]) . "</h$n>\n";
        } elseif (str_starts_with($bloc, '- ')) {
            $items = preg_split('/\R- /', substr($bloc, 2));
            $html .= "<ul>\n" . implode('', array_map(fn($i) => '<li>' . en_ligne($i) . "</li>\n", $items)) . "</ul>\n";
        } else {
            $html .= '<p>' . en_ligne(preg_replace('/\s*\R\s*/', ' ', $bloc)) . "</p>\n";
        }
    }
    return $html;
}

function en_ligne(string $texte): string
{
    $t = e($texte);
    $t = preg_replace('/\*\*(.+?)\*\*/', '<strong>$1</strong>', $t);
    return preg_replace('/\[([^\]]+)\]\(([^)\s]+)\)/', '<a href="$2">$1</a>', $t);
}

// ---------- Gabarits ----------

/** Rend src/layouts/<nom>.php ; le gabarit appelle $contenu() là où va le contenu de la page. */
function layout(string $nom, array $props, callable $contenu): void
{
    require ROOT . "/src/layouts/$nom.php";
}

/** Rend src/components/<nom>.php avec ses propriétés dans $props. */
function component(string $nom, array $props = []): void
{
    require ROOT . "/src/components/$nom.php";
}

// ---------- Dates et agenda ----------

function date_fr(string $iso, bool $avecJour = true): string
{
    static $jours = ['dimanche', 'lundi', 'mardi', 'mercredi', 'jeudi', 'vendredi', 'samedi'];
    static $mois = ['janvier', 'février', 'mars', 'avril', 'mai', 'juin', 'juillet',
        'août', 'septembre', 'octobre', 'novembre', 'décembre'];
    $d = new DateTimeImmutable($iso);
    $texte = ($d->format('j') === '1' ? '1er' : $d->format('j')) . ' ' . $mois[(int) $d->format('n') - 1] . ' ' . $d->format('Y');
    return $avecJour ? $jours[(int) $d->format('w')] . ' ' . $texte : $texte;
}
