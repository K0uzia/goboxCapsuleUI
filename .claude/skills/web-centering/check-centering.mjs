// Contrôle du centrage : marges gauche et droite de chaque bloc déclaré, à plusieurs largeurs (logique de KouziaCRM/scripts/check-desktop.ts).
// Usage : node check-centering.mjs <url de base> <pages.json>
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const require = createRequire(process.cwd() + '/');
let chromium;
try { ({ chromium } = require('playwright-core')); } catch { ({ chromium } = await import('playwright-core')); }

const [base, file] = process.argv.slice(2);
if (!base || !file) {
  console.error('Usage : node check-centering.mjs <url de base> <pages.json>');
  process.exit(2);
}
// pages.json : soit une liste de pages, soit { login?: { path, fill: { sélecteur: valeur }, submit }, pages: [...] }.
const config = JSON.parse(readFileSync(file, 'utf8'));
const pages = Array.isArray(config) ? config : config.pages;
const login = Array.isArray(config) ? null : config.login;
const WIDTHS = [360, 800, 1280, 1920];
const TOLERANCE = 3;

const browser = await chromium.launch();
let total = 0;
let failures = 0;
try {
  for (const width of WIDTHS) {
    const page = await browser.newPage({ viewport: { width, height: 900 } });
    if (login) {
      await page.goto(base + login.path);
      for (const [selector, value] of Object.entries(login.fill)) await page.fill(selector, value);
      await page.click(login.submit);
      await page.waitForLoadState('networkidle');
    }
    for (const p of pages) {
      await page.goto(base + p.path, { waitUntil: 'networkidle' });
      if (p.click) { await page.click(p.click).catch(() => {}); await page.waitForTimeout(300); }
      const scrollX = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
      for (const c of p.check) {
        total++;
        const m = await page.evaluate(({ selector, within }) => {
          const el = document.querySelector(selector);
          if (!el || !el.getClientRects().length) return null;
          const r = el.getBoundingClientRect();
          let left = 0;
          // Largeur utile de la page (corps) : exclut la place réservée à la barre de défilement (scrollbar-gutter).
          let right = document.body.getBoundingClientRect().right;
          if (within) {
            const box = el.closest(within) || document.querySelector(within);
            const b = box.getBoundingClientRect();
            const s = getComputedStyle(box);
            left = b.left + parseFloat(s.paddingLeft) + parseFloat(s.borderLeftWidth);
            right = b.right - parseFloat(s.paddingRight) - parseFloat(s.borderRightWidth);
          }
          return { gapLeft: r.left - left, gapRight: right - r.right };
        }, c);
        const issues = [];
        if (m === null) issues.push('élément absent ou masqué');
        else if (Math.abs(m.gapLeft - m.gapRight) > TOLERANCE) issues.push(`décentré : ${m.gapLeft.toFixed(1)} px à gauche, ${m.gapRight.toFixed(1)} px à droite`);
        if (scrollX > 0) issues.push(`défilement horizontal de ${scrollX} px`);
        if (issues.length) failures++;
        console.log(`${issues.length ? 'KO' : 'OK'}  ${width}px  ${p.path}  ${c.selector}${issues.length ? '  ' + issues.join(' ; ') : ''}`);
      }
    }
    await page.close();
  }
} finally {
  await browser.close();
}
console.log(`\n${total - failures}/${total} blocs centrés (${WIDTHS.join(', ')} px, tolérance ${TOLERANCE} px).`);
if (failures) process.exitCode = 1;
