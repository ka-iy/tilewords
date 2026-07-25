// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"tilewords/engine"
)

// TestHistoryAutoScroll verifies the move-history panel follows its newest line: after the
// log grows past the panel height, the scroll offset must sit at the bottom so the latest
// move stays visible without manual scrolling.
func TestHistoryAutoScroll(t *testing.T) {
	gs := newRackHarness(t)
	// Render the screen so the history scroll and label have real sizes.
	w := test.NewWindow(gs.build())
	defer w.Close()
	w.Resize(fyne.NewSize(900, 400))

	// Enough entries that the history is taller than its panel.
	for i := 0; i < 50; i++ {
		gs.logCommand("You", &engine.PassCommand{})
	}

	contentH := gs.historyLabel.MinSize().Height
	viewH := gs.historyScroll.Size().Height
	if contentH <= viewH {
		t.Fatalf("setup: history (%.0f) not taller than its panel (%.0f); cannot verify scrolling",
			contentH, viewH)
	}
	wantBottom := contentH - viewH
	if got := gs.historyScroll.Offset.Y; got < wantBottom-1 {
		t.Errorf("history not scrolled to end after logging: offset.Y=%.1f, want ~%.1f (content %.0f, view %.0f)",
			got, wantBottom, contentH, viewH)
	}
}
