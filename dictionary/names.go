// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package dictionary is documented in doc.go.
package dictionary

// DictName identifies one of the supported word list dictionaries.
type DictName string

const (
	// DictAtebits selects the atebits word list (the public-domain list shipped with
	// the Letterpress game). The value matches its wordlists/<name>.txt stem so the
	// build compiles it to assets/dictionaries/atebits-letterpress.bin.
	DictAtebits DictName = "atebits-letterpress"

	// DictENABLE selects the ENABLE2K word list (Enhanced North American Benchmark LExicon).
	// This list is public domain and downloaded automatically by 'make gaddag-free'.
	DictENABLE DictName = "enable"

	// DictWordnik selects the Wordnik word list (crowd-sourced open dictionary).
	DictWordnik DictName = "wordnik"
)

// AllDictNames is the ordered list of dictionary names.
// Used by the UI to populate the dictionary selection menu.
var AllDictNames = []DictName{
	DictAtebits,
	DictENABLE,
	DictWordnik,
}
