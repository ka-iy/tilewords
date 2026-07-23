package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// TestTabPanelCopyActiveTab verifies the Copy button copies the active tab's text and that
// switching tabs changes which text is copied.
func TestTabPanelCopyActiveTab(t *testing.T) {
	app := test.NewApp()

	copied := 0
	p := newTabPanel(
		func() { copied++ },
		tabItem{title: "A", content: widget.NewLabel("a"), copyText: func() string { return "alpha" }},
		tabItem{title: "B", content: widget.NewLabel("b"), copyText: func() string { return "bravo" }},
	)

	p.doCopy()
	if got := app.Clipboard().Content(); got != "alpha" {
		t.Errorf("copy of tab A = %q, want %q", got, "alpha")
	}
	if copied != 1 {
		t.Errorf("onCopied calls = %d, want 1", copied)
	}

	p.selectTab(1)
	p.doCopy()
	if got := app.Clipboard().Content(); got != "bravo" {
		t.Errorf("copy of tab B = %q, want %q", got, "bravo")
	}
	if copied != 2 {
		t.Errorf("onCopied calls = %d, want 2", copied)
	}
}

// TestTabPanelCopyEmptyIsNoOp verifies the Copy button does nothing (no clipboard write, no
// feedback) when the active tab's text is empty.
func TestTabPanelCopyEmptyIsNoOp(t *testing.T) {
	app := test.NewApp()
	app.Clipboard().SetContent("sentinel")

	copied := 0
	p := newTabPanel(
		func() { copied++ },
		tabItem{title: "A", content: widget.NewLabel("a"), copyText: func() string { return "" }},
	)
	p.doCopy()

	if got := app.Clipboard().Content(); got != "sentinel" {
		t.Errorf("clipboard = %q, want it untouched (%q)", got, "sentinel")
	}
	if copied != 0 {
		t.Errorf("onCopied calls = %d, want 0", copied)
	}
}
