// Package ui is documented in doc.go.
package ui

import (
	_ "embed"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"tilewords/buildinfo"
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

// aboutBannerRuleLen is the number of '=' characters in a section banner rule. It
// matches the width the Makefile 'ui/about.txt' target uses for the embedded sections,
// so a section composed here at runtime lines up with them.
const aboutBannerRuleLen = 30

// aboutSection renders a titled section using the same banner layout as the embedded
// About text (see the Makefile 'ui/about.txt' target): a rule line, the two-space-indented
// title, a rule line, a blank line, the body, then a trailing blank line. It lets the
// runtime-composed Build Info section match the embedded sections visually.
func aboutSection(title, body string) string {
	rule := strings.Repeat("=", aboutBannerRuleLen)
	return rule + "\n  " + title + "\n" + rule + "\n\n" + body + "\n\n"
}

// aboutDialogText returns the full text the About dialog shows: the embedded aboutText,
// led by a BUILD INFO section. Unlike aboutText (embedded at build time), the build
// metadata is injected at link time, so it is composed here at runtime.
//
// A build with no version injected — a plain `go build` rather than one of the Makefile
// targets — leaves it blank, and has no metadata worth reporting, so the section is
// dropped entirely instead of showing empty values.
func aboutDialogText() string {
	if buildinfo.BuildVersion() == "" {
		return aboutText
	}

	return aboutSection("BUILD INFO", strings.Join(buildinfo.BuildInfoAsStringSlice(), "\n")) + aboutText
}

// showAbout pops a scrollable dialog with the app's About and Lexicon text. The body
// is selectable and a button copies the whole text to the clipboard, so a user can
// reach the source URLs — a plain label cannot open a link, and the URLs are long
// enough to be awkward to select by hand on a phone.
func (a *App) showAbout() {
	fullText := aboutDialogText()

	body := widget.NewLabel(fullText)
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
		fyne.CurrentApp().Clipboard().SetContent(fullText)
		showCopied()
	})

	// On touch, a finger drag pans the scroll instead of selecting text, and a long
	// press copies the whole panel; double/triple tap still selects a word/line. This
	// mirrors the move-history panel (see dragscroll.go). On desktop it is a no-op —
	// the wheel scrolls, click-drag selects, and the button copies.
	enableTouchScroll(scroll, func() string { return fullText }, showCopied)

	content := container.NewBorder(copyBtn, nil, nil, nil, scroll)

	d := dialog.NewCustom("About TileWords", "Close", content, a.win)
	d.Resize(fyne.NewSize(360, 620))
	d.Show()
}
