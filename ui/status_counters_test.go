// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// newTestCounters builds a statusCounters with representative label text.
func newTestCounters() *statusCounters {
	return newStatusCounters(
		widget.NewLabel("▶ You: 0"),
		widget.NewLabel("CPU: 0"),
		widget.NewLabel("Bag: 86"),
		widget.NewLabel("Move: 0"),
		widget.NewLabel("CPU Lv: 5"),
	)
}

// TestStatusCounters_WrapsOnlyWhenTooNarrow verifies the counters stay on one row while the
// width fits the single row, and wrap to two rows once it does not.
func TestStatusCounters_WrapsOnlyWhenTooNarrow(t *testing.T) {
	_ = test.NewApp()
	sc := newTestCounters()
	w := test.NewWindow(sc)
	defer w.Close()

	single := sc.singleRowWidth()

	// Comfortably wider than the single row: one row.
	w.Resize(fyne.NewSize(single+40, 300))
	if sc.twoRow {
		t.Errorf("width %.0f fits the single row (%.0f) but wrapped to two rows", single+40, single)
	}

	// Narrower than the single row needs: two rows.
	w.Resize(fyne.NewSize(single-40, 300))
	if !sc.twoRow {
		t.Errorf("width %.0f cannot fit the single row (%.0f) but stayed on one row", single-40, single)
	}

	// Back to wide: returns to a single row.
	w.Resize(fyne.NewSize(single+40, 300))
	if sc.twoRow {
		t.Errorf("width %.0f fits the single row but did not return from two rows", single+40)
	}
}

// TestStatusCounters_WrapReservesHeight verifies that when the counters wrap, the widget's
// reported height grows so a sibling below it (the status message) is not overlapped.
//
// The wrap decision is made in Resize, one layout pass after the parent positions the
// sibling from the counters' (still single-row) minimum height; the widget then reports its
// new minimum and the following layout pass — which the render loop performs before
// painting — repositions the sibling. The second Resize below stands in for that pass.
func TestStatusCounters_WrapReservesHeight(t *testing.T) {
	_ = test.NewApp()
	sc := newTestCounters()
	below := widget.NewLabel("status message")
	col := container.NewVBox(sc, below)
	w := test.NewWindow(col)
	defer w.Close()

	narrow := sc.singleRowWidth() - 40
	w.Resize(fyne.NewSize(narrow, 300))
	if !sc.twoRow {
		t.Fatal("setup: counters did not wrap at a narrow width")
	}
	w.Resize(fyne.NewSize(narrow-1, 300)) // settle: the layout pass after the wrap switch

	// The sibling must sit at or below the counters' full (two-row) height.
	if y := below.Position().Y; y < sc.MinSize().Height-0.5 {
		t.Errorf("sibling overlaps wrapped counters: sibling Y=%.1f, counters height=%.1f", y, sc.MinSize().Height)
	}
}
