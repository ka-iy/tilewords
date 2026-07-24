package ui

import (
	"image/color"
	"math/rand"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// TestPhoneColumn_BoardFillsWidth verifies the board fills the phone column width as a
// square, both when there is ample room and when the column is narrower than the board's
// preferred minimum — where it shrinks to fit rather than overflowing the viewport.
func TestPhoneColumn_BoardFillsWidth(t *testing.T) {
	board := canvas.NewRectangle(color.Black)
	other := canvas.NewRectangle(color.White)
	l := phoneColumnLayout{board: board, minBoard: minBoardPx}

	l.Layout([]fyne.CanvasObject{other, board}, fyne.NewSize(390, 1000))
	if board.Size().Width != 390 || board.Size().Height != 390 {
		t.Errorf("fill: board = %v, want 390x390", board.Size())
	}

	// Narrower than minBoardPx: the board shrinks to the width, it does NOT clamp up (which
	// would overflow the viewport, since the vertical scroll cannot pan horizontally).
	l.Layout([]fyne.CanvasObject{board}, fyne.NewSize(300, 1000))
	if board.Size().Width != 300 {
		t.Errorf("narrow: board width = %v, want 300 (fills the width, no overflow)", board.Size().Width)
	}
}

// TestPhoneColumn_FitsSubMinimumViewport is a regression guard: on a phone whose width is
// below the board's preferred minimum (e.g. 352 vs a 360 minimum, as measured on an
// emulator), the column must not advertise a width wider than the viewport, and the board
// must fit within it. Otherwise the vertical-only scroll — which sizes content to
// MinSize().Max(viewport) — makes the board and tab row wider than the screen and clips
// their right edge.
func TestPhoneColumn_FitsSubMinimumViewport(t *testing.T) {
	board := canvas.NewRectangle(color.Black)
	board.SetMinSize(fyne.NewSize(minBoardPx, minBoardPx))
	l := phoneColumnLayout{board: board, minBoard: minBoardPx}
	const viewport = minBoardPx - 8 // a viewport just under the board's preferred minimum

	if w := l.MinSize([]fyne.CanvasObject{board}).Width; w > viewport {
		t.Fatalf("MinSize width %.0f exceeds sub-minimum viewport %.0f; content would be forced past the screen", w, viewport)
	}
	l.Layout([]fyne.CanvasObject{board}, fyne.NewSize(viewport, 1000))
	if bw := board.Size().Width; bw > viewport {
		t.Errorf("board width %.0f exceeds viewport %.0f (right edge clipped)", bw, viewport)
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

	if w := l.MinSize(objs).Width; w != 0 {
		t.Fatalf("MinSize width = %.0f, want 0 (the column advertises no width floor, so a wide child cannot inflate it)", w)
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
