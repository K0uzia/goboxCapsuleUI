<?php
// Gabarit commun. Props : titre, description, [noindex].
$site = data('site');
$chemin = chemin_courant();
?>
<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title><?= e($props['titre'] . ' · ' . $site['nom']) ?></title>
  <meta name="description" content="<?= e($props['description']) ?>">
<?php if (!empty($props['noindex'])): ?>
  <meta name="robots" content="noindex">
<?php else: ?>
  <link rel="canonical" href="<?= e(url_absolue($chemin)) ?>">
<?php endif ?>
  <link rel="stylesheet" href="<?= e(css_href()) ?>">
</head>
<body>
  <a class="evitement" href="#contenu">Aller au contenu</a>
  <header><a href="/"><?= e($site['nom']) ?></a></header>
  <main id="contenu">
<?php $contenu(); ?>
  </main>
  <footer><a href="mailto:<?= e($site['contact']['email']) ?>"><?= e($site['contact']['email']) ?></a></footer>
</body>
</html>
