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

	gs.flashPress(gs.playBtn)
	if got := gs.playBtn.Importance; got != pressFlashImportance {
		t.Fatalf("during flash: importance = %v, want %v", got, pressFlashImportance)
	}

	// Wait past the flash and let the scheduled revert run.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gs.playBtn.Importance == widget.HighImportance {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("after flash: importance = %v, want it reverted to HighImportance", gs.playBtn.Importance)
}
