// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// groupTappable is a control a touchGroup can activate on behalf of a press the touch driver
// misdirected. Controls are activated through this rather than through fyne.Tappable so that
// a press the group has already resolved cannot be resolved a second time.
type groupTappable interface {
	fyne.CanvasObject
	// tapFromGroup activates the control for a press at pos, in the control's own coordinates.
	tapFromGroup(pos fyne.Position)
}

// groupJoiner is a control that hands a misdirected press back to its group, and so needs to
// be told which group it was laid out in.
type groupJoiner interface {
	joinTouchGroup(*touchGroup)
}

// touchGroup wraps a set of controls laid out together and repairs presses that the touch
// driver's upward compensation (see touchYCompensation) delivered to the wrong place.
//
// The compensation makes the top touchYCompensation of every control belong to whatever sits
// above it, which fails in two distinct ways. The group answers both:
//   - The press lands in the padding between controls, where nothing is tappable, and is
//     dropped. The group is itself tappable and spans that padding, so it receives the press
//     and replays it at the finger's true position.
//   - The press lands on the control above, which acts although it was not the control the
//     player pressed. That control notices the finger was below itself and hands the press
//     back here, where the control actually under the finger can be found; see
//     touchButton.Tapped.
//
// Being an ancestor of its members, the group is only ever the hit-test result for a press
// that reached none of them: the driver keeps the deepest match. It therefore cannot take a
// press away from a control that was pressed properly.
//
// It deliberately implements neither mobile.Touchable nor fyne.Draggable. Either would make
// the driver hand it gestures belonging to an enclosing Scroll, and the page would stop
// panning where the group covers it.
type touchGroup struct {
	widget.BaseWidget
	// content is the group's own layout, rendered exactly as given.
	content fyne.CanvasObject
	// members are the controls a misdirected press can be resolved to.
	members []groupTappable
}

// touchGroup must satisfy fyne.Tappable to catch the presses that reach no member; the
// assertion fails the build if a refactor drops Tapped.
var _ fyne.Tappable = (*touchGroup)(nil)

// newTouchGroup wraps content, whose controls are members. Members that a group cannot
// activate — labels, spacers, separators — are ignored, so a caller can hand over a layout's
// whole object list.
func newTouchGroup(content fyne.CanvasObject, members ...fyne.CanvasObject) *touchGroup {
	g := &touchGroup{content: content}
	for _, m := range members {
		t, ok := m.(groupTappable)
		if !ok {
			continue
		}
		g.members = append(g.members, t)
		if j, ok := m.(groupJoiner); ok {
			j.joinTouchGroup(g)
		}
	}
	g.ExtendBaseWidget(g)
	return g
}

// hitInset is how far above its content the group's touch area reaches: the compensation on a
// touch screen, nothing on desktop, where the pointer position is exact and an inflated area
// would swallow clicks meant for whatever sits above.
func (g *touchGroup) hitInset() float32 {
	if !deviceIsMobile() {
		return 0
	}
	return touchYCompensation
}

// Resize claims hitInset more height than the layout allocated. Together with Move this
// stretches the group's touch area up over the empty space above its content, which is where
// the driver puts a press aimed at the top edge of the first row of controls. The content is
// laid out at its allocated size and position regardless, so nothing on screen moves.
func (g *touchGroup) Resize(size fyne.Size) {
	size.Height += g.hitInset()
	g.BaseWidget.Resize(size)
}

// Move places the group hitInset above where the layout put it; see Resize.
func (g *touchGroup) Move(pos fyne.Position) {
	pos.Y -= g.hitInset()
	g.BaseWidget.Move(pos)
}

func (g *touchGroup) CreateRenderer() fyne.WidgetRenderer {
	return &touchGroupRenderer{group: g}
}

// Tapped replays a press that reached none of the group's controls, at the position the
// finger was really at. Nothing is replayed on desktop: the pointer position is already exact,
// so a click here landed on no control and belongs to no control.
func (g *touchGroup) Tapped(ev *fyne.PointEvent) {
	if ev == nil || !deviceIsMobile() {
		return
	}
	g.tapAtFinger(ev, nil)
}

// tapAtFinger activates the member the finger was over when ev was reported, skipping from —
// the control handing the press back, which is nil for a press the group caught itself. It
// reports whether a member was found; when none was, the press belonged to nothing here.
//
// The finger was touchYCompensation below the point the driver reported, so that is where the
// members are looked for. Member positions are read from the live widget tree, so a group
// whose controls are also laid out in another arrangement (the game screen builds one per
// viewport shape) resolves against the arrangement actually on screen.
func (g *touchGroup) tapAtFinger(ev *fyne.PointEvent, from fyne.CanvasObject) bool {
	finger := ev.AbsolutePosition.AddXY(0, touchYCompensation)
	drv := fyne.CurrentApp().Driver()
	for _, m := range g.members {
		if m == from || !m.Visible() {
			continue
		}
		origin := drv.AbsolutePositionForObject(m)
		size := m.Size()
		if finger.X < origin.X || finger.X >= origin.X+size.Width {
			continue
		}
		if finger.Y < origin.Y || finger.Y >= origin.Y+size.Height {
			continue
		}
		m.tapFromGroup(finger.Subtract(origin))
		return true
	}
	return false
}

// touchGroupRenderer draws the group's content at its allocated position and size, inside the
// group's taller touch area.
type touchGroupRenderer struct {
	group *touchGroup
}

// Layout places the content below the group's touch inset, so it occupies exactly the
// rectangle the parent layout allocated for the group.
func (r *touchGroupRenderer) Layout(size fyne.Size) {
	inset := r.group.hitInset()
	r.group.content.Move(fyne.NewPos(0, inset))
	r.group.content.Resize(fyne.NewSize(size.Width, size.Height-inset))
}

// MinSize is the content's own minimum: the inset is taken from the space above the group, not
// requested from the layout, so the group asks for no more room than its content needs.
func (r *touchGroupRenderer) MinSize() fyne.Size { return r.group.content.MinSize() }

func (r *touchGroupRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.group.content}
}

func (r *touchGroupRenderer) Refresh() { r.group.content.Refresh() }

func (r *touchGroupRenderer) Destroy() {}
