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

	// Load the player's saved defaults (or the built-in defaults) so every control below
	// opens on the previously-chosen value; see FR-15 / settings.go.
	gs := a.defaultsFor(avail)

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

	dictRadio := newTouchRadio(labels, func(l string) {
		selectedDict = byLabel[l]
		dictDesc.SetText(dictDescription(selectedDict))
	})
	if len(labels) > 0 {
		// gs.Dict is guaranteed available (sanitised on load), so its display label is one of
		// the radio labels; selecting it fires the callback above, setting selectedDict and
		// the description.
		dictRadio.SetSelected(dictDisplayName(gs.Dict))
	}

	// Game mode: board layout + tile economy. Classic is the standard board and 100-tile
	// set; Interesting is the alternative pinwheel board and 110-tile economy. A single
	// two-option radio gives natural mutual exclusion; its two rows are laid out each beside
	// a small per-mode "Info" button.
	const classicOpt, interestingOpt = "Classic Mode", "Interesting Mode"
	selectedMode := gs.Mode
	modeRadio := newTouchRadio([]string{classicOpt, interestingOpt}, func(s string) {
		if s == interestingOpt {
			selectedMode = engine.InterestingMode
		} else {
			selectedMode = engine.ClassicMode
		}
	})
	modeLabel := classicOpt
	if gs.Mode == engine.InterestingMode {
		modeLabel = interestingOpt
	}
	modeRadio.SetSelected(modeLabel)

	classicInfo := newBevelButton("Info", func() { a.showModeInfo(engine.ClassicMode) })
	interestingInfo := newBevelButton("Info", func() { a.showModeInfo(engine.InterestingMode) })

	modeSection := container.NewVBox(
		container.NewHBox(modeRadio.buttons[0], classicInfo),
		container.NewHBox(modeRadio.buttons[1], interestingInfo),
	)

	// Difficulty 1–10 via a slider with a live value label.
	level := gs.Difficulty
	levelLabel := widget.NewLabelWithStyle(fmt.Sprintf("Difficulty: %d  (1 = easy, 10 = hard)", level), fyne.TextAlignCenter, fyne.TextStyle{})
	levelSlider := widget.NewSlider(1, 10)
	levelSlider.Step = 1
	levelSlider.SetValue(float64(level))
	levelSlider.OnChanged = func(v float64) {
		level = int(v)
		levelLabel.SetText(fmt.Sprintf("Difficulty: %d  (1 = easy, 10 = hard)", level))
	}

	// Move-history format: plain word list by default, Scrabble coordinate notation when
	// checked (e.g. "8D UNMIX +28").
	notationCheck := newTouchCheck("Show move history in Scrabble notation", nil)
	notationCheck.Checked = gs.Notation

	// When checked at Start Game, the current selections are persisted as the player's
	// defaults (FR-15). It opens checked every time and is not itself persisted.
	saveDefaultsCheck := newTouchCheck("Save these as my defaults", nil)
	saveDefaultsCheck.Checked = true

	var startBtn, backBtn *touchButton

	startBtn = newTouchButton("Start Game", func() {
		if len(avail) == 0 {
			status.SetText("No dictionaries are available in this build.")
			return
		}
		if saveDefaultsCheck.Checked && a.settings != nil {
			a.settings.save(GameSettings{
				Dict:       selectedDict,
				Mode:       selectedMode,
				Difficulty: level,
				Notation:   notationCheck.Checked,
			})
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
		dictRadio.list(),
		dictDesc,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Game Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		modeSection,
		widget.NewSeparator(),
		levelLabel,
		levelSlider,
		widget.NewSeparator(),
		notationCheck,
		saveDefaultsCheck,
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), backBtn, startBtn),
		status,
	)
	return container.NewPadded(container.NewVScroll(form))
}
