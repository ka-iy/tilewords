// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeTemp writes content to a file named name under a fresh temp dir and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// An exclusion entry may carry trailing columns naming the word lists it came from, so
// only the first field is the word; otherwise the whole line would be compared against
// the word list and nothing would ever match.
func TestReadExclusionsTakesFirstFieldOnly(t *testing.T) {
	path := writeTemp(t, "exclude.txt", "# a comment\n\nludss\twordnik\nKose  wordnik,enable\nkaa\n")

	excluded, err := readExclusions(path)
	if err != nil {
		t.Fatalf("readExclusions: %v", err)
	}

	for _, want := range []string{"LUDSS", "KOSE", "KAA"} {
		if _, ok := excluded[want]; !ok {
			t.Errorf("excluded set is missing %q; entries are uppercased and cut at the first field", want)
		}
	}
	if len(excluded) != 3 {
		t.Errorf("got %d exclusions, want 3; comments and blank lines must be skipped", len(excluded))
	}
}

// An absent -exclude path is how every build that names no exclusion file behaves, so it
// must yield an empty set rather than an error.
func TestReadExclusionsEmptyPath(t *testing.T) {
	excluded, err := readExclusions("")
	if err != nil {
		t.Fatalf("readExclusions(%q): %v", "", err)
	}
	if len(excluded) != 0 {
		t.Errorf("got %d exclusions for an empty path, want 0", len(excluded))
	}
}

// A word named in the exclusion set must not reach the asset, and must be reported, while
// every other word survives. Otherwise a word no dictionary defines would still be
// accepted as a play.
func TestReadWordsOmitsExcluded(t *testing.T) {
	path := writeTemp(t, "words.txt", "lud\nluds\nludss\nkaa\nkos\n")

	words, dropped, err := readWords([]string{path}, map[string]struct{}{"LUDSS": {}, "KAA": {}})
	if err != nil {
		t.Fatalf("readWords: %v", err)
	}

	want := []string{"KOS", "LUD", "LUDS"}
	if !slices.Equal(words, want) {
		t.Errorf("got words %v, want %v", words, want)
	}
	wantDropped := []string{"KAA", "LUDSS"}
	if !slices.Equal(dropped, wantDropped) {
		t.Errorf("got dropped %v, want %v", dropped, wantDropped)
	}
}

// An exclusion entry matching no word in the list must not be reported as dropped, so a
// stale entry can be spotted by its absence from the run's report.
func TestReadWordsDoesNotReportUnmatchedExclusion(t *testing.T) {
	path := writeTemp(t, "words.txt", "lud\nluds\n")

	words, dropped, err := readWords([]string{path}, map[string]struct{}{"SOAPOLITE": {}})
	if err != nil {
		t.Fatalf("readWords: %v", err)
	}

	if !slices.Equal(words, []string{"LUD", "LUDS"}) {
		t.Errorf("got words %v, want [LUD LUDS]", words)
	}
	if len(dropped) != 0 {
		t.Errorf("got dropped %v, want none: nothing in the list matched the exclusion", dropped)
	}
}

// Every word in the committed exclusion list must survive a round trip through the
// parser, so a formatting slip in that file shows up here rather than as a word that
// silently stays playable.
func TestCommittedExclusionListParses(t *testing.T) {
	excluded, err := readExclusions(filepath.Join("..", "..", "defs", "possibly_invalid_words.txt"))
	if err != nil {
		t.Fatalf("readExclusions: %v", err)
	}

	for _, want := range []string{"KAA", "KOSE", "LUDSS", "SOAPOLITE"} {
		if _, ok := excluded[want]; !ok {
			t.Errorf("committed exclusion list does not yield %q", want)
		}
	}
}
