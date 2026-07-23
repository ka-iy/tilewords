package defs

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"strings"
)

// DB is a set of headword definitions with the inflection edges needed to resolve
// a played word to one of them. It is immutable after construction and safe for
// concurrent use by Lookup.
type DB struct {
	// entries maps a lowercase headword to its definitions.
	entries map[string]*Entry
	// formOf maps a lowercase inflected form to a lemma headword present in entries.
	formOf map[string]string
}

// gobDB is the on-disk shape of a DB. Only the two maps are stored; the prefix
// index is rebuilt on decode so it never drifts from entries.
type gobDB struct {
	// Entries is the persisted headword-to-definitions map.
	Entries map[string]*Entry
	// FormOf is the persisted inflected-form-to-lemma map.
	FormOf map[string]string
}

// NewDB builds a DB from a headword map and an inflection-edge map. Both maps are
// retained, not copied; callers must not mutate them afterwards.
func NewDB(entries map[string]*Entry, formOf map[string]string) *DB {
	return &DB{entries: entries, formOf: formOf}
}

// Len returns the number of headwords in the DB.
func (db *DB) Len() int { return len(db.entries) }

// WithSupplement returns a new DB that layers supplemental headword entries and
// inflection edges on top of db without mutating db. Existing keys take
// precedence: a supplemental entry or edge whose key is already present in db is
// dropped, so the authoritative (primary-source) definitions and edges are never
// overwritten by a lower-priority source. Both argument maps are read, not
// retained. It is used to fold definitions from secondary public-domain
// dictionaries into the DB for words the primary source does not cover.
func (db *DB) WithSupplement(entries map[string]*Entry, forms map[string]string) *DB {
	mergedEntries := make(map[string]*Entry, len(db.entries)+len(entries))
	for k, v := range db.entries {
		mergedEntries[k] = v
	}
	for k, v := range entries {
		if _, exists := mergedEntries[k]; !exists {
			mergedEntries[k] = v
		}
	}

	mergedForms := make(map[string]string, len(db.formOf)+len(forms))
	for k, v := range db.formOf {
		mergedForms[k] = v
	}
	for k, v := range forms {
		if _, exists := mergedForms[k]; !exists {
			mergedForms[k] = v
		}
	}

	return NewDB(mergedEntries, mergedForms)
}

// FormLemma returns the lemma an inflected form maps to and whether such an edge
// exists. It reports the raw edge regardless of whether word is also a headword,
// so callers can detect a word that is both a headword and an inflection.
func (db *DB) FormLemma(word string) (string, bool) {
	lemma, ok := db.formOf[strings.ToLower(strings.TrimSpace(word))]
	return lemma, ok
}

// FormCount returns the number of inflection edges in the DB.
func (db *DB) FormCount() int { return len(db.formOf) }

// Lookup resolves word to a definition, trying each resolution layer in order of
// reliability (exact, form-of, stem, fuzzy). The match is case-insensitive.
// The second return is false when no layer produced a match.
func (db *DB) Lookup(word string) (Result, bool) {
	lw := strings.ToLower(strings.TrimSpace(word))
	if lw == "" {
		return Result{}, false
	}

	if e := db.entries[lw]; e != nil {
		res := Result{Entry: e, Headword: lw, Kind: MatchExact}
		// A word can be both a headword and an inflected form of another lemma
		// (e.g. "mice" is a rare verb yet chiefly the plural of "mouse"). Surface
		// the lemma's senses too rather than hiding the common reading behind the
		// homograph. The word's own senses stay primary, so this never demotes a
		// common homograph (e.g. "rose", "found") to its inflected reading.
		if lemma := db.formOf[lw]; lemma != "" && lemma != lw {
			if le := db.entries[lemma]; le != nil {
				res.AlsoForm = le
				res.AlsoFormWord = lemma
			}
		}
		return res, true
	}

	if lemma := db.formOf[lw]; lemma != "" {
		if e := db.entries[lemma]; e != nil {
			return Result{Entry: e, Headword: lemma, Kind: MatchFormOf}, true
		}
	}

	for _, c := range candidateStems(lw) {
		if r, ok := db.resolveHeadword(c, MatchStem); ok {
			return r, true
		}
	}

	for _, c := range variantCandidates(lw) {
		if r, ok := db.resolveHeadword(c, MatchFuzzy); ok {
			return r, true
		}
	}

	return Result{}, false
}

// resolveHeadword returns the definition reached by candidate cand — either as a
// headword directly or via an inflection edge to a lemma — tagged with kind.
func (db *DB) resolveHeadword(cand string, kind MatchKind) (Result, bool) {
	if e := db.entries[cand]; e != nil {
		return Result{Entry: e, Headword: cand, Kind: kind}, true
	}
	if lemma := db.formOf[cand]; lemma != "" {
		if e := db.entries[lemma]; e != nil {
			return Result{Entry: e, Headword: lemma, Kind: kind}, true
		}
	}
	return Result{}, false
}

// Encode writes the DB to w as gzip-compressed gob (the on-disk asset is roughly a
// third the size of the raw gob). The content is deterministic, but the exact bytes
// are not stable across runs (gob map order, gzip framing); the decoded DB is
// always equivalent.
func (db *DB) Encode(w io.Writer) error {
	gz := gzip.NewWriter(w)
	if err := gob.NewEncoder(gz).Encode(gobDB{Entries: db.entries, FormOf: db.formOf}); err != nil {
		gz.Close()
		return fmt.Errorf("defs.Encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("defs.Encode: flush gzip: %w", err)
	}
	return nil
}

// Decode reads a DB previously written by Encode (gzip-compressed gob).
func Decode(r io.Reader) (*DB, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	defer gz.Close()
	var g gobDB
	if err := gob.NewDecoder(gz).Decode(&g); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	if g.Entries == nil {
		g.Entries = make(map[string]*Entry)
	}
	if g.FormOf == nil {
		g.FormOf = make(map[string]string)
	}
	return NewDB(g.Entries, g.FormOf), nil
}
