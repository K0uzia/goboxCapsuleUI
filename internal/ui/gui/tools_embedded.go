//go:build embedtools

package gui

import _ "embed"

// Archive préparée par packaging/build-tools.sh pendant `make build`
//
//go:embed outils.tar.gz
var embeddedToolsArchive []byte
