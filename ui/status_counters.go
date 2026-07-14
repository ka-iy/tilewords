// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// statusCounters arranges the five score-counter labels in a single centred row when the
// available width fits it, and wraps them onto two centred rows — the two scores, then the
// bag/move/level counters — only when the single row would not fit. Wrapping is chosen from
// the widget's actual width, so the block stays within a narrow phone width without
// horizontal scrolling while keeping one row wherever there is room.
//
// Like responsiveContainer, the shared labels are re-parented into a freshly built
// arrangement only when the row count changes, so a label is never in two containers at
// once and game state is untouched.
type statusCounters struct {
	widget.BaseWidget

	you   fyne.CanvasObject
	ai    fyne.CanvasObject
	bag   fyne.CanvasObject
	move  fyne.CanvasObject
	level fyne.CanvasObject

	holder *fyne.Container // a stack holding the current arrangement
	twoRow bool
	inited bool
}

func newStatusCounters(you, ai, bag, move, level fyne.CanvasObject) *statusCounters {
	s := &statusCounters{
		you: you, ai: ai, bag: bag, move: move, level: level,
		holder: container.NewStack(),
	}
	s.ExtendBaseWidget(s)
	return s
}

// arrange builds the one- or two-row arrangement, re-parenting the shared labels.
func (s *statusCounters) arrange(twoRow bool) fyne.CanvasObject {
	if twoRow {
		return container.NewVBox(
			container.NewCenter(container.NewHBox(s.you, s.ai)),
			container.NewCenter(container.NewHBox(s.bag, s.move, s.level)),
		)
	}
	return container.NewCenter(container.NewHBox(s.you, s.ai, s.bag, s.move, s.level))
}

func (s *statusCounters) set(twoRow bool) {
	s.inited = true
	s.twoRow = twoRow
	s.holder.Objects = []fyne.CanvasObject{s.arrange(twoRow)}
	s.holder.Refresh()
}

// singleRowWidth is the width the one-row arrangement needs: the label widths plus the
// theme padding an HBox inserts between them (see layout.hBoxLayout.MinSize).
func (s *statusCounters) singleRowWidth() float32 {
	labels := [...]fyne.CanvasObject{s.you, s.ai, s.bag, s.move, s.level}
	w := float32(0)
	for i, o := range labels {
		w += o.MinSize().Width
		if i > 0 {
			w += theme.Padding()
		}
	}
	return w
}

// Resize switches to two rows when the width can no longer fit the single row (and back).
func (s *statusCounters) Resize(size fyne.Size) {
	if twoRow := size.Width < s.singleRowWidth(); !s.inited || twoRow != s.twoRow {
		s.set(twoRow)
	}
	s.BaseWidget.Resize(size)
}

func (s *statusCounters) CreateRenderer() fyne.WidgetRenderer {
	if !s.inited {
		// Default to a single row until the first Resize reports the real width.
		s.set(false)
	}
	return widget.NewSimpleRenderer(s.holder)
}
