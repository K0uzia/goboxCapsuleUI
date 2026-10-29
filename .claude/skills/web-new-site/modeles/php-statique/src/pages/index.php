<?php
layout('Base', [
    'titre' => 'Accueil',
    'description' => data('site')['description'],
], function () { ?>
    <h1><?= e(data('site')['nom']) ?></h1>
<?php });
