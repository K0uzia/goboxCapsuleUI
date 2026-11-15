#!/usr/bin/env python3
"""Démo en ligne de gobox (demo-gobox.kouzia.com) : compile l'application pour
le navigateur (GOOS=js GOARCH=wasm, étiquette demo), la compresse et assemble
le site statique dans build/demo-web (ou le dossier donné en argument).

Usage : python scripts/demo-web.py [dossier]
"""

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

    raw = out / "gobox.wasm"
    env = dict(os.environ, GOOS="js", GOARCH="wasm")
    subprocess.run(["go", "build", "-tags", "demo", "-trimpath", "-ldflags=-s -w", "-o", str(raw), "./cmd/gobox-demo"],
                   cwd=ROOT, env=env, check=True)
    data = raw.read_bytes()
    raw.unlink()
    packed = gzip.compress(data, compresslevel=9, mtime=0)
    (out / "gobox.wasm.gz").write_bytes(packed)
    version = hashlib.sha256(packed).hexdigest()[:10]

    for f in SRC.iterdir():
        if f.is_file():
            shutil.copy2(f, out / f.name)
    shutil.copy2(ROOT / "packaging" / "gobox.svg", out / "favicon.svg")

    goroot = subprocess.run(["go", "env", "GOROOT"], cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    runtime = (Path(goroot) / "lib" / "wasm" / "wasm_exec.js").read_text(encoding="utf-8")
    # commentaires de travail du runtime Go qui renvoient à son suivi de bogues
    runtime = "\n".join(l for l in runtime.splitlines() if not (l.strip().startswith("//") and "://" in l)) + "\n"
    (out / "wasm_exec.js").write_text(runtime, encoding="utf-8", newline="\n")

    js = (out / "demo.js").read_text(encoding="utf-8")
    js = js.replace("__VERSION__", version).replace("__GZ_SIZE__", str(len(packed))).replace("__RAW_SIZE__", str(len(data)))
    (out / "demo.js").write_text(js, encoding="utf-8", newline="\n")

    # contrôles : ni tiret cadratin ou demi-cadratin, ni lien vers l'hébergeur du code
    bad = []
    for f in out.iterdir():
        if f.suffix in {".html", ".js", ".css", ".txt", ".xml", ".svg"} or f.name == ".htaccess":
            text = f.read_text(encoding="utf-8")
            if "\u2014" in text or "\u2013" in text or re.search("git" + "hub", text, re.I):
                bad.append(f.name)
    if bad:
        print("Contenu refusé dans :", ", ".join(bad), file=sys.stderr)
        return 1

    print(f"{out}")
    print(f"gobox.wasm : {len(data):,} octets, gobox.wasm.gz : {len(packed):,} octets (version {version})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
