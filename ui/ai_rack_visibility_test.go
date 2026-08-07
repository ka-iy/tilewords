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

// The narrow layout leaves the AI rack out until the player asks for it, and puts it away
// again on a second press.
func TestNarrowLayout_AIRackHiddenUntilRequested(t *testing.T) {
	gs, _ := newLayoutHarness(t, phoneSize)
	if gs.page == nil {
		t.Fatal("setup: expected the narrow arrangement at a phone size")
	}

	if gs.aiRackBox.Visible() {
		t.Error("the AI rack is shown in the narrow layout before it was asked for")
	}

	gs.toggleAIRack()
	if !gs.aiRackBox.Visible() {
		t.Error("the AI rack is still hidden after the AI Rack button was pressed")
	}

	gs.toggleAIRack()
	if gs.aiRackBox.Visible() {
		t.Error("the AI rack is still shown after a second press of the AI Rack button")
	}
}

// The height the AI rack would occupy goes to the move-history/definitions pane while the
// rack is hidden: the pane starts directly under the action buttons, with no blank band and
// no gap left behind.
func TestNarrowLayout_HistoryTakesTheAIRackSpace(t *testing.T) {
	gs, _ := newLayoutHarness(t, phoneSize)
	if gs.page == nil {
		t.Fatal("setup: expected the narrow arrangement at a phone size")
	}
	column := gs.page.column
	width := phoneSize.Width

	// Locate the AI rack block and the pane below it within the column.
	aiIdx, histIdx := -1, -1
	for i, o := range column.Objects {
		switch o {
		case fyne.CanvasObject(gs.aiRackBox):
			aiIdx = i
		case fyne.CanvasObject(gs.page.histWrap):
			histIdx = i
		}
	}
	if aiIdx < 0 || histIdx != aiIdx+1 {
		t.Fatalf("setup: expected the history pane directly after the AI rack in the column (ai=%d, hist=%d)",
			aiIdx, histIdx)
	}
	// The child above the AI rack is the action buttons, which the pane must sit under
	// while the rack is hidden.
	controls := column.Objects[aiIdx-1]

	layout := func() {
		column.Resize(fyne.NewSize(width, column.MinSize().Height))
	}

	// Hidden: the pane follows the action buttons by exactly one gap.
	layout()
	hiddenNonHist := gs.page.nonHistoryHeight(width)
	wantY := controls.Position().Y + controls.Size().Height + phoneColGap
	if got := gs.page.histWrap.Position().Y; got != wantY {
		t.Errorf("with the AI rack hidden, the history pane starts at y=%.1f, want %.1f (directly under the action buttons)",
			got, wantY)
	}

	// Shown: the pane is pushed down by the rack's full height plus its gap, and exactly
	// that much less of the viewport is left for it.
	gs.toggleAIRack()
	layout()
	rackH := gs.aiRackBox.MinSize().Height
	if rackH <= 0 {
		t.Fatal("setup: the AI rack block has no height, so there is no space to reclaim")
	}
	if got, want := gs.page.histWrap.Position().Y, wantY+rackH+phoneColGap; got != want {
		t.Errorf("with the AI rack shown, the history pane starts at y=%.1f, want %.1f", got, want)
	}
	if got, want := gs.page.nonHistoryHeight(width), hiddenNonHist+rackH+phoneColGap; got != want {
		t.Errorf("with the AI rack shown, the space left for the history pane shrank by %.1f, want %.1f",
			got-hiddenNonHist, want-hiddenNonHist)
	}
}

// The wide layout always shows the AI rack: it has room for it beside the board, and the
// history pane is the split's other side, so hiding it would gain that pane nothing. Both
// arrangements re-parent the same widget, so this has to survive arriving from a narrow
// window that had the rack hidden.
func TestWideLayout_AIRackAlwaysVisible(t *testing.T) {
	gs, content := newLayoutHarness(t, desktopSize)
	if gs.page != nil {
		t.Fatal("setup: expected the wide arrangement at a desktop size")
	}
	if !gs.aiRackBox.Visible() {
		t.Error("the AI rack is hidden in the wide layout")
	}

	// Toggling face-up/face-down must not remove the block in the wide layout.
	gs.toggleAIRack()
	if !gs.aiRackBox.Visible() {
		t.Error("the AI rack disappeared from the wide layout when the AI Rack button was pressed")
	}
	gs.toggleAIRack()
	if !gs.aiRackBox.Visible() {
		t.Error("the AI rack disappeared from the wide layout on a second press")
	}

	// Hide it in the narrow layout, then widen: the wide layout must show it again.
	content.Resize(phoneSize)
	if gs.page == nil {
		t.Fatal("setup: expected the narrow arrangement at a phone size")
	}
	if gs.aiRackBox.Visible() {
		t.Fatal("setup: the AI rack should be hidden in the narrow layout")
	}
	content.Resize(desktopSize)
	if !gs.aiRackBox.Visible() {
		t.Error("the AI rack stayed hidden after widening from a narrow window that had hidden it")
	}
}

// The button is named for what pressing it would do now: while the rack is face down each
// arrangement names it for how it reveals the rack, and once the rack is face up both read
// "Hide AI".
func TestAIRackButtonLabelTracksRackState(t *testing.T) {
	for _, tc := range []struct {
		name       string
		size       fyne.Size
		whenHidden string
	}{
		{"narrow", phoneSize, aiRackBtnLabelNarrow},
		{"wide", desktopSize, aiRackBtnLabelWide},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs, _ := newLayoutHarness(t, tc.size)
			if got := gs.toggleBtn.Text; got != tc.whenHidden {
				t.Errorf("with the rack face down, button label = %q, want %q", got, tc.whenHidden)
			}

			gs.toggleAIRack()
			if !gs.showAIRack {
				t.Fatal("setup: the rack should be face up after one press")
			}
			if got := gs.toggleBtn.Text; got != aiRackBtnLabelShown {
				t.Errorf("with the rack face up, button label = %q, want %q", got, aiRackBtnLabelShown)
			}

			gs.toggleAIRack()
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
func TestAIRackButtonLabelFollowsArrangementChange(t *testing.T) {
	gs, content := newLayoutHarness(t, phoneSize)
	if got := gs.toggleBtn.Text; got != aiRackBtnLabelNarrow {
		t.Fatalf("setup: narrow button label = %q, want %q", got, aiRackBtnLabelNarrow)
	}

	content.Resize(desktopSize)
	if got := gs.toggleBtn.Text; got != aiRackBtnLabelWide {
		t.Errorf("after widening, button label = %q, want %q", got, aiRackBtnLabelWide)
	}

	content.Resize(phoneSize)
	if got := gs.toggleBtn.Text; got != aiRackBtnLabelNarrow {
		t.Errorf("after narrowing again, button label = %q, want %q", got, aiRackBtnLabelNarrow)
	}

	// Face up: both arrangements read "Hide AI", so the crossing must not rename it.
	gs.toggleAIRack()
	content.Resize(desktopSize)
	if got := gs.toggleBtn.Text; got != aiRackBtnLabelShown {
		t.Errorf("after widening with the rack face up, button label = %q, want %q",
			got, aiRackBtnLabelShown)
	}
}
