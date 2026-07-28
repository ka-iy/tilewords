// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
)

// touchYCompensation is how far above the finger the mobile driver hit-tests a touch.
//
// Fyne's Android and iOS drivers subtract a fixed 8 density-independent pixels from a
// touch's Y coordinate before hit testing it — "to compensate for how we hold our fingers
// on the device" (internal/driver/mobile/device_android.go), applied to the touch-down,
// touch-move and touch-up paths alike.
//
// A widget's touch area is exactly its position and size, so the compensation gives the top
// 8 DIP of every widget to whatever is drawn above it. Where widgets are separated by
// padding the press is simply lost; where they are packed edge to edge, as on the board, it
// is delivered to the wrong widget. A widget that can work out where the finger really was
// adds this back; see cellWidget.Tapped.
const touchYCompensation = 8

// deviceIsMobile reports whether the app is running on a touch device, i.e. whether the
// touch compensation above is in play. It is a variable so tests can exercise the touch-only
// paths: the test driver's device reports desktop unless the package is built with the
// "mobile" build tag.
var deviceIsMobile = func() bool { return fyne.CurrentDevice().IsMobile() }
