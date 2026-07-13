// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// buildMainMenu returns the main-menu content. errMsg, when non-empty, is shown
// below the buttons (e.g. a forwarded load failure).
func (a *App) buildMainMenu(errMsg string) fyne.CanvasObject {
	title := canvas.NewText("SQUABBLE", titleColor())
	title.TextSize = 48
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := canvas.NewText("A word game", bodyTextColor())
	subtitle.Alignment = fyne.TextAlignCenter

	status := widget.NewLabel("")
	status.Alignment = fyne.TextAlignCenter
	status.Wrapping = fyne.TextWrapWord
	if errMsg != "" {
		status.SetText(errMsg)
	}

	var newBtn, loadBtn, deleteBtn *widget.Button

	newBtn = widget.NewButton("New Game", func() {
		a.showSetup()
	})
	newBtn.Importance = widget.HighImportance

	loadBtn = widget.NewButton("Load Game", func() {
		loadBtn.Disable()
		newBtn.Disable()
		status.SetText("Loading…")
		a.loadSavedGame(func(msg string) {
			status.SetText(msg)
			loadBtn.Enable()
			newBtn.Enable()
		})
	})

	// Delete Save removes the saved game after a confirmation. On success the menu is
	// rebuilt, which re-disables both Load and Delete since no save then exists.
	deleteBtn = widget.NewButton("Delete Save", func() {
		dialog.ShowConfirm(
			"Delete saved game",
			"Delete the saved game? This cannot be undone.",
			func(ok bool) {
				if !ok {
					return
				}
				if err := a.sm.Delete(); err != nil {
					a.showMainMenu(sanitiseError(err))
					return
				}
				a.showMainMenu("Saved game deleted.")
			},
			a.win,
		)
	})
	deleteBtn.Importance = widget.DangerImportance

	// Load and Delete act on the save slot, so both are only enabled when one exists.
	if !a.sm.Exists() {
		loadBtn.Disable()
		deleteBtn.Disable()
	}

	// A centred column of equal-width buttons (the VBox stretches its children to
	// its own width; NewCenter shrinks the VBox to its widest child).
	buttons := []fyne.CanvasObject{newBtn, loadBtn, deleteBtn}

	// Quit is only offered on desktop. Android/iOS guidelines forbid an app quitting
	// itself, so Fyne's mobile driver ignores App.Quit — a "Quit" button there does
	// nothing and reads as a hang. Mobile users leave via the OS home/back gesture.
	if !fyne.CurrentDevice().IsMobile() {
		quitBtn := widget.NewButton("Quit", func() {
			a.quit()
		})
		buttons = append(buttons, quitBtn)
	}

	buttonCol := container.NewCenter(container.NewVBox(buttons...))

	content := container.NewVBox(
		layout.NewSpacer(),
		title,
		subtitle,
		layout.NewSpacer(),
		buttonCol,
		status,
		layout.NewSpacer(),
	)
	return container.NewPadded(content)
}
