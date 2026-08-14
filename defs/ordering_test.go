// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
)

// reorderedAsset returns a valid encoded asset whose headword blob has had two equal-length
// headwords transposed, leaving every count, length and offset in the file consistent.
func reorderedAsset(t *testing.T) []byte {
	t.Helper()
	db := NewDB(map[string]*Entry{
		"cat": {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "feline"}}},
		"dog": {Word: "dog", Senses: []Sense{{POS: "noun", Gloss: "canine"}}},
	}, map[string]string{"cats": "cat"})

	var enc bytes.Buffer
	if err := db.Encode(&enc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(enc.Bytes()))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("inflate: %v", err)
	}

	// Both headwords are three bytes, so transposing them keeps every offset valid.
	i := bytes.Index(raw, []byte("catdog"))
	if i < 0 {
		t.Fatal("could not locate the headword blob in the encoded asset")
	}
	copy(raw[i:], []byte("dogcat"))

	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	if _, err := zw.Write(raw); err != nil {
		t.Fatalf("deflate: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close deflate: %v", err)
	}
	return out.Bytes()
}

// TestDecodeRejectsUnsortedHeadwords guards the ordering invariant findHead and findForm rely on.
//
// Both are binary searches, and a binary search over unsorted data does not fail — it reports a
// word that is present as missing, and through formLemma answers an inflected form with the
// definition of a different headword. Every count, length and index in the asset can still be
// self-consistent when that happens, so nothing else in Decode catches it. Refusing the asset is
// the only way the error reaches anyone; otherwise the game silently shows a wrong definition.
func TestDecodeRejectsUnsortedHeadwords(t *testing.T) {
	_, err := Decode(bytes.NewReader(reorderedAsset(t)))
	if err == nil {
		t.Fatal("an asset whose headwords are out of order decoded cleanly; lookups would " +
			"silently answer with the wrong word")
	}
	if !strings.Contains(err.Error(), "sort strictly after") {
		t.Errorf("error = %v, want it to name the ordering problem", err)
	}
}

// TestDecodeAcceptsSortedAsset is the companion control: the same builder, left alone, must still
// decode, so TestDecodeRejectsUnsortedHeadwords is known to fail for the ordering and not because
// the fixture was malformed to begin with.
func TestDecodeAcceptsSortedAsset(t *testing.T) {
	db := NewDB(map[string]*Entry{
		"cat": {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "feline"}}},
		"dog": {Word: "dog", Senses: []Sense{{POS: "noun", Gloss: "canine"}}},
	}, map[string]string{"cats": "cat"})
	var enc bytes.Buffer
	if err := db.Encode(&enc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := Decode(bytes.NewReader(enc.Bytes()))
	if err != nil {
		t.Fatalf("a well-formed asset was refused: %v", err)
	}
	if _, ok := got.Lookup("cat"); !ok {
		t.Error("decoded DB does not resolve a headword it was built with")
	}
}
