// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package dictionary

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// --- Example-based tests ---

func TestLoadGADDAG_ValidData(t *testing.T) {
	// Verify that a GADDAG built from testWords loads without error.
	var buf bytes.Buffer
	if err := Build(testWords, &buf); err != nil {
		t.Fatalf("Build: %v", err)
	}
	g, err := loadGADDAG(buf.Bytes())
	if err != nil {
		t.Fatalf("loadGADDAG: %v", err)
	}
	if g.Root() != RootNodeID {
		t.Errorf("Root() = %d, want %d", g.Root(), RootNodeID)
	}
}

func TestLoadGADDAG_CorruptData(t *testing.T) {
	// Malformed bytes must return a descriptive error, not panic.
	_, err := loadGADDAG([]byte("not gob data at all"))
	if err == nil {
		t.Fatal("expected error for corrupt data, got nil")
	}
}

func TestValidate_KnownWords(t *testing.T) {
	cases := []struct {
		word  string
		valid bool
	}{
		{"CAT", true},
		{"DOG", true},
		{"SQUAB", true},
		{"QUEEN", true},
		{"ZZZZZ", false},
		{"NOTAWORD", false},
		{"XYZ", false},
	}
	for _, tc := range cases {
		got := testDict.Validate(tc.word)
		if got != tc.valid {
			t.Errorf("Validate(%q) = %v, want %v", tc.word, got, tc.valid)
		}
	}
}

func TestValidate_CaseInsensitive(t *testing.T) {
	variants := []string{"cat", "Cat", "CAT", "cAt", "CaT"}
	for _, v := range variants {
		if !testDict.Validate(v) {
			t.Errorf("Validate(%q) = false, want true", v)
		}
	}
}

func TestValidate_Boundaries(t *testing.T) {
	// Single letter: always false (MinWordLen == 2)
	if testDict.Validate("A") {
		t.Error("Validate(single letter) = true, want false")
	}
	// Two-letter word in list: true
	if !testDict.Validate("AA") {
		t.Error("Validate(AA) = false, want true")
	}
	// 16-letter string: always false (MaxWordLen == 15)
	if testDict.Validate("AAAAAAAAAAAAAAAA") {
		t.Error("Validate(16-letter) = true, want false")
	}
}

func TestValidate_NonAlpha(t *testing.T) {
	cases := []string{"CAT-DOG", "cat1", "has space", "café", ""}
	for _, c := range cases {
		if testDict.Validate(c) {
			t.Errorf("Validate(%q) = true, want false", c)
		}
	}
}

func TestSuccessor_ArcSep(t *testing.T) {
	// For word "CAT": GADDAG stores "C+AT" (k=1), "TC+AT" is not quite right —
	// let's verify the arc-separator arc exists after the reversed prefix.
	// k=1 for "CAT": sequence is C '+' A T — node after C should have ArcSep edge.
	root := testGADDAG.Root()
	afterC, ok := testGADDAG.Successor(root, 'C')
	if !ok {
		t.Fatal("no edge from root on 'C'")
	}
	_, ok = testGADDAG.Successor(afterC, ArcSep)
	if !ok {
		t.Error("no ArcSep edge after 'C' from root — expected for word CAT (k=1 string: C+AT)")
	}
}

func TestWordCount(t *testing.T) {
	// WordCount must equal the number of words in testWords (all unique).
	if testDict.WordCount() != len(testWords) {
		t.Errorf("WordCount() = %d, want %d", testDict.WordCount(), len(testWords))
	}
}

// --- PBT tests ---

// TestPBT_RoundTrip: serialise GADDAG → deserialise → Contains results identical (PBT-02).
func TestPBT_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		word := wordFromDictGen().Draw(t, "word")
		// Encode testGADDAG to bytes and reload.
		var buf bytes.Buffer
		if err := Build(testWords, &buf); err != nil {
			t.Fatalf("Build: %v", err)
		}
		g2, err := loadGADDAG(buf.Bytes())
		if err != nil {
			t.Fatalf("loadGADDAG round-trip: %v", err)
		}
		if g2.contains(word) != testGADDAG.contains(word) {
			t.Fatalf("round-trip divergence for %q", word)
		}
	})
}

// TestPBT_ContainsOracle: every word in testWords must be found; random non-words compared
// to brute-force set membership (PBT-05).
func TestPBT_ContainsOracle(t *testing.T) {
	oracle := testWordSet()
	rapid.Check(t, func(t *rapid.T) {
		word := randomAlphaGen(MinWordLen, MaxWordLen).Draw(t, "word")
		got := testGADDAG.contains(word)
		want := oracle[word]
		if got != want {
			t.Fatalf("contains(%q) = %v, oracle says %v", word, got, want)
		}
	})
}

// TestPBT_ContainsOracle_ValidWords: every known valid word returns true.
func TestPBT_ContainsOracle_ValidWords(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		word := wordFromDictGen().Draw(t, "word")
		if !testGADDAG.contains(word) {
			t.Fatalf("contains(%q) = false for known valid word", word)
		}
	})
}

// TestPBT_CaseInvariance: Contains(upper) == Contains(lower) for all alpha strings (PBT-03).
func TestPBT_CaseInvariance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		word := randomAlphaGen(MinWordLen, MaxWordLen).Draw(t, "word")
		upper := toUpper(word)
		lower := toLowerASCII(word)
		if testDict.Validate(upper) != testDict.Validate(lower) {
			t.Fatalf("case variance: Validate(%q) != Validate(%q)", upper, lower)
		}
	})
}

// TestPBT_InvalidRejection: Contains(string with non-A-Z) always false (PBT-03).
func TestPBT_InvalidRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := invalidStringGen().Draw(t, "invalid")
		if testGADDAG.contains(s) {
			t.Fatalf("contains(%q) = true for invalid string (non-A-Z byte present)", s)
		}
	})
}

// TestPBT_DedupInvariant: Building from a word list with duplicates produces the same
// GADDAG as building from the deduplicated list (PBT-03 / BR-05).
func TestPBT_DedupInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		word := wordFromDictGen().Draw(t, "word")

		// Double every word to create duplicates.
		doubled := make([]string, 0, len(testWords)*2)
		doubled = append(doubled, testWords...)
		doubled = append(doubled, testWords...)

		var buf bytes.Buffer
		if err := Build(doubled, &buf); err != nil {
			t.Fatalf("Build (doubled): %v", err)
		}
		g2, err := loadGADDAG(buf.Bytes())
		if err != nil {
			t.Fatalf("loadGADDAG (doubled): %v", err)
		}
		if g2.contains(word) != testGADDAG.contains(word) {
			t.Fatalf("dedup invariant violated for %q", word)
		}
	})
}

// TestPBT_IdempotentLoad: Load(bytes) twice produces identical Contains results (PBT-04).
func TestPBT_IdempotentLoad(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		word := wordFromDictGen().Draw(t, "word")

		var buf bytes.Buffer
		if err := Build(testWords, &buf); err != nil {
			t.Fatalf("Build: %v", err)
		}
		data := buf.Bytes()

		g1, err := loadGADDAG(data)
		if err != nil {
			t.Fatalf("loadGADDAG (1st): %v", err)
		}
		g2, err := loadGADDAG(data)
		if err != nil {
			t.Fatalf("loadGADDAG (2nd): %v", err)
		}
		if g1.contains(word) != g2.contains(word) {
			t.Fatalf("idempotent load failed for %q: g1=%v g2=%v", word, g1.contains(word), g2.contains(word))
		}
	})
}

// TestLoad_CachesSameDictionary verifies Load returns the cached instance when the same
// dictionary is loaded again — as in start a game, save, then load that save. Without the
// cache the reload decodes a second multi-hundred-MB GADDAG while the first is still live,
// which exhausts memory and is killed on a phone.
func TestLoad_CachesSameDictionary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: decodes a full embedded dictionary (hundreds of MB in memory)")
	}
	if !Available(DictENABLE) {
		t.Skip("enable dictionary asset not embedded in this build")
	}

	d1, err := Load(DictENABLE)
	if err != nil {
		t.Fatalf("Load #1: %v", err)
	}
	d2, err := Load(DictENABLE)
	if err != nil {
		t.Fatalf("Load #2: %v", err)
	}
	if d1 != d2 {
		t.Fatal("re-Load of the same dictionary returned a different instance: the cache " +
			"is not reused, so a reload decodes a second copy and can exhaust memory")
	}
}

// TestBuild_Deterministic verifies that building the same word list twice produces byte-
// identical output. The compressed-sparse-row encoding emits nodes in id order with each
// node's edges sorted by letter, so the result must not depend on Go map iteration order.
func TestBuild_Deterministic(t *testing.T) {
	words := []string{"CAT", "CATS", "DOG", "QUEEN", "SQUAB", "ZEBRA", "AA", "AB", "WORLD"}
	var b1, b2 bytes.Buffer
	if err := Build(words, &b1); err != nil {
		t.Fatalf("Build #1: %v", err)
	}
	if err := Build(words, &b2); err != nil {
		t.Fatalf("Build #2: %v", err)
	}
	if !bytes.Equal(b1.Bytes(), b2.Bytes()) {
		t.Fatal("Build is not deterministic: identical input produced different bytes")
	}
}

// TestBuild_Minimized verifies that Build produces a minimized automaton, not a prefix
// trie. In a trie every word's dead-end accept state is a distinct node; minimization
// merges all equivalent accept sinks (terminal, no out-edges) into one. Counting such sinks
// therefore distinguishes a minimized graph (<=1) from a trie (one per distinct ending) and
// guards against minimization being silently disabled.
func TestBuild_Minimized(t *testing.T) {
	var buf bytes.Buffer
	if err := Build([]string{"CAT", "CATS", "DOG", "DOGS", "HAT", "HATS", "RAT", "RATS"}, &buf); err != nil {
		t.Fatalf("Build: %v", err)
	}
	g, err := loadGADDAG(buf.Bytes())
	if err != nil {
		t.Fatalf("loadGADDAG: %v", err)
	}
	sinks := 0
	for id := NodeID(1); uint32(id) < g.nodeCount; id++ {
		if g.edgeOffsets[id] == g.edgeOffsets[id+1] && g.IsTerminal(id) {
			sinks++
		}
	}
	if sinks > 1 {
		t.Fatalf("found %d terminal dead-end nodes, want <=1: minimization did not merge accept sinks (graph is a trie)", sinks)
	}
}

// rawAsset is a hand-built GADDAG asset stream, mirroring the layout writeGADDAG produces
// so a test can corrupt one field at a time.
type rawAsset struct {
	magic      string
	nodeCount  uint64
	edgeCount  uint64
	wordCount  uint64
	root       uint64
	edgeCounts []uint64
	letters    string
	targets    []uint64
	termWords  int
}

// validAsset describes a two-node, one-edge graph that loadGADDAG must accept.
func validAsset() rawAsset {
	return rawAsset{
		magic:      assetMagic,
		nodeCount:  2,
		edgeCount:  1,
		wordCount:  1,
		root:       uint64(RootNodeID),
		edgeCounts: []uint64{0, 1},
		letters:    "a",
		targets:    []uint64{0},
		termWords:  1,
	}
}

// encode serialises the asset exactly as writeGADDAG would, without validating anything.
func (a rawAsset) encode() []byte {
	var out []byte
	out = append(out, a.magic...)
	for _, v := range []uint64{a.nodeCount, a.edgeCount, a.wordCount, a.root} {
		out = binary.AppendUvarint(out, v)
	}
	for _, v := range a.edgeCounts {
		out = binary.AppendUvarint(out, v)
	}
	out = append(out, a.letters...)
	for _, v := range a.targets {
		out = binary.AppendUvarint(out, v)
	}
	for i := 0; i < a.termWords; i++ {
		out = binary.LittleEndian.AppendUint64(out, 0)
	}
	return out
}

// TestLoadGADDAG_AcceptsValidAsset checks the fixture the corruption cases below are derived
// from is itself valid, so those cases fail for the reason intended.
func TestLoadGADDAG_AcceptsValidAsset(t *testing.T) {
	g, err := loadGADDAG(validAsset().encode())
	if err != nil {
		t.Fatalf("loadGADDAG rejected the valid fixture: %v", err)
	}
	if g.nodeCount != 2 || len(g.edgeLetters) != 1 {
		t.Errorf("decoded %d nodes / %d edges, want 2 / 1", g.nodeCount, len(g.edgeLetters))
	}
	if target, ok := g.Successor(RootNodeID, 'a'); !ok || target != 0 {
		t.Errorf("Successor(root,'a') = %d,%v; want 0,true", target, ok)
	}
}

// TestLoadGADDAG_RejectsMalformedAsset verifies that a malformed asset is rejected at load.
// Without these guards a bad edge count or target would let Successor index out of range and
// panic during traversal, so each case corrupts one field of an otherwise valid stream.
func TestLoadGADDAG_RejectsMalformedAsset(t *testing.T) {
	cases := []struct {
		name    string
		corrupt func(*rawAsset)
		want    string
	}{
		{"wrong magic", func(a *rawAsset) { a.magic = "TWGDDG\x00\n" }, "not a GADDAG asset"},
		{"edge counts overrun", func(a *rawAsset) { a.edgeCounts = []uint64{0, 5} }, "overrun the edge total"},
		{"edge counts short", func(a *rawAsset) { a.edgeCounts = []uint64{0, 0} }, "cover 0 of 1 edges"},
		{"target out of range", func(a *rawAsset) { a.targets = []uint64{9} }, "targets node 9 of 2"},
		{"bad root", func(a *rawAsset) { a.root = 0 }, "invalid root node"},
		{"root beyond node count", func(a *rawAsset) { a.nodeCount = 1; a.edgeCounts = []uint64{1} }, "root node"},
		{"node count beyond maximum", func(a *rawAsset) { a.nodeCount = maxAssetCount + 1 }, "beyond the"},
		// At the maximum the absolute ceiling does not fire, so only the asset-size bound
		// stands between a corrupt count and an 8 GiB make().
		{"node count at the maximum", func(a *rawAsset) { a.nodeCount = maxAssetCount }, "than a"},
		{"node count beyond the asset size", func(a *rawAsset) { a.nodeCount = 1 << 20 }, "than a"},
		{"edge count beyond the asset size", func(a *rawAsset) { a.edgeCount = 1 << 20 }, "than a"},
		{
			"edge labels not ascending",
			func(a *rawAsset) {
				a.edgeCount = 2
				a.edgeCounts = []uint64{0, 2}
				a.letters = "ba"
				a.targets = []uint64{0, 0}
			},
			"not strictly ascending",
		},
		{
			"duplicate edge labels",
			func(a *rawAsset) {
				a.edgeCount = 2
				a.edgeCounts = []uint64{0, 2}
				a.letters = "aa"
				a.targets = []uint64{0, 0}
			},
			"not strictly ascending",
		},
		{"truncated bitset", func(a *rawAsset) { a.termWords = 0 }, "terminal bitset"},
		{"truncated letters", func(a *rawAsset) { a.letters = ""; a.targets = nil; a.termWords = 0 }, "edge letters"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := validAsset()
			tc.corrupt(&a)
			_, err := loadGADDAG(a.encode())
			if err == nil {
				t.Fatal("loadGADDAG accepted a malformed asset")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// toLowerASCII converts an uppercase A-Z string to lowercase for case-invariance tests.
func toLowerASCII(s string) string {
	bs := []byte(s)
	for i, b := range bs {
		if b >= 'A' && b <= 'Z' {
			bs[i] = b + 32
		}
	}
	return string(bs)
}
