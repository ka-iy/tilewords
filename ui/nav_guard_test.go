// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import "testing"

// TestScreenToken_InvalidatedByNavigation verifies that leaving a screen invalidates the token
// an asynchronous load captured. Without this, a dictionary load that finishes after the
// player has navigated away installs a game screen over whatever they moved to — including
// the game whose save they just deleted.
func TestScreenToken_InvalidatedByNavigation(t *testing.T) {
	a := &App{}

	token := a.screenToken()
	if !a.screenIsCurrent(token) {
		t.Fatal("a freshly captured token should be current")
	}

	a.leaveScreen()
	if a.screenIsCurrent(token) {
		t.Error("token still current after leaving the screen; a stale load result would be applied")
	}

	// A token taken after the transition is current again, and each further transition
	// invalidates it in turn.
	token = a.screenToken()
	if !a.screenIsCurrent(token) {
		t.Fatal("a token captured after the transition should be current")
	}
	a.leaveScreen()
	a.leaveScreen()
	if a.screenIsCurrent(token) {
		t.Error("token survived two transitions")
	}
}

// TestReportOnCurrentScreen_AttachedTree verifies that when the widget tree that started a load
// is still installed, the callback updates it directly — which is what preserves whatever the
// player had entered on the screen.
func TestReportOnCurrentScreen_AttachedTree(t *testing.T) {
	a := &App{}
	gen := a.uiGen

	called := false
	a.reportOnCurrentScreen(gen, "could not read the word list", func() { called = true })

	if !called {
		t.Error("callback not invoked while its widget tree was still current")
	}
	if a.screenMsg != "" {
		t.Errorf("screenMsg = %q, want empty: the message was delivered directly", a.screenMsg)
	}
}

// TestReportOnCurrentScreen_RebuiltTree verifies that a load result arriving after its screen
// was rebuilt is handed to the next build instead of written into detached widgets, where the
// player would never see it. A theme variant settling mid-load is the usual cause, and it
// rebuilds the same screen, so the navigation counter cannot detect it.
func TestReportOnCurrentScreen_RebuiltTree(t *testing.T) {
	a := &App{}
	rebuilt := 0
	a.redraw = func() { rebuilt++ }

	gen := a.uiGen
	a.redrawNow() // a theme change rebuilds the screen while the load is in flight

	called := false
	a.reportOnCurrentScreen(gen, "could not read the word list", func() { called = true })

	if called {
		t.Error("callback wrote to the detached widget tree; the message would be invisible")
	}
	if a.screenMsg != "could not read the word list" {
		t.Errorf("screenMsg = %q, want the message queued for the next build", a.screenMsg)
	}
	if rebuilt != 2 {
		t.Errorf("rebuilt %d times, want 2 (the theme change, then the one showing the message)", rebuilt)
	}
}

// TestTakeScreenMsg_ShownOnce verifies a queued message is consumed by the build that shows it,
// so it does not reappear on every later rebuild.
func TestTakeScreenMsg_ShownOnce(t *testing.T) {
	a := &App{screenMsg: "save file is corrupt"}
	if got := a.takeScreenMsg(); got != "save file is corrupt" {
		t.Fatalf("takeScreenMsg = %q, want the queued message", got)
	}
	if got := a.takeScreenMsg(); got != "" {
		t.Errorf("takeScreenMsg = %q on the second call, want empty", got)
	}
}

// TestLeaveScreen_DiscardsPendingMessage verifies a message queued for a screen the player then
// leaves is dropped rather than surfacing on an unrelated screen.
func TestLeaveScreen_DiscardsPendingMessage(t *testing.T) {
	a := &App{screenMsg: "could not read the word list"}
	a.leaveScreen()
	if a.screenMsg != "" {
		t.Errorf("screenMsg = %q after leaving the screen, want empty", a.screenMsg)
	}
}

// TestLeaveScreen_StopsOutgoingGameWorker verifies the departing game screen is marked
// abandoned and has its definitions worker stopped. A superseded screen that keeps its worker
// alive pins the whole widget tree for the process lifetime and goes on delivering lookups.
func TestLeaveScreen_StopsOutgoingGameWorker(t *testing.T) {
	a := &App{}
	gs := &gameScreen{app: a, defsWordCh: make(chan defsRequest, 1)}
	a.game = gs

	a.leaveScreen()

	if !gs.abandoned {
		t.Error("outgoing game screen was not marked abandoned")
	}
	if !gs.defsClosed {
		t.Error("outgoing game screen's definitions worker was not stopped")
	}
	if a.game != nil {
		t.Error("App still references the outgoing game screen")
	}
	if _, open := <-gs.defsWordCh; open {
		t.Error("definitions channel is still open")
	}

	// Leaving again with no installed game must be harmless (it happens on every menu
	// transition), in particular it must not close the channel twice.
	a.leaveScreen()
}
