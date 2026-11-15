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

(async () => {
  const host = window.parent;
  try {
    const bytes = host.goboxWasm;
    if (!bytes) {
      // cadre ouvert seul : retour à la page de la démo
      location.replace('./');
      return;
    }
    host.goboxWasm = null;
    const seen = new MutationObserver(() => {
      if (document.querySelector('canvas')) {
        seen.disconnect();
        requestAnimationFrame(() => requestAnimationFrame(() => host.goboxReady()));
      }
    });
    seen.observe(document.documentElement, { childList: true, subtree: true });
    const go = new Go();
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
    await go.run(instance);
  } catch (err) {
    if (host.goboxFailed) host.goboxFailed(err);
  }
})();
