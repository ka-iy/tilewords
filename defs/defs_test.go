package defs

import (
	"bytes"
	"testing"
)

// containsStr reports whether s is in xs.
func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func TestCandidateStems(t *testing.T) {
	cases := []struct {
		word string
		want string // a lemma that must appear among the candidates
	}{
		{"cats", "cat"},
		{"boxes", "box"},
		{"houses", "house"},
		{"berries", "berry"},
		{"wolves", "wolf"},
		{"knives", "knife"},
		{"baking", "bake"},
		{"running", "run"},
		{"walked", "walk"},
		{"hopped", "hop"},
		{"cried", "cry"},
		{"happiest", "happy"},
		{"nicer", "nice"},
		{"quickly", "quick"},
		{"happily", "happy"},
		{"sadness", "sad"},
		{"happiness", "happy"},
	}
	for _, c := range cases {
		got := candidateStems(c.word)
		if !containsStr(got, c.want) {
			t.Errorf("candidateStems(%q) = %v, want to contain %q", c.word, got, c.want)
		}
	}
}

func TestCandidateStemsExcludesSelf(t *testing.T) {
	for _, w := range []string{"cats", "running", "ss", "as"} {
		if containsStr(candidateStems(w), w) {
			t.Errorf("candidateStems(%q) must not contain the word itself", w)
		}
	}
}

func TestClassicalPluralStems(t *testing.T) {
	cases := []struct{ word, want string }{
		{"acanthae", "acantha"},
		{"acalephae", "acalephe"},
		{"aerotaxes", "aerotaxis"},
		{"agamogeneses", "agamogenesis"},
		{"aerenchymata", "aerenchyma"},
		{"cacti", "cactus"},
		{"addenda", "addendum"},
		{"phenomena", "phenomenon"},
	}
	for _, c := range cases {
		if got := candidateStems(c.word); !containsStr(got, c.want) {
			t.Errorf("candidateStems(%q) = %v, want to contain %q", c.word, got, c.want)
		}
	}
}

func TestVariantCandidates(t *testing.T) {
	cases := []struct{ word, want string }{
		{"activise", "activize"},
		{"activize", "activise"},
		{"flavour", "flavor"},
		{"encyclopaedia", "encyclopedia"},
		{"catalogue", "catalog"},
	}
	for _, c := range cases {
		if got := variantCandidates(c.word); !containsStr(got, c.want) {
			t.Errorf("variantCandidates(%q) = %v, want to contain %q", c.word, got, c.want)
		}
	}
}

func TestVariantCandidatesExcludesSelf(t *testing.T) {
	for _, w := range []string{"cat", "activate"} {
		if containsStr(variantCandidates(w), w) {
			t.Errorf("variantCandidates(%q) must not contain the word itself", w)
		}
	}
}

func TestLookupLayers(t *testing.T) {
	entries := map[string]*Entry{
		"cat":    {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "a small feline"}}},
		"bake":   {Word: "bake", Senses: []Sense{{POS: "verb", Gloss: "to cook in an oven"}}},
		"flavor": {Word: "flavor", Senses: []Sense{{POS: "noun", Gloss: "taste"}}},
		"child":  {Word: "child", Senses: []Sense{{POS: "noun", Gloss: "a young human"}}},
	}
	formOf := map[string]string{"children": "child"}
	db := NewDB(entries, formOf)

	cases := []struct {
		word     string
		wantHead string
		wantKind MatchKind
	}{
		{"cat", "cat", MatchExact},
		{"CAT", "cat", MatchExact},
		{"children", "child", MatchFormOf},
		{"baking", "bake", MatchStem},
		{"cats", "cat", MatchStem},
		{"flavour", "flavor", MatchFuzzy},
	}
	for _, c := range cases {
		res, ok := db.Lookup(c.word)
		if !ok {
			t.Errorf("Lookup(%q) not found", c.word)
			continue
		}
		if res.Headword != c.wantHead || res.Kind != c.wantKind {
			t.Errorf("Lookup(%q) = head %q kind %v; want head %q kind %v",
				c.word, res.Headword, res.Kind, c.wantHead, c.wantKind)
		}
	}

	if _, ok := db.Lookup("xyzzy"); ok {
		t.Error("Lookup(xyzzy) should not resolve")
	}
}

func TestLookupExactMergesInflection(t *testing.T) {
	// "mice" is both a (rare) headword and the plural of "mouse"; an exact match
	// must still surface the "mouse" reading via AlsoForm.
	entries := map[string]*Entry{
		"mice":  {Word: "mice", Senses: []Sense{{POS: "verb", Gloss: "to be distracted"}}},
		"mouse": {Word: "mouse", Senses: []Sense{{POS: "noun", Gloss: "a small rodent"}}},
		"cat":   {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "a small feline"}}},
	}
	db := NewDB(entries, map[string]string{"mice": "mouse"})

	res, ok := db.Lookup("mice")
	if !ok || res.Kind != MatchExact || res.Headword != "mice" {
		t.Fatalf("Lookup(mice) = %+v,%v; want exact mice", res, ok)
	}
	if res.AlsoForm == nil || res.AlsoFormWord != "mouse" {
		t.Fatalf("Lookup(mice) AlsoForm = %v word %q; want mouse", res.AlsoForm, res.AlsoFormWord)
	}

	// A plain headword with no inflection edge must not carry AlsoForm.
	res, _ = db.Lookup("cat")
	if res.AlsoForm != nil {
		t.Errorf("Lookup(cat) AlsoForm = %v; want nil", res.AlsoForm)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	entries := map[string]*Entry{
		"cat":   {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "a small feline"}}},
		"child": {Word: "child", Senses: []Sense{{POS: "noun", Gloss: "a young human"}}},
	}
	db := NewDB(entries, map[string]string{"children": "child"})

	var buf bytes.Buffer
	if err := db.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Len() != db.Len() || got.FormCount() != db.FormCount() {
		t.Fatalf("round-trip sizes differ: got (%d,%d) want (%d,%d)",
			got.Len(), got.FormCount(), db.Len(), db.FormCount())
	}
	res, ok := got.Lookup("children")
	if !ok || res.Headword != "child" || res.Kind != MatchFormOf {
		t.Errorf("decoded Lookup(children) = %+v,%v", res, ok)
	}
}

func TestCleanGlossTruncates(t *testing.T) {
	long := make([]byte, maxGlossLen+50)
	for i := range long {
		long[i] = 'a'
	}
	got := cleanGloss(string(long))
	if r := []rune(got); len(r) > maxGlossLen {
		t.Errorf("cleanGloss produced %d runes, want <= %d", len(r), maxGlossLen)
	}
}

func TestFormTargetExplicit(t *testing.T) {
	s := kaikkiSense{Glosses: []string{"plural of cat"}, Tags: []string{"form-of", "plural"}, FormOf: []kaikkiFormOf{{Word: "cat"}}}
	if lemma, ok := formTarget(s); !ok || lemma != "cat" {
		t.Errorf("formTarget explicit = %q,%v; want cat,true", lemma, ok)
	}
}

func TestFormTargetGlossFallback(t *testing.T) {
	s := kaikkiSense{Glosses: []string{"simple past of bake"}, Tags: []string{"form-of", "past"}}
	if lemma, ok := formTarget(s); !ok || lemma != "bake" {
		t.Errorf("formTarget fallback = %q,%v; want bake,true", lemma, ok)
	}
}

func TestFormTargetNotAForm(t *testing.T) {
	s := kaikkiSense{Glosses: []string{"a small feline"}}
	if _, ok := formTarget(s); ok {
		t.Error("formTarget must report false for a real definition")
	}
}
