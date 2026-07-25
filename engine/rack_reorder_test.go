// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"math/rand"
	"sort"
	"testing"
)

// rackOf builds a rack from a string of letters (one tile per byte).
func rackOf(letters string) *Rack {
	r := &Rack{}
	tiles := make([]Tile, len(letters))
	for i := 0; i < len(letters); i++ {
		tiles[i] = Tile{Letter: letters[i], Points: LetterPoints(letters[i])}
	}
	if err := r.Add(tiles); err != nil {
		panic(err)
	}
	return r
}

// rackLetters returns the rack's tiles as a string in order.
func rackLetters(r *Rack) string {
	tiles := r.Tiles()
	b := make([]byte, len(tiles))
	for i, t := range tiles {
		b[i] = t.Letter
	}
	return string(b)
}

func TestRackMoveTile(t *testing.T) {
	cases := []struct {
		name     string
		start    string
		from, to int
		want     string
	}{
		{"move first to middle", "ABCD", 0, 2, "BCAD"},
		{"move last to second", "ABCD", 3, 1, "ADBC"},
		{"adjacent swap", "ABCD", 1, 2, "ACBD"},
		{"no-op same index", "ABCD", 2, 2, "ABCD"},
		{"out of range ignored", "ABCD", 0, 9, "ABCD"},
		{"move middle to front", "ABCDE", 2, 0, "CABDE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := rackOf(tc.start)
			r.MoveTile(tc.from, tc.to)
			if got := rackLetters(r); got != tc.want {
				t.Errorf("MoveTile(%d,%d) on %q = %q, want %q", tc.from, tc.to, tc.start, got, tc.want)
			}
		})
	}
}

func TestRackShuffle_PreservesMultiset(t *testing.T) {
	const letters = "RETAINS"
	r := rackOf(letters)
	r.Shuffle(rand.New(rand.NewSource(12345)))

	got := []byte(rackLetters(r))
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	want := []byte(letters)
	sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
	if string(got) != string(want) {
		t.Fatalf("Shuffle changed the multiset: got sorted %q, want %q", got, want)
	}
	if r.Count() != len(letters) {
		t.Fatalf("Shuffle changed count: got %d, want %d", r.Count(), len(letters))
	}
}
