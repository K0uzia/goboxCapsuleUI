<?php
// SPDX-License-Identifier: GPL-3.0-or-later
// Routeur du serveur de dev : php -S localhost:8000 -t public bin/dev.php
// Rend les pages à chaque requête (pas de build nécessaire pour voir une modification).
declare(strict_types=1);

require __DIR__ . '/../src/lib/routes.php';

$chemin = rawurldecode(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH) ?: '/');

// Fichier de public/ (PDF, images…) : servi directement par php -S
if ($chemin !== '/' && is_file(ROOT . '/public' . $chemin)) {
    return false;
}
if ($chemin === '/assets/site.css') {
    header('Content-Type: text/css; charset=utf-8');
    echo css_bundle();
    return true;
}
// Anciennes adresses Wix → 301, comme en production
foreach (data('redirections') as $r) {
    if ($r['de'] === $chemin) {
        header('Location: ' . $r['vers'], true, 301);
        return true;
    }
}
// Forme canonique avec barre finale
if (!str_ends_with($chemin, '/') && !str_contains(basename($chemin), '.')) {
    header('Location: ' . $chemin . '/', true, 301);
    return true;
}
$routes = routes();
if (isset($routes[$chemin])) {
    echo render($routes[$chemin][0], $routes[$chemin][1], $chemin);
} else {
    http_response_code(404);
    echo render(ROOT . '/src/pages/404.php', null, $chemin);
}
return true;
