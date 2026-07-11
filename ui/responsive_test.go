package ui

import (
	"image/color"
	"math/rand"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"squabble/dictionary"
	"squabble/engine"
)

// TestPhoneColumn_BoardFillsWidthAndClamps verifies the board scales up to fill the
// phone width as a square, and clamps to the tappable minimum when the column is
// narrower than that minimum.
func TestPhoneColumn_BoardFillsWidthAndClamps(t *testing.T) {
	board := canvas.NewRectangle(color.Black)
	other := canvas.NewRectangle(color.White)
	l := phoneColumnLayout{board: board, minBoard: minBoardPx}

	l.Layout([]fyne.CanvasObject{other, board}, fyne.NewSize(390, 1000))
	if board.Size().Width != 390 || board.Size().Height != 390 {
		t.Errorf("fill: board = %v, want 390x390", board.Size())
	}

	l.Layout([]fyne.CanvasObject{board}, fyne.NewSize(300, 1000))
	if board.Size().Width != float32(minBoardPx) {
		t.Errorf("clamp: board width = %v, want %d", board.Size().Width, minBoardPx)
	}
}

// TestGameScreen_ResponsiveFitsPhone verifies the game screen adopts a phone-friendly
// minimum size when sized to a phone (so it fits an average phone width and scrolls
// vertically), in both portrait and short-landscape, and tolerates the wide desktop
// size without panicking.
func TestGameScreen_ResponsiveFitsPhone(t *testing.T) {
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewSource(1)))
	content := newGameScreen(nil, state, dict).build()

	const phoneWidth = 412 // generous average-phone portrait width (dp)
	for _, sz := range []fyne.Size{
		fyne.NewSize(390, 844), // phone portrait
		fyne.NewSize(840, 390), // phone landscape (short height)
	} {
		content.Resize(sz)
		if w := content.MinSize().Width; w > phoneWidth {
			t.Errorf("at %v: min width %v exceeds an average phone width (%d)", sz, w, phoneWidth)
		}
	}

	// Plenty of room: the wide layout is allowed; just must not panic or over-shrink.
	content.Resize(fyne.NewSize(960, 760))
	_ = content.MinSize()

	// Switching back to a phone size must return to the narrow, fits-phone layout.
	content.Resize(fyne.NewSize(390, 844))
	if w := content.MinSize().Width; w > phoneWidth {
		t.Errorf("after switching back to phone size, min width %v exceeds %d", w, phoneWidth)
	}
}
