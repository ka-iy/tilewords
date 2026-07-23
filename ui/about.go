// Package ui is documented in doc.go.
package ui

import (
	_ "embed"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// aboutText is the About dialog's content. It is generated at build time from the
// top-level ABOUT.txt and LEXICON.txt (see the Makefile 'ui/about.txt' target), so
// those files remain the single source of truth. A //go:embed directive cannot
// reach a parent directory, which is why the generated copy lives in this package.
//
//go:embed about.txt
var aboutText string

// copyResetDelay is how long the copy button shows its confirmation label before
// reverting, so a tap gives visible feedback without lingering.
const copyResetDelay = 1500 * time.Millisecond

// showAbout pops a scrollable dialog with the app's About and Lexicon text. The body
// is selectable and a button copies the whole text to the clipboard, so a user can
// reach the source URLs — a plain label cannot open a link, and the URLs are long
// enough to be awkward to select by hand on a phone.
func (a *App) showAbout() {
	body := widget.NewLabel(aboutText)
	body.Wrapping = fyne.TextWrapWord
	body.Selectable = true

	scroll := container.NewVScroll(body)
	scroll.SetMinSize(fyne.NewSize(300, 480))

	const copyLabel = "Copy to clipboard"
	var copyBtn *touchButton
	// showCopied flashes the button label as confirmation, reverting on the UI
	// goroutine after a short delay. Both the button and a long-press copy use it.
	showCopied := func() {
		copyBtn.SetText("Copied to clipboard")
		time.AfterFunc(copyResetDelay, func() {
			fyne.Do(func() { copyBtn.SetText(copyLabel) })
		})
	}
	copyBtn = newTouchButtonWithIcon(copyLabel, theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Clipboard().SetContent(aboutText)
		showCopied()
	})

	// On touch, a finger drag pans the scroll instead of selecting text, and a long
	// press copies the whole panel; double/triple tap still selects a word/line. This
	// mirrors the move-history panel (see dragscroll.go). On desktop it is a no-op —
	// the wheel scrolls, click-drag selects, and the button copies.
	enableTouchScroll(scroll, func() string { return aboutText }, showCopied)

	content := container.NewBorder(copyBtn, nil, nil, nil, scroll)

	d := dialog.NewCustom("About TileWords", "Close", content, a.win)
	d.Resize(fyne.NewSize(360, 620))
	d.Show()
}
