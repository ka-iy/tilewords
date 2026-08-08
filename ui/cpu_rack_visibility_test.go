// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand/v2"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// newLayoutHarness returns a game screen whose responsive content has been sized to size,
// along with that content so the caller can resize it again.
func newLayoutHarness(t *testing.T, size fyne.Size) (*gameScreen, fyne.CanvasObject) {
	t.Helper()
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
	gs := newGameScreen(nil, state, dict)
	content := gs.build()
	content.Resize(size)
	return gs, content
}

// phoneSize and desktopSize are viewports either side of the narrow/wide threshold.
var (
	phoneSize   = fyne.NewSize(400, 800)
	desktopSize = fyne.NewSize(960, 760)
)

// The narrow layout leaves the CPU rack out until the player asks for it, and puts it away
// again on a second press.
func TestNarrowLayout_AIRackHiddenUntilRequested(t *testing.T) {
	gs, _ := newLayoutHarness(t, phoneSize)
	if gs.page == nil {
		t.Fatal("setup: expected the narrow arrangement at a phone size")
	}

	if gs.cpuRackBox.Visible() {
		t.Error("the CPU rack is shown in the narrow layout before it was asked for")
	}

	gs.toggleCPURack()
	if !gs.cpuRackBox.Visible() {
		t.Error("the CPU rack is still hidden after the CPU Rack button was pressed")
	}

	gs.toggleCPURack()
	if gs.cpuRackBox.Visible() {
		t.Error("the CPU rack is still shown after a second press of the CPU Rack button")
	}
}

// The height the CPU rack would occupy goes to the move-history/definitions pane while the
// rack is hidden: the pane starts directly under the action buttons, with no blank band and
// no gap left behind.
func TestNarrowLayout_HistoryTakesTheCPURackSpace(t *testing.T) {
	gs, _ := newLayoutHarness(t, phoneSize)
	if gs.page == nil {
		t.Fatal("setup: expected the narrow arrangement at a phone size")
	}
	column := gs.page.column
	width := phoneSize.Width

	// Locate the CPU rack block and the pane below it within the column.
	cpuIdx, histIdx := -1, -1
	for i, o := range column.Objects {
		switch o {
		case fyne.CanvasObject(gs.cpuRackBox):
			cpuIdx = i
		case fyne.CanvasObject(gs.page.histWrap):
			histIdx = i
		}
	}
	if cpuIdx < 0 || histIdx != cpuIdx+1 {
		t.Fatalf("setup: expected the history pane directly after the CPU rack in the column (cpu=%d, hist=%d)",
			cpuIdx, histIdx)
	}
	// The child above the CPU rack is the action buttons, which the pane must sit under
	// while the rack is hidden.
	controls := column.Objects[cpuIdx-1]

	layout := func() {
		column.Resize(fyne.NewSize(width, column.MinSize().Height))
	}

	// Hidden: the pane follows the action buttons by exactly one gap.
	layout()
	hiddenNonHist := gs.page.nonHistoryHeight(width)
	wantY := controls.Position().Y + controls.Size().Height + phoneColGap
	if got := gs.page.histWrap.Position().Y; got != wantY {
		t.Errorf("with the CPU rack hidden, the history pane starts at y=%.1f, want %.1f (directly under the action buttons)",
			got, wantY)
	}

	// Shown: the pane is pushed down by the rack's full height plus its gap, and exactly
	// that much less of the viewport is left for it.
	gs.toggleCPURack()
	layout()
	rackH := gs.cpuRackBox.MinSize().Height
	if rackH <= 0 {
		t.Fatal("setup: the CPU rack block has no height, so there is no space to reclaim")
	}
	if got, want := gs.page.histWrap.Position().Y, wantY+rackH+phoneColGap; got != want {
		t.Errorf("with the CPU rack shown, the history pane starts at y=%.1f, want %.1f", got, want)
	}
	if got, want := gs.page.nonHistoryHeight(width), hiddenNonHist+rackH+phoneColGap; got != want {
		t.Errorf("with the CPU rack shown, the space left for the history pane shrank by %.1f, want %.1f",
			got-hiddenNonHist, want-hiddenNonHist)
	}
}

// The wide layout always shows the CPU rack: it has room for it beside the board, and the
// history pane is the split's other side, so hiding it would gain that pane nothing. Both
// arrangements re-parent the same widget, so this has to survive arriving from a narrow
// window that had the rack hidden.
func TestWideLayout_AIRackAlwaysVisible(t *testing.T) {
	gs, content := newLayoutHarness(t, desktopSize)
	if gs.page != nil {
		t.Fatal("setup: expected the wide arrangement at a desktop size")
	}
	if !gs.cpuRackBox.Visible() {
		t.Error("the CPU rack is hidden in the wide layout")
	}

	// Toggling face-up/face-down must not remove the block in the wide layout.
	gs.toggleCPURack()
	if !gs.cpuRackBox.Visible() {
		t.Error("the CPU rack disappeared from the wide layout when the CPU Rack button was pressed")
	}
	gs.toggleCPURack()
	if !gs.cpuRackBox.Visible() {
		t.Error("the CPU rack disappeared from the wide layout on a second press")
	}

	// Hide it in the narrow layout, then widen: the wide layout must show it again.
	content.Resize(phoneSize)
	if gs.page == nil {
		t.Fatal("setup: expected the narrow arrangement at a phone size")
	}
	if gs.cpuRackBox.Visible() {
		t.Fatal("setup: the CPU rack should be hidden in the narrow layout")
	}
	content.Resize(desktopSize)
	if !gs.cpuRackBox.Visible() {
		t.Error("the CPU rack stayed hidden after widening from a narrow window that had hidden it")
	}
}

// The button is named for what pressing it would do now: while the rack is face down each
// arrangement names it for how it reveals the rack, and once the rack is face up both read
// "Hide CPU".
func TestCPURackButtonLabelTracksRackState(t *testing.T) {
	for _, tc := range []struct {
		name       string
		size       fyne.Size
		whenHidden string
	}{
		{"narrow", phoneSize, cpuRackBtnLabelNarrow},
		{"wide", desktopSize, cpuRackBtnLabelWide},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs, _ := newLayoutHarness(t, tc.size)
			if got := gs.toggleBtn.Text; got != tc.whenHidden {
				t.Errorf("with the rack face down, button label = %q, want %q", got, tc.whenHidden)
			}

			gs.toggleCPURack()
			if !gs.showCPURack {
				t.Fatal("setup: the rack should be face up after one press")
			}
			if got := gs.toggleBtn.Text; got != cpuRackBtnLabelShown {
				t.Errorf("with the rack face up, button label = %q, want %q", got, cpuRackBtnLabelShown)
			}

			gs.toggleCPURack()
			if got := gs.toggleBtn.Text; got != tc.whenHidden {
				t.Errorf("after hiding the rack again, button label = %q, want %q", got, tc.whenHidden)
			}
		})
	}
}

// Both arrangements share the one button, so crossing the wide/narrow threshold has to
// rename it — otherwise it keeps whichever name the arrangement it was built in gave it.
// Once the rack is face up the name no longer depends on the arrangement, so it survives
// the crossing unchanged.
func TestCPURackButtonLabelFollowsArrangementChange(t *testing.T) {
	gs, content := newLayoutHarness(t, phoneSize)
	if got := gs.toggleBtn.Text; got != cpuRackBtnLabelNarrow {
		t.Fatalf("setup: narrow button label = %q, want %q", got, cpuRackBtnLabelNarrow)
	}

	content.Resize(desktopSize)
	if got := gs.toggleBtn.Text; got != cpuRackBtnLabelWide {
		t.Errorf("after widening, button label = %q, want %q", got, cpuRackBtnLabelWide)
	}

	content.Resize(phoneSize)
	if got := gs.toggleBtn.Text; got != cpuRackBtnLabelNarrow {
		t.Errorf("after narrowing again, button label = %q, want %q", got, cpuRackBtnLabelNarrow)
	}

	// Face up: both arrangements read "Hide CPU", so the crossing must not rename it.
	gs.toggleCPURack()
	content.Resize(desktopSize)
	if got := gs.toggleBtn.Text; got != cpuRackBtnLabelShown {
		t.Errorf("after widening with the rack face up, button label = %q, want %q",
			got, cpuRackBtnLabelShown)
	}
}
