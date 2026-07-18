// Package ui is documented in doc.go.
package ui

import (
	"log"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"squabble/dictionary"
	"squabble/engine"
)

// windowTitle is the application/window title.
const windowTitle = "Squabble"

// App owns the single Fyne application and window and routes between screens.
// Every screen is rendered by calling win.SetContent with the screen's content.
type App struct {
	fapp fyne.App
	win  fyne.Window
	sm   *SaveManager

	// redraw re-renders the current screen. It is invoked when the theme changes so the
	// screens' canvas.Text elements (title, labels, rack cue) — which, unlike widgets, do
	// not re-query the theme on their own — pick up the new variant's colours. On mobile
	// the system light/dark variant often settles only after the first screen is built, so
	// this is what makes the initial screen adopt the right colours.
	redraw func()
}

// Run constructs the application, shows the main menu and runs the Fyne event
// loop. It blocks until the window is closed and returns any setup error.
func Run() error {
	// NewWithID (rather than New) sets the app's unique ID in code, so the Preferences and
	// storage APIs have a stable ID regardless of build mode or whether FyneApp.toml is
	// present on disk at runtime (a plain `go build`/`go run` binary reads that file from
	// the filesystem and does not embed it).
	fapp := app.NewWithID("fyi.squabble.game")

	// Set the app icon from a resource bundled into the binary (ui/bundled_icon.go), so the
	// window/taskbar icon is available regardless of the working directory or build mode.
	// Relying on Fyne's dev-mode FyneApp.toml lookup is not enough: it loads the icon via a
	// path resolved from the current directory and is skipped entirely in release builds.
	fapp.SetIcon(resourceIconPng)

	// Respect the system light/dark theme, but brighten the dark variant's text and
	// enlarge the status line (see squabbleTheme).
	fapp.Settings().SetTheme(squabbleTheme{})

	// NewSaveManager("") defaults the save directory to os.UserConfigDir(), but on
	// Android/iOS neither $HOME nor $XDG_CONFIG_HOME is set, so that lookup fails and
	// would abort startup before any window appears. On mobile, use the Fyne app's
	// per-app storage root instead — a writable, platform-appropriate location.
	configRoot := ""
	if fyne.CurrentDevice().IsMobile() {
		configRoot = fapp.Storage().RootURI().Path()
	}

	sm, err := NewSaveManager(configRoot)
	if err != nil {
		return err
	}

	a := &App{
		fapp: fapp,
		sm:   sm,
	}
	a.win = a.fapp.NewWindow(windowTitle)
	a.win.Resize(fyne.NewSize(960, 760))
	a.win.SetMaster()

	// Re-render the current screen when settings (notably the light/dark theme variant)
	// change, so canvas.Text colours track the theme. See App.redraw.
	a.fapp.Settings().AddListener(func(fyne.Settings) {
		fyne.Do(func() {
			if a.redraw != nil {
				a.redraw()
			}
		})
	})

	a.showMainMenu("")

	// ShowAndRun blocks on the main goroutine until the window is closed.
	a.win.ShowAndRun()
	return nil
}

// quit terminates the application (used by the main-menu Quit button).
func (a *App) quit() {
	a.fapp.Quit()
}

// ---------- Screen transitions ----------

// showMainMenu installs the main-menu screen. errMsg, when non-empty, is shown
// to the player (e.g. a forwarded load failure).
func (a *App) showMainMenu(errMsg string) {
	a.redraw = func() { a.win.SetContent(a.buildMainMenu(errMsg)) }
	a.redraw()
}

// showSetup installs the new-game setup screen.
func (a *App) showSetup() {
	a.redraw = func() { a.win.SetContent(a.buildSetup()) }
	a.redraw()
}

// showGame installs the gameplay screen for an initialised state and dictionary. The
// move-history format is taken from state.ScrabbleNotation. If it is the AI's turn (e.g.
// the AI won the opening draw, or a saved game was the AI's move), the AI turn is started
// immediately.
func (a *App) showGame(state *engine.GameState, dict *dictionary.Dictionary) {
	gs := newGameScreen(a, state, dict)
	content := gs.build()
	// Overlay the drag ghost above the content in a no-layout layer so it can float to
	// any pixel and follow the cursor during a drag.
	a.win.SetContent(container.NewStack(content, container.NewWithoutLayout(gs.ghost)))
	// A theme change only needs to recolour this screen's canvas.Text (rack cue) and
	// status line; gs.refresh() does exactly that without disturbing the game or the AI.
	a.redraw = gs.refresh
	if state.CurrentTurn == engine.AITurn {
		gs.startAITurn()
	}
}

// ---------- Asynchronous dictionary loading ----------
//
// The GADDAG dictionary can be large to decode, so it is always loaded on a
// background goroutine. The result is delivered back on the UI goroutine with
// fyne.Do. onErr reports a sanitised, user-facing error string on the UI
// goroutine; the caller uses it to display the message on the current screen.

// startNewGame loads dictName asynchronously and, on success, creates a fresh
// game and shows the game screen. scrabbleNotation selects the move-history format.
func (a *App) startNewGame(dictName dictionary.DictName, level int, scrabbleNotation bool, onErr func(string)) {
	go func() {
		dict, err := dictionary.Load(dictName)
		fyne.Do(func() {
			if err != nil {
				onErr(sanitiseError(err))
				return
			}
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			state := engine.New(dict.Name(), level, rng)
			state.ScrabbleNotation = scrabbleNotation
			logOpeningDraw(state)
			a.showGame(state, dict)
		})
	}()
}

// loadSavedGame reads the save file (synchronously — it is small) and then loads
// the matching dictionary asynchronously before showing the game screen.
func (a *App) loadSavedGame(onErr func(string)) {
	state, err := a.sm.Load()
	if err != nil {
		onErr(sanitiseError(err))
		return
	}
	dictName := state.DictName
	go func() {
		dict, err := dictionary.Load(dictName)
		fyne.Do(func() {
			if err != nil {
				onErr(sanitiseError(err))
				return
			}
			// state.ScrabbleNotation was persisted with the save, so the resumed game keeps
			// the same move-history format the player chose.
			a.showGame(state, dict)
		})
	}()
}

// logOpeningDraw writes the opening-draw result — the letter each player drew and
// who plays first — to the standard log, so the decided start order is recorded.
func logOpeningDraw(state *engine.GameState) {
	od := state.OpeningDraw
	if od == nil {
		return
	}
	firstMsg := "you go first"
	if od.First == engine.AITurn {
		firstMsg = "AI goes first"
	}
	log.Printf("opening draw: you drew %s, AI drew %s — %s",
		drawnLetterName(od.HumanLetter), drawnLetterName(od.AILetter), firstMsg)
}

// drawnLetterName renders a drawn tile's letter for display, mapping the blank
// sentinel (0) to "(blank)".
func drawnLetterName(letter byte) string {
	if letter == 0 {
		return "(blank)"
	}
	return string(letter)
}
