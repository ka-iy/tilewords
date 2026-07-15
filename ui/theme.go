// Package ui is documented in doc.go.
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// squabbleTheme wraps Fyne's default theme and respects the system light/dark variant
// (per Fyne's guideline that apps should follow the system theme), fixing two issues:
//
//   - In the DARK variant the default foreground/success/error colours render
//     low-contrast and are nearly invisible on the dark background. This is most
//     apparent when Fyne cannot read the OS variant (e.g. the "Failed to load user
//     locales: no current JVM" warning on some Android hosts) and falls back to dark.
//     We brighten those colours for the dark variant only; the light variant, which
//     already reads well, is passed through unchanged.
//   - The status line (sub-heading text) is too small to read on device, so its size
//     is increased for both variants.
type squabbleTheme struct{}

var _ fyne.Theme = squabbleTheme{}

func (squabbleTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantDark {
		switch name {
		case theme.ColorNameForeground:
			return color.NRGBA{R: 235, G: 235, B: 235, A: 255} // bright grey — high contrast
		case theme.ColorNameSuccess:
			return color.NRGBA{R: 120, G: 230, B: 130, A: 255} // bright green status (your points)
		case theme.ColorNameError:
			return color.NRGBA{R: 255, G: 120, B: 120, A: 255} // bright red error
		case theme.ColorNameWarning:
			return color.NRGBA{R: 255, G: 190, B: 90, A: 255} // bright amber (AI points)
		}
	} else {
		// Light variant: darken the status colours so they stay legible on white (the
		// default success/error/warning colours are too light against a white background).
		switch name {
		case theme.ColorNameSuccess:
			return colorGreenLight
		case theme.ColorNameError:
			return colorRedLight
		case theme.ColorNameWarning:
			return colorAmberLight
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}

// isLightTheme reports whether the app is currently rendering with the light variant.
func isLightTheme() bool {
	return fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantLight
}

// The following accessors return an explicit text colour appropriate to the current
// theme variant. The app draws titles, labels, and the rack cue with canvas.Text using
// fixed colours designed for the dark background; on the light theme they must be
// darkened for contrast against white.

func titleColor() color.Color {
	if isLightTheme() {
		return colorTitleLight
	}
	return colorTitle
}

func bodyTextColor() color.Color {
	if isLightTheme() {
		return colorTextLight
	}
	return colorText
}

func turnCueColor() color.Color {
	if isLightTheme() {
		return colorGreenLight
	}
	return colorTurnYou
}

func gameOverColor() color.Color {
	if isLightTheme() {
		return colorRedLight
	}
	return colorGameOver
}

func (squabbleTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (squabbleTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (squabbleTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameSubHeadingText {
		return 20 // enlarge the status line (default is ~16)
	}
	return theme.DefaultTheme().Size(name)
}
