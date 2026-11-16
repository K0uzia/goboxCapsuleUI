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

// ---- Écran tactile ------------------------------------------------------------
// Le pilote web de Fyne ne lit que la souris et la molette : un toucher bref
// devient déjà un clic (événements de souris du navigateur). Ici, glisser le
// doigt fait défiler (molette, avec élan), un appui long fait un clic droit et
// un toucher à deux doigts un clic molette.
(function touch() {
  const SLOP = 8;         // déplacement sous lequel un toucher reste un toucher
  const LONG = 550;       // appui long (ms)
  let start = null;       // { x, y, t } du doigt posé
  let last = null;        // dernière position
  let moved = false;
  let swallow = false;    // geste traité ici : pas de clic du navigateur ensuite
  let longTimer = 0;
  let twoFinger = null;
  let speed = { x: 0, y: 0 };
  let glide = 0;

  const mouse = (type, x, y, button = 0) => {
    document.dispatchEvent(new MouseEvent(type, { clientX: x, clientY: y, button, bubbles: true, cancelable: true }));
  };
  // Le pilote compte 0,2 point par unité de molette, et 2 points par unité
  // (accélération) à partir de 5 unités : chaque déplacement du doigt est
  // converti pour que le contenu suive le doigt
  const units = (d) => {
    if (Math.abs(d) >= 10) return [d / 2];
    const total = d * 5;
    const n = Math.max(1, Math.ceil(Math.abs(total) / 4.9));
    return new Array(n).fill(total / n);
  };
  const wheel = (dx, dy, x, y) => {
    const send = (deltaX, deltaY) => document.dispatchEvent(new WheelEvent('wheel', { deltaX: -deltaX, deltaY: -deltaY, clientX: x, clientY: y, bubbles: true, cancelable: true }));
    if (dy) for (const u of units(dy)) send(0, u);
    if (dx) for (const u of units(dx)) send(u, 0);
  };
  const stopGlide = () => { if (glide) cancelAnimationFrame(glide); glide = 0; };

  document.addEventListener('touchstart', (e) => {
    stopGlide();
    clearTimeout(longTimer);
    if (e.touches.length === 2) {
      const [a, b] = e.touches;
      twoFinger = { x: (a.clientX + b.clientX) / 2, y: (a.clientY + b.clientY) / 2, t: Date.now() };
      start = null;
      return;
    }
    if (e.touches.length !== 1) return;
    const p = e.touches[0];
    start = { x: p.clientX, y: p.clientY, t: Date.now() };
    last = { x: p.clientX, y: p.clientY, t: start.t };
    moved = false;
    swallow = false;
    speed = { x: 0, y: 0 };
    // le pilote défile ce qui est sous la souris : la souris va sous le doigt
    mouse('mousemove', p.clientX, p.clientY);
    longTimer = setTimeout(() => {
      if (!start || moved) return;
      swallow = true;
      mouse('mousedown', start.x, start.y, 2);
      mouse('mouseup', start.x, start.y, 2);
    }, LONG);
  }, { passive: false });

  document.addEventListener('touchmove', (e) => {
    if (!start || e.touches.length !== 1) return;
    const p = e.touches[0];
    if (!moved && Math.hypot(p.clientX - start.x, p.clientY - start.y) < SLOP) return;
    if (!moved) {
      moved = true;
      swallow = true;
      clearTimeout(longTimer);
    }
    e.preventDefault();
    const now = Date.now();
    const dx = p.clientX - last.x;
    const dy = p.clientY - last.y;
    const dt = Math.max(1, now - last.t);
    speed = { x: 0.8 * (dx / dt) + 0.2 * speed.x, y: 0.8 * (dy / dt) + 0.2 * speed.y };
    last = { x: p.clientX, y: p.clientY, t: now };
    wheel(dx, dy, p.clientX, p.clientY);
  }, { passive: false });

  document.addEventListener('touchend', (e) => {
    clearTimeout(longTimer);
    if (twoFinger && e.touches.length === 0) {
      if (Date.now() - twoFinger.t < 400) {
        mouse('mousemove', twoFinger.x, twoFinger.y);
        mouse('mousedown', twoFinger.x, twoFinger.y, 1);
        mouse('mouseup', twoFinger.x, twoFinger.y, 1);
      }
      twoFinger = null;
      e.preventDefault();
      return;
    }
    if (!start) return;
    const at = last;
    start = null;
    if (swallow) e.preventDefault();
    if (!moved || Date.now() - at.t > 80) return;
    // élan : la vitesse du doigt décroît image par image
    let vx = speed.x * 16;
    let vy = speed.y * 16;
    const step = () => {
      vx *= 0.94;
      vy *= 0.94;
      if (Math.abs(vx) < 0.5 && Math.abs(vy) < 0.5) { glide = 0; return; }
      wheel(vx, vy, at.x, at.y);
      glide = requestAnimationFrame(step);
    };
    glide = requestAnimationFrame(step);
  }, { passive: false });

  document.addEventListener('touchcancel', () => {
    clearTimeout(longTimer);
    start = null;
    twoFinger = null;
  });
})();

// ---- Clavier du téléphone -------------------------------------------------------
// Fyne ouvre le clavier virtuel en donnant le focus au champ caché #dummyEntry,
// pour tout widget qui prend le focus : on ne l'ouvre que pour un champ de
// saisie (goboxWantsKeyboard, fourni par l'application). Les claviers Android
// envoient souvent des touches « Unidentified » : le texte saisi est alors
// relu dans le champ et transmis lettre par lettre.
(function virtualKeyboard() {
  const field = document.getElementById('dummyEntry');
  if (!field) return;
  const focus = HTMLElement.prototype.focus;
  // décision hors de l'appel du pilote : interroger l'application pendant
  // qu'elle appelle la page la bloquerait
  field.focus = function () {
    setTimeout(() => {
      if (typeof window.goboxWantsKeyboard === 'function' && !window.goboxWantsKeyboard()) {
        if (document.activeElement === field) field.blur();
        return;
      }
      focus.call(field, { preventScroll: true });
    }, 0);
  };
  let composing = false;
  field.addEventListener('keydown', (e) => {
    composing = e.isTrusted && (e.key === 'Unidentified' || e.keyCode === 229);
  }, true);
  // le pilote lit keyCode, que le constructeur de KeyboardEvent ne règle pas
  const key = (k, code, keyCode = 0) => {
    for (const type of ['keydown', 'keyup']) {
      const ev = new KeyboardEvent(type, { key: k, code, bubbles: true, cancelable: true });
      Object.defineProperty(ev, 'keyCode', { get: () => keyCode });
      document.dispatchEvent(ev);
    }
  };
  field.addEventListener('input', (e) => {
    if (composing) {
      if (e.inputType === 'deleteContentBackward') {
        key('Backspace', 'Backspace', 8);
      } else if (typeof e.data === 'string') {
        for (const ch of e.data) key(ch, '');
      }
    }
    field.value = '';
  });
})();
