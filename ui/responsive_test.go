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

// TestPhoneColumn_MinWidthIgnoresWideChildren verifies a child wider than the viewport
// (e.g. the status row at a large system font) does not inflate the column's minimum
// width. Otherwise the vertical scroll — which sizes content to MinSize().Max(viewport) —
// would make the column and the board wider than the screen and grow them on re-layout.
func TestPhoneColumn_MinWidthIgnoresWideChildren(t *testing.T) {
	board := canvas.NewRectangle(color.Black)
	board.SetMinSize(fyne.NewSize(minBoardPx, minBoardPx))
	wide := canvas.NewRectangle(color.White)
	wide.SetMinSize(fyne.NewSize(minBoardPx*2, 40)) // far wider than any phone viewport
	l := phoneColumnLayout{board: board, minBoard: minBoardPx}
	objs := []fyne.CanvasObject{board, wide}

	if w := l.MinSize(objs).Width; w != float32(minBoardPx) {
		t.Fatalf("MinSize width = %.0f, want %d (a wide child must not inflate the column)", w, minBoardPx)
	}

	// At a normal phone width the board fills the viewport and the wide child is clamped to
	// it — neither is forced past the viewport.
	l.Layout(objs, fyne.NewSize(400, 1000))
	if bw := board.Size().Width; bw != 400 {
		t.Errorf("board width = %.0f, want 400 (fills viewport, not forced wider)", bw)
	}
	if ww := wide.Size().Width; ww != 400 {
		t.Errorf("wide child width = %.0f, want 400 (clamped to viewport)", ww)
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
