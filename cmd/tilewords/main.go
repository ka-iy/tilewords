// Command tilewords launches the TileWords word game.
//
// Usage:
//
//	tilewords
//
// The UI is built with the Fyne toolkit, which renders an event-driven, resizable
// window on desktop platforms and a touch UI on mobile (Android/iOS). Save files
// are stored under os.UserConfigDir()/tilewords/savegame.gob.
package main

import (
	"log"

	"tilewords/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		log.Fatalf("tilewords: %v", err)
	}
}
