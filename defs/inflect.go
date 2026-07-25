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
//
// Order matters: the caller takes the FIRST candidate that is a headword, so when a word
// yields two candidates that are both headwords, whichever comes first wins. Silent-e
// restoration is therefore offered before the bare stem for -ed/-ing/-er/-est. English drops
// a silent e before those suffixes, so a stem that still ends in a consonant reaches them by
// doubling it instead — which undouble already covers. Offering the bare stem first resolves
// "used" to "us" rather than "use" and "reded" to "red" rather than "rede", picking an
// unrelated headword that merely happens to exist.
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
	// Plural and third-person forms. The spelling-change readings are offered before the
	// plain "-es"/"-s" ones, but they do NOT exclude them: a word ending in "ves" or "ies"
	// is just as often an ordinary "-es" plural of a stem that itself ends in "v" or "i".
	// Treating these as alternatives rather than as a switch is what lets "aboves" reach
	// "above" — read only as a "-ves" plural it yields "abof", and nothing else was tried.
	if hasSuffix(word, "ies") && n > 4 {
		add(word[:n-3] + "y")  // "berries" -> "berry"
		add(word[:n-3] + "ie") // "movies" -> "movie"
	}
	if hasSuffix(word, "ves") && n > 4 {
		add(word[:n-3] + "f")  // "wolves" -> "wolf"
		add(word[:n-3] + "fe") // "knives" -> "knife"
	}
	switch {
	case hasSuffix(word, "es") && n > 3:
		// English inserts the "e" only after a sibilant ("box" to "boxes", "church" to
		// "churches"); a stem already ending in "e" just takes "s" ("barde" to "bardes").
		// So which candidate to offer first depends on what the bare form ends in — trimming
		// both letters unconditionally first resolves "bardes" to "bard" and "agapes" to
		// "agap", each a real headword but the wrong one.
		if endsInSibilant(word[:n-2]) {
			add(word[:n-2]) // "boxes" -> "box"
			add(word[:n-1])
		} else {
			add(word[:n-1]) // "bardes" -> "barde"
			add(word[:n-2])
		}
	case hasSuffix(word, "s") && !hasSuffix(word, "ss") && n > 2:
		add(word[:n-1]) // "cats" -> "cat"
	}

	if hasSuffix(word, "ied") && n > 4 {
		add(word[:n-3] + "y") // "cried" -> "cry"
	}
	if hasSuffix(word, "ed") && n > 3 {
		base := word[:n-2]
		add(base + "e")     // "baked" -> "bake"
		add(base)           // "walked" -> "walk"
		add(undouble(base)) // "hopped" -> "hop"
	}
	if hasSuffix(word, "ing") && n > 4 {
		base := word[:n-3]
		add(base + "e")     // "baking" -> "bake"
		add(base)           // "walking" -> "walk"
		add(undouble(base)) // "running" -> "run"
	}

	if hasSuffix(word, "iest") && n > 5 {
		add(word[:n-4] + "y") // "happiest" -> "happy"
	}
	if hasSuffix(word, "est") && n > 4 {
		base := word[:n-3]
		add(base + "e") // "nicest" -> "nice"
		add(base)
		add(undouble(base))
	}
	if hasSuffix(word, "ier") && n > 4 {
		add(word[:n-3] + "y") // "happier" -> "happy"
	}
	if hasSuffix(word, "er") && n > 4 {
		base := word[:n-2]
		add(base + "e") // "nicer" -> "nice"
		add(base)
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
	addDerivationStems(word, add)
	return out
}

// minDerivedBase is the shortest base the derivation rules will propose. Those rules trim or
// swap a suffix rather than undo an inflection, so a short result is far more often an
// unrelated word than the same one: dropping the final e of "frae" (Scots "from") reaches
// "fra" (a friar), of "idee" reaches "ide" (a fish), of "cide" reaches "cid" (a lord). Every
// wrong pairing found while measuring these rules against the shipped word lists had a base of
// three letters or fewer, while the longer bases were right — "gentlenesse" to "gentleness",
// "riboflavine" to "riboflavin", "pterodactyle" to "pterodactyl".
const minDerivedBase = 4

// addDerivationStems adds candidates for words derived from a base by a suffix, rather than
// inflected from it. They are offered after the inflectional rules above because they are the
// weaker inference: an inflected form IS its lemma, whereas a derived word merely shares a
// root, so the base's definition explains it without being a definition of it.
//
// Included are only the families measured to be reliable — a variant spelling that gains or
// loses a silent final e, and suffixes whose base gloss genuinely explains the derived word:
//
//   - "-e"/"" either way: an archaic or dialect spelling, or the -ine/-in chemical pair
//     ("gentlenesse", "riboflavine", "alkalin").
//   - "-al", "-ical", "-ically": adjective forms of the same term ("albitical" to "albitic").
//   - "-able", "-ible": "capable of being X-ed" ("alkalisable" to "alkalise").
//   - "-ous": "having or pertaining to X" ("myriapodous" to "myriapod").
//   - "-ment": "the act or result of X-ing" ("appetisement" to "appetise").
//   - "-ish", "-like", "-ful": "resembling" or "full of X" ("werwolfish", "hoblike").
//
// Deliberately excluded, having been measured as unreliable or misleading:
//
//   - "-ist", "-ism": these name a person or a doctrine, not the root — "aerobicist" is not
//     "aerobic", and "breist" (Scots for breast) is not "bree".
//   - "-ity": mixed at best ("aminity" is not "amin", "perseity" is not "perse").
//   - "-less": the suffix negates the base, so showing the base's gloss states the opposite of
//     what the word means — "chaiseless" would be explained as "a light open carriage".
func addDerivationStems(word string, add func(string)) {
	n := len(word)
	// Only propose a base that is long enough to be the same word; see minDerivedBase.
	addLong := func(s string) {
		if len(s) >= minDerivedBase {
			add(s)
		}
	}
	// A variant spelling differing only by a silent final e, in either direction.
	if hasSuffix(word, "e") {
		addLong(word[:n-1])
	} else if n >= minDerivedBase {
		addLong(word + "e")
	}

	trim := func(suffix string, replacements ...string) {
		if !hasSuffix(word, suffix) || n <= len(suffix)+2 {
			return
		}
		stem := word[:n-len(suffix)]
		for _, r := range replacements {
			addLong(stem + r)
		}
	}
	// Where a suffix begins with a vowel, English drops the base's silent e before it
	// ("alkalise" plus "-able" gives "alkalisable"), so the e-restored base is offered ahead
	// of the bare stem for the same reason as in candidateStems: the bare stem otherwise wins
	// whenever it happens to be a word too, resolving "alkalisable" to "alkalis" — hence
	// "alkali" — instead of "alkalise".
	trim("ically", "ic", "y")
	trim("ical", "ic", "y")
	trim("ally", "al", "e", "")
	trim("al", "e", "")
	trim("able", "e", "", "ate")
	trim("ible", "e", "")
	trim("ously", "ous", "e", "")
	trim("ous", "e", "", "y")
	trim("ment", "e", "")
	trim("ish", "e", "")
	trim("like", "")
	trim("ful", "e", "")
}

// minClassicalLen is the shortest word the ending-replacement classical-plural rules will
// rewrite. Those rules swap one ending for another rather than trimming, so unlike the
// trimming rules they can land on a real but wholly unrelated headword — and short words are
// where that goes wrong, because a four- or five-letter word ending in -a, -ae or -i is far
// more often an ordinary word than a Latin plural.
//
// The threshold is a measured precision trade, not a clean separation. Across the shipped
// word lists, holding these rules back below six letters gives up 18 resolutions: 13 were
// wrong ("acta" to "acton", a padded jacket; "ursa" to "urson", a porcupine; "frae", Scots
// for "from", to "fra", a friar; "targa" to "targum"; "hansa" to "hanson") and 5 were right
// ("animi" to "animus", "areae" to "area", "micra" to "micron", "culti" to "cultus", "momi"
// to "momus"). Those five now report no definition rather than a guess.
//
// A threshold of five was measured too: it recovers those five but readmits four wrong
// rewrites, i.e. coin-flip precision. Six is chosen because a confidently wrong definition is
// worse than an honest "no definition" — the same judgement the offline merge tool makes when
// it accepts these rewrites only under its -fuzzy flag. Longer words are unaffected, and
// keep resolutions like "cementa" to "cementum", "acanthae" to "acantha" and "amphisbaenae"
// to "amphisbaena".
const minClassicalLen = 6

// addClassicalPluralStems adds lemma candidates for the Greek- and Latin-derived
// plurals common in scientific and technical vocabulary, undoing:
//
//   - "-ae" to "-a" or "-e" ("acanthae" to "acantha", "acalephae" to "acalephe").
//   - "-es" to "-is" ("aerotaxes" to "aerotaxis", "agamogeneses" to "agamogenesis").
//   - "-ata" to "-a" ("aerenchymata" to "aerenchyma", "stomata" to "stoma").
//   - "-i" to "-us" ("cacti" to "cactus", "fungi" to "fungus").
//   - "-a" to "-um" or "-on" ("addenda" to "addendum", "phenomena" to "phenomenon").
//
// The rules that replace an ending rather than trim one are limited to words of at least
// minClassicalLen; see that constant for why. The trimming rules ("-ata", "-es") are not, as
// they cannot reach an unrelated stem the same way.
func addClassicalPluralStems(word string, add func(string)) {
	n := len(word)
	long := n >= minClassicalLen
	switch {
	case hasSuffix(word, "ae") && long:
		add(word[:n-1])       // "-ae" -> "-a"
		add(word[:n-2] + "e") // "-ae" -> "-e"
	case hasSuffix(word, "ata") && n > 4:
		add(word[:n-2]) // "-ata" -> "-a"
	case hasSuffix(word, "es") && n > 4:
		add(word[:n-2] + "is") // "-es" -> "-is"
	case hasSuffix(word, "i") && long:
		add(word[:n-1] + "us") // "-i" -> "-us"
	case hasSuffix(word, "a") && long:
		add(word[:n-1] + "um") // "-a" -> "-um"
		add(word[:n-1] + "on") // "-a" -> "-on"
	}
}

// endsInSibilant reports whether s ends in one of the sounds before which English inserts an
// "e" to form the plural or third-person singular: s, x, z, ch or sh.
func endsInSibilant(s string) bool {
	if hasSuffix(s, "ch") || hasSuffix(s, "sh") {
		return true
	}
	n := len(s)
	if n == 0 {
		return false
	}
	switch s[n-1] {
	case 's', 'x', 'z':
		return true
	default:
		return false
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
