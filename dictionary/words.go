// Package dictionary is documented in doc.go.
package dictionary

import (
	"bytes"
	"fmt"
)

// NewFromWords builds a Dictionary directly from a word list without requiring pre-built
// .bin assets, for tests and tooling where embedded assets are not available.
//
// words are sorted and deduplicated but NOT normalised: as with Build, each must already be
// uppercase A-Z and of a length Validate accepts. A word that is not is stored but can never
// match, so a list of lowercase words yields a Dictionary whose WordCount looks right and
// whose every lookup fails. tools/buildgaddag normalises before calling Build; callers here
// must do the same.
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
