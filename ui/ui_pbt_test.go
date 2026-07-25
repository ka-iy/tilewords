// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"testing"

	"pgregory.net/rapid"
)

// PBT-UI-01: boardGeometry always produces a square board that fits within the
// available area with non-negative centring offsets.
func TestPBT_BoardGeometry_FitsAndCentred(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := rapid.Float32Range(0, 4000).Draw(t, "w")
		h := rapid.Float32Range(0, 4000).Draw(t, "h")
		cell, offX, offY := boardGeometry(w, h)

		if cell < 0 {
			t.Fatalf("cell must be non-negative, got %v", cell)
		}
		boardSide := cell * boardDim
		if boardSide > w+1e-3 || boardSide > h+1e-3 {
			t.Fatalf("board side %v does not fit in %vx%v", boardSide, w, h)
		}
		if offX < -1e-3 || offY < -1e-3 {
			t.Fatalf("offsets must be non-negative, got (%v,%v)", offX, offY)
		}
	})
}

// PBT-UI-02: rackGeometry always produces square slots whose row fits within the
// available width with a non-negative centring offset.
func TestPBT_RackGeometry_FitsAndCentred(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := rapid.Float32Range(0, 4000).Draw(t, "w")
		h := rapid.Float32Range(0, 4000).Draw(t, "h")
		n := rapid.IntRange(1, 7).Draw(t, "n")
		slot, offX := rackGeometry(w, h, n)

		if slot < 0 {
			t.Fatalf("slot must be non-negative, got %v", slot)
		}
		if slot > h+1e-3 {
			t.Fatalf("slot %v taller than height %v", slot, h)
		}
		// When slots have a positive size the whole row must fit the width. (If the
		// area is narrower than the fixed gaps, slots collapse to zero and only the
		// gaps remain, which legitimately cannot fit — so guard on slot > 0.)
		total := slot*float32(n) + rackGapPx*float32(n-1)
		if slot > 0 && total > w+1e-3 {
			t.Fatalf("rack row %v wider than width %v", total, w)
		}
		if offX < -1e-3 {
			t.Fatalf("offset must be non-negative, got %v", offX)
		}
	})
}

// PBT-UI-03: sanitiseError always returns a bounded, non-empty string for a
// non-nil error.
func TestPBT_SanitiseError_Invariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		msg := rapid.StringN(0, 300, -1).Draw(t, "msg")
		err := &simpleErr{msg}
		result := sanitiseError(err)
		if result == "" {
			t.Fatal("sanitiseError: non-nil error produced empty string")
		}
		if len([]rune(result)) > 124 {
			t.Fatalf("sanitiseError: result too long (%d runes): %q", len([]rune(result)), result)
		}
	})
}

// simpleErr implements the error interface for PBT use.
type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
