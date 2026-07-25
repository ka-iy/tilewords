package defs

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"strings"
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
		// "bacilli" rather than the shorter "cacti": the ending-replacement rules are held
		// back below minClassicalLen, where they only ever reached unrelated headwords. Short
		// classical plurals are unaffected in practice because Wiktionary records them as
		// inflected forms, so they resolve at the form-of layer before these rules run —
		// see TestShortClassicalPluralsResolveByFormOf.
		{"bacilli", "bacillus"},
		{"addenda", "addendum"},
		{"phenomena", "phenomenon"},
	}
	for _, c := range cases {
		if got := candidateStems(c.word); !containsStr(got, c.want) {
			t.Errorf("candidateStems(%q) = %v, want to contain %q", c.word, got, c.want)
		}
	}
}

// TestShortClassicalPluralsResolveByFormOf documents why holding the ending-replacement rules
// back below minClassicalLen costs no coverage for common short plurals: the form-of layer
// already resolves them, and it runs first.
func TestShortClassicalPluralsResolveByFormOf(t *testing.T) {
	entries := map[string]*Entry{
		"cactus": {Word: "cactus", Senses: []Sense{{POS: "noun", Gloss: "a spiny plant"}}},
	}
	db := NewDB(entries, map[string]string{"cacti": "cactus"})

	res, ok := db.Lookup("cacti")
	if !ok {
		t.Fatal(`Lookup("cacti") not found`)
	}
	if res.Headword != "cactus" || res.Kind != MatchFormOf {
		t.Errorf(`Lookup("cacti") = head %q kind %v; want head "cactus" kind %v`,
			res.Headword, res.Kind, MatchFormOf)
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

func TestWithSupplement(t *testing.T) {
	base := NewDB(
		map[string]*Entry{
			"cat": {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "authoritative feline"}}},
		},
		map[string]string{"cats": "cat"},
	)

	sup := base.WithSupplement(
		map[string]*Entry{
			// New headword the base lacks — must be added.
			"abrege": {Word: "abrege", Senses: []Sense{{Gloss: "an abridgment"}}},
			// Collides with an existing headword — base must win, not be overwritten.
			"cat": {Word: "cat", Senses: []Sense{{Gloss: "lower-priority feline"}}},
		},
		map[string]string{
			// New edge for a word the base cannot resolve — must be added.
			"abreges": "abrege",
			// Collides with an existing edge — base edge must win.
			"cats": "dog",
		},
	)

	// The supplement leaves the original DB untouched.
	if base.Len() != 1 || base.FormCount() != 1 {
		t.Fatalf("base mutated by WithSupplement: len=%d forms=%d", base.Len(), base.FormCount())
	}

	// New headword resolves from the supplement.
	if res, ok := sup.Lookup("abrege"); !ok || res.Kind != MatchExact {
		t.Errorf("Lookup(abrege) = %+v,%v; want exact", res, ok)
	}
	// New edge resolves the inflected form to the supplemental headword.
	if res, ok := sup.Lookup("abreges"); !ok || res.Headword != "abrege" || res.Kind != MatchFormOf {
		t.Errorf("Lookup(abreges) = %+v,%v; want formof abrege", res, ok)
	}
	// Colliding headword keeps the base (authoritative) definition.
	res, ok := sup.Lookup("cat")
	if !ok || len(res.Entry.Senses) == 0 || res.Entry.Senses[0].Gloss != "authoritative feline" {
		t.Errorf("Lookup(cat) = %+v; want base definition preserved", res)
	}
	// Colliding edge keeps the base target ("cats" -> "cat", not "dog").
	if res, ok := sup.Lookup("cats"); !ok || res.Headword != "cat" {
		t.Errorf("Lookup(cats) = %+v,%v; want base edge to cat preserved", res, ok)
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

// TestEncodeIsDeterministic guards a property the committed asset depends on: encoding
// an unchanged DB reproduces the same bytes, so regenerating an asset whose sources have
// not moved leaves no diff behind, and a diff therefore means the inputs really changed.
// The flat layout has a fixed order, unlike the map-based format it replaced.
func TestEncodeIsDeterministic(t *testing.T) {
	db := NewDB(map[string]*Entry{
		"cat":   {Word: "cat", Senses: []Sense{{POS: "noun", Gloss: "a small feline"}}},
		"child": {Word: "child", Senses: []Sense{{POS: "noun", Gloss: "a young human"}}},
		"dog":   {Word: "dog", Senses: []Sense{{POS: "noun", Gloss: "a canine"}, {POS: "verb", Gloss: "to follow"}}},
	}, map[string]string{"children": "child", "dogs": "dog"})

	var first, second bytes.Buffer
	if err := db.Encode(&first); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if err := db.Encode(&second); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Errorf("re-encoding the same DB produced different bytes (%d vs %d)", first.Len(), second.Len())
	}
}

// TestNewDBDropsUnresolvableEdge verifies that an inflection edge whose lemma is not a
// headword is dropped: Lookup could never report such an edge, since it needs the lemma's
// entry to answer with, so keeping it would only inflate FormCount.
func TestNewDBDropsUnresolvableEdge(t *testing.T) {
	db := NewDB(
		map[string]*Entry{"child": {Word: "child", Senses: []Sense{{POS: "noun", Gloss: "a young human"}}}},
		map[string]string{"children": "child", "geese": "goose"},
	)

	if got, want := db.FormCount(), 1; got != want {
		t.Errorf("FormCount = %d, want %d (the edge to the absent lemma must be dropped)", got, want)
	}
	if _, ok := db.FormLemma("geese"); ok {
		t.Error("FormLemma(geese) resolved, but its lemma goose is not a headword")
	}
	if lemma, ok := db.FormLemma("children"); !ok || lemma != "child" {
		t.Errorf("FormLemma(children) = %q,%v; want child,true", lemma, ok)
	}
}

// rawAsset is a hand-built asset stream, mirroring the layout Encode writes so a test can
// corrupt one field at a time. Every field is what Decode validates against the others.
type rawAsset struct {
	magic      string
	nHead      uint64
	nSense     uint64
	nForm      uint64
	posTable   []string
	headBlob   string
	headLens   []uint64
	senseCount []uint64
	glossBlob  string
	glossLens  []uint64
	sensePOS   []uint64
	formBlob   string
	formLens   []uint64
	formLemma  []uint64
}

// validAsset describes a two-headword, one-edge asset that Decode must accept.
func validAsset() rawAsset {
	return rawAsset{
		magic:      assetMagic,
		nHead:      2,
		nSense:     2,
		nForm:      1,
		posTable:   []string{"noun"},
		headBlob:   "catchild",
		headLens:   []uint64{3, 5},
		senseCount: []uint64{1, 1},
		glossBlob:  "felinehuman",
		glossLens:  []uint64{6, 5},
		sensePOS:   []uint64{0, 0},
		formBlob:   "children",
		formLens:   []uint64{8},
		formLemma:  []uint64{1},
	}
}

// encode serialises the asset exactly as Encode would, without validating anything.
func (a rawAsset) encode(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	scratch := make([]byte, binary.MaxVarintLen64)
	put := func(v uint64) {
		n := binary.PutUvarint(scratch, v)
		if _, err := gz.Write(scratch[:n]); err != nil {
			t.Fatalf("write varint: %v", err)
		}
	}
	putAll := func(vs []uint64) {
		for _, v := range vs {
			put(v)
		}
	}
	if _, err := gz.Write([]byte(a.magic)); err != nil {
		t.Fatalf("write magic: %v", err)
	}
	put(a.nHead)
	put(a.nSense)
	put(a.nForm)
	put(uint64(len(a.posTable)))
	put(uint64(len(a.headBlob)))
	put(uint64(len(a.glossBlob)))
	put(uint64(len(a.formBlob)))
	for _, pos := range a.posTable {
		put(uint64(len(pos)))
		if _, err := gz.Write([]byte(pos)); err != nil {
			t.Fatalf("write pos: %v", err)
		}
	}
	write := func(s string) {
		if _, err := gz.Write([]byte(s)); err != nil {
			t.Fatalf("write blob: %v", err)
		}
	}
	write(a.headBlob)
	putAll(a.headLens)
	putAll(a.senseCount)
	write(a.glossBlob)
	putAll(a.glossLens)
	putAll(a.sensePOS)
	write(a.formBlob)
	putAll(a.formLens)
	putAll(a.formLemma)
	if err := gz.Close(); err != nil {
		t.Fatalf("close fixture: %v", err)
	}
	return &buf
}

// TestDecodeAcceptsValidAsset checks the fixture the corruption cases are derived from is
// itself valid, so those cases fail for the reason intended rather than a broken fixture.
func TestDecodeAcceptsValidAsset(t *testing.T) {
	db, err := Decode(validAsset().encode(t))
	if err != nil {
		t.Fatalf("Decode rejected the valid fixture: %v", err)
	}
	if db.Len() != 2 || db.FormCount() != 1 {
		t.Errorf("decoded (%d headwords, %d forms), want (2, 1)", db.Len(), db.FormCount())
	}
	if res, ok := db.Lookup("children"); !ok || res.Headword != "child" || res.Kind != MatchFormOf {
		t.Errorf("Lookup(children) = %+v,%v", res, ok)
	}
	if res, ok := db.Lookup("cat"); !ok || len(res.Entry.Senses) != 1 || res.Entry.Senses[0].Gloss != "feline" {
		t.Errorf("Lookup(cat) = %+v,%v", res, ok)
	}
}

// TestDecodeRejectsMalformedAsset covers the checks Decode performs while reading. The
// flat form indexes blobs by offset, so a corrupt or truncated asset must be reported as
// an error at load rather than panicking on an out-of-range slice at first lookup.
func TestDecodeRejectsMalformedAsset(t *testing.T) {
	cases := []struct {
		name    string
		corrupt func(*rawAsset)
		want    string
	}{
		{"wrong magic", func(a *rawAsset) { a.magic = "TWDEFS\x00\n" }, "not a definitions asset"},
		{"headword lengths overrun", func(a *rawAsset) { a.headLens = []uint64{3, 99} }, "headword lengths overrun"},
		{"headword lengths short", func(a *rawAsset) { a.headLens = []uint64{3, 2} }, "headword lengths cover"},
		{"gloss lengths overrun", func(a *rawAsset) { a.glossLens = []uint64{6, 99} }, "gloss lengths overrun"},
		{"sense counts exceed senses", func(a *rawAsset) { a.senseCount = []uint64{1, 9} }, "sense lengths overrun"},
		{"sense counts short", func(a *rawAsset) { a.senseCount = []uint64{1, 0} }, "sense lengths cover"},
		{"form lengths overrun", func(a *rawAsset) { a.formLens = []uint64{99} }, "form lengths overrun"},
		{"pos index out of range", func(a *rawAsset) { a.sensePOS = []uint64{0, 7} }, "part-of-speech index"},
		{"form lemma out of range", func(a *rawAsset) { a.formLemma = []uint64{9} }, "form lemma"},
		{"truncated mid-stream", func(a *rawAsset) { a.formLemma = nil }, "form lemma"},
		// A count is rejected before it is allocated from. Without a bound tight enough to
		// keep the value inside an int, a 32-bit build (armeabi-v7a) converts it to a
		// negative length and make() panics instead of the asset being reported as corrupt;
		// on 64-bit it would allocate gigabytes from a header alone.
		{"headword count beyond maximum", func(a *rawAsset) { a.nHead = maxItemCount + 1 }, "beyond the"},
		{"sense count beyond maximum", func(a *rawAsset) { a.nSense = maxItemCount + 1 }, "beyond the"},
		{"form count beyond maximum", func(a *rawAsset) { a.nForm = maxItemCount + 1 }, "beyond the"},
		// At 2^31 the value is exactly what an int32 cannot hold; the old bound admitted it.
		{"headword count at the 32-bit boundary", func(a *rawAsset) { a.nHead = 1 << 31 }, "beyond the"},
		// More headwords than the blob describing them could possibly hold.
		{"headword count exceeds its blob", func(a *rawAsset) { a.nHead = 1 << 20 }, "more than the"},
		{"form count exceeds its blob", func(a *rawAsset) { a.nForm = 1 << 20 }, "more than the"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := validAsset()
			tc.corrupt(&a)

			_, err := Decode(a.encode(t))
			if err == nil {
				t.Fatalf("Decode accepted a malformed asset")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Decode error = %q, want it to mention %q", err, tc.want)
			}
		})
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
