<?php
// SPDX-License-Identifier: GPL-3.0-or-later
// Routeur de prévisualisation du build : php -S localhost:8001 -t dist bin/preview.php
// Sert dist/ comme un hébergeur statique (index.html par dossier, 404.html sinon).
declare(strict_types=1);

$dist = __DIR__ . '/../dist';
$chemin = rawurldecode(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH) ?: '/');
if (is_file($dist . $chemin) || is_file($dist . rtrim($chemin, '/') . '/index.html')) {
    return false;
}
http_response_code(404);
readfile($dist . '/404.html');
return true;
