// Package dictionary is documented in doc.go.
package dictionary

import (
	"bytes"
	"fmt"
)

// NewFromWords builds a Dictionary directly from a word list without requiring
// pre-built .bin assets. Words are normalised, sorted, and deduplicated before
// GADDAG construction. This is useful in tests and tooling where embedded assets
// are not available.
func NewFromWords(name DictName, words []string) (*Dictionary, error) {
	var buf bytes.Buffer
	if err := Build(words, &buf); err != nil {
		return nil, fmt.Errorf("dictionary.NewFromWords: %w", err)
	}
	g, err := loadGADDAG(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("dictionary.NewFromWords: %w", err)
	}
	return &Dictionary{name: name, gaddag: g, wordCount: g.words()}, nil
}
