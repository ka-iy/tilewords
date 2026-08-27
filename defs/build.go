// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package defs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
)

// MaxSensesPerEntry caps how many definitions are kept per headword, keeping the
// shipped asset small while retaining a word's most common senses.
const MaxSensesPerEntry = 4

// maxGlossLen caps a gloss at this many runes, truncating longer Wiktionary
// definitions (which can run to a paragraph) with an ellipsis.
const maxGlossLen = 200

// WordList names a word list file to measure coverage against and to filter the
// definitions down to.
type WordList struct {
	// Name is the label used in the coverage report (e.g. "atebits-letterpress").
	Name string
	// Path is the file path of the newline-separated word list.
	Path string
}

// ListCoverage is the per-list breakdown of how words resolved to definitions.
type ListCoverage struct {
	// Name is the word list's label.
	Name string
	// Total is the number of words in the list.
	Total int
	// Exact counts words that are themselves headwords.
	Exact int
	// FormOf counts words Wiktionary explicitly records as inflected forms.
	FormOf int
	// Stem counts words resolved by rule-based de-inflection.
	Stem int
	// Fuzzy counts words resolved by a guarded edit-distance match.
	Fuzzy int
	// Miss counts words with no definition from any layer.
	Miss int
	// SampleMisses holds up to sampleMissCount example unmatched words for inspection.
	SampleMisses []string
}

// Covered returns the number of words that resolved to a definition by any layer.
func (c ListCoverage) Covered() int { return c.Exact + c.FormOf + c.Stem + c.Fuzzy }

// Report summarises a build: per-list coverage plus headword/edge counts before
// and after filtering to the word lists.
type Report struct {
	// Lists holds one ListCoverage per input word list.
	Lists []ListCoverage
	// FullHeadwords is the number of English headwords parsed from the extract.
	FullHeadwords int
	// FullForms is the number of inflection edges parsed from the extract.
	FullForms int
	// ShippedHeadwords is the number of headwords retained after filtering.
	ShippedHeadwords int
	// ShippedForms is the number of inflection edges retained after filtering.
	ShippedForms int
	// RedirectSenses is the number of senses that only point at another word, which
	// DB.Lookup joins that word's definition onto. Like FullHeadwords it counts the whole
	// parse, not the filtered asset.
	RedirectSenses int
	// DroppedInitialisms is the number of senses discarded as initialism definitions.
	DroppedInitialisms int
	// RedirectTargets is the number of extra headwords retained because a retained
	// headword points at them; see the closure in BuildFilteredDB.
	RedirectTargets int
}

// sampleMissCount is how many example misses to retain per list for the report.
const sampleMissCount = 30

// inflectionTagOrder lists the form-table tags that mark an entry as an inflection of its
// lemma (as opposed to an alternative spelling, romanization, or misspelling, which are
// excluded to avoid mapping unrelated words together).
//
// They are listed in the order they read in as a description rather than alphabetically,
// because tagRelation joins them in it: a row tagged both "past" and "participle" has to
// come out as a past participle, not a participle past, and the order the row happens to
// list them in is not dependable.
var inflectionTagOrder = []string{
	"third-person", "singular", "plural",
	"comparative", "superlative",
	"present", "past",
	"participle", "present-participle", "past-participle",
	"gerund",
}

// inflectionTags is inflectionTagOrder as a set, for the membership tests. It is derived
// from the slice so the two cannot come to disagree about which tags mark an inflection.
var inflectionTags = func() map[string]struct{} {
	m := make(map[string]struct{}, len(inflectionTagOrder))
	for _, t := range inflectionTagOrder {
		m[t] = struct{}{}
	}
	return m
}()

// kaikkiEntry is the subset of a Wiktionary extract line that this package reads.
type kaikkiEntry struct {
	// Word is the headword.
	Word string `json:"word"`
	// Pos is the part of speech.
	Pos string `json:"pos"`
	// LangCode is the language; only "en" entries are used.
	LangCode string `json:"lang_code"`
	// Senses are the entry's definitions and form-of pointers.
	Senses []kaikkiSense `json:"senses"`
	// Forms is the lemma's inflection table (present on lemma entries).
	Forms []kaikkiForm `json:"forms"`
}

// kaikkiSense is one sense within a kaikkiEntry.
type kaikkiSense struct {
	// Glosses are the human-readable definition lines for this sense.
	Glosses []string `json:"glosses"`
	// Tags annotate the sense (e.g. "form-of").
	Tags []string `json:"tags"`
	// FormOf lists the lemmas this sense is an inflected form of.
	FormOf []kaikkiFormOf `json:"form_of"`
}

// kaikkiFormOf names a lemma a sense inflects.
type kaikkiFormOf struct {
	// Word is the lemma headword.
	Word string `json:"word"`
}

// kaikkiForm is one row of a lemma's inflection table.
type kaikkiForm struct {
	// Form is the inflected spelling.
	Form string `json:"form"`
	// Tags describe the inflection (e.g. "plural", "past").
	Tags []string `json:"tags"`
}

// lineMsg carries one extract line from the reader to a parse worker, tagged with
// its file position so the collector can restore file order.
type lineMsg struct {
	// seq is the zero-based line number in the extract file.
	seq int
	// data is the raw JSON line, owned by the message.
	data []byte
}

// formEdge records that an inflected form maps to a lemma headword.
type formEdge struct {
	// form is the inflected spelling.
	form string
	// lemma is the headword it inflects.
	lemma string
	// relation describes the inflection in Wiktionary's own words, or is empty when the
	// sense or table row this edge came from did not say. See Inflection.Relation.
	relation string
}

// rawExtract is the lightweight per-line result a parse worker emits to the collector.
type rawExtract struct {
	// seq is the extract's line number in the file. The collector orders a word's
	// entries by seq so the primary sense follows Wiktionary's own etymology order,
	// making the build deterministic despite concurrent parsing.
	seq int
	// word is the entry headword (lowercase).
	word string
	// primary holds ordinary-language senses, shown first.
	primary []Sense
	// secondary holds low-value senses (initialisms, abbreviations), shown only
	// when primary senses do not fill the entry.
	secondary []Sense
	// edges are inflection edges discovered for this entry, already filtered to
	// forms present in the word lists.
	edges []formEdge
	// redirects counts the senses kept that only point at another word, for the report.
	redirects int
	// initialisms counts the senses dropped as initialism definitions, which is
	// reported rather than left silent.
	initialisms int
}

// senseAccum gathers a word's senses across its several Wiktionary entries (one per
// etymology) before they are ranked and capped.
type senseAccum struct {
	// primary holds the ordinary-language senses, shown first.
	primary []Sense
	// secondary holds the low-value senses, shown only when primary does not fill the
	// entry; see lowValueGlossPrefixes and the redirect ranking in extractLine.
	secondary []Sense
}

// senseStats counts the senses the parse treated specially, for the build report. Both
// fields count over the whole extract, not just the words a list can form, so they
// describe the parse rather than the filtered asset.
type senseStats struct {
	// redirects is the number of senses kept that only point at another word.
	redirects int
	// initialisms is the number of senses dropped as initialism definitions.
	initialisms int
}

// properNounPOS is the part of speech Wiktionary gives proper nouns. Their senses
// (surnames, placenames, brands, sports clubs) are never the intended meaning of a
// lowercase game word, so entries with this part of speech are dropped.
const properNounPOS = "name"

// lowValueGlossPrefixes flag senses that define a word only as an abbreviation or
// symbol of another term. Such senses are kept but ranked below ordinary ones, so a
// common meaning (e.g. "za" as pizza) is shown ahead of an abbreviation.
//
// Most senses of this shape never reach the ranking, because they are redirects and the
// word they name supplies the definition instead. What is left here is the remainder
// whose term is not a single word the DB can be keyed by — "Abbreviation of
// air-conditioned" — for which the gloss is the only thing there is to show.
var lowValueGlossPrefixes = []string{
	"abbreviation of", "symbol for",
	"clipping of", "contraction of", "short for", "ellipsis of",
}

// classifySense reports whether a sense should be kept and, when kept, whether it
// is a primary (ordinary-language) sense as opposed to a low-value one.
func classifySense(pos, gloss string) (keep, primary bool) {
	if pos == properNounPOS {
		return false, false
	}
	lower := strings.ToLower(gloss)
	for _, p := range lowValueGlossPrefixes {
		if strings.HasPrefix(lower, p) {
			return true, false
		}
	}
	return true, true
}

// BuildFilteredDB parses the Wiktionary extract at kaikkiPath, measures how well
// each word list's words resolve to definitions, and returns a DB filtered to just
// the definitions those lists can reach.
//
// The returned DB is the shippable subset: only headwords some list word resolves
// to (and the inflection edges needed to reach them) are retained. The Report
// describes coverage against the full parse.
func BuildFilteredDB(kaikkiPath string, lists []WordList) (*DB, *Report, error) {
	listWords, needed, err := readLists(lists)
	if err != nil {
		return nil, nil, err
	}

	entries, formOf, stats, err := parseExtract(kaikkiPath, needed)
	if err != nil {
		return nil, nil, err
	}
	full := NewDB(entries, formOf)

	report := &Report{
		FullHeadwords:      len(entries),
		FullForms:          len(formOf),
		RedirectSenses:     stats.redirects,
		DroppedInitialisms: stats.initialisms,
	}
	keep := make(map[string]struct{}) // headwords any list word resolves to

	for i, wl := range lists {
		cov := ListCoverage{Name: wl.Name, Total: len(listWords[i])}
		for _, w := range listWords[i] {
			res, ok := full.Lookup(w)
			if !ok {
				cov.Miss++
				if len(cov.SampleMisses) < sampleMissCount {
					cov.SampleMisses = append(cov.SampleMisses, w)
				}
				continue
			}
			keep[res.Headword] = struct{}{}
			switch res.Kind {
			case MatchExact:
				cov.Exact++
			case MatchFormOf:
				cov.FormOf++
			case MatchStem:
				cov.Stem++
			case MatchFuzzy:
				cov.Fuzzy++
			}
		}
		report.Lists = append(report.Lists, cov)
	}

	report.RedirectTargets = keepRedirectTargets(keep, entries)

	shipEntries := make(map[string]*Entry, len(keep))
	for hw := range keep {
		if e := entries[hw]; e != nil {
			shipEntries[hw] = e
		}
	}
	shipForms := make(map[string]Inflection)
	for form, inf := range formOf {
		if _, ok := keep[inf.Lemma]; ok {
			shipForms[form] = inf
		}
	}
	report.ShippedHeadwords = len(shipEntries)
	report.ShippedForms = len(shipForms)

	return NewDB(shipEntries, shipForms), report, nil
}

// keepRedirectTargets adds to keep every word a kept headword only points at, and then
// every word those point at in turn, returning how many were added.
//
// A retained entry that says only "Alternative form of X" is answered by joining X's
// definition onto it at lookup, so X has to be in the asset even when no word list holds
// it. Following the closure rather than one step keeps the intermediates of a chain too,
// which is what lets the lookup walk from one end of it to the other.
func keepRedirectTargets(keep map[string]struct{}, entries map[string]*Entry) int {
	queue := make([]string, 0, len(keep))
	for w := range keep {
		queue = append(queue, w)
	}
	added := 0
	for len(queue) > 0 {
		word := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		e := entries[word]
		if e == nil {
			continue
		}
		for _, s := range e.Senses {
			target, ok := redirectTarget(s.Gloss)
			if !ok || target == word || entries[target] == nil {
				continue
			}
			if _, done := keep[target]; done {
				continue
			}
			keep[target] = struct{}{}
			queue = append(queue, target)
			added++
		}
	}
	return added
}

// readLists loads each word list, returning the per-list ordered words and the
// lowercase union of all words (the set whose coverage matters and to which
// inflection edges are limited).
func readLists(lists []WordList) ([][]string, map[string]struct{}, error) {
	listWords := make([][]string, len(lists))
	needed := make(map[string]struct{})
	for i, wl := range lists {
		f, err := os.Open(wl.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("open word list %q: %w", wl.Path, err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			w := strings.ToLower(strings.TrimSpace(sc.Text()))
			if w == "" {
				continue
			}
			listWords[i] = append(listWords[i], w)
			needed[w] = struct{}{}
		}
		err = sc.Err()
		f.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("read word list %q: %w", wl.Path, err)
		}
	}
	return listWords, needed, nil
}

// parseExtract streams the Wiktionary extract and returns the English headword
// definitions plus the resolved inflection edges (form to a lemma that has an
// entry). Only edges whose form is in needed are kept, bounding memory to the
// word lists' scope. Parsing runs across worker goroutines; a single collector
// merges their results. The returned stats describe what the redirect pass did.
func parseExtract(path string, needed map[string]struct{}) (map[string]*Entry, map[string]Inflection, senseStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, senseStats{}, fmt.Errorf("open extract %q: %w", path, err)
	}
	defer f.Close()

	lines := make(chan lineMsg, 4096)
	extracts := make(chan rawExtract, 4096)

	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range lines {
				if ex, ok := extractLine(msg.data, needed); ok {
					ex.seq = msg.seq
					extracts <- ex
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(extracts)
	}()

	// Reader goroutine: feed newline-delimited lines to the workers, tagging each
	// with its position so the collector can restore file order.
	var readErr error
	go func() {
		defer close(lines)
		r := bufio.NewReaderSize(f, 1<<20)
		seq := 0
		for {
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				buf := make([]byte, len(line))
				copy(buf, line)
				lines <- lineMsg{seq: seq, data: buf}
				seq++
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				readErr = fmt.Errorf("read extract %q: %w", path, err)
				return
			}
		}
	}()

	// Drain every extract, then restore file order by seq. Concurrent workers emit
	// out of order; sorting makes both the primary-sense choice and edge resolution
	// deterministic.
	all := make([]rawExtract, 0, 1<<19)
	for ex := range extracts {
		all = append(all, ex)
	}
	if readErr != nil {
		return nil, nil, senseStats{}, readErr
	}
	sort.Slice(all, func(i, j int) bool { return all[i].seq < all[j].seq })

	// collectCap bounds retained senses per tier during collection, keeping memory
	// bounded while leaving headroom to prefer primary senses at ranking time.
	const collectCap = MaxSensesPerEntry * 2

	var stats senseStats
	acc := make(map[string]*senseAccum)
	edges := make([]formEdge, 0, 1<<20)
	for _, ex := range all {
		if len(ex.primary) > 0 || len(ex.secondary) > 0 {
			a := acc[ex.word]
			if a == nil {
				a = &senseAccum{}
				acc[ex.word] = a
			}
			a.primary = appendCapped(a.primary, ex.primary, collectCap)
			a.secondary = appendCapped(a.secondary, ex.secondary, collectCap)
		}
		edges = append(edges, ex.edges...)
		stats.redirects += ex.redirects
		stats.initialisms += ex.initialisms
	}

	entries := make(map[string]*Entry, len(acc))
	for word, a := range acc {
		senses := a.primary
		if len(senses) < MaxSensesPerEntry {
			senses = append(senses, a.secondary...)
		}
		if len(senses) > MaxSensesPerEntry {
			senses = senses[:MaxSensesPerEntry]
		}
		if len(senses) == 0 {
			continue
		}
		entries[word] = &Entry{Word: word, Senses: senses}
	}

	formOf := resolveEdges(edges, entries)
	return entries, formOf, stats, nil
}

// appendCapped appends items from src to dst, stopping once dst reaches limit.
func appendCapped[T any](dst, src []T, limit int) []T {
	for _, s := range src {
		if len(dst) >= limit {
			break
		}
		dst = append(dst, s)
	}
	return dst
}

// extractLine parses one extract line into a rawExtract, or reports ok=false when
// the line is not a usable English single-word entry. Form-of senses become
// inflection edges (kept only when the form is in needed); the remaining senses
// become real definitions.
func extractLine(line []byte, needed map[string]struct{}) (rawExtract, bool) {
	line = trimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return rawExtract{}, false
	}
	var ke kaikkiEntry
	if err := json.Unmarshal(line, &ke); err != nil {
		return rawExtract{}, false
	}
	if ke.LangCode != "en" {
		return rawExtract{}, false
	}
	word := strings.ToLower(ke.Word)
	if !isAlpha(word) {
		return rawExtract{}, false
	}

	ex := rawExtract{word: word}
	_, wordNeeded := needed[word]

	for _, s := range ke.Senses {
		if lemma, isForm := formTarget(s); isForm {
			if wordNeeded && lemma != "" {
				ex.edges = append(ex.edges, formEdge{
					form:     word,
					lemma:    lemma,
					relation: inflectionRelation(senseGloss(s.Glosses)),
				})
			}
			continue
		}
		for _, g := range s.Glosses {
			clean := cleanGloss(g)
			if clean == "" {
				continue
			}
			keep, primary := classifySense(ke.Pos, clean)
			if !keep {
				continue
			}
			// The initialism test runs before the redirect test because an initialism
			// gloss can read as a redirect to the last word of the term it expands
			// ("Initialism of form of payment"), which is not a word the player formed.
			if isInitialismGloss(clean) {
				ex.initialisms++
				continue
			}
			sense := Sense{POS: ke.Pos, Gloss: clean}
			// A sense that only points at another word is ranked with the low-value
			// ones however Wiktionary phrased it: on its own it says nothing about the
			// word, and what makes it useful — the other word's definition — is joined
			// on at lookup, not here.
			if _, isRedirect := redirectTarget(clean); isRedirect {
				ex.redirects++
				primary = false
			}
			if primary {
				ex.primary = append(ex.primary, sense)
			} else {
				ex.secondary = append(ex.secondary, sense)
			}
		}
	}

	for _, fm := range ke.Forms {
		form := strings.ToLower(fm.Form)
		if form == word || !isAlpha(form) || !hasInflectionTag(fm.Tags) {
			continue
		}
		if _, ok := needed[form]; ok {
			ex.edges = append(ex.edges, formEdge{form: form, lemma: word, relation: tagRelation(fm.Tags)})
		}
	}

	if len(ex.primary) == 0 && len(ex.secondary) == 0 && len(ex.edges) == 0 &&
		ex.initialisms == 0 {
		return rawExtract{}, false
	}
	return ex, true
}

// formTarget reports whether a sense is an inflected-form pointer and, if so, the
// lemma it points at. It reads the explicit form_of field first, falling back to
// parsing a "... of <lemma>" gloss when the sense is tagged form-of without one.
func formTarget(s kaikkiSense) (string, bool) {
	if len(s.FormOf) > 0 {
		return strings.ToLower(s.FormOf[0].Word), true
	}
	if !containsTag(s.Tags, "form-of") {
		return "", false
	}
	for _, g := range s.Glosses {
		fields := strings.Fields(g)
		if len(fields) >= 3 && fields[len(fields)-2] == "of" {
			return strings.ToLower(strings.Trim(fields[len(fields)-1], ".,;:\"'()")), true
		}
	}
	return "", true
}

// maxRelationLen caps how long an inflection description may be. Wiktionary phrases these
// from a handful of templates, the longest of which runs to about forty characters
// ("third-person singular simple present indicative"); anything beyond this is a gloss
// that only looks like one, and is no label to put in front of a definition.
const maxRelationLen = 64

// senseGloss returns the first of a sense's gloss lines, or "" when it has none. It is
// named apart from DB.firstGloss, which answers the same question of a built entry.
func senseGloss(glosses []string) string {
	if len(glosses) == 0 {
		return ""
	}
	return glosses[0]
}

// inflectionRelation returns how a form-of gloss describes the inflection: the words
// before the lemma it names, so "simple past and past participle of abandon" gives
// "simple past and past participle". It returns "" when the gloss is not of that shape,
// or describes the inflection at a length no label should carry.
func inflectionRelation(gloss string) string {
	fields := strings.Fields(strings.TrimSuffix(strings.TrimSpace(gloss), "."))
	if len(fields) < 3 || !strings.EqualFold(fields[len(fields)-2], "of") {
		return ""
	}
	rel := strings.ToLower(strings.Join(fields[:len(fields)-2], " "))
	if len(rel) > maxRelationLen {
		return ""
	}
	return rel
}

// tagRelation describes an inflection-table row in the tags Wiktionary gave it, so a form
// reached through a lemma's table is labelled like one reached through a gloss: the tags
// ["past", "participle"] give "past participle". The description is assembled in
// inflectionTagOrder rather than the row's own order, tags that name no inflection are
// left out, and a row naming none gives "".
func tagRelation(tags []string) string {
	parts := make([]string, 0, len(tags))
	for _, want := range inflectionTagOrder {
		if slices.Contains(tags, want) {
			parts = append(parts, want)
		}
	}
	return strings.Join(parts, " ")
}

// resolveEdges turns raw inflection edges into a form-to-lemma map, keeping for
// each form the first lemma candidate that has a real entry (preferring a lemma
// that carries definitions). Edges to lemmas absent from entries are dropped.
func resolveEdges(edges []formEdge, entries map[string]*Entry) map[string]Inflection {
	formOf := make(map[string]Inflection)
	for _, e := range edges {
		if e.form == "" || e.lemma == "" {
			continue
		}
		if _, dup := formOf[e.form]; dup {
			continue
		}
		if le := entries[e.lemma]; le != nil && len(le.Senses) > 0 {
			formOf[e.form] = Inflection{Lemma: e.lemma, Relation: e.relation}
		}
	}
	// Second pass: forms whose first-choice lemma had no definitions may still map
	// to a lemma that exists as a headword; accept those so the edge is usable.
	for _, e := range edges {
		if e.form == "" || e.lemma == "" {
			continue
		}
		if _, done := formOf[e.form]; done {
			continue
		}
		if entries[e.lemma] != nil {
			formOf[e.form] = Inflection{Lemma: e.lemma, Relation: e.relation}
		}
	}
	// Third pass: one inflection is often recorded twice, by the form's own entry and by
	// the lemma's inflection table, and only one of the two need describe it. Fill a
	// missing description in from another edge that agrees on the lemma, so which of them
	// the extract happened to list first does not decide whether the form is labelled.
	for _, e := range edges {
		if e.relation == "" {
			continue
		}
		inf, ok := formOf[e.form]
		if !ok || inf.Relation != "" || inf.Lemma != e.lemma {
			continue
		}
		inf.Relation = e.relation
		formOf[e.form] = inf
	}
	return formOf
}

// cleanGloss normalises whitespace and truncates a gloss to maxGlossLen runes.
func cleanGloss(g string) string {
	g = strings.Join(strings.Fields(g), " ")
	if g == "" {
		return ""
	}
	runes := []rune(g)
	if len(runes) > maxGlossLen {
		return strings.TrimRight(string(runes[:maxGlossLen-1]), " ") + "…"
	}
	return g
}

// hasInflectionTag reports whether tags contains any recognised inflection tag.
func hasInflectionTag(tags []string) bool {
	for _, t := range tags {
		if _, ok := inflectionTags[t]; ok {
			return true
		}
	}
	return false
}

// containsTag reports whether tags contains want.
func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// isAlpha reports whether s is non-empty and all lowercase ASCII letters.
func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}
	return true
}

// trimSpace trims leading and trailing ASCII whitespace from a byte slice.
func trimSpace(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\r' || b[i] == '\n') {
		i++
	}
	j := len(b)
	for j > i && (b[j-1] == ' ' || b[j-1] == '\t' || b[j-1] == '\r' || b[j-1] == '\n') {
		j--
	}
	return b[i:j]
}
