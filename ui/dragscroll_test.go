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

// TestDragScrollerLongPressCopiesAll verifies a long press (secondary tap) on the panel
// copies the whole panel text and reports via onCopied.
func TestDragScrollerLongPressCopiesAll(t *testing.T) {
	app := test.NewApp()
	copied := 0
	scroll := container.NewVScroll(widget.NewLabel("x"))
	d := newDragScroller(scroll, func() string { return "full history text" }, func() { copied++ }, nil)

	d.TappedSecondary(&fyne.PointEvent{})

	if got := app.Clipboard().Content(); got != "full history text" {
		t.Errorf("long-press copy = %q, want %q", got, "full history text")
	}
	if copied != 1 {
		t.Errorf("onCopied calls = %d, want 1", copied)
	}
}

// TestDragScrollerLongPressEmptyIsNoOp verifies a long press does nothing when the panel is empty.
func TestDragScrollerLongPressEmptyIsNoOp(t *testing.T) {
	app := test.NewApp()
	app.Clipboard().SetContent("sentinel")
	copied := 0
	scroll := container.NewVScroll(widget.NewLabel("x"))
	d := newDragScroller(scroll, func() string { return "" }, func() { copied++ }, nil)

	d.TappedSecondary(&fyne.PointEvent{})

	if got := app.Clipboard().Content(); got != "sentinel" {
		t.Errorf("clipboard = %q, want it untouched (%q)", got, "sentinel")
	}
	if copied != 0 {
		t.Errorf("onCopied calls = %d, want 0", copied)
	}
}
