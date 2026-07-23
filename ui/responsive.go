// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// narrowThresholdW and narrowThresholdH are the size below which the game screen
// switches from the wide (side-by-side) layout to the narrow (stacked, scrollable)
// layout. Both exceed the wide layout's own minimum size so the switch always
// happens before the window can be pinned at the wide minimum.
const (
	narrowThresholdW = 700
	narrowThresholdH = 680
)

// useNarrow reports whether the stacked phone layout should be used for size.
func useNarrow(size fyne.Size) bool {
	return size.Width < narrowThresholdW || size.Height < narrowThresholdH
}

// responsiveContainer renders one of two arrangements of the SAME child widgets,
// chosen by the available size: a wide side-by-side layout when there is room, and
// a narrow stacked+scrollable layout for phone-sized screens. The arrangement is
// rebuilt (re-parenting the shared widgets) only when the size crosses the
// threshold, so switching is cheap and game state is preserved.
type responsiveContainer struct {
	widget.BaseWidget
	build  func(narrow bool) fyne.CanvasObject
	holder *fyne.Container // a stack holding the current arrangement
	narrow bool
	inited bool
}

func newResponsiveContainer(build func(narrow bool) fyne.CanvasObject) *responsiveContainer {
	r := &responsiveContainer{build: build, holder: container.NewStack()}
	r.ExtendBaseWidget(r)
	return r
}

func (r *responsiveContainer) set(narrow bool) {
	r.inited = true
	r.narrow = narrow
	r.holder.Objects = []fyne.CanvasObject{r.build(narrow)}
	r.holder.Refresh()
}

// Resize swaps the arrangement when the size crosses the wide/narrow threshold.
func (r *responsiveContainer) Resize(size fyne.Size) {
	if narrow := useNarrow(size); !r.inited || narrow != r.narrow {
		r.set(narrow)
	}
	r.BaseWidget.Resize(size)
}

func (r *responsiveContainer) CreateRenderer() fyne.WidgetRenderer {
	if !r.inited {
		// Default to the narrow layout: its minimum size is small (it scrolls), so
		// the window is never forced wider than a phone screen before the first
		// Resize corrects the choice.
		r.set(true)
	}
	return widget.NewSimpleRenderer(r.holder)
}

// minHeightLayout lays out a single child to fill the space while advertising a
// fixed minimum height (and the child's minimum width). Used to give the move-history
// panel a usable height inside the stacked phone layout.
type minHeightLayout struct{ minH float32 }

func (m minHeightLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func (m minHeightLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	w := float32(0)
	for _, o := range objs {
		if ms := o.MinSize(); ms.Width > w {
			w = ms.Width
		}
	}
	return fyne.NewSize(w, m.minH)
}

// phoneColGap is the vertical gap between sections in the stacked phone layout.
const phoneColGap = 4

// phoneColumnLayout stacks children vertically for the narrow phone layout. The
// board child is rendered as a square that fills the column width — scaling up from
// minBoard so cells stay tappable — while every other child gets its own minimum
// height. The column lives inside a vertical scroll, so the page scrolls when it is
// taller than the screen.
type phoneColumnLayout struct {
	board    fyne.CanvasObject
	minBoard float32
}

// boardSide returns the square edge for a column of the given width: the board always
// fills the full column width. On a screen narrower than the board's preferred minimum
// (minBoard) the cells shrink to fit rather than the board overflowing the viewport —
// the column lives in a vertical-only scroll, so any horizontal overflow is clipped.
func (p phoneColumnLayout) boardSide(width float32) float32 {
	return width
}

func (p phoneColumnLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, o := range objs {
		w, h := size.Width, o.MinSize().Height
		if o == p.board {
			side := p.boardSide(size.Width)
			w, h = side, side
		}
		o.Resize(fyne.NewSize(w, h))
		o.Move(fyne.NewPos(0, y))
		y += h + phoneColGap
	}
}

func (p phoneColumnLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	// The width floor is zero, so the column never advertises a minimum wider than the
	// viewport. The column lives in a vertical scroll, which sizes its content to
	// MinSize().Max(viewport): any positive width floor larger than the viewport — such as
	// the board's tappable minimum on a phone narrower than that (measured 352 vs a 360
	// minimum) — would make the scroll size the content, and the board and tabs that fill
	// it, past the screen; the vertical-only scroll then clips the overflow. The board and
	// every other child instead fill/clamp to the actual width in Layout, so a wide child
	// (e.g. the status row at a large system font) can never inflate the column either.
	//
	// Only the height is summed, so the column scrolls when it is taller than the screen.
	total := float32(0)
	for i, o := range objs {
		if o == p.board {
			total += p.minBoard
		} else {
			total += o.MinSize().Height
		}
		if i > 0 {
			total += phoneColGap
		}
	}
	return fyne.NewSize(0, total)
}
