// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package dictionary

import (
	"bytes"
	"os"
	"testing"

	"pgregory.net/rapid"
)

// testWords is a small curated word list embedded in the test binary.
// These words have no licensing encumbrances and are used to build a test GADDAG.
var testWords = []string{
	"AA", "AB", "AD", "AE", "AG", "AH", "AI", "AL", "AM", "AN",
	"AR", "AS", "AT", "AW", "AX", "AY",
	"CAT", "DOG", "FOX", "HAT", "JAB", "MAT", "PAT", "RAT", "SAT", "VAT",
	"CATS", "DOGS", "HATS", "MATS", "RATS",
	"QUICK", "QUACK", "QUEEN", "QUEST",
	"SQUAB", "SQUAD", "SQUAT",
	"BOARD", "BLANK", "BRICK", "BRUSH",
	"VALID", "VIRAL", "VISIT", "VISTA",
	"WORD", "WORDS", "WORLD", "WORTH",
	"ZEBRA", "ZONAL", "ZONED", "ZONES",
}

// testGADDAG is loaded once in TestMain and shared across all tests.
var testGADDAG *GADDAG

// testDict is the Dictionary wrapping testGADDAG.
var testDict *Dictionary

func TestMain(m *testing.M) {
	var buf bytes.Buffer
	if err := Build(testWords, &buf); err != nil {
		panic("TestMain: Build failed: " + err.Error())
	}
	g, err := loadGADDAG(buf.Bytes())
	if err != nil {
		panic("TestMain: loadGADDAG failed: " + err.Error())
	}
	testGADDAG = g
	testDict = &Dictionary{name: "test", gaddag: g, wordCount: g.words()}

	os.Exit(m.Run())
}

// testWordSet returns a set of all test words for oracle comparisons.
func testWordSet() map[string]bool {
	s := make(map[string]bool, len(testWords))
	for _, w := range testWords {
		s[w] = true
	}
	return s
}

// wordFromDictGen is a rapid generator that samples valid words from testWords.
func wordFromDictGen() *rapid.Generator[string] {
	return rapid.SampledFrom(testWords)
}

// randomAlphaGen generates uppercase A-Z strings with length in [minLen, maxLen].
func randomAlphaGen(minLen, maxLen int) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		n := rapid.IntRange(minLen, maxLen).Draw(t, "len")
		bs := make([]byte, n)
		for i := range bs {
			bs[i] = byte(rapid.IntRange(0, 25).Draw(t, "letter")) + 'A'
		}
		return string(bs)
	})
}

// invalidStringGen generates strings that contain at least one non-A-Z byte.
func invalidStringGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Start with a valid alpha string then inject a non-A-Z byte at a random position.
		n := rapid.IntRange(1, MaxWordLen+2).Draw(t, "len")
		bs := make([]byte, n)
		for i := range bs {
			bs[i] = byte(rapid.IntRange(0, 25).Draw(t, "letter")) + 'A'
		}
		// Inject a non-A-Z byte (space, digit, punctuation — anything outside 65-90)
		badPos := rapid.IntRange(0, n-1).Draw(t, "badPos")
		badByte := byte(rapid.IntRange(0, 64).Draw(t, "badByte")) // 0-64 excludes 65('A')-90('Z')
		bs[badPos] = badByte
		return string(bs)
	})
}
