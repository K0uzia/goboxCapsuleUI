// Lance GoBox (WebAssembly déjà téléchargé par la page) et prévient la page
// dès que la fenêtre de l'application est dessinée
'use strict';

// Le pilote web de Fyne traite la perte de focus de la fenêtre sans rendre la
// main au navigateur : avec un champ de saisie actif, un clic sur le bandeau ou
// un changement d'onglet figeait la page. L'application garde donc le focus.
for (const type of ['focus', 'blur']) {
  window.addEventListener(type, (e) => {
    if (e.target === window) e.stopImmediatePropagation();
  }, true);
}

// Le pilote web pose ses écouteurs de molette et de toucher sans { passive: false } :
// Chrome les traite alors en passif et chaque preventDefault écrit une erreur dans
// la console. On les déclare non passifs, ce qu'ils sont.
const addListener = EventTarget.prototype.addEventListener;
EventTarget.prototype.addEventListener = function (type, fn, opts) {
  if ((type === 'wheel' || type === 'touchstart' || type === 'touchmove') && (opts === undefined || typeof opts === 'boolean')) {
    return addListener.call(this, type, fn, { capture: opts === true, passive: false });
  }
  return addListener.call(this, type, fn, opts);
};

// La page envoie l'application par message (seule voie possible en file://,
// où page et cadre n'ont pas la même origine)
const host = window.parent;
const target = location.protocol === 'file:' ? '*' : location.origin;
const tell = (msg) => host.postMessage(msg, target);

async function run(bytes) {
  try {
    const seen = new MutationObserver(() => {
      if (document.querySelector('canvas')) {
        seen.disconnect();
        requestAnimationFrame(() => requestAnimationFrame(() => tell({ type: 'gobox-ready' })));
      }
    });
    seen.observe(document.documentElement, { childList: true, subtree: true });
    const go = new Go();
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
    await go.run(instance);
  } catch (err) {
    tell({ type: 'gobox-failed', message: String(err && err.message ? err.message : err) });
  }
}

if (host === window) {
  // cadre ouvert seul : retour à la page de la démo
  location.replace('./');
} else {
  window.addEventListener('message', function onWasm(e) {
    if (e.source !== host || !e.data || e.data.type !== 'gobox-wasm') return;
    window.removeEventListener('message', onWasm);
    run(e.data.bytes);
  });
  tell({ type: 'gobox-hello' });
}
