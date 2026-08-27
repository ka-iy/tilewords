// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import "strings"

// This file classifies the two kinds of Wiktionary sense that say nothing about the word
// a player actually formed:
//
//   - A redirect, whose whole text names another word ("Alternative form of argh.",
//     "Synonym of nomogram.", "UK standard spelling of analyze."). The build replaces it
//     with an inflection edge to the word it names, so the game answers with that word's
//     real definitions instead of repeating the pointer back at the player.
//   - An initialism, whose text expands the word's letters into a longer term
//     ("Initialism of counselor-in-training."). That is dropped outright; see
//     initialismPrefixes for why it is not redirected.
//
// A gloss counts as a redirect only when ALL of it is the redirect. Its last two words
// must be "of" or "for" followed by a single all-letter target, and every word before
// them must be a recognised qualifier, at least one of which names the kind of redirect.
// Anchoring at both ends is what keeps ordinary definitions built from the same words
// out: "A classical form of dance", "In the form of heat" and "One who plays a form of
// football" each fail on a leading word that is no qualifier, and "South of France" on
// having no kind word at all. The cost of the anchoring is a redirect dressed up with
// prose ("The ordinal form of myriad") staying as it is, which is the safe direction to
// err in — a kept redirect is merely unhelpful, whereas a mis-parsed definition would
// answer a played word with an unrelated one.

// redirectKinds are the nouns that name what kind of redirect a gloss is. A gloss needs
// one of them to be a redirect at all, which is what separates "Alternative form of argh"
// from a phrase like "South of France" built only from qualifiers.
var redirectKinds = map[string]bool{
	// Variant spellings and forms, the bulk of them.
	"form": true, "spelling": true, "misspelling": true, "typography": true,
	"synonym": true, "eggcorn": true,
	// Shortenings. An initialism is deliberately absent — see initialismPrefixes.
	"abbreviation": true, "clipping": true, "contraction": true, "ellipsis": true,
	"apocope": true, "short": true,
	// Inflections, for the entries where Wiktionary spells the inflection out in the
	// gloss instead of tagging the sense (which formTarget handles).
	"plural": true, "singular": true, "participle": true, "gerund": true, "indicative": true,
}

// redirectQualifiers are the words a gloss may use ahead of its kind word. Every word
// before the target has to be one of these or a kind word, so an ordinary definition
// cannot be mistaken for a redirect.
var redirectQualifiers = map[string]bool{
	// How current or standard the form is.
	"alternative": true, "alternate": true, "variant": true, "standard": true,
	"nonstandard": true, "non-standard": true, "obsolete": true, "archaic": true,
	"dated": true, "rare": true, "uncommon": true, "common": true, "superseded": true,
	"proscribed": true, "informal": true, "formal": true, "colloquial": true,
	"dialectal": true, "regional": true, "early": true, "modern": true,
	// How the form was arrived at.
	"deliberate": true, "elongated": true, "shortened": true, "aphetic": true,
	"apocopic": true, "syllabic": true, "censored": true, "filter-avoidance": true,
	"eye": true, "dialect": true, "letter-case": true, "pronunciation": true,
	"feminist": true, "euphemistic": true, "humorous": true, "egyptological": true,
	// Where the form is used.
	"us": true, "uk": true, "american": true, "america": true, "british": true,
	"britain": true, "english": true, "oxford": true, "non-oxford": true,
	"canada": true, "canadian": true, "commonwealth": true, "ireland": true,
	"irish": true, "australia": true, "australian": true, "new": true, "zealand": true,
	"south": true, "africa": true, "african-american": true, "vernacular": true,
	"india": true, "indian": true, "malaysia": true, "singapore": true,
	"philippines": true, "scottish": true, "scots": true, "northern": true,
	// Which inflection the form is.
	"simple": true, "past": true, "present": true, "third-person": true,
	"second-person": true, "first-person": true, "comparative": true,
	"superlative": true, "infinitive": true, "preterite": true, "subjunctive": true,
	"imperative": true,
	// Joins two of the above.
	"and": true, "or": true,
}

// redirectTarget reports whether gloss is nothing but a redirect to another word, and
// returns that word in lower case.
//
// The target must be a single unhyphenated word, because that is what the DB is keyed by:
// "Abbreviation of air-conditioned" names something no lookup could reach, so it is left
// alone as a gloss rather than turned into an edge that could never resolve.
func redirectTarget(gloss string) (string, bool) {
	fields := strings.Fields(strings.TrimSuffix(strings.TrimSpace(gloss), "."))
	// The shortest redirect is a kind word, a connective and the target.
	if len(fields) < 3 {
		return "", false
	}
	if c := strings.ToLower(fields[len(fields)-2]); c != "of" && c != "for" {
		return "", false
	}
	target := strings.ToLower(fields[len(fields)-1])
	if !isAlpha(target) {
		return "", false
	}

	named := false
	for _, f := range fields[:len(fields)-2] {
		// Wiktionary punctuates the qualifier run — "(Canadian spelling, common) present
		// participle and gerund of fuel" — so the brackets and separators come off before
		// a word is recognised.
		w := strings.ToLower(strings.Trim(f, "(),"))
		if w == "" {
			continue
		}
		switch {
		case redirectKinds[w]:
			named = true
		case redirectQualifiers[w]:
		default:
			return "", false
		}
	}
	if !named {
		return "", false
	}
	return target, true
}

// maxRedirectHops bounds how far a chain of redirects is followed. A redirect to another
// redirect is ordinary (a variant spelling of a variant spelling); a chain longer than
// this is not, and is abandoned rather than walked.
const maxRedirectHops = 8

// joinRedirect renders a redirect gloss followed by the definition it points at. The
// redirect's full stop goes, so the colon reads as the join it is.
func joinRedirect(redirect, definition string) string {
	return strings.TrimRight(redirect, ".") + ": " + definition
}

// initialismPrefixes begin a gloss that defines a word only as the initial letters of a
// longer term. Such a sense is dropped rather than redirected to the term it expands to:
// an initialism is not itself an English word, so the expansion is not what the player
// formed, and the few initialisms that did enter the language as words (scuba, laser)
// carry ordinary senses of their own that the drop leaves untouched.
var initialismPrefixes = []string{"initialism of ", "acronym of "}

// isInitialismGloss reports whether gloss defines its word only as an initialism.
func isInitialismGloss(gloss string) bool {
	lower := strings.ToLower(strings.TrimSpace(gloss))
	for _, p := range initialismPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}
