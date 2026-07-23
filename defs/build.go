package defs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
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
}

// sampleMissCount is how many example misses to retain per list for the report.
const sampleMissCount = 30

// inflectionTags are the form-table tags that mark an entry as an inflection of
// its lemma (as opposed to an alternative spelling, romanization, or misspelling,
// which are excluded to avoid mapping unrelated words together).
var inflectionTags = map[string]struct{}{
	"plural": {}, "singular": {}, "past": {}, "participle": {},
	"gerund": {}, "present": {}, "comparative": {}, "superlative": {},
	"third-person": {}, "past-participle": {}, "present-participle": {},
}

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
}

// properNounPOS is the part of speech Wiktionary gives proper nouns. Their senses
// (surnames, placenames, brands, sports clubs) are never the intended meaning of a
// lowercase game word, so entries with this part of speech are dropped.
const properNounPOS = "name"

// lowValueGlossPrefixes flag senses that define a word only as an abbreviation or
// symbol. Such senses are kept but ranked below ordinary ones, so a common meaning
// (e.g. "za" as pizza) is shown ahead of an initialism ("za" as zinc-aluminium).
var lowValueGlossPrefixes = []string{
	"initialism of", "abbreviation of", "acronym of", "symbol for",
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

	entries, formOf, err := parseExtract(kaikkiPath, needed)
	if err != nil {
		return nil, nil, err
	}
	full := NewDB(entries, formOf)

	report := &Report{FullHeadwords: len(entries), FullForms: len(formOf)}
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

	shipEntries := make(map[string]*Entry, len(keep))
	for hw := range keep {
		if e := entries[hw]; e != nil {
			shipEntries[hw] = e
		}
	}
	shipForms := make(map[string]string)
	for form, lemma := range formOf {
		if _, ok := keep[lemma]; ok {
			shipForms[form] = lemma
		}
	}
	report.ShippedHeadwords = len(shipEntries)
	report.ShippedForms = len(shipForms)

	return NewDB(shipEntries, shipForms), report, nil
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
// merges their results.
func parseExtract(path string, needed map[string]struct{}) (map[string]*Entry, map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open extract %q: %w", path, err)
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
		return nil, nil, readErr
	}
	sort.Slice(all, func(i, j int) bool { return all[i].seq < all[j].seq })

	// senseAccum gathers a word's senses across its several Wiktionary entries
	// (one per etymology) before they are ranked and capped.
	type senseAccum struct {
		primary   []Sense
		secondary []Sense
	}
	// collectCap bounds retained senses per tier during collection, keeping memory
	// bounded while leaving headroom to prefer primary senses at ranking time.
	const collectCap = MaxSensesPerEntry * 2

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
	return entries, formOf, nil
}

// appendCapped appends senses from src to dst, stopping once dst reaches limit.
func appendCapped(dst, src []Sense, limit int) []Sense {
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
				ex.edges = append(ex.edges, formEdge{form: word, lemma: lemma})
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
			sense := Sense{POS: ke.Pos, Gloss: clean}
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
			ex.edges = append(ex.edges, formEdge{form: form, lemma: word})
		}
	}

	if len(ex.primary) == 0 && len(ex.secondary) == 0 && len(ex.edges) == 0 {
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

// resolveEdges turns raw inflection edges into a form-to-lemma map, keeping for
// each form the first lemma candidate that has a real entry (preferring a lemma
// that carries definitions). Edges to lemmas absent from entries are dropped.
func resolveEdges(edges []formEdge, entries map[string]*Entry) map[string]string {
	formOf := make(map[string]string)
	for _, e := range edges {
		if e.form == "" || e.lemma == "" {
			continue
		}
		if _, dup := formOf[e.form]; dup {
			continue
		}
		if le := entries[e.lemma]; le != nil && len(le.Senses) > 0 {
			formOf[e.form] = e.lemma
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
			formOf[e.form] = e.lemma
		}
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
