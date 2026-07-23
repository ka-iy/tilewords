// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"tilewords/engine"
)

// buildMainMenu returns the main-menu content. errMsg, when non-empty, is shown
// below the buttons (e.g. a forwarded load failure).
func (a *App) buildMainMenu(errMsg string) fyne.CanvasObject {
	title := menuTitleTiles()

	subtitle := widget.NewLabelWithStyle(
		"A two-player almost-free-form cross×word game, quite like That Game We Shall "+
			"Not Name For Fear Of Being Sued By That Company That Sounds Like It Has A Male Sibling",
		fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	subtitle.Wrapping = fyne.TextWrapWord

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

	aboutBtn := widget.NewButton("About", func() {
		a.showAbout()
	})

	// A centred column of equal-width buttons (the VBox stretches its children to
	// its own width; NewCenter shrinks the VBox to its widest child).
	buttons := []fyne.CanvasObject{newBtn, loadBtn, deleteBtn, aboutBtn}

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

// menuTitleTiles renders the app name "TILEWORDS" as a row of game tiles (each letter with
// its Classic-mode point value), so the main-menu title uses the actual tile styling. The
// tiles are kept small enough that all nine fit one row on a phone.
func menuTitleTiles() fyne.CanvasObject {
	const word = "TILEWORDS"
	const sz = 34
	cells := make([]fyne.CanvasObject, 0, len(word))
	for i := 0; i < len(word); i++ {
		bg, letter, points := newTileObjects()
		styleAsTile(bg, letter, points, engine.Tile{Letter: word[i], Points: engine.LetterPoints(word[i])}, false)
		tile := container.New(tileFillLayout{}, bg, letter, points)
		cells = append(cells, container.NewGridWrap(fyne.NewSize(sz, sz), tile))
	}
	return container.NewCenter(container.NewHBox(cells...))
}
