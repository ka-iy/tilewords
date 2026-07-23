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
// digraph or extra letter) reduces to the right form. Only the contraction is
// applied: expanding every "e" to "ae" or every "or" to "our" would generate far
// too many spurious candidates.
var orthographicContractions = [][2]string{
	{"ae", "e"},
	{"oe", "e"},
	{"our", "or"},
}

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
	for _, p := range orthographicContractions {
		if strings.Contains(word, p[0]) {
			add(strings.ReplaceAll(word, p[0], p[1]))
		}
	}
	return out
}
