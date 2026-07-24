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

// statusRowGap is the vertical gap between the word list and the score counters in the
// status block. It is negative so the two rows sit tighter than a default VBox would
// place them: a VBox adds theme padding between children, and each label carries its own
// top/bottom padding on top of that, which together leave a large gap. Overlapping those
// padded boxes slightly closes the space without clipping the text.
const statusRowGap = -20

// statusMoveGap is the vertical gap above the current-move line (the last status row). It
// is kept looser than statusRowGap so the move line is set off from the score counters
// (rather than tucked right up against them like the word list is), but not so loose that
// it drifts down toward the rack.
const statusMoveGap = -10

// statusRackGap is the vertical gap between the status block and the rack in the stacked
// phone layout (in place of the usual phoneColGap). It is negative to pull the rack up
// closer to the current-move line so the block reads as one group.
const statusRackGap = -12

// tightColumnLayout stacks its children vertically, each stretched to the full width.
// gaps[i] is the vertical gap between child i and child i+1; a negative value pulls the
// children's padded boxes closer than a standard VBox (which always separates by theme
// padding). Boundaries with no matching entry (index beyond len(gaps)) get no gap. Used
// to control the status block's inter-row spacing row by row.
type tightColumnLayout struct{ gaps []float32 }

// gapAfter returns the gap to insert after child i (before child i+1), or 0 when i has no
// configured gap.
func (l tightColumnLayout) gapAfter(i int) float32 {
	if i >= 0 && i < len(l.gaps) {
		return l.gaps[i]
	}
	return 0
}

func (l tightColumnLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for i, o := range objs {
		h := o.MinSize().Height
		o.Resize(fyne.NewSize(size.Width, h))
		o.Move(fyne.NewPos(0, y))
		y += h
		if i < len(objs)-1 {
			y += l.gapAfter(i)
		}
	}
}

func (l tightColumnLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	w := float32(0)
	h := float32(0)
	for i, o := range objs {
		m := o.MinSize()
		if m.Width > w {
			w = m.Width
		}
		h += m.Height
		if i < len(objs)-1 {
			h += l.gapAfter(i)
		}
	}
	if h < 0 {
		h = 0
	}
	return fyne.NewSize(w, h)
}

// phoneColGap is the vertical gap between sections in the stacked phone layout.
const phoneColGap = 4

// phoneColumnLayout stacks children vertically for the narrow phone layout. The
// board child is rendered as a square that fills the column width — scaling up from
// minBoard so cells stay tappable — while every other child gets its own minimum
// height. The column is hosted by phoneColumnScroll, which scrolls it when it is taller
// than the viewport and otherwise stretches the history pane to fill the spare height.
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

// portraitHistoryMinH is the smallest history/definitions pane worth showing without
// scrolling in the stacked phone layout. When the rest of the column leaves at least this
// much room, the pane grows to fill the spare height and nothing scrolls; when it leaves
// less, the column must scroll anyway, so the pane is instead grown to twice this height
// (see phoneColumnScroll.Resize).
const portraitHistoryMinH = 160

// phoneColumnScroll hosts the stacked phone column (laid out by phoneColumnLayout). When
// the column's natural height fits the viewport it lays the column out directly and grows
// the history pane to fill the spare height down to the viewport bottom; when it does not
// fit it hosts the column in a vertical scroll with the history pane at its minimum
// height, so the page scrolls.
//
// A plain VScroll cannot do this on its own: it sizes its content to the content's
// MinSize, and a layout's MinSize cannot express the board's width-dependent (square)
// height, so the leftover height is only knowable from the widget's actual viewport size.
type phoneColumnScroll struct {
	widget.BaseWidget
	// column is the phoneColumnLayout container whose last child is histWrap.
	column *fyne.Container
	// board is the column's board child (its height equals the column width).
	board fyne.CanvasObject
	// histWrap wraps the history pane; its minimum height is adjusted to fill or floor.
	histWrap *fyne.Container
	// holder presents either the bare column (fill mode) or a scroll of it (scroll mode).
	holder *fyne.Container
	// scrolled records the current mode so it only switches when the mode changes.
	scrolled bool
	// inited guards the first mode selection.
	inited bool
}

func newPhoneColumnScroll(column *fyne.Container, board fyne.CanvasObject, histWrap *fyne.Container) *phoneColumnScroll {
	p := &phoneColumnScroll{column: column, board: board, histWrap: histWrap, holder: container.NewStack()}
	p.ExtendBaseWidget(p)
	return p
}

// nonHistoryHeight is the total height every column child except the history pane occupies
// (including the per-child gap), using the actual board square edge — the column width —
// for the board.
func (p *phoneColumnScroll) nonHistoryHeight(width float32) float32 {
	total := float32(0)
	for _, o := range p.column.Objects {
		if o == p.histWrap {
			continue
		}
		h := o.MinSize().Height
		if o == p.board {
			h = width
		}
		total += h + phoneColGap
	}
	return total
}

// setHistoryMin sets the history wrapper's minimum height and re-lays the column.
func (p *phoneColumnScroll) setHistoryMin(h float32) {
	p.histWrap.Layout = minHeightLayout{minH: h}
	p.histWrap.Refresh()
}

// setScrolled switches between the fill (bare column) and scroll arrangements, only
// rebuilding when the mode actually changes so the shared column is never double-parented.
func (p *phoneColumnScroll) setScrolled(scroll bool) {
	if p.inited && scroll == p.scrolled {
		return
	}
	p.inited = true
	p.scrolled = scroll
	if scroll {
		p.holder.Objects = []fyne.CanvasObject{container.NewVScroll(p.column)}
	} else {
		p.holder.Objects = []fyne.CanvasObject{p.column}
	}
	p.holder.Refresh()
}

// Resize chooses fill vs scroll for the new viewport, sizing the history pane to suit.
//   - Fits (the rest of the column plus a usable history leave no overflow): stretch the
//     history to exactly fill the spare height, so nothing scrolls.
//   - Overflows (even a minimal history would push the column past the viewport): the page
//     must scroll regardless, so give the history a bit more room — twice its minimum — so
//     more of it shows. A full viewport-tall pane is deliberately avoided: it would fill
//     the screen and swallow the drag gestures used to scroll the page itself.
func (p *phoneColumnScroll) Resize(size fyne.Size) {
	if size.Width > 0 && size.Height > 0 {
		fill := size.Height - p.nonHistoryHeight(size.Width)
		if fill < portraitHistoryMinH {
			p.setHistoryMin(portraitHistoryMinH * 2)
			p.setScrolled(true)
		} else {
			p.setHistoryMin(fill)
			p.setScrolled(false)
		}
	}
	p.BaseWidget.Resize(size)
}

func (p *phoneColumnScroll) CreateRenderer() fyne.WidgetRenderer {
	if !p.inited {
		// Default to scrolling until the first Resize reports the real viewport: the scroll
		// advertises a small minimum, so the window is never forced open before the choice
		// is corrected.
		p.setScrolled(true)
	}
	return widget.NewSimpleRenderer(p.holder)
}

// MinSize keeps the widget from forcing the window taller than the screen: the parent
// gives it the full viewport via Resize, and the scroll arrangement handles any overflow.
func (p *phoneColumnScroll) MinSize() fyne.Size { return fyne.NewSize(0, 0) }
