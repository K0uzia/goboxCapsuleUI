package gui

import (
	"embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Icônes Font Awesome Free 6.7.2 (CC BY 4.0, icons/LICENSE.txt), en SVG
// intégrés au binaire : ni police d'icônes ni réseau.
//
//go:embed icons/*.svg
var iconFiles embed.FS

var iconCache = map[string]fyne.Resource{}

// icon ressource SVG d'une icône (nom du fichier sans .svg)
func icon(name string) fyne.Resource {
	if r, ok := iconCache[name]; ok {
		return r
	}
	b, err := iconFiles.ReadFile("icons/" + name + ".svg")
	if err != nil {
		fyne.LogError("icône absente : "+name, err)
		return theme.QuestionIcon()
	}
	r := fyne.NewStaticResource(name+".svg", b)
	iconCache[name] = r
	return r
}

// coloredIcon icône à la couleur d'un rôle du thème (suit le changement de thème)
func coloredIcon(name string, role fyne.ThemeColorName) fyne.Resource {
	return theme.NewColoredResource(icon(name), role)
}

// iconOf icône colorée de taille fixe, centrée : environ 1,2 fois la hauteur
// du texte voisin (11 px de texte : 13 px d'icône)
func iconOf(name string, role fyne.ThemeColorName, size float32) fyne.CanvasObject {
	return sized(widget.NewIcon(coloredIcon(name, role)), size)
}

// sized impose une taille carrée à une icône
func sized(o fyne.CanvasObject, size float32) fyne.CanvasObject {
	return container.NewCenter(container.NewGridWrap(fyne.NewSize(size, size), o))
}

// buttonIcon icône pour un bouton : sa couleur suit celle du bouton
func buttonIcon(name string) fyne.Resource {
	return theme.NewThemedResource(icon(name))
}
