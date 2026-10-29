package gui

import (
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Thème Kouzia (../kouzia-boite-a-outils/base/tokens.css) : sombre par
// défaut, variante claire. Les couleurs sont nommées par rôle et résolues au
// rafraîchissement, ce qui permet de changer de thème sans reconstruire la
// fenêtre.

// Rôles de couleur propres à gobox (en plus des noms standard de Fyne)
const (
	cBg         = theme.ColorNameBackground
	cText       = theme.ColorNameForeground
	cSurface    = fyne.ThemeColorName("kz-surface")
	cSurface2   = fyne.ThemeColorName("kz-surface-2")
	cLine       = fyne.ThemeColorName("kz-line")
	cLineStrong = fyne.ThemeColorName("kz-line-strong")
	cBorder     = fyne.ThemeColorName("kz-border")
	cMuted      = fyne.ThemeColorName("kz-muted")
	cAccent     = fyne.ThemeColorName("kz-accent")
	cAccentSoft = fyne.ThemeColorName("kz-accent-soft")
	cAccentText = fyne.ThemeColorName("kz-accent-text")
	cSuccess    = fyne.ThemeColorName("kz-success")
	cSuccessBg  = fyne.ThemeColorName("kz-success-soft")
	cWarning    = fyne.ThemeColorName("kz-warning")
	cWarningBg  = fyne.ThemeColorName("kz-warning-soft")
	cDanger     = fyne.ThemeColorName("kz-danger")
	cDangerBg   = fyne.ThemeColorName("kz-danger-soft")
	cInfo       = fyne.ThemeColorName("kz-info")
	cInfoBg     = fyne.ThemeColorName("kz-info-soft")
	cLog        = fyne.ThemeColorName("kz-log")

	// Catégories de tests (séries de la charte)
	cPeripheral   = fyne.ThemeColorName("kz-cat-peripheral")
	cPeripheralBg = fyne.ThemeColorName("kz-cat-peripheral-soft")
	cStorage      = fyne.ThemeColorName("kz-cat-storage")
	cStorageBg    = fyne.ThemeColorName("kz-cat-storage-soft")
	cSystem       = fyne.ThemeColorName("kz-cat-system")
	cSystemBg     = fyne.ThemeColorName("kz-cat-system-soft")
)

func hex(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// soft couleur à opacité réduite, posée sur une surface
func soft(v uint32, alpha float64) color.NRGBA {
	c := hex(v)
	c.A = uint8(alpha * 255)
	return c
}

var darkPalette = map[fyne.ThemeColorName]color.Color{
	cBg:           hex(0x0b1020),
	cText:         hex(0xe8eaf8),
	cSurface:      hex(0x121a2e),
	cSurface2:     hex(0x19233c),
	cLine:         hex(0x232d47),
	cLineStrong:   hex(0x3a4566),
	cBorder:       hex(0x6f7897),
	cMuted:        hex(0x9aa1bf),
	cAccent:       hex(0xfe9a00),
	cAccentSoft:   soft(0xfe9a00, 0.12),
	cAccentText:   hex(0x1a1103),
	cSuccess:      hex(0x5ee0a0),
	cSuccessBg:    soft(0x5ee0a0, 0.12),
	cWarning:      hex(0xffcf6b),
	cWarningBg:    soft(0xffcf6b, 0.12),
	cDanger:       hex(0xff9b8f),
	cDangerBg:     soft(0xff9b8f, 0.12),
	cInfo:         hex(0x8b93ff),
	cInfoBg:       soft(0x8b93ff, 0.12),
	cLog:          hex(0x070b16),
	cPeripheral:   hex(0x8b93ff),
	cPeripheralBg: soft(0x8b93ff, 0.12),
	cStorage:      hex(0x2ec4b6),
	cStorageBg:    soft(0x2ec4b6, 0.12),
	cSystem:       hex(0xf472b6),
	cSystemBg:     soft(0xf472b6, 0.12),
}

var lightPalette = map[fyne.ThemeColorName]color.Color{
	cBg:           hex(0xf7f5f0),
	cText:         hex(0x141a2e),
	cSurface:      hex(0xffffff),
	cSurface2:     hex(0xefece4),
	cLine:         hex(0xe2ddd2),
	cLineStrong:   hex(0xc9c2b3),
	cBorder:       hex(0x767b8f),
	cMuted:        hex(0x4a5270),
	cAccent:       hex(0xb45309),
	cAccentSoft:   soft(0xb45309, 0.08),
	cAccentText:   hex(0xffffff),
	cSuccess:      hex(0x1e7a46),
	cSuccessBg:    soft(0x1e7a46, 0.08),
	cWarning:      hex(0x8a5300),
	cWarningBg:    soft(0x8a5300, 0.08),
	cDanger:       hex(0xb3261e),
	cDangerBg:     soft(0xb3261e, 0.07),
	cInfo:         hex(0x3640b8),
	cInfoBg:       soft(0x3640b8, 0.07),
	cLog:          hex(0xefece4),
	cPeripheral:   hex(0x4048c8),
	cPeripheralBg: soft(0x4048c8, 0.08),
	cStorage:      hex(0x0f766e),
	cStorageBg:    soft(0x0f766e, 0.08),
	cSystem:       hex(0xbe185d),
	cSystemBg:     soft(0xbe185d, 0.08),
}

// Rôles standard de Fyne exprimés avec les rôles Kouzia
var standardRoles = map[fyne.ThemeColorName]fyne.ThemeColorName{
	theme.ColorNamePrimary:             cAccent,
	theme.ColorNameForegroundOnPrimary: cAccentText,
	theme.ColorNameHyperlink:           cAccent,
	theme.ColorNameButton:              cSurface2,
	theme.ColorNameInputBackground:     cBg,
	theme.ColorNameInputBorder:         cBorder,
	theme.ColorNameMenuBackground:      cSurface2,
	theme.ColorNameOverlayBackground:   cSurface,
	theme.ColorNameHeaderBackground:    cSurface2,
	theme.ColorNamePlaceHolder:         cMuted,
	theme.ColorNameSeparator:           cLine,
	theme.ColorNameScrollBar:           cLineStrong,
	theme.ColorNameError:               cDanger,
	theme.ColorNameForegroundOnError:   cBg,
	theme.ColorNameSuccess:             cSuccess,
	theme.ColorNameForegroundOnSuccess: cBg,
	theme.ColorNameWarning:             cWarning,
	theme.ColorNameForegroundOnWarning: cBg,
}

//go:embed fonts/Exo2-Regular.ttf
var exoRegular []byte

//go:embed fonts/Exo2-Bold.ttf
var exoBold []byte

//go:embed fonts/Exo2-Italic.ttf
var exoItalic []byte

//go:embed fonts/ShareTechMono-Regular.ttf
var shareTechMono []byte

var (
	fontRegular = fyne.NewStaticResource("Exo2-Regular.ttf", exoRegular)
	fontBold    = fyne.NewStaticResource("Exo2-Bold.ttf", exoBold)
	fontItalic  = fyne.NewStaticResource("Exo2-Italic.ttf", exoItalic)
	fontMono    = fyne.NewStaticResource("ShareTechMono-Regular.ttf", shareTechMono)
)

// kouziaTheme thème de l'application ; dark choisit la palette
type kouziaTheme struct{ dark bool }

func newTheme(dark bool) fyne.Theme { return &kouziaTheme{dark: dark} }

// isDark thème courant de l'application
func isDark() bool {
	t, ok := fyne.CurrentApp().Settings().Theme().(*kouziaTheme)
	return !ok || t.dark
}

func (t *kouziaTheme) palette() map[fyne.ThemeColorName]color.Color {
	if t.dark {
		return darkPalette
	}
	return lightPalette
}

func (t *kouziaTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	p := t.palette()
	if role, ok := standardRoles[n]; ok {
		n = role
	}
	if c, ok := p[n]; ok {
		return c
	}
	switch n {
	case theme.ColorNameFocus, theme.ColorNameSelection:
		return p[cAccentSoft]
	case theme.ColorNameHover, theme.ColorNamePressed:
		if t.dark {
			return soft(0xffffff, 0.06)
		}
		return soft(0x141a2e, 0.05)
	case theme.ColorNameDisabled: // sert aussi aux textes secondaires (Label LowImportance) : 4,5:1
		return p[cMuted]
	case theme.ColorNameDisabledButton:
		return p[cLine]
	}
	variant := theme.VariantLight
	if t.dark {
		variant = theme.VariantDark
	}
	return theme.DefaultTheme().Color(n, variant)
}

func (t *kouziaTheme) Font(s fyne.TextStyle) fyne.Resource {
	switch {
	case s.Monospace:
		return fontMono
	case s.Symbol:
		return theme.DefaultTheme().Font(s)
	case s.Bold:
		return fontBold
	case s.Italic:
		return fontItalic
	}
	return fontRegular
}

func (t *kouziaTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (t *kouziaTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 17
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 7
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius:
		return 7
	case theme.SizeNameSelectionRadius:
		return 4
	case theme.SizeNameInlineIcon: // icônes des boutons, au niveau d'un texte de 14 px
		return 16
	case theme.SizeNameInputBorder:
		return 1
	}
	return theme.DefaultTheme().Size(n)
}
