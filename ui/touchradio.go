// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// touchRadio is a vertical single-select control built from touchButtons so its options
// tap reliably inside a Scroll on touch screens.
//
// Fyne's widget.RadioGroup cannot be given the same fix as touchButton/touchCheck: it taps
// through unexported radioItem widgets that are Tappable but not mobile.Touchable, so a
// slightly-moved press inside a Scroll is stolen by the scroll's pan (see touchButton) and
// there is no exported type to make Touchable. Rebuilding the group from touchButtons — one
// per option, each a flat row with a leading radio-circle icon — keeps the radio look while
// making every row a reliable touch target.
type touchRadio struct {
	// options are the selectable labels, parallel to buttons.
	options []string
	// buttons are the per-option rows, parallel to options. Exposed so a caller can lay
	// them out itself (e.g. each row beside a trailing Info button) instead of using list().
	buttons []*touchButton
	// selected is the index of the chosen option, or -1 when nothing is selected yet.
	selected int
	// onChange, when set, is called with the newly-selected option's label after a change.
	onChange func(string)
}

// newTouchRadio builds a radio group over options that reports selection changes to
// onChange (which may be nil). Nothing is selected until SetSelected or a tap.
func newTouchRadio(options []string, onChange func(string)) *touchRadio {
	tr := &touchRadio{options: options, onChange: onChange, selected: -1}
	for i, opt := range options {
		idx := i
		b := newTouchButton(opt, func() { tr.selectIndex(idx) })
		b.Icon = theme.RadioButtonIcon()
		b.Importance = widget.LowImportance // flat row, so it reads as a radio not a raised button
		b.Alignment = widget.ButtonAlignLeading
		b.IconPlacement = widget.ButtonIconLeadingText
		tr.buttons = append(tr.buttons, b)
	}
	return tr
}

// list returns the options stacked in a VBox — the default layout for a plain radio group.
func (tr *touchRadio) list() *fyne.Container {
	objs := make([]fyne.CanvasObject, len(tr.buttons))
	for i, b := range tr.buttons {
		objs[i] = b
	}
	return container.NewVBox(objs...)
}

// selectIndex selects option i, updates the row icons and fires onChange. Selecting the
// already-selected option (or an out-of-range index) is a no-op, matching a Required
// RadioGroup that cannot be deselected by re-tapping its current choice.
func (tr *touchRadio) selectIndex(i int) {
	if i < 0 || i >= len(tr.options) || i == tr.selected {
		return
	}
	tr.selected = i
	tr.refreshIcons()
	if tr.onChange != nil {
		tr.onChange(tr.options[i])
	}
}

// SetSelected selects the option equal to opt, if present, firing onChange like a tap.
func (tr *touchRadio) SetSelected(opt string) {
	for i, o := range tr.options {
		if o == opt {
			tr.selectIndex(i)
			return
		}
	}
}

// Selected returns the currently-selected option's label, or "" when none is selected.
func (tr *touchRadio) Selected() string {
	if tr.selected < 0 {
		return ""
	}
	return tr.options[tr.selected]
}

// refreshIcons shows the filled radio icon on the selected row and the empty circle on the
// rest, so selection is visible exactly as a RadioGroup shows it.
func (tr *touchRadio) refreshIcons() {
	for i, b := range tr.buttons {
		if i == tr.selected {
			b.SetIcon(theme.RadioButtonCheckedIcon())
		} else {
			b.SetIcon(theme.RadioButtonIcon())
		}
	}
}
