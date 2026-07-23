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
// (reporting via onCopied). The catcher implements only Draggable and SecondaryTappable,
// so other gestures fall through to the label beneath it (double-tap word / triple-tap
// line selection keep working); drag-to-extend-selection is traded for drag-to-scroll.
//
// On desktop it is a no-op: the mouse wheel already scrolls, click-drag selects, and the
// tab bar's Copy button copies the whole panel.
func enableTouchScroll(s *container.Scroll, copyText func() string, onCopied func()) {
	if !fyne.CurrentDevice().IsMobile() {
		return
	}
	s.Content = container.NewStack(s.Content, newDragScroller(s, copyText, onCopied))
	s.Refresh()
}

// dragScroller handles drag (pan) and secondary tap (copy); it deliberately implements no
// other pointer interface so double-tap/triple-tap selection falls through to the label.
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
}

// newDragScroller returns a catcher that pans target and copies copyText() on a long press.
func newDragScroller(target *container.Scroll, copyText func() string, onCopied func()) *dragScroller {
	d := &dragScroller{target: target, copyText: copyText, onCopied: onCopied}
	d.ExtendBaseWidget(d)
	return d
}

// Dragged pans the target by the drag delta, clamped to the scrollable range. The
// offset moves opposite the drag (dragging down reveals earlier content), matching
// the Scroll's own touch panning.
func (d *dragScroller) Dragged(e *fyne.DragEvent) {
	s := d.target
	outer := s.Size()
	inner := s.Content.MinSize()
	s.Offset = fyne.NewPos(
		clampOffset(s.Offset.X-e.Dragged.DX, outer.Width, inner.Width),
		clampOffset(s.Offset.Y-e.Dragged.DY, outer.Height, inner.Height),
	)
	s.Refresh()
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
