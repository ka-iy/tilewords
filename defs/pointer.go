// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import "strings"

// This file classifies the kinds of Wiktionary sense that say nothing about the word a
// player actually formed:
//
//   - A redirect, whose whole text names another word ("Alternative form of argh.",
//     "Synonym of nomogram.", "UK standard spelling of analyze."). The build replaces it
//     with an inflection edge to the word it names, so the game answers with that word's
//     real definitions instead of repeating the pointer back at the player.
//   - An initialism, whose text expands the word's letters into a longer term
//     ("Initialism of counselor-in-training."). That is dropped outright; see
//     initialismPrefixes for why it is not redirected.
//   - An abbreviation, whose text gives the longer term the word is a written short form
//     of ("Abbreviation of postgraduate."). That is dropped outright too; see
//     abbreviationMarker.
//   - A letter-case pointer, whose text names the same word under a different
//     capitalisation ("Alternative letter-case form of ELINT."). Dropped outright as
//     well; see letterCaseMarker.
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
	// Shortenings. An initialism and an abbreviation are deliberately absent — a gloss
	// of either kind is dropped before it can be classified, see initialismPrefixes and
	// abbreviationMarker.
	"clipping": true, "contraction": true, "ellipsis": true,
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
	// How the form was arrived at. A letter-case pointer is deliberately absent — such a
	// gloss is dropped before it can be classified, see letterCaseMarker.
	"deliberate": true, "elongated": true, "shortened": true, "aphetic": true,
	"apocopic": true, "syllabic": true, "censored": true, "filter-avoidance": true,
	"eye": true, "dialect": true, "pronunciation": true,
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
// "Ellipsis of air-conditioned unit" names something no lookup could reach, so it is left
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

// abbreviationMarker is the phrase Wiktionary uses to say that a word is the written
// short form of a longer term: "Abbreviation of postgraduate.", "Syllabic abbreviation of
// parallax second.", "An early TV camera tube. (Abbreviation of orthoconoscope)."
//
// A sense carrying it is dropped rather than kept or redirected, for the reason
// initialismPrefixes gives: an abbreviation is not a play, so naming the term it stands
// for does not tell a player what the word they formed means. The word itself is often
// still a legal play — parsec, forex, postgrad — and those words are given a real
// definition by the supplemental glossary instead.
const abbreviationMarker = "abbreviation of"

// IsAbbreviationGloss reports whether gloss defines its word as an abbreviation.
//
// The marker is looked for ANYWHERE in the gloss, not just at its start, because
// Wiktionary qualifies it ("Syllabic abbreviation of parallax second"), joins it to a
// pointer ("Alternative form of sALS; Abbreviation of sporadic amyotrophic lateral
// sclerosis") and parenthesises it after a definition. The cost of matching that widely
// is that an ordinary definition which happens to use the phrase — "initials: An
// abbreviation of a person's name" — goes with them; every word this was measured to
// affect keeps other senses or is given a glossary definition.
//
// It is exported because the same rule has to hold for every source the asset is built
// from, not only the Wiktionary parse in this package: mergedefs applies it to the
// supplemental dictionaries.
func IsAbbreviationGloss(gloss string) bool {
	return strings.Contains(strings.ToLower(gloss), abbreviationMarker)
}

// letterCaseMarker is the phrase Wiktionary uses for a word that differs from another
// only in capitalisation: "Alternative letter-case form of ELINT.", "Alternative
// letter-case form of Alpine: Of, relating to, or inhabiting the Alps."
//
// A sense carrying it is dropped. The DB is keyed by lowercase words, so the word such a
// pointer names is always the played word itself: nothing can ever be joined onto it, and
// what the player is shown is a pointer back at the word they just played. The words this
// affects are mostly a proper noun or trade name that has entered general use — november,
// quebec, novocaine, vegemite — and those are given a real definition by the supplemental
// glossary instead.
const letterCaseMarker = "letter-case form of"

// IsLetterCaseGloss reports whether gloss defines its word only as another capitalisation
// of itself. Like IsAbbreviationGloss it looks for the marker anywhere in the text, and is
// exported for the same reason: the rule holds for every source, not just this package's
// Wiktionary parse.
func IsLetterCaseGloss(gloss string) bool {
	return strings.Contains(strings.ToLower(gloss), letterCaseMarker)
}
