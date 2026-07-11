// Package dictionary is documented in doc.go.
package dictionary

import (
	"embed"
	"fmt"
)

//go:embed all:assets/dictionaries
var embeddedAssets embed.FS

// assetPath maps a DictName to its embedded .gob file path.
func assetPath(name DictName) string {
	return "assets/dictionaries/" + string(name) + ".gob"
}

// Available reports whether the embedded GADDAG asset for name exists in the binary.
// Returns false for any name whose .gob was not present when the binary was compiled.
func Available(name DictName) bool {
	f, err := embeddedAssets.Open(assetPath(name))
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// Load loads a dictionary by name and returns it.
// Returns a descriptive error if the asset is missing or malformed.
func Load(name DictName) (*Dictionary, error) {
	path := assetPath(name)
	data, err := embeddedAssets.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("dictionary.Load: asset %q not found (run 'make gaddag' to build it): %w", path, err)
	}

	g, err := loadGADDAG(data)
	if err != nil {
		return nil, fmt.Errorf("dictionary.Load: %w", err)
	}

	return &Dictionary{
		name:      name,
		gaddag:    g,
		wordCount: g.words(),
	}, nil
}
