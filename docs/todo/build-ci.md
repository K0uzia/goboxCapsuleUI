# Build, CI et distribution

## Fait

- [x] `make deps` : outils système (apt), Go 1.25.2, modules
- [x] `make dev` : `go run` sans fichier dans `build/`, lancé en sudo avec accès à l'écran
- [x] `make build`, `make test`, `make vet`, `make fmt`, `make clean`, `make update`
- [x] Fyne compilé en **X11 seul** par défaut (`GO_TAGS=x11`) : pas besoin des en-têtes Wayland sur Mint

## À faire

- [ ] **Workflow GitHub Actions** (dépôt `K0uzia/goboxCapsuleUI`) : `go vet` + `go test` + build à chaque push / PR
- [ ] **Build multi-OS** demandé :
  - Linux amd64 / arm64 (runner Ubuntu, paquets `libgl1-mesa-dev xorg-dev`)
  - Windows et macOS : Fyne compile, **mais les sondes (`internal/probe`) ne lisent que `/sys` et `/proc`** → il faut des sondes par OS (fichiers `_windows.go` / `_darwin.go`, WMI / IOKit) ou désactiver ces fonctions hors Linux
  - outil possible : `fyne-cross` (Docker), ou un runner natif par OS
- [ ] Artefacts de release : binaire Linux, `.deb` (nfpm / goreleaser), AppImage ?
- [ ] Lancement avec **pkexec** + règle PolicyKit au lieu de `sudo` (une seule demande, comme SecMet)
- [ ] `.desktop` + icône ; remplacer l'entrée LACAPSULE dans le menu
- [ ] Version du binaire (`-ldflags -X main.version=$(git describe)`) affichée dans l'interface et sur la fiche
- [ ] Taille du binaire : 32 Mo actuellement (Fyne) — vérifier avec `-s -w` + `upx` éventuel
