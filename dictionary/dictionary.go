// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package dictionary is documented in doc.go.
package dictionary

// Dictionary wraps a loaded GADDAG and exposes word validation for game use.
// It is immutable after construction and safe for concurrent use.
type Dictionary struct {
	name      DictName
	gaddag    *GADDAG
	wordCount int
}

// Name returns the dictionary identifier used to load this Dictionary.
func (d *Dictionary) Name() DictName { return d.name }

// WordCount returns the number of words in this dictionary.
func (d *Dictionary) WordCount() int { return d.wordCount }

// Validate reports whether word is a valid dictionary entry.
// The check is case-insensitive; "word", "WORD", and "Word" are equivalent.
// Returns false for any string containing non-alphabetic characters, or with
// length outside [MinWordLen, MaxWordLen].
func (d *Dictionary) Validate(word string) bool {
	return d.gaddag.contains(toUpper(word))
}

// GADDAG returns the underlying graph for direct traversal by the AI move generator.
// The returned pointer is read-only; callers must not modify the graph.
func (d *Dictionary) GADDAG() *GADDAG { return d.gaddag }
