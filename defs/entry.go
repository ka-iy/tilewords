// Package defs resolves a played word to a Wiktionary definition.
//
// The same resolution logic runs in two places:
//
//   - The builddefs tool uses it to filter the full Wiktionary extract down to
//     the definitions actually reachable from a word list, and to report coverage.
//   - The game uses DB.Lookup at runtime to show the definition of a word a
//     player has formed.
//
// Resolution is layered, from most to least reliable:
//
//   - Exact: the word is itself a headword.
//   - FormOf: Wiktionary records the word as an inflected form of a lemma
//     (from a "plural of X" sense or a lemma's inflection table).
//   - Stem: rule-based de-inflection (including classical plurals) produces a
//     lemma that is a headword.
//   - Fuzzy: a known orthographic correspondence (e.g. -ise/-ize, -our/-or,
//     ae/e) rewrites the word into a headword. Both this and the stem layer only
//     ever accept a rewrite that is itself a real headword, so neither can invent
//     a definition from an unrelated near-spelling.
package defs

// Sense is one definition of a headword.
type Sense struct {
	// POS is the part of speech (e.g. "noun", "verb"); empty when Wiktionary recorded none.
	POS string
	// Gloss is a single human-readable definition line.
	Gloss string
}

// Entry holds every definition Wiktionary records for one headword.
type Entry struct {
	// Word is the lowercase headword these senses define.
	Word string
	// Senses are the headword's definitions in Wiktionary order, capped at MaxSensesPerEntry.
	Senses []Sense
}

// MatchKind records how DB.Lookup resolved a query to an Entry.
type MatchKind uint8

const (
	// MatchNone means no definition was found.
	MatchNone MatchKind = iota
	// MatchExact means the queried word is itself a headword.
	MatchExact
	// MatchFormOf means Wiktionary explicitly records the word as an inflected form of the headword.
	MatchFormOf
	// MatchStem means rule-based de-inflection reduced the word to the headword.
	MatchStem
	// MatchFuzzy means an orthographic-variant rewrite reached the headword.
	MatchFuzzy
)

// String returns a short lowercase label for the match kind.
func (k MatchKind) String() string {
	switch k {
	case MatchExact:
		return "exact"
	case MatchFormOf:
		return "formof"
	case MatchStem:
		return "stem"
	case MatchFuzzy:
		return "fuzzy"
	default:
		return "none"
	}
}

// Result is the outcome of DB.Lookup.
type Result struct {
	// Entry is the definitions found, or nil when Kind is MatchNone.
	Entry *Entry
	// Headword is the DB key that supplied Entry; it may differ from the queried word.
	Headword string
	// Kind records which resolution layer produced the match.
	Kind MatchKind
	// AlsoForm is set only on an exact match whose word is additionally a recorded
	// inflected form of a different lemma. It carries that lemma's definitions so a
	// caller can show the inflected-form reading (e.g. "mice" as the plural of
	// "mouse") alongside the word's own senses. It is nil otherwise.
	AlsoForm *Entry
	// AlsoFormWord is the lemma AlsoForm belongs to, or "" when AlsoForm is nil.
	AlsoFormWord string
}
