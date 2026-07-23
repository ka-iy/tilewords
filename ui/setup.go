// Package ui is documented in doc.go.
package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"tilewords/dictionary"
	"tilewords/engine"
)

// dictPlayableWords is the number of playable words (2–15 letters, A–Z only, after
// dedup) in each dictionary's embedded GADDAG asset, shown in the setup menu. These are
// fixed properties of the committed .gob assets; recompute for a rebuilt list with:
//
//	tr a-z A-Z < wordlists/<name>.txt | grep -xE '[A-Z]{2,15}' | sort -u | wc -l
var dictPlayableWords = map[dictionary.DictName]int{
	dictionary.DictENABLE:  169266,
	dictionary.DictWordnik: 194152,
	dictionary.DictAtebits: 270652,
}

// dictShortName returns the compact name of a dictionary. It is kept short because it
// forms the radio-button label, and the widest radio label sets the control's minimum
// width — a long label would stop the desktop window from shrinking to a narrow width.
func dictShortName(name dictionary.DictName) string {
	switch name {
	case dictionary.DictENABLE:
		return "ENABLE"
	case dictionary.DictWordnik:
		return "Wordnik"
	case dictionary.DictAtebits:
		return "atebits"
	default:
		return string(name)
	}
}

// dictDisplayName returns the setup-menu radio label for a dictionary: its short name
// plus playable-word count. The longer prose description is shown separately, and
// word-wrapped, by dictDescription so a long line never truncates on a phone nor forces
// a minimum window width on desktop.
func dictDisplayName(name dictionary.DictName) string {
	if n, ok := dictPlayableWords[name]; ok {
		return fmt.Sprintf("%s — %s words", dictShortName(name), groupThousands(n))
	}
	return dictShortName(name)
}

// dictDescription returns a one-line prose description of a dictionary, shown under the
// dictionary radio in a word-wrapping label. Empty for an unknown dictionary.
func dictDescription(name dictionary.DictName) string {
	switch name {
	case dictionary.DictENABLE:
		return "Public-domain, uncensored friendly-word-game word list."
	case dictionary.DictWordnik:
		return "Crowd-sourced open dictionary."
	case dictionary.DictAtebits:
		return "Public-domain list with similarities to a certain porcine official UK/Euro English list."
	default:
		return ""
	}
}

// groupThousands formats a non-negative integer with commas grouping every three digits
// (e.g. 270652 → "270,652") for readable word counts in the menu.
func groupThousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	out := s[:lead]
	for i := lead; i < len(s); i += 3 {
		out += "," + s[i:i+3]
	}
	return out
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

	// Prose description of the selected dictionary, shown under the radio and word-wrapped
	// so a long line neither truncates on a phone nor forces a minimum window width.
	dictDesc := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	dictDesc.Wrapping = fyne.TextWrapWord

	dictRadio := widget.NewRadioGroup(labels, func(l string) {
		selectedDict = byLabel[l]
		dictDesc.SetText(dictDescription(selectedDict))
	})
	if len(labels) > 0 {
		dictRadio.SetSelected(labels[0])
		selectedDict = avail[0]
		dictDesc.SetText(dictDescription(selectedDict))
	}

	// Game mode: board layout + tile economy. Classic is the standard board and 100-tile
	// set; Interesting is the alternative pinwheel board and 110-tile economy. Each mode is
	// its own single-option radio so a small per-mode "Info" button can sit inline; the two
	// radios are kept mutually exclusive by hand (Required blocks deselect-by-tapping).
	selectedMode := engine.ClassicMode
	var classicRadio, interestingRadio *widget.RadioGroup
	classicRadio = widget.NewRadioGroup([]string{"Classic Mode"}, func(s string) {
		if s == "" {
			return // ignore the deselection callback raised when the other radio is chosen
		}
		selectedMode = engine.ClassicMode
		interestingRadio.SetSelected("")
	})
	interestingRadio = widget.NewRadioGroup([]string{"Interesting Mode"}, func(s string) {
		if s == "" {
			return
		}
		selectedMode = engine.InterestingMode
		classicRadio.SetSelected("")
	})
	classicRadio.Required, interestingRadio.Required = true, true
	classicRadio.SetSelected("Classic Mode")

	classicInfo := newBevelButton("Info", func() { a.showModeInfo(engine.ClassicMode) })
	interestingInfo := newBevelButton("Info", func() { a.showModeInfo(engine.InterestingMode) })

	modeSection := container.NewVBox(
		container.NewHBox(classicRadio, classicInfo),
		container.NewHBox(interestingRadio, interestingInfo),
	)

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

	var startBtn, backBtn *touchButton

	startBtn = newTouchButton("Start Game", func() {
		if len(avail) == 0 {
			status.SetText("No dictionaries are available in this build.")
			return
		}
		startBtn.Disable()
		backBtn.Disable()
		status.SetText("Loading dictionary…")
		a.startNewGame(selectedDict, level, selectedMode, notationCheck.Checked, func(msg string) {
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

	backBtn = newTouchButton("Back", func() {
		a.showMainMenu("")
	})

	form := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Dictionary", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		dictRadio,
		dictDesc,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Game Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		modeSection,
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
