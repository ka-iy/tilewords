// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// promptBlank shows a modal dialog letting the player assign a letter to the
// blank tile just staged from rack slot fromRackIdx. Choosing a letter assigns
// it; dismissing the dialog without choosing recalls the staged tile.
func (gs *gameScreen) promptBlank(fromRackIdx int) {
	gs.blankOpen = true
	gs.refresh() // disable the action buttons while the modal is up

	var d dialog.Dialog

	grid := container.NewGridWithColumns(7)
	for i := 0; i < 26; i++ {
		letter := byte('A' + i)
		btn := widget.NewButton(string(letter), func() {
			gs.assignBlank(fromRackIdx, letter)
			d.Hide()
		})
		grid.Add(btn)
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Choose a letter for your blank tile:", fyne.TextAlignCenter, fyne.TextStyle{}),
		grid,
	)

	d = dialog.NewCustom("Blank tile", "Cancel", content, gs.app.win)
	d.SetOnClosed(func() {
		gs.blankOpen = false
		// If the tile is still an unassigned blank, the dialog was cancelled —
		// recall the staged tile.
		if gs.stagedBlankUnassigned(fromRackIdx) {
			gs.recallStagedTile(fromRackIdx)
		}
		gs.refresh()
	})
	d.Show()
}
