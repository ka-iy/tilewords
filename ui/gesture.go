// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

// gestureOwner remembers the widget a touch gesture began on, so a drag can be delivered to it
// even when the driver hands the drag somewhere else.
//
// The mobile driver decides who owns a gesture at the first move it gets round to processing —
// measured on an emulator at a median of 75 ms after the touch went down, worst case 132 ms — and
// it decides by hit-testing wherever the pointer is *then*. A rack tile is 44 DIP, so a pointer
// heading for the board has usually left it by that point, and the gesture goes to whatever now
// sits under the pointer: the page scroll, which pans instead. The tile is never offered the drag
// at all.
//
// TouchDown is the fix, because it fires on the widget the touch actually went down on. Recording
// the owner there turns "who should get this drag" from a guess about geometry into a fact, and
// makes the outcome independent of how long the driver took to react.
type gestureOwner struct {
	// owner is the widget the current gesture began on, or nil when a gesture began somewhere
	// with no claim on it.
	owner fyne.Draggable
}

// claim records d as the owner of the gesture now beginning.
//
// Nothing needs to clear this when the pointer leaves d: the driver's TouchCancel fires as soon as
// that happens, which is exactly while the gesture is still live and the ownership still matters.
// It is cleared instead when the gesture ends, or replaced when the next one begins.
func (g *gestureOwner) claim(d fyne.Draggable) {
	g.owner = d
}

// current returns the widget owning the gesture in progress, or nil.
func (g *gestureOwner) current() fyne.Draggable {
	return g.owner
}

// release forgets the current owner. It is called when a gesture ends, and when one begins
// somewhere that makes no claim, so a completed gesture cannot redirect a later one.
func (g *gestureOwner) release() {
	g.owner = nil
}

// deliverTo reports whether receiver should hand this drag to another widget, returning that
// widget. A widget calls it on its own drag events: the answer is nil when it owns the gesture
// itself (the ordinary case) or when nothing claimed one.
func (g *gestureOwner) deliverTo(receiver fyne.Draggable) fyne.Draggable {
	if g.owner == nil || g.owner == receiver {
		return nil
	}
	return g.owner
}

// pageDragRouter is a transparent widget covering the scrolling page, sitting *under* the page's
// content. It routes a drag to the widget the gesture began on, and pans the page when no widget
// claimed it.
//
// The stacking order is what makes this work, and it is the opposite of the panel catchers in
// dragscroll.go. The driver's hit test keeps the LAST match in a tree walk, so:
//
//   - the page content is stacked after this, and therefore still wins wherever the pointer is
//     over a tile, a button or a pane — nothing about their behaviour changes;
//   - this is deeper in the tree than the enclosing Scroll, so it wins over the Scroll's own
//     panning on the dead space between and around those widgets, which is exactly where a drag
//     that started on a tile used to be captured as a pan.
type pageDragRouter struct {
	widget.BaseWidget

	// target is the page scroll this pans when a drag belongs to no widget.
	target *container.Scroll

	// gesture is the shared record of which widget the current gesture began on.
	gesture *gestureOwner
}

// pageDragRouter routes drags and observes touch phases. It implements no tap interface, so taps
// on the page are unaffected by it.
var (
	_ fyne.Draggable   = (*pageDragRouter)(nil)
	_ mobile.Touchable = (*pageDragRouter)(nil)
)

// newPageDragRouter returns a router that pans target, deferring to gesture's owner.
func newPageDragRouter(target *container.Scroll, gesture *gestureOwner) *pageDragRouter {
	r := &pageDragRouter{target: target, gesture: gesture}
	r.ExtendBaseWidget(r)
	return r
}

// TouchDown releases any previous claim: a touch reaching this router landed on page background
// rather than on a widget that wants the gesture, since anything deeper would have received it
// instead. Without this, a stale owner from an earlier gesture could redirect this one.
func (r *pageDragRouter) TouchDown(*mobile.TouchEvent) {
	r.gesture.release()
}

// TouchUp ends the gesture's claim.
func (r *pageDragRouter) TouchUp(*mobile.TouchEvent) {
	r.gesture.release()
}

// TouchCancel deliberately keeps the claim: it fires as soon as the pointer leaves the object it
// went down on, which is while the gesture is still running.
func (r *pageDragRouter) TouchCancel(*mobile.TouchEvent) {}

// Dragged hands the drag to the widget the gesture began on, or pans the page when there is none.
func (r *pageDragRouter) Dragged(e *fyne.DragEvent) {
	if owner := r.gesture.current(); owner != nil {
		owner.Dragged(e)
		return
	}
	r.pan(e)
}

// pan scrolls the page by the drag delta, clamped to its travel. Other gesture handlers call it
// with drags they cannot use themselves, so a gesture aimed at the page still moves it.
func (r *pageDragRouter) pan(e *fyne.DragEvent) {
	s := r.target
	outer := s.Size()
	inner := s.Content.MinSize()
	s.ScrollToOffset(fyne.NewPos(
		clampOffset(s.Offset.X-e.Dragged.DX, outer.Width, inner.Width),
		clampOffset(s.Offset.Y-e.Dragged.DY, outer.Height, inner.Height),
	))
}

// DragEnd completes a routed gesture. Panning is applied incrementally in Dragged, so only a
// forwarded gesture needs anything here.
func (r *pageDragRouter) DragEnd() {
	if owner := r.gesture.current(); owner != nil {
		owner.DragEnd()
	}
	r.gesture.release()
}

// CreateRenderer returns a transparent renderer: the router is invisible and only routes drags.
func (r *pageDragRouter) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}
