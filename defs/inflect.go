package defs

// minStemLen is the shortest candidate a de-inflection rule will emit. Candidates
// shorter than this (e.g. "a" from stripping the "s" of "as") are never real
// lemmas and only risk spurious matches.
const minStemLen = 2

// candidateStems returns lowercase lemma candidates for an inflected word, most
// likely first, excluding the word itself. Each rule undoes a common English
// inflection; the caller keeps only candidates that are real headwords, so
// over-generating here is harmless (a candidate that is not a headword is dropped).
//
// The rules cover:
//
//   - Noun plurals and third-person verbs (-s, -es, -ies, -ves).
//   - Past tense and participles (-ed, -ied).
//   - Gerunds and present participles (-ing).
//   - Comparatives and superlatives (-er, -est, -ier, -iest).
//   - Adverbs (-ly, -ily) and the noun suffix -ness.
//
// Consonant doubling ("running" to "run") and silent-e restoration ("baking" to
// "bake") are both attempted where they apply.
func candidateStems(word string) []string {
	var out []string
	seen := map[string]struct{}{word: {}}
	add := func(s string) {
		if len(s) < minStemLen {
			return
		}
		if _, dup := seen[s]; dup {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	n := len(word)
	switch {
	case hasSuffix(word, "ies") && n > 4:
		add(word[:n-3] + "y")  // "berries" -> "berry"
		add(word[:n-3] + "ie") // "movies" -> "movie"
	case hasSuffix(word, "ves") && n > 4:
		add(word[:n-3] + "f")  // "wolves" -> "wolf"
		add(word[:n-3] + "fe") // "knives" -> "knife"
	case hasSuffix(word, "es") && n > 3:
		add(word[:n-2]) // "boxes" -> "box"
		add(word[:n-1]) // "houses" -> "house"
	case hasSuffix(word, "s") && !hasSuffix(word, "ss") && n > 2:
		add(word[:n-1]) // "cats" -> "cat"
	}

	if hasSuffix(word, "ied") && n > 4 {
		add(word[:n-3] + "y") // "cried" -> "cry"
	}
	if hasSuffix(word, "ed") && n > 3 {
		base := word[:n-2]
		add(base)           // "walked" -> "walk"
		add(base + "e")     // "baked" -> "bake"
		add(undouble(base)) // "hopped" -> "hop"
	}
	if hasSuffix(word, "ing") && n > 4 {
		base := word[:n-3]
		add(base)           // "walking" -> "walk"
		add(base + "e")     // "baking" -> "bake"
		add(undouble(base)) // "running" -> "run"
	}

	if hasSuffix(word, "iest") && n > 5 {
		add(word[:n-4] + "y") // "happiest" -> "happy"
	}
	if hasSuffix(word, "est") && n > 4 {
		base := word[:n-3]
		add(base)
		add(base + "e") // "nicest" -> "nice"
		add(undouble(base))
	}
	if hasSuffix(word, "ier") && n > 4 {
		add(word[:n-3] + "y") // "happier" -> "happy"
	}
	if hasSuffix(word, "er") && n > 4 {
		base := word[:n-2]
		add(base)
		add(base + "e") // "nicer" -> "nice"
		add(undouble(base))
	}

	if hasSuffix(word, "ily") && n > 4 {
		add(word[:n-3] + "y") // "happily" -> "happy"
	}
	if hasSuffix(word, "ly") && n > 3 {
		add(word[:n-2]) // "quickly" -> "quick"
	}

	if hasSuffix(word, "iness") && n > 6 {
		add(word[:n-5] + "y") // "happiness" -> "happy"
	}
	if hasSuffix(word, "ness") && n > 5 {
		add(word[:n-4]) // "sadness" -> "sad"
	}

	addClassicalPluralStems(word, add)
	return out
}

// addClassicalPluralStems adds lemma candidates for the Greek- and Latin-derived
// plurals common in scientific and technical vocabulary, undoing:
//
//   - "-ae" to "-a" or "-e" ("acanthae" to "acantha", "acalephae" to "acalephe").
//   - "-es" to "-is" ("aerotaxes" to "aerotaxis", "agamogeneses" to "agamogenesis").
//   - "-ata" to "-a" ("aerenchymata" to "aerenchyma", "stomata" to "stoma").
//   - "-i" to "-us" ("cacti" to "cactus", "fungi" to "fungus").
//   - "-a" to "-um" or "-on" ("addenda" to "addendum", "phenomena" to "phenomenon").
func addClassicalPluralStems(word string, add func(string)) {
	n := len(word)
	switch {
	case hasSuffix(word, "ae") && n > 3:
		add(word[:n-1])       // "-ae" -> "-a"
		add(word[:n-2] + "e") // "-ae" -> "-e"
	case hasSuffix(word, "ata") && n > 4:
		add(word[:n-2]) // "-ata" -> "-a"
	case hasSuffix(word, "es") && n > 4:
		add(word[:n-2] + "is") // "-es" -> "-is"
	case hasSuffix(word, "i") && n > 3:
		add(word[:n-1] + "us") // "-i" -> "-us"
	case hasSuffix(word, "a") && n > 3:
		add(word[:n-1] + "um") // "-a" -> "-um"
		add(word[:n-1] + "on") // "-a" -> "-on"
	}
}

// undouble collapses a trailing doubled consonant, mapping the spelling change
// English applies before "-ed"/"-ing"/"-er" back to the base ("hopp" to "hop").
// It leaves a word ending in a doubled non-consonant or a single letter unchanged.
func undouble(s string) string {
	n := len(s)
	if n < 2 {
		return s
	}
	last := s[n-1]
	if last == s[n-2] && isConsonant(last) {
		return s[:n-1]
	}
	return s
}

// isConsonant reports whether b is a lowercase ASCII consonant.
func isConsonant(b byte) bool {
	if b < 'a' || b > 'z' {
		return false
	}
	return !isVowel(b)
}

// isVowel reports whether b is a lowercase ASCII vowel (excluding y, which is
// treated as a consonant for the doubling rule).
func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

// hasSuffix reports whether s ends with suffix. It avoids importing strings for
// this hot path used across every de-inflection candidate.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
