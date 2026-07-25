// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// TestTileWordsTheme_Colors confirms the high-contrast overrides for each variant: the
// dark variant brightens foreground/success/error; the light variant darkens the status
// success/error (and leaves the foreground to the default light theme).
func TestTileWordsTheme_Colors(t *testing.T) {
	th := tileWordsTheme{}

	// Dark variant: bright overrides.
	dark := []struct {
		name fyne.ThemeColorName
		want color.Color
	}{
		{theme.ColorNameForeground, color.NRGBA{R: 235, G: 235, B: 235, A: 255}},
		{theme.ColorNameSuccess, color.NRGBA{R: 120, G: 230, B: 130, A: 255}},
		{theme.ColorNameError, color.NRGBA{R: 255, G: 120, B: 120, A: 255}},
	}
	for _, c := range dark {
		if got := th.Color(c.name, theme.VariantDark); got != c.want {
			t.Errorf("dark %s = %v, want %v", c.name, got, c.want)
		}
	}

	// Light variant: status colours darkened for contrast on white.
	if got := th.Color(theme.ColorNameSuccess, theme.VariantLight); got != color.Color(colorGreenLight) {
		t.Errorf("light success = %v, want %v", got, colorGreenLight)
	}
	if got := th.Color(theme.ColorNameError, theme.VariantLight); got != color.Color(colorRedLight) {
		t.Errorf("light error = %v, want %v", got, colorRedLight)
	}
	// Light foreground is left to the default theme (already dark on white).
	if got, want := th.Color(theme.ColorNameForeground, theme.VariantLight),
		theme.DefaultTheme().Color(theme.ColorNameForeground, theme.VariantLight); got != want {
		t.Errorf("light foreground = %v, want default %v", got, want)
	}

	if got := th.Size(theme.SizeNameSubHeadingText); got != 20 {
		t.Errorf("SubHeadingText size = %v, want 20 (enlarged status line)", got)
	}
}
