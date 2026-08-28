// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedirectTargetAcceptsWholeGlossRedirects(t *testing.T) {
	cases := []struct{ gloss, want string }{
		{"Alternative form of argh.", "argh"},
		{"Synonym of nomogram.", "nomogram"},
		{"Obsolete spelling of acre.", "acre"},
		{"Commonwealth and Ireland standard spelling of color.", "color"},
		{"US, Canada, and Oxford British English standard spelling of aluminise.", "aluminise"},
		{"(Canadian spelling, common) present participle and gerund of fuel", "fuel"},
		{"plural of bap", "bap"},
		{"Deliberate misspelling of penis.", "penis"},
		{"Short for microphone.", "microphone"},
	}
	for _, c := range cases {
		got, ok := redirectTarget(c.gloss)
		if !ok || got != c.want {
			t.Errorf("redirectTarget(%q) = %q,%v; want %q,true", c.gloss, got, ok, c.want)
		}
	}
}

// TestRedirectTargetRejectsOrdinaryDefinitions covers the definitions built from the same
// words a redirect uses. Mistaking one for a redirect would answer the word a player formed
// with an unrelated word's definition, which is worse than showing the gloss as it stands.
func TestRedirectTargetRejectsOrdinaryDefinitions(t *testing.T) {
	for _, gloss := range []string{
		"A classical form of dance.",
		"In the form of heat.",
		"One who plays a form of football.",
		"Having the shape or form of coral.",
		"South of France.",
		"The act of acceding.",
		"A native or inhabitant of Wales.",
		"A small feline.",
		// The term is not a single word, so no lookup could reach it.
		"Ellipsis of air-conditioned unit.",
		// An abbreviation is no longer a kind of redirect: such a sense is dropped
		// before it is classified, so nothing may turn it into an edge.
		"Abbreviation of postgraduate.",
		// Nor is a letter-case pointer, for the same reason.
		"Alternative letter-case form of ELINT.",
		// Two redirects in one gloss: which word the entry stands for is ambiguous.
		"Obsolete form of jambeaux, plural of jambeau.",
		// A redirect dressed up with prose is left alone; see the file comment.
		"The ordinal form of myriad.",
	} {
		if got, ok := redirectTarget(gloss); ok {
			t.Errorf("redirectTarget(%q) = %q,true; want false", gloss, got)
		}
	}
}

func TestIsInitialismGloss(t *testing.T) {
	for _, gloss := range []string{
		"Initialism of counselor-in-training.",
		"Acronym of cyclooxygenase.",
		// Reads as a redirect to "payment"; the initialism test runs first so it cannot.
		"Initialism of form of payment.",
	} {
		if !isInitialismGloss(gloss) {
			t.Errorf("isInitialismGloss(%q) = false; want true", gloss)
		}
	}
	for _, gloss := range []string{
		"Abbreviation of abstract.",
		"An apparatus for breathing underwater.",
		"Alternative form of argh.",
	} {
		if isInitialismGloss(gloss) {
			t.Errorf("isInitialismGloss(%q) = true; want false", gloss)
		}
	}
}

func TestIsAbbreviationGloss(t *testing.T) {
	for _, gloss := range []string{
		"Abbreviation of postgraduate.",
		// The marker is qualified, so it does not begin the gloss.
		"Syllabic abbreviation of parallax second.",
		// A definition with the marker parenthesised after it.
		"An early TV camera tube. (Abbreviation of orthoconoscope).",
		// A pointer with the marker joined onto it.
		"Alternative form of sALS; Abbreviation of sporadic amyotrophic lateral sclerosis.",
		// Wiktionary writes the marker in either case.
		"abbreviation of carabiner",
	} {
		if !IsAbbreviationGloss(gloss) {
			t.Errorf("IsAbbreviationGloss(%q) = false; want true", gloss)
		}
	}
	for _, gloss := range []string{
		"Initialism of counselor-in-training.",
		"Clipping of ketogenic.",
		"A classical form of dance.",
		"An apparatus for breathing underwater.",
	} {
		if IsAbbreviationGloss(gloss) {
			t.Errorf("IsAbbreviationGloss(%q) = true; want false", gloss)
		}
	}
}

func TestIsLetterCaseGloss(t *testing.T) {
	for _, gloss := range []string{
		"Alternative letter-case form of ELINT.",
		// The marker sits ahead of a sub-gloss rather than ending the text.
		"Alternative letter-case form of Alpine: Of, relating to, or inhabiting the Alps.",
		"alternative letter-case form of november",
	} {
		if !IsLetterCaseGloss(gloss) {
			t.Errorf("IsLetterCaseGloss(%q) = false; want true", gloss)
		}
	}
	for _, gloss := range []string{
		"Alternative form of argh.",
		"Alternative spelling of estivated.",
		"A classical form of dance.",
	} {
		if IsLetterCaseGloss(gloss) {
			t.Errorf("IsLetterCaseGloss(%q) = true; want false", gloss)
		}
	}
}

// writeFixture writes lines to a file in dir and returns its path.
func writeFixture(t *testing.T, dir, name string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// kaikkiLine renders one extract line: an English entry with the given senses.
func kaikkiLine(word, pos string, glosses ...string) string {
	var b strings.Builder
	b.WriteString(`{"word":"` + word + `","pos":"` + pos + `","lang_code":"en","senses":[`)
	for i, g := range glosses {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"glosses":["` + g + `"]}`)
	}
	b.WriteString(`]}`)
	return b.String()
}

// kaikkiFormLine renders an extract line for an inflected form: a sense Wiktionary tags as
// a form of lemma, which the build turns into an inflection edge rather than a definition.
func kaikkiFormLine(word, pos, gloss, lemma string) string {
	return `{"word":"` + word + `","pos":"` + pos + `","lang_code":"en","senses":[` +
		`{"glosses":["` + gloss + `"],"tags":["form-of","plural"],"form_of":[{"word":"` + lemma + `"}]}]}`
}

// buildFixtureDB runs a whole build over a hand-written extract, so the redirect pass is
// exercised the way builddefs drives it rather than through its parts.
func buildFixtureDB(t *testing.T) (*DB, *Report) {
	t.Helper()
	dir := t.TempDir()
	extract := writeFixture(t, dir, "extract.jsonl", []string{
		kaikkiLine("color", "noun", "The spectral composition of visible light."),
		kaikkiLine("colour", "noun", "Commonwealth and Ireland standard spelling of color."),
		kaikkiLine("colorize", "verb", "To add colour to."),
		kaikkiLine("colourize", "verb", "Standard spelling of colorize."),
		kaikkiLine("colourise", "verb", "Non-Oxford British English standard spelling of colourize."),
		// "aby" is absent from the extract, so this redirect reaches nothing.
		kaikkiLine("abegge", "verb", "Obsolete form of aby."),
		// A letter-case pointer, which names the word itself once the case distinction
		// is gone and so can never be joined to anything.
		kaikkiLine("abelian", "adj", "Alternative letter-case form of abelian."),
		kaikkiLine("scuba", "noun", "Initialism of self-contained underwater breathing apparatus.",
			"An apparatus for breathing underwater."),
		kaikkiLine("zyzzyt", "noun", "Initialism of zeta y zeta."),
		// An abbreviation of a term that IS a single word: nothing may turn it into a
		// redirect to that term, and the word is left with no definition at all.
		kaikkiLine("postgrad", "noun", "Abbreviation of postgraduate."),
		// The marker in the middle of an otherwise ordinary definition. The sense goes
		// with the rest, and the word keeps the senses that do not carry it.
		kaikkiLine("macro", "noun", "Very large in scale or scope.",
			"A human-friendly abbreviation of complex input to a computer program."),
		// Points at "postgrad", which the abbreviation rule empties, so the pointer is
		// orphaned and goes too.
		kaikkiLine("pgrad", "noun", "Alternative form of postgrad."),
		// A pointer at itself, and a word pointing at that: the first can never be joined,
		// and removing it orphans the second in turn.
		kaikkiLine("esports", "noun", "Alternative form of esports."),
		kaikkiLine("cyberathletics", "noun", "Synonym of esports."),
		kaikkiLine("ballet", "noun", "A classical form of dance."),
		// An inflection whose lemma is itself only a redirect: ceca -> cecum -> caecum.
		kaikkiLine("caecum", "noun", "A blind pouch of the intestine."),
		kaikkiLine("cecum", "noun", "Alternative form of caecum."),
		kaikkiFormLine("ceca", "noun", "plural of cecum", "cecum"),
		// A pointer at an inflected form. The form is what the extract named, so the
		// lookup follows the edge and answers with the lemma's definition. "estivate" is
		// left out of the word list below: it reaches the asset as the lemma "estivated"
		// resolves to.
		kaikkiLine("estivate", "verb", "To spend the summer in a dormant state."),
		kaikkiFormLine("estivated", "verb", "past participle of estivate", "estivate"),
		kaikkiLine("aestivated", "verb", "Alternative spelling of estivated."),
		// A pointer written in one part of speech at a headword whose primary sense is
		// another. Both senses are shown, each labelled, rather than one being chosen.
		kaikkiLine("anemic", "adj", "Of, pertaining to, or suffering from anemia."),
		kaikkiLine("anemic", "noun", "A person who has anemia."),
		kaikkiLine("anaemic", "noun", "Alternative spelling of anemic."),
		// The same pointer written in both parts of speech. The adjective sense shows the
		// target's primary gloss, so the noun sense shows only the noun one.
		kaikkiLine("anaimic", "adj", "Alternative spelling of anemic."),
		kaikkiLine("anaimic", "noun", "Alternative spelling of anemic."),
		// A pointer at a spelling nothing records: "barcas" is neither a headword nor a
		// form of one, so the pointer goes. The stem rule would still rewrite "barcae"
		// itself to the headword "barca" for a player who typed it, which is what makes
		// the entry's own kind the thing to assert below.
		kaikkiLine("barca", "noun", "A small boat."),
		kaikkiLine("barcae", "noun", "Alternative form of barcas."),
		// "wobble" is deliberately left out of the word list below, so only the closure
		// over redirect targets can keep it in the asset.
		kaikkiLine("wibble", "verb", "Alternative form of wobble."),
		kaikkiLine("wobble", "verb", "To move erratically from side to side."),
	})
	list := writeFixture(t, dir, "words.txt", []string{
		"color", "colour", "colorize", "colourize", "colourise",
		"abegge", "abelian", "scuba", "zyzzyt", "postgrad", "pgrad", "macro", "ballet",
		"esports", "cyberathletics",
		"caecum", "cecum", "ceca", "wibble",
		"estivated", "aestivated", "barca", "barcae", "anemic", "anaemic", "anaimic",
	})

	db, report, err := BuildFilteredDB(extract, []WordList{{Name: "fixture", Path: list}})
	if err != nil {
		t.Fatalf("BuildFilteredDB: %v", err)
	}
	return db, report
}

// TestBuildJoinsRedirectsToWhatTheyPointAt is the behaviour a player sees: a word whose
// Wiktionary entry says only "standard spelling of color" is told both that and what
// "color" means, rather than being sent to look the second word up.
func TestBuildJoinsRedirectsToWhatTheyPointAt(t *testing.T) {
	db, _ := buildFixtureDB(t)

	cases := []struct{ word, wantHead, wantGloss string }{
		{"colour", "colour",
			"Commonwealth and Ireland standard spelling of color: The spectral composition of visible light."},
		{"colourize", "colourize", "Standard spelling of colorize: To add colour to."},
		// Reached through a chain: colourise -> colourize -> colorize. The definition
		// joined on is the one at the end of the chain, not the next redirect along it.
		{"colourise", "colourise",
			"Non-Oxford British English standard spelling of colourize: To add colour to."},
		{"cecum", "cecum", "Alternative form of caecum: A blind pouch of the intestine."},
	}
	for _, c := range cases {
		res, ok := db.Lookup(c.word)
		if !ok {
			t.Errorf("Lookup(%q) not found", c.word)
			continue
		}
		if res.Kind != MatchExact || res.Headword != c.wantHead {
			t.Errorf("Lookup(%q) = head %q kind %v; want head %q kind %v",
				c.word, res.Headword, res.Kind, c.wantHead, MatchExact)
			continue
		}
		if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != c.wantGloss {
			t.Errorf("Lookup(%q) senses = %v;\n want %q", c.word, res.Entry.Senses, c.wantGloss)
		}
	}
}

// TestBuildJoinsARedirectToAnInflectedForm is the mirror of the test above: there the
// inflection points at a redirect, here a redirect points at an inflection. The word the
// gloss names is not a headword, so the join follows the edge the extract recorded for it
// and answers with the lemma's definition rather than leaving the pointer bare.
func TestBuildJoinsARedirectToAnInflectedForm(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("aestivated")
	if !ok || res.Kind != MatchExact {
		t.Fatalf("Lookup(aestivated) = %+v,%v; want an exact match", res, ok)
	}
	want := "Alternative spelling of estivated: To spend the summer in a dormant state."
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != want {
		t.Errorf("Lookup(aestivated) senses = %v;\n want %q", res.Entry.Senses, want)
	}
}

// TestBuildJoinsBothSensesWhenThePartsOfSpeechDiffer covers the choice the join refuses to
// make: the pointer is written as a noun, the target's primary sense is an adjective, and
// both are shown with their own labels. Dropping either would answer half the readings of a
// word that has both.
func TestBuildJoinsBothSensesWhenThePartsOfSpeechDiffer(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("anaemic")
	if !ok || res.Kind != MatchExact {
		t.Fatalf("Lookup(anaemic) = %+v,%v; want an exact match", res, ok)
	}
	want := "Alternative spelling of anemic: (adj) Of, pertaining to, or suffering from anemia; " +
		"(noun) A person who has anemia."
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != want {
		t.Errorf("Lookup(anaemic) senses = %v;\n want %q", res.Entry.Senses, want)
	}
}

// TestBuildDoesNotRepeatASenseAnotherPointerShows is the counterpart: "anaimic" points at
// "anemic" in both parts of speech, so the adjective sense already carries the target's
// primary gloss and the noun sense carries only the noun one. Showing both in both would
// print the adjective definition twice.
func TestBuildDoesNotRepeatASenseAnotherPointerShows(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("anaimic")
	if !ok || len(res.Entry.Senses) != 2 {
		t.Fatalf("Lookup(anaimic) = %+v,%v; want two senses", res, ok)
	}
	want := []string{
		"Alternative spelling of anemic: Of, pertaining to, or suffering from anemia.",
		"Alternative spelling of anemic: A person who has anemia.",
	}
	for i, w := range want {
		if got := res.Entry.Senses[i].Gloss; got != w {
			t.Errorf("Lookup(anaimic) sense %d = %q;\n want %q", i, got, w)
		}
	}
}

// TestRedirectDefinitionStopsAtRecordedWords fixes where a join may look for its definition:
// a headword, or an inflected form the extract recorded, and nowhere else. A redirect names
// one specific word, so reaching a definition from that name by this package's own stem rule
// would put a meaning nothing recorded in front of the player as though a source had stated
// it — even though Lookup resolves the same spelling for a player who types it.
func TestRedirectDefinitionStopsAtRecordedWords(t *testing.T) {
	db := NewDB(
		map[string]*Entry{
			"barca":    {Word: "barca", Senses: []Sense{{POS: "noun", Gloss: "A small boat."}}},
			"estivate": {Word: "estivate", Senses: []Sense{{POS: "verb", Gloss: "To spend the summer dormant."}}},
		},
		map[string]Inflection{"estivated": {Lemma: "estivate", Relation: "past participle"}},
	)

	if i, ok := db.redirectEntry("barque", "barca"); !ok || db.definitionFor(i, "noun", false) != "A small boat." {
		t.Errorf("redirectEntry(barque, barca) = %d,%v; want the headword", i, ok)
	}
	if i, ok := db.redirectEntry("aestivated", "estivated"); !ok ||
		db.definitionFor(i, "verb", false) != "To spend the summer dormant." {
		t.Errorf("redirectEntry(aestivated, estivated) = %d,%v; want the lemma", i, ok)
	}
	// "barcas" is neither, though candidateStems rewrites it to the headword "barca".
	if i, ok := db.redirectEntry("barcae", "barcas"); ok {
		t.Errorf("redirectEntry(barcae, barcas) = %d,true; a stem rewrite is a guess, not a record", i)
	}
	if i, ok := db.redirectEntry("barca", "barca"); ok {
		t.Errorf("redirectEntry(barca, barca) = %d,true; a word cannot answer itself", i)
	}
}

// TestBuildResolvesAnInflectionOfARedirect guards the second hop: "ceca" is recorded as the
// plural of "cecum", and "cecum" is itself only a redirect, so the inflection must still
// land on an entry that says what the word means.
func TestBuildResolvesAnInflectionOfARedirect(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("ceca")
	if !ok || res.Kind != MatchFormOf || res.Headword != "cecum" {
		t.Fatalf("Lookup(ceca) = %+v,%v; want formof cecum", res, ok)
	}
	// The extract said how the two relate, so the asset carries it through the build.
	if res.Relation != "plural" {
		t.Errorf("Lookup(ceca) Relation = %q, want %q", res.Relation, "plural")
	}
	want := "Alternative form of caecum: A blind pouch of the intestine."
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != want {
		t.Errorf("Lookup(ceca) senses = %v;\n want %q", res.Entry.Senses, want)
	}
}

// TestBuildDropsRedirectsThatReachNothing covers a redirect naming a word the extract does
// not carry: nothing can be joined onto it, so telling a player their word is "aby" and
// stopping there says less than nothing. The sense goes, and the word is left to the
// supplemental glossary to define.
func TestBuildDropsRedirectsThatReachNothing(t *testing.T) {
	db, _ := buildFixtureDB(t)

	if res, ok := db.Lookup("abegge"); ok {
		t.Errorf("Lookup(abegge) = %+v; a pointer at a word nothing defines must not resolve", res)
	}
}

// TestBuildDropsInitialismSenses covers the second half of the rule: an initialism expansion
// is not a definition of the word played, so it is dropped — leaving a word's ordinary senses
// (scuba) and leaving nothing at all behind a word that had only the initialism.
func TestBuildDropsInitialismSenses(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("scuba")
	if !ok || res.Kind != MatchExact {
		t.Fatalf("Lookup(scuba) = %+v,%v; want an exact match", res, ok)
	}
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != "An apparatus for breathing underwater." {
		t.Errorf("Lookup(scuba) senses = %v; want only the ordinary sense", res.Entry.Senses)
	}

	if res, ok := db.Lookup("zyzzyt"); ok {
		t.Errorf("Lookup(zyzzyt) = %+v; a word defined only as an initialism must not resolve", res)
	}
}

// TestBuildDropsAbbreviationSenses covers the same rule for abbreviations: naming the term
// a word is the written short form of says nothing about the word a player formed, so the
// sense goes — taking the whole entry with it when there is nothing else (postgrad), and
// leaving the other senses behind when there is (macro).
func TestBuildDropsAbbreviationSenses(t *testing.T) {
	db, _ := buildFixtureDB(t)

	if res, ok := db.Lookup("postgrad"); ok {
		t.Errorf("Lookup(postgrad) = %+v; a word defined only as an abbreviation must not resolve", res)
	}

	res, ok := db.Lookup("macro")
	if !ok || res.Kind != MatchExact {
		t.Fatalf("Lookup(macro) = %+v,%v; want an exact match", res, ok)
	}
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != "Very large in scale or scope." {
		t.Errorf("Lookup(macro) senses = %v; want only the sense without the marker", res.Entry.Senses)
	}
}

// TestBuildDropsLetterCaseSenses covers the third drop: a pointer at another capitalisation
// of the same word names the played word itself, because the DB is keyed in lower case, so
// it can never be joined and is removed rather than shown back to the player.
func TestBuildDropsLetterCaseSenses(t *testing.T) {
	db, _ := buildFixtureDB(t)

	if res, ok := db.Lookup("abelian"); ok {
		t.Errorf("Lookup(abelian) = %+v; a word defined only by a letter-case pointer must not resolve", res)
	}
}

// TestBuildDropsPointersAtDroppedWords guards the consequence of the drops: a redirect at a
// word the rules emptied can never be answered, so it goes too, rather than telling a player
// their word is another word and stopping there.
func TestBuildDropsPointersAtDroppedWords(t *testing.T) {
	db, _ := buildFixtureDB(t)

	if res, ok := db.Lookup("pgrad"); ok {
		t.Errorf("Lookup(pgrad) = %+v; a pointer at a dropped word must not resolve", res)
	}
	if res, ok := db.Lookup("esports"); ok {
		t.Errorf("Lookup(esports) = %+v; a pointer at itself must not resolve", res)
	}
	// Reached only by the repeat: "esports" is emptied by the pass that drops its own
	// self-pointer, which is what orphans this one.
	if res, ok := db.Lookup("cyberathletics"); ok {
		t.Errorf("Lookup(cyberathletics) = %+v; a pointer at an emptied word must not resolve", res)
	}
	// "barcae" still answers a player who types it, through the stem rule that reaches
	// "barca"; what must be gone is its own entry, which held only the pointer.
	if res, ok := db.Lookup("barcae"); ok && res.Kind == MatchExact {
		t.Errorf("Lookup(barcae) = %+v; the entry holding only an unanswerable pointer must be gone", res)
	}
}

// TestBuildLeavesOrdinaryDefinitionsAlone is the control for the two tests above: a definition
// that merely reads like a redirect must survive the pass untouched.
func TestBuildLeavesOrdinaryDefinitionsAlone(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("ballet")
	if !ok || res.Kind != MatchExact {
		t.Fatalf("Lookup(ballet) = %+v,%v; want an exact match", res, ok)
	}
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != "A classical form of dance." {
		t.Errorf("Lookup(ballet) senses = %v; want the gloss unchanged", res.Entry.Senses)
	}
}

func TestBuildReportsSenseCounts(t *testing.T) {
	_, report := buildFixtureDB(t)

	if got, want := report.DroppedInitialisms, 2; got != want {
		t.Errorf("Report.DroppedInitialisms = %d, want %d (scuba, zyzzyt)", got, want)
	}
	if got, want := report.DroppedAbbreviations, 2; got != want {
		t.Errorf("Report.DroppedAbbreviations = %d, want %d (postgrad, macro)", got, want)
	}
	if got, want := report.DroppedLetterCase, 1; got != want {
		t.Errorf("Report.DroppedLetterCase = %d, want %d (abelian)", got, want)
	}
	if got, want := report.OrphanedRedirects, 5; got != want {
		t.Errorf("Report.OrphanedRedirects = %d, want %d (pgrad, esports, cyberathletics, abegge, barcae)",
			got, want)
	}
	// Every pointer is counted here by the parse, whether the prune later removes it or
	// not; "abelian" is not counted at all, being dropped before it can be classified.
	if got, want := report.RedirectSenses, 14; got != want {
		t.Errorf("Report.RedirectSenses = %d, want %d", got, want)
	}
	// "wobble" is in no word list; it ships only because "wibble" points at it.
	if got, want := report.RedirectTargets, 1; got != want {
		t.Errorf("Report.RedirectTargets = %d, want %d (wobble)", got, want)
	}
}

// TestBuildKeepsAnUnlistedRedirectTarget guards the closure that decides what ships: the
// definition joined onto a redirect comes from the word it names, so that word has to be
// in the asset even when no word list can form it.
func TestBuildKeepsAnUnlistedRedirectTarget(t *testing.T) {
	db, _ := buildFixtureDB(t)

	res, ok := db.Lookup("wibble")
	if !ok {
		t.Fatal("Lookup(wibble) not found")
	}
	want := "Alternative form of wobble: To move erratically from side to side."
	if len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != want {
		t.Errorf("Lookup(wibble) senses = %v;\n want %q", res.Entry.Senses, want)
	}
}

// TestRedirectDefinitionStopsOnACycle guards the chain walk against a pair of words that
// name each other, which the extract is free to contain and which would otherwise not
// terminate. Neither word can be joined to a definition, so both keep their text alone.
func TestRedirectDefinitionStopsOnACycle(t *testing.T) {
	db := NewDB(map[string]*Entry{
		"aye": {Word: "aye", Senses: []Sense{{Gloss: "Alternative form of ay."}}},
		"ay":  {Word: "ay", Senses: []Sense{{Gloss: "Alternative form of aye."}}},
	}, nil)

	if i, ok := db.redirectEntry("aye", "ay"); ok {
		t.Errorf("redirectEntry over a cycle = %d,true; want false", i)
	}
	res, ok := db.Lookup("aye")
	if !ok || len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != "Alternative form of ay." {
		t.Errorf("Lookup(aye) = %+v; want the gloss left as it stands", res)
	}
}

// TestLookupLeavesAnUndefinedRedirectAlone covers a redirect whose target the DB does not
// hold at all: there is nothing to join, and the pointer is then all there is to say.
func TestLookupLeavesAnUndefinedRedirectAlone(t *testing.T) {
	db := NewDB(map[string]*Entry{
		"abegge": {Word: "abegge", Senses: []Sense{{Gloss: "Obsolete form of aby."}}},
	}, nil)

	res, ok := db.Lookup("abegge")
	if !ok || len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != "Obsolete form of aby." {
		t.Errorf("Lookup(abegge) = %+v; want the gloss left as it stands", res)
	}
}
