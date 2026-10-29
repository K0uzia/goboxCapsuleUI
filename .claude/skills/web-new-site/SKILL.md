---
name: web-new-site
description: Cadre un nouveau site ou une refonte, quel que soit le domaine (vitrine, e-commerce, blog, média, application, portail associatif, événementiel, documentation) - questions de cadrage, choix d'architecture, rôles à mobiliser, exigences propres au type de site, plan de livraison. À utiliser au démarrage d'un projet web ou quand une demande touche plusieurs métiers à la fois.
---

# Démarrer un site

Source : AGENTS.md (arbre de décision), équipe et skill `coordinator` de CapsuleOS, généralisés.

## 1. Cadrage : skill `interview`

Rien ne se code avant un mode d'emploi validé (`.claude/rules/projet.md`). Les questions, l'ordre et la condition d'arrêt sont dans le skill `interview`.

## 2. Architecture : rule `web-architecture`

Le socle (statique PHP à l'architecture Astro, Astro sur demande, dynamique PHP) se tranche à l'interview. Pour un socle statique PHP, partir du modèle `modeles/php-statique/` (voir son README).

| Besoin | Réponse dans le socle choisi |
|--------|------------------------------|
| Contenu édité par un non-développeur | fichiers Markdown et JSON dans `src/content/` et `src/data/`, rebuild à la publication |
| Formulaire de contact | `mailto:` et téléphone, ou service de formulaire ; traitement sur place seulement en socle dynamique (skill `web-forms-security`) |
| Comptes, paiement, données métier | socle dynamique PHP |
| Interactivité ponctuelle | JavaScript vanilla en amélioration progressive, pas d'application client entière |

## 3. Exigences par type de site

| Type | En plus du socle (rules `web-*`) |
|------|-------------------------------|
| Vitrine, artisan, profession libérale | `LocalBusiness`/`ProfessionalService`, NAP identique partout, fiche Google Business, appel à l'action visible sans défilement |
| E-commerce | CGV, prix TTC, rétractation, médiateur ; `Product`/`Offer` ; paiement par prestataire (jamais de carte stockée) ; pages produit indexables, filtres en `noindex` ; perf des listes d'images |
| Blog, média | `Article` (auteur, dates), flux RSS, pagination indexable, images avec crédits, temps de lecture optionnel |
| Application, SaaS, espace client | pages publiques statiques séparées de l'app ; app en `noindex` ; sessions, CSRF, limites de débit ; états vides et erreurs soignés |
| Association, collectivité, service public | déclaration d'accessibilité obligatoire, RGAA, contraste et lisibilité renforcés, documents PDF accessibles ou alternatives HTML |
| Événement | `Event` (date, lieu, statut), page qui reste utile après la date (archive), billetterie externe en `noopener` |
| Documentation | recherche, navigation latérale, ancres stables, `BreadcrumbList`, code copiable |
| Multilingue | une URL par langue, `hreflang` réciproques, pas de traduction automatique non relue |

## 4. Rôles à mobiliser

| Rôle | Responsabilité | Skills |
|------|----------------|--------|
| Chef de projet | périmètre, priorités P0/P1/P2, recette | `web-audit`, `web-release` |
| Designer UX / contenu | parcours, hiérarchie, textes réels, états vides | `web-ui-states` |
| Web designer | tokens, grilles, typo, responsive, mouvement | `web-design-system` |
| Graphiste | logo, icônes, photos, formats | `web-media` |
| Développeur | HTML/CSS/JS, build, données, formulaires | rules `web-code`, `web-forms-security` |
| SEO | balisage, données structurées, indexation | `web-seo-structured-data` |
| Sécurité / exploitation | en-têtes, secrets, déploiement | `web-security-headers`, `web-repo-hygiene`, `web-release` |
| Conformité | légal, RGPD, accessibilité | `web-legal-fr`, `web-a11y-audit` |

Une demande multi-métiers se découpe : classer, charger les skills des rôles concernés, séquencer, livrer la liste des fichiers touchés.

## 5. Ordre de livraison

1. Interview terminée, mode d'emploi validé, données sources créées (même incomplètes, éléments manquants dans « À fournir »).
2. Tokens et composants de base (skill `web-design-system`).
3. Gabarits de pages avec contenu réel, mobile d'abord.
4. SEO, données structurées, pages légales.
5. Formulaires et sécurité.
6. Recette complète (skill `web-audit`), livraison (skill `livraison`), puis mise en production (skill `web-release`).
