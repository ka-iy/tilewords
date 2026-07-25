package defs

import "strings"

// orthographicPairs are spelling correspondences that map a word to a variant of
// the same word (British/American and Latinate orthography). Each is applied in
// both directions. They are distinctive multi-letter sequences, so replacing them
// rarely produces an unrelated word — and any candidate that is not a real
// headword is discarded anyway.
var orthographicPairs = [][2]string{
	{"ise", "ize"},
	{"isation", "ization"},
	{"yse", "yze"},
	{"ogue", "og"},
}

// orthographicContractions are one-directional spellings where the left form (a
// digraph) reduces to the right form. Only the contraction is applied: expanding every "e"
// to "ae" would generate far too many spurious candidates.
//
// These are the Latinate ligature spellings — "haemoglobin" for "hemoglobin", "oecology" for
// "ecology" — so they are limited to words of at least minContractionLen. In a short word the
// same two letters are usually not a ligature at all but ordinary adjacent vowels, most often
// in Scots: contracting those turns "haen" into "hen", "kae" into "ke", "thae" into "the" and
// "haeres" (Latin for heir) into "here", each a real headword and none of them the right one.
//
// "our" to "or" is handled separately, by suffixContractions, because it is only the British
// noun ending that reduces — not the same three letters anywhere in a word.
var orthographicContractions = [][2]string{
	{"ae", "e"},
	{"oe", "e"},
}

// suffixContractions are contractions that apply only at the end of a word. "our" to "or" is
// the British-to-American noun ending ("colour", "flavour", "honour", "stentour"), and
// anchoring it there is what keeps it from firing mid-word, where those three letters are
// usually not that ending at all: unanchored it turned "stoure" into "store", "stoury" into
// "story", "courd" into "cord", "couries" into "cory", "touries" into "tory" and "avoure" into
// "avore" — six wrong rewrites against one right one across the shipped word lists.
//
// The trailing "s" is accepted so a plural contracts too ("colours" to "colors").
var suffixContractions = [][2]string{
	{"our", "or"},
}

// minContractionLen is the shortest word the ligature contractions will rewrite. Measured
// against the shipped word lists, the threshold gives up exactly one correct resolution
// ("caerule" to "cerule") and removes roughly fifteen wrong ones. The words it still serves
// are the long scientific and technical spellings the rule is for — "gynaecocratic",
// "hypaesthesia", "spelaeological", "homoeomorph", "phaenotype", "oecologist",
// "somaesthesia", "quaestionary".
const minContractionLen = 8

// variantCandidates returns spelling variants of word produced by known
// orthographic correspondences, excluding the word itself. As with candidateStems,
// over-generation is safe: the caller keeps only candidates that are real headwords.
func variantCandidates(word string) []string {
	var out []string
	seen := map[string]struct{}{word: {}}
	add := func(s string) {
		if s == "" || len(s) < minStemLen {
			return
		}
		if _, dup := seen[s]; dup {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	for _, p := range orthographicPairs {
		if strings.Contains(word, p[0]) {
			add(strings.ReplaceAll(word, p[0], p[1]))
		}
		if strings.Contains(word, p[1]) {
			add(strings.ReplaceAll(word, p[1], p[0]))
		}
	}
	if len(word) >= minContractionLen {
		for _, p := range orthographicContractions {
			if strings.Contains(word, p[0]) {
				add(strings.ReplaceAll(word, p[0], p[1]))
			}
		}
	}
	for _, p := range suffixContractions {
		// Rewrite the ending in place, allowing a plural "s" after it.
		for _, tail := range [2]string{"", "s"} {
			if hasSuffix(word, p[0]+tail) {
				add(word[:len(word)-len(p[0])-len(tail)] + p[1] + tail)
			}
		}
	}
	return out
}
