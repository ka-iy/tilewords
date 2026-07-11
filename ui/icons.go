// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// playIconResource is a green "play" triangle shown beside the rack label when it is
// the human's turn. It uses a fixed green fill (not themed) so it always reads as the
// turn cue regardless of the active theme.
var playIconResource = &fyne.StaticResource{
	StaticName:    "play-turn.svg",
	StaticContent: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#50DC5A" d="M8 5v14l11-7z"/></svg>`),
}

// blankIconResource is an empty glyph the play slot shows on the AI's turn. widget.Icon
// reserves the same inline size for any resource, so swapping to this leaves the rack
// header layout unchanged (the play icon simply disappears in place).
var blankIconResource = &fyne.StaticResource{
	StaticName:    "blank.svg",
	StaticContent: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"></svg>`),
}

// shuffleIconResource is the Material "shuffle" glyph (two crossing arrows), themed so
// it follows the foreground colour like the other rack buttons. The previous
// ViewRefresh icon read as undo/redo, which this replaces.
var shuffleIconResource = theme.NewThemedResource(&fyne.StaticResource{
	StaticName:    "shuffle.svg",
	StaticContent: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#ffffff" d="M10.59 9.17L5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41l-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z"/></svg>`),
})

// recallIconResource is a down arrow dropping into a tray — "return the placed tiles
// to your rack". Themed to match the other rack buttons. It replaces the plain undo
// arrow, which was easily confused with the separate Undo (move) button.
var recallIconResource = theme.NewThemedResource(&fyne.StaticResource{
	StaticName:    "recall.svg",
	StaticContent: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="#ffffff" d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"/></svg>`),
})
