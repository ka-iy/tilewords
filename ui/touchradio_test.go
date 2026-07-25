// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

// TestTouchRadioSelection covers the touchRadio selection model that replaces Fyne's
// RadioGroup on the setup screen: SetSelected and taps move the selection and fire onChange
// with the new label, while re-selecting the current option (or an unknown one) is a no-op —
// matching a Required RadioGroup, which cannot be deselected by re-tapping its choice.
func TestTouchRadioSelection(t *testing.T) {
	_ = test.NewApp() // selecting refreshes the row icons, which renders

	var changes []string
	tr := newTouchRadio([]string{"A", "B", "C"}, func(s string) { changes = append(changes, s) })

	if got := tr.Selected(); got != "" {
		t.Fatalf("new radio Selected() = %q, want empty", got)
	}

	tr.SetSelected("B")
	if got := tr.Selected(); got != "B" {
		t.Fatalf("after SetSelected(B): Selected() = %q, want B", got)
	}

	// Tapping option C runs its button's handler, selecting it.
	tr.buttons[2].OnTapped()
	if got := tr.Selected(); got != "C" {
		t.Fatalf("after tapping C: Selected() = %q, want C", got)
	}

	// Re-tapping the current option must not change the selection or fire onChange again.
	tr.buttons[2].OnTapped()
	if got := tr.Selected(); got != "C" {
		t.Fatalf("re-tapping C changed selection to %q", got)
	}

	// An unknown option is ignored.
	tr.SetSelected("Z")
	if got := tr.Selected(); got != "C" {
		t.Fatalf("SetSelected(unknown) changed selection to %q", got)
	}

	want := []string{"B", "C"}
	if len(changes) != len(want) {
		t.Fatalf("onChange fired %v, want %v", changes, want)
	}
	for i := range want {
		if changes[i] != want[i] {
			t.Fatalf("onChange[%d] = %q, want %q", i, changes[i], want[i])
		}
	}
}
