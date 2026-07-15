// Package ui is documented in doc.go.
package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"squabble/dictionary"
)

// dictDisplayName returns a friendly label for a dictionary name.
func dictDisplayName(name dictionary.DictName) string {
	switch name {
	case dictionary.DictPIGPODS:
		return "PIGPODS (IYKYK)"
	case dictionary.DictTWIRL06:
		return "TWIRL06 (North American)"
	case dictionary.DictENABLE:
		return "ENABLE (public domain)"
	case dictionary.DictWordnik:
		return "Wordnik (crowd-sourced)"
	default:
		return string(name)
	}
}

// availableDicts returns the dictionaries whose GADDAG assets are embedded in the
// binary, preserving the order of dictionary.AllDictNames.
func availableDicts() []dictionary.DictName {
	var out []dictionary.DictName
	for _, name := range dictionary.AllDictNames {
		if dictionary.Available(name) {
			out = append(out, name)
		}
	}
	return out
}

// buildSetup returns the new-game setup screen: dictionary choice, difficulty
// slider, and Start/Back actions.
func (a *App) buildSetup() fyne.CanvasObject {
	title := canvas.NewText("New Game Setup", titleColor())
	title.TextSize = 28
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	status := widget.NewLabel("")
	status.Alignment = fyne.TextAlignCenter
	status.Wrapping = fyne.TextWrapWord

	avail := availableDicts()

	// Map display label → dictionary name for the selection control.
	labels := make([]string, len(avail))
	byLabel := make(map[string]dictionary.DictName, len(avail))
	for i, name := range avail {
		l := dictDisplayName(name)
		labels[i] = l
		byLabel[l] = name
	}

	var selectedDict dictionary.DictName
	dictRadio := widget.NewRadioGroup(labels, func(l string) {
		selectedDict = byLabel[l]
	})
	if len(labels) > 0 {
		dictRadio.SetSelected(labels[0])
		selectedDict = avail[0]
	}

	// Difficulty 1–10 via a slider with a live value label.
	level := 5
	levelLabel := widget.NewLabelWithStyle("Difficulty: 5  (1 = easy, 10 = hard)", fyne.TextAlignCenter, fyne.TextStyle{})
	levelSlider := widget.NewSlider(1, 10)
	levelSlider.Step = 1
	levelSlider.SetValue(float64(level))
	levelSlider.OnChanged = func(v float64) {
		level = int(v)
		levelLabel.SetText(fmt.Sprintf("Difficulty: %d  (1 = easy, 10 = hard)", level))
	}

	// Move-history format: plain word list by default, Scrabble coordinate notation when
	// checked (e.g. "8D UNMIX +28").
	notationCheck := widget.NewCheck("Show move history in Scrabble notation", nil)

	var startBtn, backBtn *widget.Button

	startBtn = widget.NewButton("Start Game", func() {
		if len(avail) == 0 {
			status.SetText("No dictionaries are available in this build.")
			return
		}
		startBtn.Disable()
		backBtn.Disable()
		status.SetText("Loading dictionary…")
		a.startNewGame(selectedDict, level, notationCheck.Checked, func(msg string) {
			status.SetText(msg)
			startBtn.Enable()
			backBtn.Enable()
		})
	})
	startBtn.Importance = widget.HighImportance
	if len(avail) == 0 {
		startBtn.Disable()
		status.SetText("No dictionaries are available in this build.")
	}

	backBtn = widget.NewButton("Back", func() {
		a.showMainMenu("")
	})

	form := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Dictionary", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		dictRadio,
		widget.NewSeparator(),
		levelLabel,
		levelSlider,
		widget.NewSeparator(),
		notationCheck,
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), backBtn, startBtn),
		status,
	)
	return container.NewPadded(container.NewVScroll(form))
}
