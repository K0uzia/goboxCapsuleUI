// Démo GoBox : télécharge l'application (WebAssembly compressé), affiche la
// progression, puis l'ouvre dans un cadre sous le bandeau. Rien n'est envoyé.
'use strict';

// Valeurs remplies par scripts/demo-web.py
const WASM = { url: 'gobox.wasm.gz?v=__VERSION__', gz: __GZ_SIZE__, raw: __RAW_SIZE__ };

const $ = (id) => document.getElementById(id);
const mo = (n) => (n / 1e6).toLocaleString('fr-FR', { minimumFractionDigits: 1, maximumFractionDigits: 1 });

function setProgress(done, total) {
  const pct = Math.min(100, Math.floor((done / total) * 100));
  $('bar-fill').style.transform = `scaleX(${pct / 100})`;
  $('bar').setAttribute('aria-valuenow', String(pct));
  $('status').textContent = `Chargement de GoBox : ${pct} % (${mo(done)} sur ${mo(total)} Mo)`;
}

function fail(err) {
  $('loading').hidden = true;
  const box = $('error');
  box.hidden = false;
  box.textContent = "La démo n'a pas pu démarrer dans ce navigateur. Essayez avec une version récente de Firefox, Chrome, Edge ou Safari. ";
  const detail = document.createElement('span');
  detail.textContent = `(${err && err.message ? err.message : err})`;
  box.append(detail);
  console.warn(err);
}

async function download() {
  const res = await fetch(WASM.url);
  if (!res.ok) throw new Error(`téléchargement refusé (${res.status})`);
  const reader = res.body.getReader();
  const chunks = [];
  let done = 0;
  for (;;) {
    const { done: end, value } = await reader.read();
    if (end) break;
    chunks.push(value);
    done += value.length;
    // le serveur a pu décompresser lui-même : la taille attendue est alors la taille brute
    setProgress(done, done > WASM.gz ? WASM.raw : WASM.gz);
  }
  let bytes = new Uint8Array(done);
  let at = 0;
  for (const c of chunks) { bytes.set(c, at); at += c.length; }
  if (bytes[0] === 0x1f && bytes[1] === 0x8b) {
    $('status').textContent = 'Décompression…';
    const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('gzip'));
    bytes = new Uint8Array(await new Response(stream).arrayBuffer());
  }
  return bytes;
}

async function start() {
  $('phone').hidden = true;
  $('loading').hidden = false;
  try {
    if (!('WebAssembly' in window) || !('DecompressionStream' in window)) {
      throw new Error('WebAssembly ou DecompressionStream absent');
    }
    window.goboxWasm = await download();
    $('status').textContent = "Démarrage de l'application…";
    const frame = document.createElement('iframe');
    frame.className = 'app-frame';
    frame.title = 'Application GoBox (démo)';
    frame.src = 'app.html';
    $('stage').append(frame);
    fit();
    window.addEventListener('resize', fit);
  } catch (err) {
    fail(err);
  }
}

// Écran étroit : l'application garde sa largeur d'ordinateur, réduite pour
// tenir dans la page (les clics restent justes, le cadre fait la conversion)
const APP_MIN_WIDTH = 1280;
function fit() {
  const frame = document.querySelector('.app-frame');
  if (!frame) return;
  const stage = $('stage');
  const w = stage.clientWidth;
  const h = stage.clientHeight;
  if (w >= APP_MIN_WIDTH) {
    frame.style.width = '';
    frame.style.height = '';
    frame.style.transform = '';
    return;
  }
  const scale = w / APP_MIN_WIDTH;
  frame.style.width = `${APP_MIN_WIDTH}px`;
  frame.style.height = `${Math.round(h / scale)}px`;
  frame.style.transform = `scale(${scale})`;
}

// appelé par app.html quand la fenêtre de GoBox est dessinée
window.goboxReady = () => {
  $('intro').classList.add('done');
  setTimeout(() => { $('intro').hidden = true; }, 350);
  const frame = document.querySelector('.app-frame');
  if (frame) {
    frame.focus();
    frame.contentWindow.focus();
  }
};
window.goboxFailed = fail;

$('reset').addEventListener('click', () => {
  try { localStorage.clear(); sessionStorage.clear(); } catch (e) { /* stockage indisponible */ }
  location.reload();
});

$('launch').addEventListener('click', start);

if (window.matchMedia('(max-width: 799px)').matches) {
  $('phone').hidden = false;
} else {
  start();
}
