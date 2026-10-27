# Bugs et dette dans le code existant

- [ ] `internal/diagnostic/disk/grading.go` : fonctions `TODO` sans `return` → `go vet ./...` / `go build ./...` échouent (le binaire GUI n'en dépend pas)
- [ ] `internal/probe/cpu.go` : nombre de cœurs faux (3 au lieu de 4 sur un i5-6500T)
- [ ] `internal/stress_test.go` et `internal/diagnostic/cpu/stress_test.go` : le suffixe `_test.go` les exclut de la compilation normale → renommer ou supprimer (le stress test est maintenant dans `internal/ui/gui/test_stress.go`)
- [ ] `main.go` à la racine : ancien prototype qui importe `gobox/bin` (inexistant) → à supprimer
- [ ] Fichiers vides d'une ligne : `diagnostic/runner.go`, `network/connectivity.go`, `ram/stress.go`, `disk/benchmark.go`, `disk/models.go`, `export/*.go`, `diagnostic/te.go`
- [ ] `internal/ui/display/*` (affichage terminal) : à garder pour `cmd/goboxcli` ou à supprimer
- [ ] `configs/config.yml` vide : y mettre les réglages (durée stress, seuils, dossier des fiches) ou supprimer
- [ ] README à écrire (installation, `make deps`, `make dev`)
