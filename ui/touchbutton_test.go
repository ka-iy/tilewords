// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/test"
)

// assertTouchable fails if obj does not implement mobile.Touchable. The obj argument is
// typed as the interface fyne.CanvasObject rather than the concrete widget type so the
// check is a real runtime assertion: reverting a control back to a plain widget.Button
// (which is not Touchable) makes it fail instead of being erased by the compiler.
func assertTouchable(t *testing.T, name string, obj fyne.CanvasObject) {
	t.Helper()
	if obj == nil {
		t.Fatalf("%s: widget was not built", name)
	}
	if _, ok := obj.(mobile.Touchable); !ok {
		t.Errorf("%s is not mobile.Touchable; a tap on it inside a Scroll would be stolen "+
			"by the scroll's pan on mobile and the button would need multiple presses", name)
	}
}

// TestControlButtonsAreTouchable guards the mobile tap-in-scroll fix for the game screen:
// every control button lives inside the phone layout's vertical Scroll, so each must be
// mobile.Touchable or Fyne's mobile driver hands a slightly-moved press to the enclosing
// Scroll and drops the tap. See touchButton for the mechanism.
func TestControlButtonsAreTouchable(t *testing.T) {
	gs := newRackHarness(t)
	assertTouchable(t, "play", gs.playBtn)
	assertTouchable(t, "exchange", gs.exchBtn)
	assertTouchable(t, "pass", gs.passBtn)
	assertTouchable(t, "undo", gs.undoBtn)
	assertTouchable(t, "save", gs.saveBtn)
	assertTouchable(t, "toggle", gs.toggleBtn)
	assertTouchable(t, "menu", gs.menuBtn)
	assertTouchable(t, "shuffle", gs.shuffleBtn)
	assertTouchable(t, "recall", gs.recallBtn)
}

// TestTabButtonsAreTouchable guards the same fix for the move-history / definitions tab
// panel, whose selector and copy buttons sit inside the same phone Scroll.
func TestTabButtonsAreTouchable(t *testing.T) {
	_ = test.NewApp()
	p := newTabPanel(nil,
		tabItem{title: "One", content: container.NewWithoutLayout()},
		tabItem{title: "Two", content: container.NewWithoutLayout()},
	)
	if len(p.buttons) == 0 {
		t.Fatal("tab panel built no buttons")
	}
	for _, b := range p.buttons {
		assertTouchable(t, "tab button", b)
	}
}

// TestBevelButtonIsTouchable guards the fix for the DOS-style Info buttons on the setup
// screen, which also sit inside that screen's vertical Scroll.
func TestBevelButtonIsTouchable(t *testing.T) {
	assertTouchable(t, "bevel Info button", newBevelButton("Info", func() {}))
}

// TestSetupControlsAreTouchable guards the fix for the setup screen's non-button controls:
// the dictionary and game-mode radio rows and the notation checkbox all sit inside that
// screen's vertical Scroll, so each must be mobile.Touchable to tap reliably on mobile.
func TestSetupControlsAreTouchable(t *testing.T) {
	_ = test.NewApp()
	assertTouchable(t, "notation check", newTouchCheck("Show notation", nil))
	tr := newTouchRadio([]string{"A", "B"}, nil)
	for _, b := range tr.buttons {
		assertTouchable(t, "radio row", b)
	}
}
