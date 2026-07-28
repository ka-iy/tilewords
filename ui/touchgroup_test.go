// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// groupHarness is two stacked buttons in a touchGroup, laid out for real so presses can be
// aimed at actual coordinates.
type groupHarness struct {
	group     *touchGroup
	top       *touchButton
	bottom    *touchButton
	topTaps   int
	lowerTaps int
}

// newGroupHarness lays out the harness in a window of the given size. The touch device is
// selected before layout, so the group's touch inset is in place.
func newGroupHarness(t *testing.T) *groupHarness {
	t.Helper()
	_ = test.NewApp()
	onTouchDevice(t)

	h := &groupHarness{}
	h.top = newTouchButton("Top", func() { h.topTaps++ })
	h.bottom = newTouchButton("Bottom", func() { h.lowerTaps++ })
	h.group = newTouchGroup(container.NewVBox(h.top, h.bottom), h.top, h.bottom)

	w := test.NewWindow(h.group)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(300, 300))
	if h.bottom.Size().Height <= touchYCompensation {
		t.Fatalf("setup: button height %.0f is not above the %d DIP compensation",
			h.bottom.Size().Height, touchYCompensation)
	}
	return h
}

// absTop returns the absolute position of obj.
func absTop(obj fyne.CanvasObject) fyne.Position {
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(obj)
}

// pressAimedAt builds the event the touch driver reports for a finger at the given absolute
// point: the driver hit-tests touchYCompensation above the finger, and reports the position
// within whichever widget it found there.
func pressAimedAt(finger fyne.Position, hit fyne.CanvasObject) *fyne.PointEvent {
	reported := finger.SubtractXY(0, touchYCompensation)
	return &fyne.PointEvent{
		AbsolutePosition: reported,
		Position:         reported.Subtract(absTop(hit)),
	}
}

// TestTouchGroupRedirectsPressFromControlAbove verifies that a press aimed at a button's top
// edge runs that button and not the one above it. The driver reports such a press against the
// button above, which must hand it back rather than act on it.
func TestTouchGroupRedirectsPressFromControlAbove(t *testing.T) {
	h := newGroupHarness(t)
	finger := absTop(h.bottom).AddXY(4, 2) // just inside the lower button's top edge

	// The driver reports this press against the button above.
	h.top.Tapped(pressAimedAt(finger, h.top))

	if h.lowerTaps != 1 || h.topTaps != 0 {
		t.Errorf("press on the lower button ran top=%d lower=%d, want top=0 lower=1",
			h.topTaps, h.lowerTaps)
	}
}

// TestTouchGroupCatchesPressAboveItsContent verifies a press aimed at the first row of
// controls is not lost. The compensated position lands above the content, where nothing is
// tappable, so the group's own touch area must reach up and catch it.
func TestTouchGroupCatchesPressAboveItsContent(t *testing.T) {
	h := newGroupHarness(t)
	finger := absTop(h.top).AddXY(4, 2)

	// Nothing is tappable at the compensated position, so the group is what the driver finds.
	h.group.Tapped(pressAimedAt(finger, h.group))

	if h.topTaps != 1 || h.lowerTaps != 0 {
		t.Errorf("press on the top button ran top=%d lower=%d, want top=1 lower=0",
			h.topTaps, h.lowerTaps)
	}
}

// TestTouchGroupIgnoresPressOverNoControl verifies the group does not invent a press: a touch
// on its padding, with no control under the finger either, runs nothing.
func TestTouchGroupIgnoresPressOverNoControl(t *testing.T) {
	h := newGroupHarness(t)
	// A finger in the gap between the two buttons is over neither of them.
	finger := absTop(h.bottom).SubtractXY(-4, 2)

	h.group.Tapped(pressAimedAt(finger, h.group))

	if h.topTaps != 0 || h.lowerTaps != 0 {
		t.Errorf("press over no control ran top=%d lower=%d, want both 0", h.topTaps, h.lowerTaps)
	}
}

// TestTouchGroupPressOnControlBodyIsUnchanged verifies the correction stays out of the way of
// an ordinary press: one whose finger is well inside the button the driver reported it against
// runs that button directly.
func TestTouchGroupPressOnControlBodyIsUnchanged(t *testing.T) {
	h := newGroupHarness(t)
	mid := absTop(h.top).AddXY(4, h.top.Size().Height/2)

	h.top.Tapped(pressAimedAt(mid, h.top))

	if h.topTaps != 1 || h.lowerTaps != 0 {
		t.Errorf("press in the top button's body ran top=%d lower=%d, want top=1 lower=0",
			h.topTaps, h.lowerTaps)
	}
}

// TestTouchGroupIgnoresPressBelowAControl verifies a grouped button answers only to the screen
// area it occupies: a press below it, with no control under the finger, runs nothing. The
// driver hands such a press to the button (it hit-tests above the finger), so without the
// check a button could be operated from the empty space beneath it.
func TestTouchGroupIgnoresPressBelowAControl(t *testing.T) {
	h := newGroupHarness(t)
	below := absTop(h.bottom).AddXY(4, h.bottom.Size().Height+2)

	h.bottom.Tapped(pressAimedAt(below, h.bottom))

	if h.topTaps != 0 || h.lowerTaps != 0 {
		t.Errorf("press below the lower button ran top=%d lower=%d, want both 0",
			h.topTaps, h.lowerTaps)
	}
}

// TestTouchGroupPressOnBottomEdgeRuns verifies the boundary the check above turns on: a finger
// still inside the button, however close to its bottom edge, runs it.
func TestTouchGroupPressOnBottomEdgeRuns(t *testing.T) {
	h := newGroupHarness(t)
	edge := absTop(h.bottom).AddXY(4, h.bottom.Size().Height-1)

	h.bottom.Tapped(pressAimedAt(edge, h.bottom))

	if h.lowerTaps != 1 || h.topTaps != 0 {
		t.Errorf("press on the lower button's bottom edge ran top=%d lower=%d, want top=0 lower=1",
			h.topTaps, h.lowerTaps)
	}
}

// TestTouchGroupWithoutGroupActsItself verifies a button that is not in a group is unaffected:
// it acts on every press it is given, since it has nowhere to hand one back to.
func TestTouchGroupWithoutGroupActsItself(t *testing.T) {
	_ = test.NewApp()
	onTouchDevice(t)
	taps := 0
	b := newTouchButton("Solo", func() { taps++ })
	w := test.NewWindow(b)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 300))

	// A press the driver reports against this button's bottom edge.
	b.Tapped(&fyne.PointEvent{Position: fyne.NewPos(4, b.Size().Height-1)})

	if taps != 1 {
		t.Errorf("ungrouped button ran its action %d times, want 1", taps)
	}
}

// TestTouchGroupDoesNotMoveItsContent verifies the group is invisible to layout: the controls
// inside it sit at the same heights, and are the same size, as the identical controls laid out
// beside them without a group. The group takes its touch inset from the space above itself,
// never from the layout.
func TestTouchGroupDoesNotMoveItsContent(t *testing.T) {
	_ = test.NewApp()
	onTouchDevice(t)

	plainTop := newTouchButton("Top", nil)
	plainLower := newTouchButton("Bottom", nil)
	groupedTop := newTouchButton("Top", nil)
	groupedLower := newTouchButton("Bottom", nil)
	grouped := newTouchGroup(container.NewVBox(groupedTop, groupedLower), groupedTop, groupedLower)

	// Both columns in one canvas: two windows cannot be compared, as the driver resolves
	// absolute positions only for the current one.
	w := test.NewWindow(container.NewGridWithColumns(2,
		container.NewVBox(plainTop, plainLower), grouped))
	defer w.Close()
	w.Resize(fyne.NewSize(400, 300))

	for _, c := range []struct {
		name           string
		plain, grouped fyne.CanvasObject
	}{
		{"top button", plainTop, groupedTop},
		{"lower button", plainLower, groupedLower},
	} {
		if got, want := absTop(c.grouped).Y, absTop(c.plain).Y; got != want {
			t.Errorf("%s sits at y=%.1f inside a group, want y=%.1f as laid out without one",
				c.name, got, want)
		}
		if got, want := c.grouped.Size().Height, c.plain.Size().Height; got != want {
			t.Errorf("%s is %.1f high inside a group, want %.1f as laid out without one",
				c.name, got, want)
		}
	}
}

// TestTouchGroupInertOnDesktop verifies the group changes nothing for a mouse: it claims no
// extra area, so it cannot swallow a click meant for whatever is above it, and a click that
// reaches it is not replayed anywhere.
func TestTouchGroupInertOnDesktop(t *testing.T) {
	_ = test.NewApp()
	if deviceIsMobile() {
		t.Fatal("setup: the test device reports mobile, so the desktop path cannot be checked")
	}
	taps := 0
	b := newTouchButton("Top", func() { taps++ })
	g := newTouchGroup(container.NewVBox(b), b)
	w := test.NewWindow(g)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 300))

	if got, want := absTop(g), absTop(b); got != want {
		t.Errorf("group sits at %v, want %v: it must claim no area above its content on desktop", got, want)
	}
	g.Tapped(&fyne.PointEvent{AbsolutePosition: absTop(b).AddXY(4, 2)})
	if taps != 0 {
		t.Errorf("a click on the group replayed onto a control %d times, want 0", taps)
	}
}
