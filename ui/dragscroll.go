// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// enableTouchScroll makes a scroll pannable by finger drag on touch screens, and lets a
// long press on it copy the whole panel.
//
// A selectable widget.Label inside a Scroll captures touch drags for text selection —
// the driver sends a drag to the innermost Draggable, which is the label's selection
// overlay, so the enclosing Scroll never pans and the panel is stuck wherever the last
// auto-scroll left it. On mobile this layers a transparent catcher above the scroll's
// content that forwards drags to the scroll and copies copyText() on a long press
// (reporting via onCopied).
//
// On touch this replaces the label's own tap gestures rather than coexisting with them.
// The driver hit-tests by walking the visible tree and keeping the LAST match, and the
// catcher is stacked after the label, so every tap the catcher matches resolves to the
// catcher. It matches as a SecondaryTappable, and implements neither Tappable nor
// DoubleTappable, so an ordinary tap on the panel is dropped and the label's double-tap
// word / triple-tap line selection never fires. Copying is therefore via the long press
// here and the tab bar's Copy button, not by selecting text.
//
// onExhausted receives any drag the scroll has no room for at all; see dragScroller.Dragged. Pass
// nil where there is nothing behind the panel to pan.
//
// On desktop it is a no-op: the mouse wheel already scrolls, click-drag selects, and the
// tab bar's Copy button copies the whole panel.
func enableTouchScroll(s *container.Scroll, copyText func() string, onCopied func(),
	onExhausted func(e *fyne.DragEvent),
) {
	if !fyne.CurrentDevice().IsMobile() {
		return
	}
	s.Content = container.NewStack(s.Content, newDragScroller(s, copyText, onCopied, onExhausted))
	s.Refresh()
}

// dragScroller handles drag (pan) and secondary tap (copy). It implements no other pointer
// interface, which means an ordinary tap that lands on it is discarded rather than passed
// down — see enableTouchScroll for why nothing beneath it receives taps on touch.
var (
	_ fyne.Draggable         = (*dragScroller)(nil)
	_ fyne.SecondaryTappable = (*dragScroller)(nil)
)

// dragScroller is a transparent overlay that pans a target Scroll on drag and copies the
// whole panel on a long press (secondary tap); other taps pass through to the widgets below.
type dragScroller struct {
	widget.BaseWidget
	// target is the scroll this overlay pans.
	target *container.Scroll
	// copyText returns the text a long press copies to the clipboard; may be nil.
	copyText func() string
	// onCopied, when set, is invoked after a long press copies text (for user feedback).
	onCopied func()
	// onExhausted receives a drag when the target cannot scroll at all — its content shorter than
	// its viewport, so there is no travel in either axis. The controller pans the page with it,
	// since the driver delivers the whole gesture here once it has latched on and absorbing it
	// would leave a swipe that began on the panel unable to move the page behind it. May be nil.
	onExhausted func(e *fyne.DragEvent)
}

// newDragScroller returns a catcher that pans target and copies copyText() on a long press.
func newDragScroller(target *container.Scroll, copyText func() string, onCopied func(),
	onExhausted func(e *fyne.DragEvent),
) *dragScroller {
	d := &dragScroller{target: target, copyText: copyText, onCopied: onCopied, onExhausted: onExhausted}
	d.ExtendBaseWidget(d)
	return d
}

// Dragged pans the target by the drag delta, clamped to the scrollable range. The
// offset moves opposite the drag (dragging down reveals earlier content), matching
// the Scroll's own touch panning.
// Panning goes through ScrollToOffset rather than assigning Offset and calling Refresh:
// Scroll.Refresh recursively refreshes the scrolled content, which re-wraps the long
// selectable label these panels hold, while ScrollToOffset repositions the content and bars
// without touching it. That matters per frame — after a flick the driver replays decaying
// drag events every 16 ms for up to half a second.
func (d *dragScroller) Dragged(e *fyne.DragEvent) {
	s := d.target
	outer := s.Size()
	inner := s.Content.MinSize()

	// A panel with no travel in either axis can do nothing with this drag; hand it to the page.
	// A panel merely sitting at an end of its travel still absorbs it: forwarding there would let
	// one gesture run the panel to its limit and then carry on panning the page.
	if inner.Width <= outer.Width && inner.Height <= outer.Height {
		if d.onExhausted != nil {
			d.onExhausted(e)
		}
		return
	}
	// Pre-clamping keeps ScrollToOffset's "offset unchanged" early return effective once a
	// fling reaches the end of the travel, so the replayed events stop costing anything.
	s.ScrollToOffset(fyne.NewPos(
		clampOffset(s.Offset.X-e.Dragged.DX, outer.Width, inner.Width),
		clampOffset(s.Offset.Y-e.Dragged.DY, outer.Height, inner.Height),
	))
}

// DragEnd completes the gesture; panning is applied incrementally in Dragged, so
// nothing more is needed.
func (d *dragScroller) DragEnd() {}

// TappedSecondary copies the whole panel to the clipboard on a long press (the touch
// gesture the driver reports as a secondary tap). It is a no-op when there is no
// copyText or the panel is empty.
func (d *dragScroller) TappedSecondary(*fyne.PointEvent) {
	if d.copyText == nil {
		return
	}
	text := d.copyText()
	if text == "" {
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(text)
	if d.onCopied != nil {
		d.onCopied()
	}
}

// CreateRenderer returns a transparent renderer: the overlay is invisible and only
// intercepts drags.
func (d *dragScroller) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

// clampOffset limits a scroll offset to [0, inner-outer], matching the clamping the
// Scroll applies to its own panning.
func clampOffset(offset, outer, inner float32) float32 {
	if offset+outer >= inner {
		offset = inner - outer
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
