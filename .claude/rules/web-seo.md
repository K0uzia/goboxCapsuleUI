# SEO et découverte (Google et assistants IA)

Le balisage décrit la page affichée, rien de plus. Pas de `meta keywords`.

- `title` unique < 60 caractères, description unique 140 à 160 caractères, `link rel="canonical"` absolu vers une URL qui répond 200 (jamais vers une redirection).
- Une seule URL par contenu : `/index.html`, `www`, `http` et slash final redirigent en 301 vers la forme canonique.
- Open Graph (`og:title`, `og:description`, `og:url`, `og:image` absolue 1200×630 réelle, `og:locale`) et Twitter Card.
- `robots.txt` : autorise le public et les CSS/JS nécessaires au rendu Googlebot, interdit admin, API, brouillons ; ligne `Sitemap:` absolue. `sitemap.xml` : seulement les URL canoniques indexables.
- `noindex` (meta ou `X-Robots-Tag`) pour pages privées, utilitaires, résultats de recherche interne et variantes à paramètres.
- Page d'erreur sur mesure obligatoire (404, et 500 si le serveur le permet) : vrai statut HTTP, gabarit du site (en-tête, pied de page, styles), `noindex`, liens vers l'accueil, les pages principales et le plan du site. Jamais la page par défaut du serveur ni une réponse texte. Multilingue : `hreflang` réciproques + `x-default`.
- Fil d'Ariane visible et `BreadcrumbList` en JSON-LD sur toute page sauf l'accueil.
- `llms.txt` à la racine : résumé Markdown, liens vers les pages réelles, mêmes faits que le site.
- Navigation par les agents IA (catégorie « agentic browsing » de Lighthouse) : chaque contrôle a un nom accessible, un formulaire se comprend sans le visuel (`label`, `name` unique, `autocomplete`), aucune action seulement en JavaScript quand un `a`, `button` ou `form` suffit, aucune injection tardive qui décale la page. Attributs WebMCP (`toolname` et `tooldescription`, toujours ensemble) : seulement par décision écrite, la norme est expérimentale.
- JSON-LD généré depuis la même source que le HTML visible. Un `@id` par nœud, un nœud par intention. Pas de `FAQPage` si le texte n'est pas affiché à l'identique. Aucun `Review`/`AggregateRating` sans avis publiés.
- Procédure : skill `web-seo-structured-data`.
