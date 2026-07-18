// Package dictionary is documented in doc.go.
package dictionary

// DictName identifies one of the supported word list dictionaries.
type DictName string

const (
	// DictENABLE selects the ENABLE word list (Enhanced North American Benchmark LExicon).
	// This list is public domain and downloaded automatically by 'make gaddag-free'.
	DictENABLE DictName = "enable"

	// DictWordnik selects the Wordnik word list (crowd-sourced open dictionary).
	DictWordnik DictName = "wordnik"
)

// AllDictNames is the ordered list of dictionary names.
// Used by the UI to populate the dictionary selection menu.
var AllDictNames = []DictName{
	DictENABLE,
	DictWordnik,
}
