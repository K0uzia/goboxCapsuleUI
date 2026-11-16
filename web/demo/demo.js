// Démo GoBox : télécharge l'application (WebAssembly compressé), affiche la
// progression, puis l'ouvre dans un cadre sous le bandeau. Rien n'est envoyé.
// Ouverte depuis le disque (file://), la page ne peut rien télécharger : elle
// lit alors la copie de l'application en script (gobox.wasm.js).
'use strict';

// Valeurs remplies par scripts/demo-web.py
const WASM = { url: 'gobox.wasm.gz?v=__VERSION__', script: 'gobox.wasm.js?v=__VERSION__', gz: __GZ_SIZE__, raw: __RAW_SIZE__ };
const FROM_DISK = location.protocol === 'file:';

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
  if (FROM_DISK) {
    const link = document.createElement('a');
    link.href = 'https://demo-gobox.kouzia.com/';
    link.textContent = 'demo-gobox.kouzia.com';
    box.append(" Elle s'ouvre aussi en ligne sur ", link, '.');
  }
  console.warn(err);
}

// file:// : fetch est refusé, une balise <script> classique reste permise
function loadScript() {
  $('status').textContent = 'Lecture de GoBox sur le disque…';
  return new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = WASM.script;
    s.onload = () => {
      s.remove();
      const b64 = window.goboxWasmGz;
      delete window.goboxWasmGz;
      if (typeof b64 !== 'string') { reject(new Error("copie de l'application illisible")); return; }
      let bytes;
      if (typeof Uint8Array.fromBase64 === 'function') {
        bytes = Uint8Array.fromBase64(b64);
      } else {
        const bin = atob(b64);
        bytes = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
      }
      setProgress(bytes.length, bytes.length);
      resolve(bytes);
    };
    s.onerror = () => reject(new Error('fichier gobox.wasm.js introuvable à côté de la page'));
    document.head.append(s);
  });
}

async function fetchWasm() {
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
  const bytes = new Uint8Array(done);
  let at = 0;
  for (const c of chunks) { bytes.set(c, at); at += c.length; }
  return bytes;
}

async function download() {
  let bytes = FROM_DISK ? await loadScript() : await fetchWasm();
  if (bytes[0] === 0x1f && bytes[1] === 0x8b) {
    $('status').textContent = 'Décompression…';
    const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('gzip'));
    bytes = new Uint8Array(await new Response(stream).arrayBuffer());
  }
  return bytes;
}

async function start() {
  $('loading').hidden = false;
  try {
    if (!('WebAssembly' in window) || !('DecompressionStream' in window)) {
      throw new Error('WebAssembly ou DecompressionStream absent');
    }
    wasm = await download();
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

// L'application suit la taille de la page : deux colonnes sur ordinateur,
// onglets sur tablette et téléphone, à l'échelle 1. Seul un écran plus petit
// que APP_MIN_WIDTH × APP_MIN_HEIGHT la réduit (les touchers restent justes, le
// cadre fait la conversion).
const APP_MIN_WIDTH = 320;
const APP_MIN_HEIGHT = 420;
function fit() {
  const frame = document.querySelector('.app-frame');
  if (!frame) return;
  const stage = $('stage');
  const w = stage.clientWidth;
  const h = stage.clientHeight;
  const scale = Math.min(1, w / APP_MIN_WIDTH, h / APP_MIN_HEIGHT);
  if (scale === 1) {
    frame.style.width = '';
    frame.style.height = '';
    frame.style.transform = '';
    return;
  }
  frame.style.width = `${Math.floor(w / scale)}px`;
  frame.style.height = `${Math.floor(h / scale)}px`;
  frame.style.transform = `scale(${scale})`;
}

// Échanges avec le cadre par messages : en file:// la page et le cadre n'ont
// pas la même origine et ne peuvent pas lire les variables l'un de l'autre
let wasm = null;
window.addEventListener('message', (e) => {
  const frame = document.querySelector('.app-frame');
  if (!frame || e.source !== frame.contentWindow || !e.data) return;
  const target = FROM_DISK ? '*' : location.origin;
  if (e.data.type === 'gobox-hello' && wasm) {
    const bytes = wasm;
    wasm = null;
    frame.contentWindow.postMessage({ type: 'gobox-wasm', bytes }, target, [bytes.buffer]);
  } else if (e.data.type === 'gobox-ready') {
    ready();
  } else if (e.data.type === 'gobox-failed') {
    fail(e.data.message);
  }
});

// quand la fenêtre de GoBox est dessinée
function ready() {
  $('intro').classList.add('done');
  setTimeout(() => { $('intro').hidden = true; }, 350);
  const frame = document.querySelector('.app-frame');
  if (frame) {
    frame.focus();
    frame.contentWindow.focus();
  }
}

$('reset').addEventListener('click', () => {
  try { localStorage.clear(); sessionStorage.clear(); } catch (e) { /* stockage indisponible */ }
  location.reload();
});

start();
