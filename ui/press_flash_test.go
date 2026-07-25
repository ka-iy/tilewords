// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"
)

// TestFlashPress_ChangesThenRestores verifies a tapped control button switches to the
// pressed-flash colour immediately and reverts to its resting importance after the flash.
func TestFlashPress_ChangesThenRestores(t *testing.T) {
	gs := newRackHarness(t)

	// playBtn rests at HighImportance; the flash must be a different colour and revert.
	if got := gs.playBtn.Importance; got != widget.HighImportance {
		t.Fatalf("precondition: playBtn importance = %v, want HighImportance", got)
	}

	// Capture the scheduled revert rather than let it fire on a background timer, so it
	// runs synchronously on this goroutine. The real timer's fyne.Do would otherwise be
	// executed inline on the timer goroutine by the test driver, racing the reads below.
	var revert func()
	gs.afterFunc = func(_ time.Duration, f func()) *time.Timer {
		revert = f
		return nil
	}

	gs.flashPress(gs.playBtn)
	if got := gs.playBtn.Importance; got != pressFlashImportance {
		t.Fatalf("during flash: importance = %v, want %v", got, pressFlashImportance)
	}
	if revert == nil {
		t.Fatal("flashPress did not schedule a revert")
	}

	revert() // run the scheduled revert on this goroutine
	if got := gs.playBtn.Importance; got != widget.HighImportance {
		t.Fatalf("after flash: importance = %v, want it reverted to HighImportance", got)
	}
}
