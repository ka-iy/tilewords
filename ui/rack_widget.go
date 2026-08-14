// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"

	"tilewords/engine"
)

// rackSlotWidget is a single tappable rack slot. It displays an empty slot, a
// face-down tile (hidden CPU rack), or a face-up tile with optional selection or
// exchange highlight. It reports taps with its slot index.
type rackSlotWidget struct {
	widget.BaseWidget

	idx int

	tile        *engine.Tile // nil = empty slot or staged-out
	faceDown    bool         // hide the letter (CPU rack while concealed)
	selected    bool         // currently selected for placement
	exchangeSel bool         // selected for exchange

	onTap func(idx int)
	// onDrag/onDragEnd power drag-and-drop: onDrag fires repeatedly while a drag is
	// in progress (used to drive the ghost tile), onDragEnd fires once at release with
	// the final pointer position so the controller can hit-test the drop target.
	onDrag    func(idx int, abs fyne.Position)
	onDragEnd func(idx int, abs fyne.Position)

	// onUnusableDrag receives a drag this slot cannot act on, which is any drag when neither drag
	// handler is wired — a slot that shows a tile without offering to move it, i.e. the CPU's rack.
	// The controller pans the page with it, so that row is not a dead zone for scrolling.
	onUnusableDrag func(e *fyne.DragEvent)

	// gesture is the shared record of which widget a touch gesture began on; see gestureOwner. A
	// slot claims it when a touch lands on the slot, so a drag the driver later hands to another
	// widget still reaches this one. Nil outside the game screen.
	gesture *gestureOwner

	dragging bool
	dragAbs  fyne.Position // most recent absolute pointer position during a drag
}

func newRackSlotWidget(idx int, onTap func(int)) *rackSlotWidget {
	s := &rackSlotWidget{idx: idx, onTap: onTap}
	s.ExtendBaseWidget(s)
	return s
}

// setContent updates the slot's display and refreshes it. A slot whose contents are
// unchanged is left untouched, for the same reason as cellWidget.setContent: every refresh
// calls this for all slots, and a Refresh discards and re-uploads the slot's textures. tile
// is compared by value because callers pass the address of a fresh copy each time.
func (s *rackSlotWidget) setContent(tile *engine.Tile, faceDown, selected, exchangeSel bool) {
	unchanged := faceDown == s.faceDown && selected == s.selected && exchangeSel == s.exchangeSel &&
		(tile == nil) == (s.tile == nil) &&
		(tile == nil || *tile == *s.tile)
	if unchanged {
		return
	}
	s.tile = tile
	s.faceDown = faceDown
	s.selected = selected
	s.exchangeSel = exchangeSel
	s.Refresh()
}

func (s *rackSlotWidget) Tapped(_ *fyne.PointEvent) {
	if s.onTap != nil {
		s.onTap(s.idx)
	}
}

// Dragged records the live pointer position and notifies the controller so the
// dragged tile can be shown as picked up. Implementing fyne.Draggable also means a
// slightly-moving tap (which Fyne classifies as a drag) is no longer discarded — it
// arrives here and then as DragEnd, which the controller resolves like a tap.
func (s *rackSlotWidget) Dragged(e *fyne.DragEvent) {
	// A drag belonging to a gesture that began on another widget is delivered there instead.
	if s.gesture != nil {
		if owner := s.gesture.deliverTo(s); owner != nil {
			owner.Dragged(e)
			return
		}
	}
	// A slot with no drag handlers has no use for the gesture, so it goes to the page rather than
	// being absorbed here. The slot stays not tracking a drag, so the rest of the gesture, and its
	// DragEnd, take this path too.
	if s.onDrag == nil && s.onDragEnd == nil {
		if s.onUnusableDrag != nil {
			s.onUnusableDrag(e)
		}
		return
	}
	// AbsolutePosition is (0,0) during a drag on mobile; track the pointer via the delta.
	s.dragAbs = dragAbsPosition(s, s.dragging, s.dragAbs, e)
	s.dragging = true
	if s.onDrag != nil {
		s.onDrag(s.idx, s.dragAbs)
	}
}

// cancelDrag discards the widget's in-progress drag tracking; see cellWidget.cancelDrag for
// why the controller must reset this alongside its own drag source.
func (s *rackSlotWidget) cancelDrag() {
	s.dragging = false
	s.dragAbs = fyne.Position{}
}

// DragEnd reports the gesture's final pointer position to the controller, which
// decides whether it lands on a board cell (place), another rack slot (reorder), or
// nowhere meaningful (treated as a tap).
func (s *rackSlotWidget) DragEnd() {
	if s.gesture != nil {
		if owner := s.gesture.deliverTo(s); owner != nil {
			owner.DragEnd()
			return
		}
	}
	if s.dragging && s.onDragEnd != nil {
		s.onDragEnd(s.idx, s.dragAbs)
	}
	s.dragging = false
	// Release here as well as in TouchUp: the driver delivers no TouchUp for a gesture that
	// became a drag, so this is the only end-of-gesture this slot sees for every drag it owns.
	// Without it the claim outlives the gesture and redirects a later one.
	if s.gesture != nil {
		s.gesture.releaseIf(s)
	}
}

// TouchDown claims the gesture for this slot. It is the whole point of implementing
// mobile.Touchable here: TouchDown fires on the widget the touch actually went down on, which is
// the one fact that identifies where a gesture started. See gestureOwner for why inferring it from
// the drag events instead does not work.
func (s *rackSlotWidget) TouchDown(*mobile.TouchEvent) {
	// Only a slot that handles drags claims the gesture; the CPU's rack does not.
	if s.gesture != nil && (s.onDrag != nil || s.onDragEnd != nil) {
		s.gesture.claim(s)
	}
}

// TouchUp ends this slot's claim on the gesture.
func (s *rackSlotWidget) TouchUp(*mobile.TouchEvent) {
	if s.gesture != nil {
		s.gesture.release()
	}
}

// TouchCancel deliberately keeps the claim. The driver cancels the touch as soon as the pointer
// leaves this slot — about 100 ms into every drag, successful ones included — which is exactly
// while the gesture is still running and the claim still matters.
func (s *rackSlotWidget) TouchCancel(*mobile.TouchEvent) {}

func (s *rackSlotWidget) CreateRenderer() fyne.WidgetRenderer {
	bg, letter, points := newTileObjects()
	r := &rackSlotRenderer{slot: s, bg: bg, letter: letter, points: points}
	r.applyState()
	return r
}

type rackSlotRenderer struct {
	slot   *rackSlotWidget
	bg     *canvas.Rectangle
	letter *canvas.Text
	points *canvas.Text
}

func (r *rackSlotRenderer) applyState() {
	s := r.slot

	// Empty slot or a tile that is currently staged on the board.
	if s.tile == nil {
		r.bg.FillColor = colorRackSlot
		r.bg.StrokeColor = colorTileBorder
		r.bg.StrokeWidth = 1
		r.letter.Text = ""
		r.points.Text = ""
		return
	}

	// Hidden CPU tile.
	if s.faceDown {
		r.bg.FillColor = colorTileFaceDown
		r.bg.StrokeColor = colorTileBorder
		r.bg.StrokeWidth = 1
		r.letter.Text = ""
		r.points.Text = ""
		return
	}

	// Visible tile.
	styleAsTile(r.bg, r.letter, r.points, *s.tile, s.selected)
	if s.exchangeSel {
		// Distinct highlight for exchange selection (overrides the staged border).
		r.bg.StrokeColor = colorTileExchangeSel
		r.bg.StrokeWidth = 3
	}
}

func (r *rackSlotRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
	// Rack slots display tiles, so the letter is shifted right of centre (an empty or
	// face-down slot has no letter, so the shift is moot there).
	layoutTileText(r.letter, r.points, size, 0.5, tileLetterShiftFactor)
}

func (r *rackSlotRenderer) MinSize() fyne.Size {
	return fyne.NewSize(minRackSlotPx, minRackSlotPx)
}

func (r *rackSlotRenderer) Refresh() {
	r.applyState()
	r.Layout(r.slot.Size())
	canvas.Refresh(r.slot)
}

func (r *rackSlotRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.letter, r.points}
}

func (r *rackSlotRenderer) Destroy() {}
