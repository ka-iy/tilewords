// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tabItem is one tab of a tabPanel: a header title, the content shown when selected, and
// the text its Copy button copies.
type tabItem struct {
	// title is the tab's button label.
	title string
	// content is the body shown while the tab is selected.
	content fyne.CanvasObject
	// copyText returns the text the Copy button copies while this tab is active. It is the
	// whole-panel copy path (on touch a finger drag scrolls rather than selects, so this is
	// how the full move history / definitions text reaches the clipboard). May be nil.
	copyText func() string
}

// tabPanel is a minimal tab switcher used in place of container.AppTabs. AppTabs' tab
// bar advertises a wide minimum size and its touch handling and nested scrolling do not
// behave inside the phone layout's vertical scroll — the board is pushed past the
// viewport, tab taps are missed, and the body cannot be scrolled to the top. This
// switcher instead uses ordinary buttons over a stack that shows one body at a time, so
// it inherits the same reliable tap and scroll behaviour as the rest of the screen.
type tabPanel struct {
	// root is the built panel (a button header above the body); it is what callers add to a layout.
	root *fyne.Container
	// items are the tabs, parallel to buttons; retained so doCopy can read the active tab's copyText.
	items []tabItem
	// buttons are the tab-selector buttons, one per tab, parallel to items.
	buttons []*touchButton
	// body stacks every content object and is refreshed when the selection changes.
	body *fyne.Container
	// onCopied, when set, is invoked after the Copy button copies text (for user feedback).
	onCopied func()
	// selected is the index of the visible tab, or -1 before the first selection.
	selected int
}

// newTabPanel builds a tab panel from items and selects the first. It requires at least
// one item. The header buttons share the width equally so the panel never demands more
// width than the board it sits beside; a compact Copy button sits at the right. onCopied
// may be nil.
func newTabPanel(onCopied func(), items ...tabItem) *tabPanel {
	p := &tabPanel{items: items, onCopied: onCopied, selected: -1}
	tabButtons := container.NewGridWithColumns(len(items))
	contents := make([]fyne.CanvasObject, len(items))
	for i, it := range items {
		idx := i
		b := newTouchButton(it.title, func() { p.selectTab(idx) })
		p.buttons = append(p.buttons, b)
		contents[i] = it.content
		tabButtons.Add(b)
	}
	p.body = container.NewStack(contents...)

	copyBtn := newTouchButtonWithIcon("", theme.ContentCopyIcon(), p.doCopy)
	header := container.NewBorder(nil, nil, nil, copyBtn, tabButtons)
	p.root = container.NewBorder(header, nil, nil, nil, p.body)
	p.selectTab(0)
	return p
}

// selectTab shows the body of tab idx and marks its button active, hiding every other
// body. Selecting the already-selected tab (or an out-of-range index) is a no-op.
func (p *tabPanel) selectTab(idx int) {
	if idx < 0 || idx >= len(p.items) || idx == p.selected {
		return
	}
	p.selected = idx
	for i, it := range p.items {
		if i == idx {
			it.content.Show()
		} else {
			it.content.Hide()
		}
	}
	for i, b := range p.buttons {
		if i == idx {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.MediumImportance
		}
		b.Refresh()
	}
	p.body.Refresh()
}

// doCopy copies the active tab's whole-panel text to the clipboard and reports the copy
// via onCopied. It is a no-op when the active tab has no copyText or the text is empty.
func (p *tabPanel) doCopy() {
	if p.selected < 0 || p.selected >= len(p.items) {
		return
	}
	copyText := p.items[p.selected].copyText
	if copyText == nil {
		return
	}
	text := copyText()
	if text == "" {
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(text)
	if p.onCopied != nil {
		p.onCopied()
	}
}
