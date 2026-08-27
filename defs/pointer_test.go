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
		"Abbreviation of air-conditioned.",
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
		// Points at itself once the case distinction is gone.
		kaikkiLine("abelian", "adj", "Alternative letter-case form of abelian."),
		kaikkiLine("scuba", "noun", "Initialism of self-contained underwater breathing apparatus.",
			"An apparatus for breathing underwater."),
		kaikkiLine("zyzzyt", "noun", "Initialism of zeta y zeta."),
		kaikkiLine("ballet", "noun", "A classical form of dance."),
		// An inflection whose lemma is itself only a redirect: ceca -> cecum -> caecum.
		kaikkiLine("caecum", "noun", "A blind pouch of the intestine."),
		kaikkiLine("cecum", "noun", "Alternative form of caecum."),
		kaikkiFormLine("ceca", "noun", "plural of cecum", "cecum"),
		// "wobble" is deliberately left out of the word list below, so only the closure
		// over redirect targets can keep it in the asset.
		kaikkiLine("wibble", "verb", "Alternative form of wobble."),
		kaikkiLine("wobble", "verb", "To move erratically from side to side."),
	})
	list := writeFixture(t, dir, "words.txt", []string{
		"color", "colour", "colorize", "colourize", "colourise",
		"abegge", "abelian", "scuba", "zyzzyt", "ballet",
		"caecum", "cecum", "ceca", "wibble",
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

// TestBuildKeepsRedirectsThatReachNothing guards the fallback: a redirect naming a word the
// extract does not define, or naming the word itself, has nothing to join on, so its text
// stands alone rather than leaving the word with no definition at all.
func TestBuildKeepsRedirectsThatReachNothing(t *testing.T) {
	db, _ := buildFixtureDB(t)

	for _, c := range []struct{ word, wantGloss string }{
		{"abegge", "Obsolete form of aby."},
		{"abelian", "Alternative letter-case form of abelian."},
	} {
		res, ok := db.Lookup(c.word)
		if !ok {
			t.Errorf("Lookup(%q) not found; the unresolvable redirect must be kept", c.word)
			continue
		}
		if res.Kind != MatchExact || len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != c.wantGloss {
			t.Errorf("Lookup(%q) = %+v; want the exact entry %q", c.word, res, c.wantGloss)
		}
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

	if got, want := report.RedirectSenses, 7; got != want {
		t.Errorf("Report.RedirectSenses = %d, want %d", got, want)
	}
	if got, want := report.DroppedInitialisms, 2; got != want {
		t.Errorf("Report.DroppedInitialisms = %d, want %d (scuba, zyzzyt)", got, want)
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

	if def, ok := db.redirectDefinition("aye", "ay"); ok {
		t.Errorf("redirectDefinition over a cycle = %q,true; want false", def)
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
