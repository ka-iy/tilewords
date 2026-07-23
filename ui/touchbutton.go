// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

// touchButton is a widget.Button that also implements mobile.Touchable so a tap on it
// still registers when the button sits inside a Scroll on a touch screen.
//
// Fyne's mobile driver, once a touch moves more than tapMoveThreshold (4 density-
// independent pixels — smaller than normal finger jitter), hands the gesture to the
// innermost object under the finger that is Draggable or Touchable. A plain widget.Button
// is neither, so the driver instead selects the enclosing Scroll (which is Draggable),
// treats the press as a pan, and never delivers the button's Tapped — the tap appears to
// be ignored and the user must press again perfectly still. Implementing Touchable (even
// with no-op handlers) makes the driver select the button, not the Scroll, as the gesture
// target; since the button is not Draggable no pan starts and the tap is delivered on
// release. The board and rack cells sidestep the same trap by being Touchable for their
// own drag handling.
type touchButton struct {
	widget.Button
}

// touchButton must satisfy mobile.Touchable for the tap-in-scroll fix above to hold; the
// assertion fails the build if a refactor drops the Touch* methods.
var _ mobile.Touchable = (*touchButton)(nil)

// newTouchButton returns a text button that reliably taps inside a Scroll on touch screens.
func newTouchButton(label string, tapped func()) *touchButton {
	b := &touchButton{}
	b.Text = label
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// newTouchButtonWithIcon is newTouchButton with a themed icon (and optional label).
func newTouchButtonWithIcon(label string, icon fyne.Resource, tapped func()) *touchButton {
	b := &touchButton{}
	b.Text = label
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// TouchDown satisfies mobile.Touchable. The press cue and tap handling live in the
// embedded Button, so the touch phases need no handling of their own — implementing the
// interface is what keeps the enclosing Scroll from stealing the tap.
func (b *touchButton) TouchDown(*mobile.TouchEvent) {}

// TouchUp satisfies mobile.Touchable; see TouchDown.
func (b *touchButton) TouchUp(*mobile.TouchEvent) {}

// TouchCancel satisfies mobile.Touchable; see TouchDown.
func (b *touchButton) TouchCancel(*mobile.TouchEvent) {}

// touchCheck is a widget.Check that also implements mobile.Touchable, for the same reason
// as touchButton: a plain Check inside a Scroll loses a slightly-moved tap to the scroll's
// pan on touch screens. Check taps on itself (unlike RadioGroup, whose tap target is an
// unexported radioItem), so embedding it and adding the no-op Touch* methods is enough.
type touchCheck struct {
	widget.Check
}

// touchCheck must satisfy mobile.Touchable for the tap-in-scroll fix to hold.
var _ mobile.Touchable = (*touchCheck)(nil)

// newTouchCheck returns a labelled check box that reliably taps inside a Scroll on touch
// screens. changed may be nil.
func newTouchCheck(label string, changed func(bool)) *touchCheck {
	c := &touchCheck{}
	c.Text = label
	c.OnChanged = changed
	c.ExtendBaseWidget(c)
	return c
}

// TouchDown satisfies mobile.Touchable; being Touchable is what keeps the enclosing Scroll
// from stealing the tap. The check-toggle logic lives in the embedded Check.
func (c *touchCheck) TouchDown(*mobile.TouchEvent) {}

// TouchUp satisfies mobile.Touchable; see TouchDown.
func (c *touchCheck) TouchUp(*mobile.TouchEvent) {}

// TouchCancel satisfies mobile.Touchable; see TouchDown.
func (c *touchCheck) TouchCancel(*mobile.TouchEvent) {}
