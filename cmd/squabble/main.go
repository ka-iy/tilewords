// Command squabble launches the Squabble word game.
//
// Usage:
//
//	squabble
//
// The UI is built with the Fyne toolkit, which renders an event-driven, resizable
// window on desktop platforms and a touch UI on mobile (Android/iOS). Save files
// are stored under os.UserConfigDir()/squabble/savegame.gob.
package main

import (
	"log"

	"squabble/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		log.Fatalf("squabble: %v", err)
	}
}
