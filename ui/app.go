// Package ui is documented in doc.go.
package ui

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"tilewords/buildinfo"
	"tilewords/dictionary"
	"tilewords/engine"
)

// windowTitle is the application/window title.
const windowTitle = "TileWords"

// App owns the single Fyne application and window and routes between screens.
// Every screen is rendered by calling win.SetContent with the screen's content.
type App struct {
	fapp fyne.App
	win  fyne.Window
	sm   *SaveManager

	// settings persists the New Game Setup defaults (word list, mode, difficulty, notation)
	// so the setup screen can pre-populate the player's previous choices. See settings.go.
	settings *settingsStore

	// redraw re-renders the current screen. It is invoked when the theme changes so the
	// screens' canvas.Text elements (title, labels, rack cue) — which, unlike widgets, do
	// not re-query the theme on their own — pick up the new variant's colours. On mobile
	// the system light/dark variant often settles only after the first screen is built, so
	// this is what makes the initial screen adopt the right colours.
	redraw func()

	// nav counts screen transitions. An asynchronous load captures it before starting and
	// compares it when it finishes, so a result whose screen the player has already left is
	// dropped instead of replacing whatever they navigated to. Without it, leaving the menu
	// mid-load drops the player into a game they no longer asked for — including one whose
	// save they just deleted — and orphans the previous game screen along with its
	// definitions worker.
	nav int

	// game is the gameplay screen currently installed, or nil. Kept so a new screen can shut
	// the previous one's background worker down instead of leaking it.
	game *gameScreen

	// uiGen counts widget-tree builds, including the rebuilds a theme change triggers. An
	// asynchronous callback that captured widgets from a particular build compares it to know
	// whether those widgets are still on screen; writing to a detached tree shows the player
	// nothing. Unlike nav it counts rebuilds of the same screen, which is exactly the case
	// nav cannot see.
	uiGen int

	// screenMsg is a message for the next build of the main-menu or setup screen to display.
	// It is how an asynchronous result reaches whichever widget tree is current, rather than
	// the one its closure happened to capture.
	screenMsg string
}

// redrawNow rebuilds the current screen, recording that any widgets an in-flight callback
// captured are now detached.
func (a *App) redrawNow() {
	a.uiGen++
	if a.redraw != nil {
		a.redraw()
	}
}

// reportOnCurrentScreen delivers an asynchronous load message. When the widget tree that
// started the load is still installed, onAttached updates it directly, which preserves
// whatever the player had entered. When that tree has been rebuilt — a theme variant settling
// mid-load is the usual cause — the message is handed to the next build instead, so a failure
// is still seen rather than written into detached widgets.
func (a *App) reportOnCurrentScreen(gen int, msg string, onAttached func()) {
	if a.uiGen == gen {
		a.screenMsg = ""
		onAttached()
		return
	}
	a.screenMsg = msg
	a.redrawNow()
}

// takeScreenMsg returns any pending message for a screen being built and clears it, so it is
// shown once rather than reappearing on every later rebuild.
func (a *App) takeScreenMsg() string {
	msg := a.screenMsg
	a.screenMsg = ""
	return msg
}

// Run constructs the application, shows the main menu and runs the Fyne event
// loop. It blocks until the window is closed and returns any setup error.
func Run() error {
	// Log the build metadata (git version, build type, timestamp) at startup so a
	// user-reported log pins down exactly which build produced it. The values are injected
	// at link time (see the Makefile); a build without injection logs buildinfo's defaults.
	log.Printf("%s starting; build info:\n%s", windowTitle,
		strings.Join(buildinfo.BuildInfoAsStringSlice(), "\n"))

	// NewWithID (rather than New) sets the app's unique ID in code, so the Preferences and
	// storage APIs have a stable ID regardless of build mode or whether FyneApp.toml is
	// present on disk at runtime (a plain `go build`/`go run` binary reads that file from
	// the filesystem and does not embed it).
	fapp := app.NewWithID("fyi.tilewords.game")

	// Set the app icon from a resource bundled into the binary (ui/bundled_icon.go), so the
	// window/taskbar icon is available regardless of the working directory or build mode.
	// Relying on Fyne's dev-mode FyneApp.toml lookup is not enough: it loads the icon via a
	// path resolved from the current directory and is skipped entirely in release builds.
	fapp.SetIcon(resourceIconPng)

	// Respect the system light/dark theme, but brighten the dark variant's text and
	// enlarge the status line (see tileWordsTheme).
	fapp.Settings().SetTheme(tileWordsTheme{})

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
		fapp:     fapp,
		sm:       sm,
		settings: newSettingsStore(fapp.Preferences()),
	}
	a.win = a.fapp.NewWindow(windowTitle)
	a.win.Resize(fyne.NewSize(960, 760))
	a.win.SetMaster()

	// Re-render the current screen when settings (notably the light/dark theme variant)
	// change, so canvas.Text colours track the theme. See App.redraw.
	a.fapp.Settings().AddListener(func(fyne.Settings) {
		fyne.Do(func() {
			if a.redraw != nil {
				a.redrawNow()
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
	a.leaveScreen()
	a.screenMsg = errMsg
	// The message is read from screenMsg rather than captured here, so a rebuild shows a
	// message that arrived after this screen was installed.
	a.redraw = func() { a.win.SetContent(a.buildMainMenu(a.takeScreenMsg())) }
	a.redrawNow()
}

// showSetup installs the new-game setup screen.
func (a *App) showSetup() {
	a.leaveScreen()
	a.redraw = func() { a.win.SetContent(a.buildSetup()) }
	a.redrawNow()
}

// leaveScreen records that the installed screen is being replaced, invalidating any
// asynchronous load still in flight for it, and shuts down the outgoing game screen's
// background worker so it does not outlive the screen it belongs to.
func (a *App) leaveScreen() {
	a.nav++
	// A message meant for the screen being left has nowhere to go.
	a.screenMsg = ""
	if a.game != nil {
		a.game.abandoned = true
		a.game.stopDefinitions()
		a.game = nil
	}
}

// screenToken returns the current navigation counter, to be passed to screenIsCurrent when an
// asynchronous result comes back.
func (a *App) screenToken() int { return a.nav }

// screenIsCurrent reports whether the screen that started an asynchronous load is still the
// one installed, i.e. whether its result should still be applied.
func (a *App) screenIsCurrent(token int) bool { return a.nav == token }

// showGame installs the gameplay screen for an initialised state and dictionary. The
// move-history format is taken from state.ScrabbleNotation. If it is the AI's turn (e.g.
// the AI won the opening draw, or a saved game was the AI's move), the AI turn is started
// immediately.
func (a *App) showGame(state *engine.GameState, dict *dictionary.Dictionary) {
	a.leaveScreen()
	gs := newGameScreen(a, state, dict)
	a.game = gs
	content := gs.build()
	// Start the definitions lookup worker once the screen's widgets exist. This is done
	// here rather than in build() so tests that build a screen directly do not spawn the
	// worker or load the (large) definitions asset.
	gs.startDefinitions()
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

// startNewGame loads dictName asynchronously and, on success, creates a fresh game in
// the given mode and shows the game screen. scrabbleNotation selects the move-history
// format.
func (a *App) startNewGame(dictName dictionary.DictName, level int, mode engine.GameMode, scrabbleNotation bool, onErr func(string)) {
	token := a.screenToken()
	go func() {
		dict, err := dictionary.Load(dictName)
		fyne.Do(func() {
			if !a.screenIsCurrent(token) {
				return // the player left the screen that asked for this game
			}
			if err != nil {
				onErr(sanitiseError(err))
				return
			}
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			state := engine.NewWithMode(dict.Name(), level, mode, rng)
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
	token := a.screenToken()
	go func() {
		dict, err := dictionary.Load(dictName)
		fyne.Do(func() {
			if !a.screenIsCurrent(token) {
				return // the player left the menu that asked for this game
			}
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
	log.Printf("opening draw: you drew %s, AI drew %s - %s",
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
