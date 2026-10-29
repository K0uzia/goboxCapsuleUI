<?php
// SPDX-License-Identifier: GPL-3.0-or-later
// Construit le site statique dans dist/ : php bin/build.php
// dist/ est entièrement régénéré : ne jamais y modifier un fichier à la main.
declare(strict_types=1);

require __DIR__ . '/../src/lib/routes.php';

// Le moindre avertissement PHP (clé manquante, fichier absent…) arrête le build.
set_error_handler(function (int $niveau, string $message, string $fichier, int $ligne): never {
    throw new ErrorException($message, 0, $niveau, $fichier, $ligne);
});

$dist = ROOT . '/dist';

function vider(string $dossier): void
{
    if (!is_dir($dossier)) {
        return;
    }
    $iter = new RecursiveIteratorIterator(
        new RecursiveDirectoryIterator($dossier, FilesystemIterator::SKIP_DOTS),
        RecursiveIteratorIterator::CHILD_FIRST
    );
    foreach ($iter as $f) {
        $f->isDir() ? rmdir($f->getPathname()) : unlink($f->getPathname());
    }
    rmdir($dossier);
}

function ecrire(string $fichier, string $contenu): void
{
    is_dir(dirname($fichier)) || mkdir(dirname($fichier), 0775, true);
    file_put_contents($fichier, $contenu);
}

vider($dist);

// 1. public/ copié tel quel
$public = ROOT . '/public';
$iter = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($public, FilesystemIterator::SKIP_DOTS));
foreach ($iter as $f) {
    $cible = $dist . substr($f->getPathname(), strlen($public));
    is_dir(dirname($cible)) || mkdir(dirname($cible), 0775, true);
    copy($f->getPathname(), $cible);
}

// 2. CSS avec empreinte dans le nom (cache 1 an « immutable » ; forme nom.<hash>.css attendue par le .htaccess)
$css = css_bundle();
$GLOBALS['SITE_CSS'] = '/assets/site.' . substr(hash('sha256', $css), 0, 10) . '.css';
ecrire($dist . $GLOBALS['SITE_CSS'], $css);

// 3. Pages
$routes = routes();
foreach ($routes as $chemin => [$fichier, $param]) {
    ecrire($dist . $chemin . 'index.html', render($fichier, $param, $chemin));
}
ecrire("$dist/404.html", render(ROOT . '/src/pages/404.php', null, '/404'));

// 4. Fichiers pour les moteurs et assistants
$site = data('site');
$sitemap = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n";
foreach (array_keys($routes) as $chemin) {
    $sitemap .= '  <url><loc>' . e(url_absolue($chemin)) . "</loc></url>\n";
}
ecrire("$dist/sitemap.xml", $sitemap . "</urlset>\n");
ecrire("$dist/robots.txt", "User-agent: *\nAllow: /\n\nSitemap: " . url_absolue('/sitemap.xml') . "\n");

$llms = "# {$site['nom']}\n\n> {$site['description']}\n\n"
    . "Association loi 1901, {$site['zone']}. Contact : {$site['contact']['email']}, {$site['contact']['telephoneAffiche']}.\n\n## Pages\n\n";
foreach ($routes as $chemin => [$fichier, $param]) {
    preg_match('#<title>(.*?)</title>#s', file_get_contents($dist . $chemin . 'index.html'), $t);
    $llms .= '- [' . html_entity_decode($t[1] ?? $chemin) . '](' . url_absolue($chemin) . ")\n";
}
ecrire("$dist/llms.txt", $llms);

printf("✅ %d pages construites dans dist/ (+ 404, sitemap.xml, robots.txt, llms.txt)\n", count($routes));
