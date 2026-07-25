// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import (
	"embed"
	"fmt"
	"sync"
)

// embeddedDefs holds the definitions asset compiled into the binary. The "all:"
// prefix embeds the directory even when it contains only .gitkeep, so the package
// builds whether or not 'make defs' has produced the asset; a missing asset surfaces
// at Load time rather than as a compile error.
//
//go:embed all:assets/definitions
var embeddedDefs embed.FS

// defsAssetPath is the embedded path of the definitions asset: the gzipped flat layout
// described on defs.DB, written and read by DB.Encode and Decode.
const defsAssetPath = "assets/definitions/definitions.bin.gz"

// loadOnce guards the one-time decode of the embedded DB.
var loadOnce sync.Once

// loadedDB is the decoded definitions DB, or nil until the first successful Load.
var loadedDB *DB

// loadErr is the error from the one-time decode, cached alongside loadedDB.
var loadErr error

// Available reports whether the definitions asset was embedded in this binary.
// It returns false when 'make defs' has not been run, letting the UI hide the
// definitions feature rather than fail.
func Available() bool {
	f, err := embeddedDefs.Open(defsAssetPath)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// Load decodes the embedded definitions DB and caches it. The DB is large but
// immutable, so the first call decodes it and every later call returns that same
// instance. It returns a descriptive error when the asset was not embedded.
func Load() (*DB, error) {
	loadOnce.Do(func() {
		// Stream the asset rather than ReadFile it: the embedded bytes live in the binary's
		// read-only data and cost no heap, but ReadFile converts them to a []byte, which
		// copies the whole compressed asset for the duration of the decode. Decode reads
		// sequentially and needs no more than a Reader.
		f, err := embeddedDefs.Open(defsAssetPath)
		if err != nil {
			loadErr = fmt.Errorf("defs.Load: asset not embedded, run 'make defs': %w", err)
			return
		}
		defer f.Close()

		db, err := Decode(f)
		if err != nil {
			loadErr = fmt.Errorf("defs.Load: %w", err)
			return
		}
		loadedDB = db
	})
	return loadedDB, loadErr
}
