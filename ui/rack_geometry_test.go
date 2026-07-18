package ui

import (
	"testing"

	"fyne.io/fyne/v2"

	"tilewords/engine"
)

// TestRackSlotAtRel_ToleratesVerticalDrift verifies a reorder drop is resolved to the
// correct slot even when the release drifts above or below the rack row (as real finger
// and mouse drags do), and is rejected only when it drifts well beyond the rack.
func TestRackSlotAtRel_ToleratesVerticalDrift(t *testing.T) {
	size := fyne.NewSize(700, 100)
	slot, offX := rackGeometry(size.Width, size.Height, engine.MaxRackSize)
	stride := slot + rackGapPx
	x := offX + stride*3 + slot/2 // horizontal centre of slot 3

	if idx, ok := rackSlotAtRel(fyne.NewPos(x, size.Height/2), size); !ok || idx != 3 {
		t.Fatalf("in-band: idx=%d ok=%v, want 3,true", idx, ok)
	}
	if idx, ok := rackSlotAtRel(fyne.NewPos(x, -size.Height/2), size); !ok || idx != 3 {
		t.Errorf("drift above the rack: idx=%d ok=%v, want 3,true", idx, ok)
	}
	if idx, ok := rackSlotAtRel(fyne.NewPos(x, size.Height*3/2), size); !ok || idx != 3 {
		t.Errorf("drift below the rack: idx=%d ok=%v, want 3,true", idx, ok)
	}
	if _, ok := rackSlotAtRel(fyne.NewPos(x, -size.Height*2), size); ok {
		t.Error("a release far above the rack should not resolve to a slot")
	}
}
