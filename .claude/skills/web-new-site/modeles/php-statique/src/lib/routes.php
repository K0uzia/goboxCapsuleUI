<?php
// SPDX-License-Identifier: GPL-3.0-or-later
// Routage par fichiers, comme Astro : src/pages/a/b.php → /a/b/, src/pages/a/index.php → /a/.
// Page dynamique : nom entre crochets (src/pages/actions/[slug].php). Elle renvoie
// ['chemins' => fn() => [slugs…], 'page' => function (string $slug) { … }].
declare(strict_types=1);

require_once __DIR__ . '/site.php';

/** Toutes les URL du site → [fichier de page, paramètre ou null]. La page 404 n'en fait pas partie. */
function routes(): array
{
    $dossier = ROOT . '/src/pages';
    $routes = [];
    $fichiers = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($dossier, FilesystemIterator::SKIP_DOTS));
    foreach ($fichiers as $f) {
        $relatif = substr($f->getPathname(), strlen($dossier), -4); // « /actions/[slug] »
        if ($f->getExtension() !== 'php' || $relatif === '/404') {
            continue;
        }
        $relatif = preg_replace('#/index$#', '', $relatif);
        if (preg_match('#^(.*)/\[\w+\]$#', $relatif, $m)) {
            $dynamique = require $f->getPathname();
            foreach (($dynamique['chemins'])() as $slug) {
                $routes["$m[1]/$slug/"] = [$f->getPathname(), $slug];
            }
        } else {
            $routes[$relatif . '/'] = [$f->getPathname(), null];
        }
    }
    ksort($routes);
    return $routes;
}

/** Rend une page et renvoie son HTML. */
function render(string $fichier, ?string $param, string $chemin): string
{
    $GLOBALS['SITE_CHEMIN'] = $chemin;
    ob_start();
    $page = require $fichier;
    if (is_array($page)) {
        ($page['page'])($param);
    }
    return ob_get_clean();
}

/** Feuilles de style concaténées dans l'ordre : jetons, base, composants, pages. */
function css_bundle(): string
{
    $css = '';
    foreach (['tokens', 'base', 'components', 'pages'] as $nom) {
        $css .= "/* $nom.css */\n" . file_get_contents(ROOT . "/src/styles/$nom.css") . "\n";
    }
    return $css;
}

/** Adresse de la feuille de style : fixe en dev, nom haché au build (cache long). */
function css_href(): string
{
    return $GLOBALS['SITE_CSS'] ?? '/assets/site.css';
}
