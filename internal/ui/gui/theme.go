package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Couleurs de la maquette
var (
	colInk       = rgb(0x1B, 0x1D, 0x22) // texte principal
	colMuted     = rgb(0x5B, 0x60, 0x69) // libellés
	colLine      = rgb(0xD5, 0xD8, 0xDE) // bordures
	colLineSoft  = rgb(0xEC, 0xEE, 0xF1) // séparateurs
	colPanel     = rgb(0xF3, 0xF4, 0xF6) // fonds gris clair
	colTeal      = rgb(0x0F, 0x76, 0x6E) // validé / principal
	colTealSoft  = rgb(0xE6, 0xF4, 0xF1)
	colAmber     = rgb(0xB4, 0x53, 0x09) // avec réserve
	colAmberSoft = rgb(0xFD, 0xF3, 0xE1)
	colAmberInk  = rgb(0x92, 0x40, 0x0E)
	colRed       = rgb(0xB9, 0x1C, 0x1C) // échec
	colRedSoft   = rgb(0xFE, 0xE2, 0xE2)
	colRedInk    = rgb(0x99, 0x1B, 0x1B)
	colGraySoft  = rgb(0xEE, 0xF0, 0xF3) // à faire / absent
	colGrayInk   = rgb(0x4B, 0x50, 0x59)
	colBlueSoft  = rgb(0xDB, 0xEA, 0xFE) // en cours
	colBlueInk   = rgb(0x1E, 0x40, 0xAF)

	colPeripheral = rgb(0x6D, 0x4F, 0xC2)
	colStorage    = rgb(0xA1, 0x62, 0x07)
	colSystem     = rgb(0x1D, 0x4E, 0xD8)
)

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 0xff} }

// atelierTheme thème clair de la maquette, basé sur le thème Fyne par défaut
type atelierTheme struct{}

func (atelierTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return color.White
	case theme.ColorNameForeground:
		return colInk
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return colTeal
	case theme.ColorNameInputBackground:
		return color.White
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return colLine
	case theme.ColorNameButton:
		return colPanel
	case theme.ColorNamePlaceHolder:
		return colMuted
	}
	return theme.DefaultTheme().Color(n, theme.VariantLight)
}

func (atelierTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (atelierTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (atelierTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameText {
		return 13
	}
	return theme.DefaultTheme().Size(n)
}
