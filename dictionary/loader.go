// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package dictionary is documented in doc.go.
package dictionary

import (
	"embed"
	"fmt"
	"sync"
)

//go:embed all:assets/dictionaries
var embeddedAssets embed.FS

// A loaded GADDAG's in-memory size is close to its on-disk asset, since both are the
// same flat slice-of-integers representation (the largest shipped list is on the order
// of ~10 MB resident). Decoding one still allocates and copies those slices, so Load
// caches the most recently loaded dictionary: re-loading the same name (e.g. starting a
// game, saving, then loading that save) returns the existing instance instead of
// decoding a second copy, which would otherwise briefly hold two GADDAGs live.
//
// A single slot (rather than a per-name map) bounds resident memory to one dictionary:
// switching to a different dictionary drops the previous one's reference so it can be
// garbage-collected. Dictionary is immutable after Load, so sharing the cached pointer
// across callers is safe.
var (
	// cacheMu guards cachedName and cachedDict.
	cacheMu sync.Mutex
	// cachedName is the name of the dictionary held in cachedDict; "" when the cache is empty.
	cachedName DictName
	// cachedDict is the most recently loaded dictionary, or nil when the cache is empty.
	cachedDict *Dictionary
)

// assetPath maps a DictName to its embedded asset file path.
func assetPath(name DictName) string {
	return "assets/dictionaries/" + string(name) + ".bin"
}

// Available reports whether the embedded GADDAG asset for name exists in the binary.
// Returns false for any name whose asset was not present when the binary was compiled.
func Available(name DictName) bool {
	f, err := embeddedAssets.Open(assetPath(name))
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// Load loads a dictionary by name and returns it. Re-loading the currently cached
// dictionary returns the existing instance without decoding it again (see cachedDict).
// Returns a descriptive error if the asset is missing or malformed.
func Load(name DictName) (*Dictionary, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cachedDict != nil && cachedName == name {
		return cachedDict, nil
	}

	// Open the asset before evicting anything. The likeliest failure is a name whose asset was
	// not built into this binary, and discovering that after dropping the cache would throw
	// away a perfectly good dictionary that the next load then has to decode again.
	//
	// Stream the asset rather than ReadFile it: the embedded bytes live in the binary's
	// read-only data and cost no heap, but ReadFile converts them to a []byte, which copies
	// the whole asset. Decoding then peaks at the asset plus the graph instead of the graph
	// alone — a spike the mobile low-memory killer reacts to.
	path := assetPath(name)
	f, err := embeddedAssets.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dictionary.Load: %w: asset %q not found; run 'make gaddag' to build it", err, path)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("dictionary.Load: stat asset %q: %w", path, err)
	}

	// Loading a different dictionary: drop the previously cached one before decoding, so its
	// large GADDAG can be garbage-collected rather than staying live alongside the new one.
	// Holding it back to restore on a decode failure would defeat that, and a decode failure
	// means the embedded asset itself is corrupt — a broken binary, not a routine outcome.
	cachedDict = nil
	cachedName = ""

	g, err := decodeGADDAG(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("dictionary.Load: %w", err)
	}

	d := &Dictionary{
		name:      name,
		gaddag:    g,
		wordCount: g.words(),
	}
	cachedDict = d
	cachedName = name
	return d, nil
}
