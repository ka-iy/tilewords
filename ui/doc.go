// Package ui implements the TileWords user interface using the Fyne toolkit.
//
// Fyne is a retained-mode, event-driven GUI toolkit. Unlike the previous
// Ebitengine implementation there is no per-frame game loop: the UI reacts to
// tap/click and resize events dispatched by Fyne, which also means window
// dragging and resizing never block the application.
//
// Fyne builds for every desktop platform (Linux, macOS, Windows) and for the
// mobile platforms (Android, iOS) as well as WebAssembly, satisfying the
// cross-platform requirement.
//
// # Screens
//
// A single App owns one fyne.Window. Screens are controllers that build a
// fyne.CanvasObject and install it with window.SetContent. The flow mirrors the
// original state machine:
//
//	Main Menu  →  Setup  →  Game  →  End Game
//	                          ↑          |
//	                          └──────────┘ (Play Again)
//
// # Game screen
//
// gameScreen is a controller holding all mutable game state plus the widgets
// that visualise it (a 15×15 grid of cellWidgets, two racks of rackSlotWidgets,
// the score/status bar and the control buttons). State changes happen in event
// handlers; refresh() pushes the current engine.GameState into every widget.
//
// The AI move is computed on a background goroutine: a clone of the game state
// is handed to ai.ChooseMove so the goroutine never shares mutable state with
// the UI. The result is marshalled back onto the UI goroutine with fyne.Do, and
// a 10-second timeout converts a stuck AI into a pass.
//
// # Rendering
//
// All drawing is programmatic (no external image assets): canvas.Rectangle for
// board cells and tiles, canvas.Text for letters, premium labels and points.
// The board and racks use custom layouts so cells stay square and scale to the
// available space on any screen size.
//
// # Persistence
//
// SaveManager (save.go) is unchanged from the original implementation: it writes
// engine.GameState to os.UserConfigDir()/tilewords/savegame.gob with an atomic
// temp-file rename and is injectable for headless testing.
//
// # Usage
//
//	if err := ui.Run(); err != nil {
//	    log.Fatal(err)
//	}
package ui
