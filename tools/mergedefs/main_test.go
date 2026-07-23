package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tilewords/defs"
)

func TestParseWebsterSensesNumbered(t *testing.T) {
	def := "1. To make shorter; to shorten in duration. 2. To deprive of a right or privilege."
	senses := parseWebsterSenses(def)
	if len(senses) != 2 {
		t.Fatalf("got %d senses, want 2: %+v", len(senses), senses)
	}
	if !strings.HasPrefix(senses[0].Gloss, "To make shorter") {
		t.Errorf("sense 0 = %q", senses[0].Gloss)
	}
	if !strings.HasPrefix(senses[1].Gloss, "To deprive") {
		t.Errorf("sense 1 = %q", senses[1].Gloss)
	}
	if senses[0].POS != "" {
		t.Errorf("Webster senses carry no POS, got %q", senses[0].POS)
	}
}

func TestParseWebsterSensesUnnumbered(t *testing.T) {
	senses := parseWebsterSenses("  A single definition with no numbering.  ")
	if len(senses) != 1 || senses[0].Gloss != "A single definition with no numbering." {
		t.Fatalf("got %+v", senses)
	}
}

func TestParseWebsterSensesCap(t *testing.T) {
	def := "1. one. 2. two. 3. three. 4. four. 5. five. 6. six."
	if got := len(parseWebsterSenses(def)); got != defs.MaxSensesPerEntry {
		t.Errorf("got %d senses, want cap %d", got, defs.MaxSensesPerEntry)
	}
}

func TestParseWebsterSensesEmpty(t *testing.T) {
	if got := parseWebsterSenses("   "); got != nil {
		t.Errorf("empty definition must yield no senses, got %+v", got)
	}
}

func TestParseWordNetLine(t *testing.T) {
	// A real-shaped noun line: two lemmas, gloss with a trailing example.
	line := `00002137 03 n 02 abstraction 0 abstract_entity 0 010 @ 00001740 n 0000 | a general concept; "an abstraction of the idea"`
	lemmas, pos, gloss, ok := parseWordNetLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if pos != "noun" {
		t.Errorf("pos = %q, want noun", pos)
	}
	if gloss != "a general concept" {
		t.Errorf("gloss = %q, want the definition without the example", gloss)
	}
	// abstract_entity is a collocation and must be dropped; abstraction kept.
	if len(lemmas) != 1 || lemmas[0] != "abstraction" {
		t.Errorf("lemmas = %v, want [abstraction]", lemmas)
	}
}

func TestParseWordNetLineVerb(t *testing.T) {
	line := `00001740 29 v 01 breathe 0 021 * 00005041 v 0000 | draw air into the lungs; "I can breathe better now"`
	lemmas, pos, gloss, ok := parseWordNetLine(line)
	if !ok || pos != "verb" || len(lemmas) != 1 || lemmas[0] != "breathe" {
		t.Fatalf("parse = %v %q %v %v", lemmas, pos, gloss, ok)
	}
	if gloss != "draw air into the lungs" {
		t.Errorf("gloss = %q", gloss)
	}
}

func TestParseWordNetLineRejectsNonData(t *testing.T) {
	for _, line := range []string{
		"  1 This software and database is being provided to you",
		"00001740 03 n",                  // too few fields
		"00001740 03 x 01 foo 0 000 | g", // unknown ss_type
		"no pipe here at all",
	} {
		if _, _, _, ok := parseWordNetLine(line); ok {
			t.Errorf("expected reject for %q", line)
		}
	}
}

func TestCleanGlossTruncates(t *testing.T) {
	long := strings.Repeat("word ", 100) // 500 runes
	got := cleanGloss(long)
	if r := []rune(got); len(r) > maxGlossLen+1 { // +1 for the ellipsis rune
		t.Errorf("cleanGloss produced %d runes, want <= %d", len(r), maxGlossLen+1)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated gloss must end with an ellipsis: %q", got[len(got)-10:])
	}
}

func TestCleanGlossCollapsesWhitespace(t *testing.T) {
	if got := cleanGloss("  a   b\tc\n"); got != "a b c" {
		t.Errorf("cleanGloss = %q, want %q", got, "a b c")
	}
}

func TestParseGlossarySpec(t *testing.T) {
	cases := []struct{ spec, label, path string }{
		{"scots=data/scots.tsv", "scots", "data/scots.tsv"},
		{"wordlists/supplemental-glossary.tsv", "supplemental-glossary", "wordlists/supplemental-glossary.tsv"},
		{"plain.tsv", "plain", "plain.tsv"},
	}
	for _, c := range cases {
		if label, path := parseGlossarySpec(c.spec); label != c.label || path != c.path {
			t.Errorf("parseGlossarySpec(%q) = (%q,%q); want (%q,%q)", c.spec, label, path, c.label, c.path)
		}
	}
}

func TestLoadGlossary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "g.tsv")
	content := "# a comment\n\n" +
		"aefauld\tHonest, upright\n" +
		"talaunts\tnoun\ttalons\n" + // 3-column: pos in the middle
		"cauld\tCold\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pre-seed one word to prove an earlier source keeps precedence.
	entries := map[string]*defs.Entry{"cauld": {Word: "cauld", Senses: []defs.Sense{{Gloss: "prior"}}}}
	srcOf := map[string]string{"cauld": "webster1913"}

	n, err := loadGlossary("scots", path, entries, srcOf)
	if err != nil {
		t.Fatalf("loadGlossary: %v", err)
	}
	if n != 2 { // aefauld + talaunts added; cauld skipped (already present)
		t.Errorf("added %d, want 2", n)
	}
	if e := entries["talaunts"]; e == nil || e.Senses[0].Gloss != "talons" || e.Senses[0].POS != "noun" {
		t.Errorf("talaunts entry = %+v", e)
	}
	if entries["cauld"].Senses[0].Gloss != "prior" {
		t.Errorf("precedence lost: cauld = %q, want prior", entries["cauld"].Senses[0].Gloss)
	}
	if srcOf["aefauld"] != "scots" {
		t.Errorf("source label = %q, want scots", srcOf["aefauld"])
	}
}

func TestLoadGlossaryRejectsMissingTab(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.tsv")
	if err := os.WriteFile(path, []byte("wordonly no tab here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadGlossary("x", path, map[string]*defs.Entry{}, map[string]string{}); err == nil {
		t.Error("expected an error for a line without a tab")
	}
}
