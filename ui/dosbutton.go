// Package ui is documented in doc.go.
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

// DOS-style raised-button colours: a light-grey face with a white top/left highlight and a
// dark bottom/right shadow, giving the beveled look of an old DOS/Windows 3.1 button so the
// control clearly reads as pressable. These are intentionally fixed (not theme-following).
var (
	dosFace   = color.NRGBA{R: 192, G: 192, B: 192, A: 255} // light grey face
	dosLight  = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // white top/left highlight
	dosShadow = color.NRGBA{R: 64, G: 64, B: 64, A: 255}    // dark bottom/right shadow
	dosText   = color.NRGBA{R: 30, G: 30, B: 30, A: 255}    // near-black label
)

// dosBevel is the highlight/shadow edge thickness in pixels.
const dosBevel = 2

// bevelButton is a small tappable button drawn with a DOS-style raised bevel.
type bevelButton struct {
	widget.BaseWidget
	// text is the button label.
	text string
	// onTap is invoked when the button is tapped.
	onTap func()
}

// bevelButton implements mobile.Touchable so its tap survives inside a Scroll on touch
// screens; the assertion fails the build if a refactor drops the Touch* methods. See
// touchButton for why a Tappable-only control inside a Scroll otherwise loses its tap.
var _ mobile.Touchable = (*bevelButton)(nil)

// newBevelButton returns a DOS-style button showing text that calls onTap when tapped.
func newBevelButton(text string, onTap func()) *bevelButton {
	b := &bevelButton{text: text, onTap: onTap}
	b.ExtendBaseWidget(b)
	return b
}

// Tapped reports a tap to onTap (satisfies fyne.Tappable).
func (b *bevelButton) Tapped(*fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap()
	}
}

// TouchDown satisfies mobile.Touchable. There is nothing to do on the touch phases; being
// Touchable is what stops an enclosing Scroll from stealing the tap on touch screens.
func (b *bevelButton) TouchDown(*mobile.TouchEvent) {}

// TouchUp satisfies mobile.Touchable; see TouchDown.
func (b *bevelButton) TouchUp(*mobile.TouchEvent) {}

// TouchCancel satisfies mobile.Touchable; see TouchDown.
func (b *bevelButton) TouchCancel(*mobile.TouchEvent) {}

func (b *bevelButton) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(b.text, dosText)
	label.Alignment = fyne.TextAlignCenter
	label.TextSize = 11
	return &bevelButtonRenderer{
		b:      b,
		face:   canvas.NewRectangle(dosFace),
		top:    canvas.NewRectangle(dosLight),
		left:   canvas.NewRectangle(dosLight),
		bottom: canvas.NewRectangle(dosShadow),
		right:  canvas.NewRectangle(dosShadow),
		label:  label,
	}
}

type bevelButtonRenderer struct {
	b *bevelButton
	// face is the light-grey button background.
	face *canvas.Rectangle
	// top / left are the white highlight edges.
	top, left *canvas.Rectangle
	// bottom / right are the dark shadow edges.
	bottom, right *canvas.Rectangle
	// label is the centred button text.
	label *canvas.Text
}

func (r *bevelButtonRenderer) Destroy() {}

func (r *bevelButtonRenderer) MinSize() fyne.Size {
	ts := r.label.MinSize()
	return fyne.NewSize(ts.Width+10, ts.Height+6)
}

func (r *bevelButtonRenderer) Layout(size fyne.Size) {
	r.face.Resize(size)
	r.face.Move(fyne.NewPos(0, 0))

	r.top.Resize(fyne.NewSize(size.Width, dosBevel))
	r.top.Move(fyne.NewPos(0, 0))
	r.left.Resize(fyne.NewSize(dosBevel, size.Height))
	r.left.Move(fyne.NewPos(0, 0))
	r.bottom.Resize(fyne.NewSize(size.Width, dosBevel))
	r.bottom.Move(fyne.NewPos(0, size.Height-dosBevel))
	r.right.Resize(fyne.NewSize(dosBevel, size.Height))
	r.right.Move(fyne.NewPos(size.Width-dosBevel, 0))

	lh := r.label.MinSize().Height
	r.label.Resize(fyne.NewSize(size.Width, lh))
	r.label.Move(fyne.NewPos(0, (size.Height-lh)/2))
}

func (r *bevelButtonRenderer) Refresh() { canvas.Refresh(r.b) }

// Objects draws the face first, then the bevel edges over it, then the label on top.
func (r *bevelButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.face, r.top, r.left, r.bottom, r.right, r.label}
}
