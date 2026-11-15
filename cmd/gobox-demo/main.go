//go:build demo && js

// Démo en ligne de gobox, compilée pour le navigateur. Le site complet
// (page, WebAssembly compressé) se construit avec : python scripts/demo-web.py
package main

import "gobox/internal/ui/gui"

func main() { gui.RunDemo() }
