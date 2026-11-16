#!/usr/bin/env python3
"""Démo en ligne de gobox (demo-gobox.kouzia.com) : compile l'application pour
le navigateur (GOOS=js GOARCH=wasm, étiquette demo), la compresse et assemble
le site statique dans build/demo-web (ou le dossier donné en argument).

Usage : python scripts/demo-web.py [dossier]
"""

import base64
import gzip
import hashlib
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "web" / "demo"


def main() -> int:
    out = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else ROOT / "build" / "demo-web"
    if out.exists():
        shutil.rmtree(out)
    out.mkdir(parents=True)

    # migrated_fynedo : l'application passe déjà tout par fyne.Do ; sans les
    # contrôles de fil d'exécution de Fyne (runtime.Stack à chaque Refresh), une
    # image coûte nettement moins en WebAssembly
    raw = out / "gobox.wasm"
    env = dict(os.environ, GOOS="js", GOARCH="wasm")
    # GOBOX_DEMO_DEBUG=1 : version de recette (goboxFind pour les tests
    # automatiques du navigateur), jamais publiée
    tags = "demo migrated_fynedo" + (" demodebug" if os.environ.get("GOBOX_DEMO_DEBUG") else "")
    subprocess.run(["go", "build", "-tags", tags, "-trimpath", "-ldflags=-s -w", "-o", str(raw), "./cmd/gobox-demo"],
                   cwd=ROOT, env=env, check=True)
    data = raw.read_bytes()
    raw.unlink()
    packed = gzip.compress(data, compresslevel=9, mtime=0)
    (out / "gobox.wasm.gz").write_bytes(packed)
    version = hashlib.sha256(packed).hexdigest()[:10]
    # Copie en script pour la page ouverte depuis le disque (file://) : fetch y
    # est refusé, une balise <script> classique non. Chargée seulement dans ce cas.
    b64 = base64.b64encode(packed).decode("ascii")
    (out / "gobox.wasm.js").write_bytes(b'self.goboxWasmGz="' + b64.encode("ascii") + b'";\n')

    for f in SRC.iterdir():
        if f.is_file():
            shutil.copy2(f, out / f.name)
    shutil.copy2(ROOT / "packaging" / "gobox.svg", out / "favicon.svg")

    goroot = subprocess.run(["go", "env", "GOROOT"], cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    runtime = (Path(goroot) / "lib" / "wasm" / "wasm_exec.js").read_text(encoding="utf-8")
    # commentaires de travail du runtime Go qui renvoient à son suivi de bogues
    runtime = "\n".join(l for l in runtime.splitlines() if not (l.strip().startswith("//") and "://" in l)) + "\n"
    (out / "wasm_exec.js").write_text(runtime, encoding="utf-8", newline="\n")

    # Empreinte dans chaque adresse de fichier statique (?v=) : le CDN garde ces
    # fichiers une semaine, une nouvelle version doit changer d'adresse. Un
    # fichier n'est empreinté qu'après la réécriture de ses propres liens.
    def stamp(name: str) -> str:
        return hashlib.sha256((out / name).read_bytes()).hexdigest()[:10]

    def rewrite(name: str, pairs: list[tuple[str, str]]) -> None:
        text = (out / name).read_text(encoding="utf-8")
        for old, new in pairs:
            if old not in text:
                raise SystemExit(f"{name} : lien introuvable {old!r}")
            text = text.replace(old, new)
        (out / name).write_text(text, encoding="utf-8", newline="\n")

    v = {n: stamp(n) for n in ("app.css", "demo.css", "app.js", "wasm_exec.js", "favicon.svg", "og-image.png")}
    rewrite("app.html", [
        ('href="favicon.svg"', f'href="favicon.svg?v={v["favicon.svg"]}"'),
        ('href="app.css"', f'href="app.css?v={v["app.css"]}"'),
        ('src="wasm_exec.js"', f'src="wasm_exec.js?v={v["wasm_exec.js"]}"'),
        ('src="app.js"', f'src="app.js?v={v["app.js"]}"'),
    ])
    rewrite("demo.js", [
        ("__VERSION__", version), ("__GZ_SIZE__", str(len(packed))), ("__RAW_SIZE__", str(len(data))),
        ("frame.src = 'app.html';", f"frame.src = 'app.html?v={stamp('app.html')}';"),
    ])
    v["demo.js"] = stamp("demo.js")
    rewrite("index.html", [
        ('href="favicon.svg"', f'href="favicon.svg?v={v["favicon.svg"]}"'),
        ('src="favicon.svg"', f'src="favicon.svg?v={v["favicon.svg"]}"'),
        ('href="demo.css"', f'href="demo.css?v={v["demo.css"]}"'),
        ('src="demo.js"', f'src="demo.js?v={v["demo.js"]}"'),
        ('/og-image.png"', f'/og-image.png?v={v["og-image.png"]}"'),
    ])
    rewrite("404.html", [
        ('href="/favicon.svg"', f'href="/favicon.svg?v={v["favicon.svg"]}"'),
        ('src="/favicon.svg"', f'src="/favicon.svg?v={v["favicon.svg"]}"'),
        ('href="/demo.css"', f'href="/demo.css?v={v["demo.css"]}"'),
    ])

    # contrôles : ni tiret cadratin ou demi-cadratin, ni lien vers l'hébergeur du code
    bad = []
    for f in out.iterdir():
        if f.name != "gobox.wasm.js" and (f.suffix in {".html", ".js", ".css", ".txt", ".xml", ".svg"} or f.name == ".htaccess"):
            text = f.read_text(encoding="utf-8")
            if "\u2014" in text or "\u2013" in text or re.search("git" + "hub", text, re.I):
                bad.append(f.name)
    if bad:
        print("Contenu refusé dans :", ", ".join(bad), file=sys.stderr)
        return 1

    print(f"{out}")
    print(f"gobox.wasm : {len(data):,} octets, gobox.wasm.gz : {len(packed):,} octets, gobox.wasm.js : {(out / 'gobox.wasm.js').stat().st_size:,} octets (version {version})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
