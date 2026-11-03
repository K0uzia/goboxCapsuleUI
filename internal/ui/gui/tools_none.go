//go:build !embedtools

package gui

// Sans `make build` (make dev, tests) : les outils du système
var embeddedToolsArchive []byte
