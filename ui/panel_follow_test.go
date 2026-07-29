// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"tilewords/engine"
)

// checkScrolledToEnd fails when s is not showing the end of its content.
func checkScrolledToEnd(t *testing.T, s *container.Scroll, name string) {
	t.Helper()
	contentH := s.Content.MinSize().Height
	viewH := s.Size().Height
	if contentH <= viewH {
		t.Fatalf("setup: %s content (%.0f) is not taller than its viewport (%.0f); "+
			"there is no scrolling to check", name, contentH, viewH)
	}
	if want := contentH - viewH; s.Offset.Y < want-1 {
		t.Errorf("%s is not showing its newest entry: offset.Y=%.1f, want ~%.1f (content %.0f, view %.0f)",
			name, s.Offset.Y, want, contentH, viewH)
	}
}

// longGloss is a definition entry of realistic length: one that wraps onto several lines in a
// narrow panel and fewer in a wide one, which is what makes a rotation move the end of the log.
const longGloss = "CRANE\nnoun - A tall wading bird with long legs, a long neck and a straight bill, of the family Gruidae."

// TestPanelsFollowEndAcrossLayoutChange verifies both panels still show their newest entry
// after the viewport changes shape, as a rotation does.
//
// The rotation is landscape to portrait. A phone uses the stacked column in both orientations
// (the wide split needs a viewport bigger than either), so what changes is the panel's width:
// its word-wrapped text re-wraps onto more lines, and the offset it was left on then points
// into the middle of the log. The other direction needs no help — a panel that grows wider
// holds less content than before, so Scroll.Resize clamps the offset onto the new end by itself.
//
// The definitions panel is checked as well, on the unselected tab: a stack lays out its hidden
// children too, so that panel is resized by the rotation exactly as the visible one is.
func TestPanelsFollowEndAcrossLayoutChange(t *testing.T) {
	gs := newRackHarness(t)
	w := test.NewWindow(gs.build())
	defer w.Close()
	w.Resize(fyne.NewSize(900, 400)) // landscape

	// Enough entries in both panels that each is taller than its viewport.
	for i := 0; i < 50; i++ {
		gs.logCommand("You", &engine.PassCommand{})
	}
	for i := range gs.history {
		gs.appendDefinition(defsEntry{text: longGloss, turn: i})
	}
	checkScrolledToEnd(t, gs.historyScroll, "move history")
	checkScrolledToEnd(t, gs.defsScroll, "definitions")

	// Rotate to portrait: the same arrangement, but much narrower panels.
	w.Resize(fyne.NewSize(420, 780))

	checkScrolledToEnd(t, gs.historyScroll, "move history after rotation")
	checkScrolledToEnd(t, gs.defsScroll, "definitions after rotation")
}

// TestPanelFollowIgnoresRefreshWithoutResize verifies a panel is not pinned to its end: a
// container re-runs its layout on every Refresh, and following the end on those would drag a
// player who has scrolled back to an earlier turn down to the bottom again.
func TestPanelFollowIgnoresRefreshWithoutResize(t *testing.T) {
	_ = test.NewApp()
	scroll := container.NewVScroll(widget.NewLabel(strings.Repeat("a line of history\n", 200)))
	pane := container.New(&endFollowLayout{}, scroll)
	w := test.NewWindow(pane)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 100))

	// The player scrolls back to read an earlier turn.
	scroll.ScrollToTop()
	if scroll.Offset.Y != 0 {
		t.Fatalf("setup: panel did not scroll to the top; offset.Y=%.1f", scroll.Offset.Y)
	}

	pane.Refresh() // re-runs the layout, at the same size

	if scroll.Offset.Y != 0 {
		t.Errorf("a refresh moved the panel from where the player left it: offset.Y=%.1f, want 0",
			scroll.Offset.Y)
	}
}
