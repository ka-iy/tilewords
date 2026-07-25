// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import "testing"

// indexOf returns the position of want in candidates, or -1.
func indexOf(candidates []string, want string) int {
	for i, c := range candidates {
		if c == want {
			return i
		}
	}
	return -1
}

// assertBefore checks that first is offered ahead of second. Lookup takes the first candidate
// that is a headword, so when both are real words the order decides which definition a player
// is shown.
func assertBefore(t *testing.T, word, first, second string) {
	t.Helper()
	c := candidateStems(word)
	i, j := indexOf(c, first), indexOf(c, second)
	if i < 0 {
		t.Fatalf("candidateStems(%q) = %v, missing %q", word, c, first)
	}
	if j < 0 {
		t.Fatalf("candidateStems(%q) = %v, missing %q", word, c, second)
	}
	if i > j {
		t.Errorf("candidateStems(%q) offers %q before %q; %v", word, second, first, c)
	}
}

// TestCandidateStems_SilentEBeforeBareStem verifies silent-e restoration is offered ahead of
// the bare stem for the suffixes English drops a silent e before. A stem that keeps its
// consonant reaches those suffixes by doubling it instead, which undouble covers, so trying
// the bare form first only picks up unrelated headwords that happen to exist.
func TestCandidateStems_SilentEBeforeBareStem(t *testing.T) {
	cases := []struct{ word, wantFirst, wantSecond string }{
		{"used", "use", "us"},
		{"reded", "rede", "red"},
		{"eched", "eche", "ech"},
		{"eching", "eche", "ech"},
		{"fracturer", "fracture", "fractur"},
		{"fertiler", "fertile", "fertil"},
		{"nicest", "nice", "nic"},
	}
	for _, tc := range cases {
		assertBefore(t, tc.word, tc.wantFirst, tc.wantSecond)
	}
}

// TestCandidateStems_ConsonantDoublingStillReached verifies the doubled-consonant base is still
// produced, so reordering has not cost the "hopped" to "hop" case.
func TestCandidateStems_ConsonantDoublingStillReached(t *testing.T) {
	for _, tc := range []struct{ word, want string }{
		{"hopped", "hop"},
		{"running", "run"},
		{"bigger", "big"},
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.want) < 0 {
			t.Errorf("candidateStems(%q) = %v, missing %q", tc.word, c, tc.want)
		}
	}
}

// TestCandidateStems_PluralESibilantOrder verifies which -es candidate is offered first depends
// on whether the bare form ends in a sibilant. English inserts the e only after a sibilant, so
// a stem already ending in e just takes s.
func TestCandidateStems_PluralESibilantOrder(t *testing.T) {
	// Bare form ends in a sibilant: the bare form is the likelier stem.
	assertBefore(t, "boxes", "box", "boxe")
	assertBefore(t, "churches", "church", "churche")
	assertBefore(t, "buzzes", "buzz", "buzze")
	// Bare form does not: the e-keeping form is the likelier stem.
	assertBefore(t, "bardes", "barde", "bard")
	assertBefore(t, "agapes", "agape", "agap")
	// -ine chemical names are the volume case: -in and -ine are different suffixes.
	assertBefore(t, "buprenorphines", "buprenorphine", "buprenorphin")
}

// TestCandidateStems_SpellingChangePluralsDoNotExcludePlainES verifies that a word ending in
// "ves" or "ies" is also offered its plain "-es" reading. Those endings are just as often an
// ordinary plural of a stem ending in "v" or "i" ("above", "ami", "champleve") as a
// spelling-change plural ("wolf", "berry"), and reading them only the second way left the
// first unreachable.
func TestCandidateStems_SpellingChangePluralsDoNotExcludePlainES(t *testing.T) {
	// The spelling-change reading must still come first where it is the right one.
	assertBefore(t, "wolves", "wolf", "wolve")
	assertBefore(t, "knives", "knife", "knive")
	assertBefore(t, "berries", "berry", "berrie")
	assertBefore(t, "movies", "movie", "movi")

	// The plain reading must be present for stems that genuinely end in v or i.
	for _, tc := range []struct{ word, want string }{
		{"aboves", "above"},
		{"gloves", "glove"},
		{"curves", "curve"},
		{"amies", "ami"},
		{"champleves", "champleve"},
		{"barramundies", "barramundi"},
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.want) < 0 {
			t.Errorf("candidateStems(%q) = %v, missing %q", tc.word, c, tc.want)
		}
	}
}

// TestAddClassicalPluralStems_ShortWordsNotRewritten verifies the ending-replacement rules are
// held back on short words, where they only ever landed on unrelated headwords.
func TestAddClassicalPluralStems_ShortWordsNotRewritten(t *testing.T) {
	// Each of these produced a real but wrong headword before the length gate.
	for _, tc := range []struct{ word, unwanted string }{
		{"acta", "acton"},
		{"ursa", "urson"},
		{"olea", "oleum"},
		{"sala", "salon"},
		{"frae", "fra"},
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.unwanted) >= 0 {
			t.Errorf("candidateStems(%q) still offers %q; %v", tc.word, tc.unwanted, c)
		}
	}
}

// TestAddClassicalPluralStems_LongWordsStillRewritten verifies the technical vocabulary the
// rules exist for is untouched by the length gate.
func TestAddClassicalPluralStems_LongWordsStillRewritten(t *testing.T) {
	for _, tc := range []struct{ word, want string }{
		{"cementa", "cementum"},
		{"conaria", "conarium"},
		{"acanthae", "acantha"},
		{"amphisbaenae", "amphisbaena"},
		{"phenomena", "phenomenon"},
		{"addenda", "addendum"},
		{"aerenchymata", "aerenchyma"},
		{"aerotaxes", "aerotaxis"},
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.want) < 0 {
			t.Errorf("candidateStems(%q) = %v, missing %q", tc.word, c, tc.want)
		}
	}
}

// TestAddDerivationStems_ReliableFamilies verifies the derivation families that were measured
// as reliable do produce their base, so a word derived from a defined term is explained by it
// rather than left with no definition at all.
func TestAddDerivationStems_ReliableFamilies(t *testing.T) {
	for _, tc := range []struct{ word, want string }{
		// A variant spelling differing by a silent final e, both directions.
		{"gentlenesse", "gentleness"},
		{"riboflavine", "riboflavin"},
		{"pterodactyle", "pterodactyl"},
		{"alkalin", "alkaline"},
		{"phosphocreatin", "phosphocreatine"},
		// Adjective forms of the same term.
		{"albitical", "albitic"},
		{"zoometrical", "zoometric"},
		{"orthogenically", "orthogenic"},
		{"incisural", "incisure"},
		// "capable of being X-ed".
		{"alkalisable", "alkalise"},
		{"socialisable", "socialise"},
		{"relegatable", "relegate"},
		// "having or pertaining to X".
		{"myriapodous", "myriapod"},
		{"microdontous", "microdont"},
		// "the act or result of X-ing".
		{"appetisement", "appetise"},
		// "resembling" or "full of X". A base shorter than minDerivedBase is excluded even
		// here, so "hoblike" does not reach "hob" — see TestAddDerivationStems_ShortBasesExcluded.
		{"werwolfish", "werwolf"},
		{"khakilike", "khaki"},
		{"courageful", "courage"},
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.want) < 0 {
			t.Errorf("candidateStems(%q) = %v, missing %q", tc.word, c, tc.want)
		}
	}
}

// TestAddDerivationStems_ExcludedFamilies verifies the families measured as misleading are not
// applied. Each names something other than its root, so the root's gloss would not explain it.
func TestAddDerivationStems_ExcludedFamilies(t *testing.T) {
	for _, tc := range []struct{ word, unwanted, why string }{
		{"aerobicist", "aerobic", "-ist names a person, not the adjective"},
		{"breist", "bree", "-ist must not strip a word that merely ends in those letters"},
		{"magism", "mag", "-ism names a doctrine, not the root"},
		{"aminity", "amin", "-ity is unreliable"},
		{"perseity", "perse", "-ity is unreliable"},
		{"chaiseless", "chaise", "-less negates the base, so its gloss states the opposite"},
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.unwanted) >= 0 {
			t.Errorf("candidateStems(%q) offers %q (%s); %v", tc.word, tc.unwanted, tc.why, c)
		}
	}
}

// TestAddDerivationStems_ShortBasesExcluded verifies the derivation rules are held back from
// short bases, where trimming a suffix reaches an unrelated word rather than the same one.
func TestAddDerivationStems_ShortBasesExcluded(t *testing.T) {
	for _, tc := range []struct{ word, unwanted string }{
		{"frae", "fra"}, // Scots "from" is not a friar
		{"idee", "ide"}, // not a fish
		{"cide", "cid"}, // not a lord
		{"awee", "awe"}, // Scots "a little while"
		{"tite", "tit"}, // archaic "quickly"
	} {
		if c := candidateStems(tc.word); indexOf(c, tc.unwanted) >= 0 {
			t.Errorf("candidateStems(%q) offers the short base %q; %v", tc.word, tc.unwanted, c)
		}
	}
}

// TestVariantCandidates_LigatureLengthGate verifies the ae/oe contractions apply only to the
// long Latinate spellings they target, not to short words where the same letters are ordinary
// adjacent vowels (most often Scots).
func TestVariantCandidates_LigatureLengthGate(t *testing.T) {
	// Short words must not be contracted.
	for _, tc := range []struct{ word, unwanted string }{
		{"haen", "hen"},
		{"kae", "ke"},
		{"thae", "the"},
		{"haeres", "here"},
	} {
		if c := variantCandidates(tc.word); indexOf(c, tc.unwanted) >= 0 {
			t.Errorf("variantCandidates(%q) still offers %q; %v", tc.word, tc.unwanted, c)
		}
	}
	// Long Latinate spellings still are.
	for _, tc := range []struct{ word, want string }{
		{"gynaecocratic", "gynecocratic"},
		{"hypaesthesia", "hypesthesia"},
		{"spelaeological", "speleological"},
		{"homoeomorph", "homeomorph"},
		{"oecologist", "ecologist"},
	} {
		if c := variantCandidates(tc.word); indexOf(c, tc.want) < 0 {
			t.Errorf("variantCandidates(%q) = %v, missing %q", tc.word, c, tc.want)
		}
	}
}

// TestVariantCandidates_NoOurToOrRewrite verifies the -our to -or contraction is gone. British
// spellings and their American forms are both headwords already, so the rule almost never fired
// legitimately and mostly turned Scots words into unrelated ones.
func TestVariantCandidates_NoOurToOrRewrite(t *testing.T) {
	for _, tc := range []struct{ word, unwanted string }{
		{"stoure", "store"},
		{"stoury", "story"},
		{"courd", "cord"},
		{"touries", "tories"},
	} {
		if c := variantCandidates(tc.word); indexOf(c, tc.unwanted) >= 0 {
			t.Errorf("variantCandidates(%q) still offers %q; %v", tc.word, tc.unwanted, c)
		}
	}
	// The -ise/-ize correspondences are unaffected: they are reliable in both directions.
	if c := variantCandidates("activise"); indexOf(c, "activize") < 0 {
		t.Errorf("variantCandidates(%q) = %v, missing %q", "activise", c, "activize")
	}
}

// TestEndsInSibilant covers the sound classes English inserts an e before.
func TestEndsInSibilant(t *testing.T) {
	for _, s := range []string{"box", "church", "bush", "buzz", "gas", "bus"} {
		if !endsInSibilant(s) {
			t.Errorf("endsInSibilant(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"bard", "agap", "buprenorphin", "cat", "", "a"} {
		if endsInSibilant(s) {
			t.Errorf("endsInSibilant(%q) = true, want false", s)
		}
	}
}
