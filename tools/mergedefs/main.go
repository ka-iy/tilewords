// Command mergedefs augments a definitions .gob with glosses from secondary
// public-domain / permissively-licensed dictionaries, closing part of the coverage
// gap for words the primary source (Wiktionary, via builddefs) does not define.
//
// It targets only the words that are actually missing from the given word lists,
// so it never grows the asset with definitions the game can't reach. For each
// missing word it resolves a gloss through the same layered matching the game uses
// (exact, then rule-based de-inflection, then guarded orthographic variants), so an
// inflected miss such as "abreges" can still pick up the lemma "abrege".
//
// Existing (primary-source) definitions and inflection edges always win: a
// supplemental entry is only added for a word the primary DB cannot already
// resolve. See defs.DB.WithSupplement.
//
// Sources (pass any combination):
//
//   - -webster: a Webster's 1913 Unabridged Dictionary JSON (a flat
//     {"word": "definition text", ...} object). Webster's 1913 is public domain.
//     Its glosses take precedence over WordNet's for words both define.
//   - -wordnet: a WordNet database directory (the "dict" folder holding data.noun,
//     data.verb, data.adj, data.adv). WordNet is distributed under its own
//     permissive license and requires attribution.
//   - -glossary: a supplemental glossary TSV ("word<TAB>gloss", or
//     "word<TAB>pos<TAB>gloss"; '#' comments and blank lines ignored). Repeatable,
//     and accepts "label=path" to name the source in the report. Used to fold in
//     smaller curated public-domain glossaries (e.g. Jamieson's Scots dictionary,
//     Spenser glossaries) held in the repo as reviewable text.
//
// Usage:
//
//	# Report how much each source would close, writing nothing:
//	go run ./tools/mergedefs -db defs/assets/definitions/definitions.bin.gz \
//	  -webster webster1913.json -wordnet wn/dict \
//	  -lists wordlists/enable.txt,wordlists/wordnik.txt,wordlists/atebits-letterpress.txt \
//	  -dryrun
//
//	# Produce an augmented asset:
//	go run ./tools/mergedefs -db defs/assets/definitions/definitions.bin.gz \
//	  -webster webster1913.json -wordnet wn/dict \
//	  -lists wordlists/enable.txt,wordlists/wordnik.txt,wordlists/atebits-letterpress.txt \
//	  -output defs/assets/definitions/definitions.bin.gz
//
// mergedefs is a developer build tool and is not part of the shipped app.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tilewords/defs"
)

// maxGlossLen caps a supplemental gloss at this many runes, matching the primary
// build's cap so definitions from every source read uniformly in the UI.
const maxGlossLen = 200

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "mergedefs: %v\n", err)
		os.Exit(1)
	}
}

// glossaryFlags collects the repeatable -glossary values in command-line order.
type glossaryFlags []string

// String renders the collected values for flag's usage output.
func (g *glossaryFlags) String() string { return strings.Join(*g, ",") }

// Set appends one -glossary occurrence.
func (g *glossaryFlags) Set(v string) error {
	*g = append(*g, v)
	return nil
}

// run parses flags, resolves missing words against the supplemental sources, and
// either reports (dry run) or writes the augmented asset.
func run() error {
	dbFlag := flag.String("db", "", "Path to the definitions .gob(.gz) to augment (required)")
	websterFlag := flag.String("webster", "", "Path to a Webster's 1913 JSON source (word->definition)")
	wordnetFlag := flag.String("wordnet", "", "Path to a WordNet dict directory (data.noun, data.verb, ...)")
	listsFlag := flag.String("lists", "", "Comma-separated word lists whose misses to target (required)")
	outFlag := flag.String("output", "", "Path to write the augmented .gob(.gz)")
	dryRun := flag.Bool("dryrun", false, "Report coverage gained without writing an asset")
	fuzzy := flag.Bool("fuzzy", false, "Also accept de-inflection / orthographic-variant matches to a supplemental headword (higher coverage, lower precision)")
	var glossaries glossaryFlags
	flag.Var(&glossaries, "glossary", "Supplemental glossary TSV (word<TAB>gloss, or word<TAB>pos<TAB>gloss); repeatable; accepts label=path")
	flag.Parse()

	if *dbFlag == "" {
		return fmt.Errorf("-db is required")
	}
	if *listsFlag == "" {
		return fmt.Errorf("-lists is required")
	}
	if *websterFlag == "" && *wordnetFlag == "" && len(glossaries) == 0 {
		return fmt.Errorf("at least one source (-webster, -wordnet, or -glossary) is required")
	}
	if !*dryRun && *outFlag == "" {
		return fmt.Errorf("-output is required unless -dryrun is set")
	}

	base, err := loadDB(*dbFlag)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "base: %d headwords, %d inflection edges\n", base.Len(), base.FormCount())

	// supEntries maps a supplemental headword to its parsed definition; srcOf
	// records which source it came from. Webster is loaded first so its glosses
	// win for any headword both sources define.
	supEntries := make(map[string]*defs.Entry)
	srcOf := make(map[string]string)

	if *websterFlag != "" {
		n, err := loadWebster(*websterFlag, supEntries, srcOf)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "webster: %d headwords loaded\n", n)
	}
	if *wordnetFlag != "" {
		n, err := loadWordNet(*wordnetFlag, supEntries, srcOf)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wordnet: %d new headwords loaded\n", n)
	}
	for _, spec := range glossaries {
		label, path := parseGlossarySpec(spec)
		n, err := loadGlossary(label, path, supEntries, srcOf)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%s: %d new headwords loaded\n", label, n)
	}

	// supDB reuses the game's layered Lookup to resolve a missing word to a
	// supplemental headword (exact, de-inflection, or orthographic variant).
	supDB := defs.NewDB(supEntries, nil)

	misses, listTotals, err := computeMisses(base, splitLists(*listsFlag))
	if err != nil {
		return err
	}

	addedEntries := make(map[string]*defs.Entry)
	addedForms := make(map[string]string)
	perSource := make(map[string]int)
	resolved := 0

	for _, m := range misses {
		res, ok := supDB.Lookup(m)
		if !ok {
			continue
		}
		// A supplemental DB carries no inflection edges, so its only high-precision
		// layer is an exact headword hit: the missing word is itself defined by a
		// source. Stem and orthographic-variant rewrites against the enlarged
		// headword set manufacture false matches (e.g. the classical-plural rule
		// maps "barca" onto the unrelated headword "barcon"), so they are accepted
		// only under -fuzzy. A wrong definition is worse than an honest "no
		// definition".
		if !*fuzzy && res.Kind != defs.MatchExact {
			continue
		}
		hw := res.Headword
		// Propose the supplemental headword's entry and, for an inflected miss, an
		// edge from the miss to it. WithSupplement drops either if the base already
		// has that key, so an edge can never dangle and the base always wins.
		addedEntries[hw] = supEntries[hw]
		if hw != m {
			addedForms[m] = hw
		}
		resolved++
		perSource[srcOf[hw]]++
	}

	reportMerge(listTotals, len(misses), resolved, perSource, len(addedEntries), len(addedForms))

	if *dryRun {
		return nil
	}

	aug := base.WithSupplement(addedEntries, addedForms)
	if err := writeDB(aug, *outFlag); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d headwords, %d inflection edges\n", *outFlag, aug.Len(), aug.FormCount())
	return nil
}

// splitLists splits a comma-separated -lists value into trimmed, non-empty paths.
func splitLists(csv string) []string {
	var out []string
	for _, p := range strings.Split(csv, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// listTotal is one word list's size and miss count against the base DB.
type listTotal struct {
	// Name is the list's short label (base name without a ".txt" suffix).
	Name string
	// Total is the number of non-blank words in the list.
	Total int
	// Missing is how many of those words the base DB cannot define directly (see computeMisses).
	Missing int
}

// definedDirectly reports whether a match kind means the DB holds a definition for the queried
// word itself, rather than explaining it through a related word.
func definedDirectly(k defs.MatchKind) bool {
	return k == defs.MatchExact || k == defs.MatchFormOf
}

// computeMisses returns the deduplicated, sorted set of lowercase words that the base DB
// cannot define *directly* across the given lists, plus each list's totals.
//
// A word counts as missing unless the base DB holds a definition for the word itself — an
// exact headword, or one Wiktionary records it as an inflected form of. A word that resolves
// only through the stem or orthographic-variant layers does NOT count as covered: those layers
// explain a word using a related word's gloss, so a supplement that can define the word itself
// is strictly better and must not be skipped. Treating any resolution as coverage let a
// broadening of the stem rules silently displace real definitions — "barde", "glace" and
// "geste" each lost their own Webster gloss to a stem guess at "bard", "glac" and "gest".
func computeMisses(base *defs.DB, lists []string) ([]string, []listTotal, error) {
	missing := make(map[string]bool)
	totals := make([]listTotal, 0, len(lists))
	for _, path := range lists {
		words, err := scanWords(path)
		if err != nil {
			return nil, nil, err
		}
		lt := listTotal{Name: listName(path), Total: len(words)}
		for _, w := range words {
			if res, ok := base.Lookup(w); ok && definedDirectly(res.Kind) {
				continue
			}
			lt.Missing++
			missing[strings.ToLower(w)] = true
		}
		totals = append(totals, lt)
	}
	misses := make([]string, 0, len(missing))
	for w := range missing {
		misses = append(misses, w)
	}
	sort.Strings(misses)
	return misses, totals, nil
}

// loadWebster reads a Webster's 1913 JSON ({"word": "definition"}) into entries,
// parsing each definition into senses. It only adds a headword not already present,
// so an earlier-loaded source keeps precedence. It returns the number added.
func loadWebster(path string, entries map[string]*defs.Entry, srcOf map[string]string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read webster %q: %w", path, err)
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, fmt.Errorf("parse webster %q: %w", path, err)
	}
	added := 0
	for word, def := range raw {
		hw := strings.ToLower(strings.TrimSpace(word))
		if hw == "" || entries[hw] != nil {
			continue
		}
		senses := parseWebsterSenses(def)
		if len(senses) == 0 {
			continue
		}
		entries[hw] = &defs.Entry{Word: hw, Senses: senses}
		srcOf[hw] = "webster1913"
		added++
	}
	return added, nil
}

// parseGlossarySpec splits a -glossary value into a source label and a path. The
// value may be "label=path" to set the label explicitly, or a bare path, in which
// case the label is the file's base name without its extension.
func parseGlossarySpec(spec string) (label, path string) {
	if i := strings.IndexByte(spec, '='); i >= 0 {
		return spec[:i], spec[i+1:]
	}
	base := filepath.Base(spec)
	return strings.TrimSuffix(base, filepath.Ext(base)), spec
}

// loadGlossary reads a supplemental glossary TSV into entries, one sense per line.
// Each line is "<word>\t<gloss>" or "<word>\t<pos>\t<gloss>"; blank lines and lines
// beginning with '#' are ignored. A word already supplied by an earlier source is
// skipped so precedence is preserved. It returns the number of headwords added.
func loadGlossary(label, path string, entries map[string]*defs.Entry, srcOf map[string]string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open glossary %q: %w", path, err)
	}
	defer f.Close()

	added := 0
	line := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line++
		raw := sc.Text()
		if t := strings.TrimSpace(raw); t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		fields := strings.Split(raw, "\t")
		if len(fields) < 2 {
			return 0, fmt.Errorf("%s:%d: expected <word>\\t<gloss>, got %q", path, line, raw)
		}
		word := strings.ToLower(strings.TrimSpace(fields[0]))
		pos := ""
		gloss := strings.TrimSpace(fields[1])
		if len(fields) >= 3 {
			pos = strings.TrimSpace(fields[1])
			gloss = strings.TrimSpace(fields[2])
		}
		gloss = cleanGloss(gloss)
		if word == "" || gloss == "" || entries[word] != nil {
			continue
		}
		entries[word] = &defs.Entry{Word: word, Senses: []defs.Sense{{POS: pos, Gloss: gloss}}}
		srcOf[word] = label
		added++
	}
	if err := sc.Err(); err != nil {
		return 0, fmt.Errorf("read glossary %q: %w", path, err)
	}
	return added, nil
}

// senseNumRE matches a candidate Webster sense number ("1.", "2.", ...) used as a boundary
// between numbered definitions within one entry. It requires the number to sit at
// the start of the text or after whitespace so mid-sentence figures are not split.
// Matching alone does not make a boundary — see websterSenseBoundaries.
var senseNumRE = regexp.MustCompile(`(?:^|\s)(\d{1,2})\.\s`)

// websterSenseBoundaries returns the matches in locs that are really sense numbers: those
// forming the run 1, 2, 3, … from the start. Webster's text is full of numbers that sit after
// whitespace and end in a period without introducing a sense — scripture citations ("Ps. xvi.
// 10.") and cross-references ("See Dit, n., 2.") most of all. Treating those as boundaries
// splits an entry at the wrong place, and because the text before the first boundary used to
// be dropped, it discarded the definition entirely: "sheol" shipped as "(Rev. Ver.)" and
// "ditt" as "[Obs.] Spenser.". Requiring the sequence to begin at 1 rejects both, leaving the
// entry as a single unsplit sense with its text intact.
func websterSenseBoundaries(def string, locs [][]int) [][]int {
	var out [][]int
	want := 1
	for _, loc := range locs {
		n, err := strconv.Atoi(def[loc[2]:loc[3]])
		if err != nil || n != want {
			continue
		}
		out = append(out, loc)
		want++
	}
	return out
}

// parseWebsterSenses splits a Webster's 1913 definition blob into senses on its
// numbered-definition boundaries, cleaning and capping each. Webster's compact
// text records no part of speech, so each sense's POS is left empty.
func parseWebsterSenses(def string) []defs.Sense {
	def = strings.TrimSpace(def)
	if def == "" {
		return nil
	}

	locs := websterSenseBoundaries(def, senseNumRE.FindAllStringSubmatchIndex(def, -1))
	var segments []string
	if len(locs) == 0 {
		segments = []string{def}
	} else {
		// Text before the first sense number is a definition too — Webster often leaves the
		// leading sense unnumbered — so it is emitted rather than skipped over.
		if lead := def[:locs[0][0]]; strings.TrimSpace(lead) != "" {
			segments = append(segments, lead)
		}
		for i, loc := range locs {
			start := loc[1] // text after this sense's "N. "
			end := len(def)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			segments = append(segments, def[start:end])
		}
	}

	var senses []defs.Sense
	for _, seg := range segments {
		g := cleanGloss(seg)
		if g == "" {
			continue
		}
		senses = append(senses, defs.Sense{Gloss: g})
		if len(senses) == defs.MaxSensesPerEntry {
			break
		}
	}
	return senses
}

// wordNetFiles are the WordNet data files parsed for glosses, one per open class.
var wordNetFiles = []string{"data.noun", "data.verb", "data.adj", "data.adv"}

// loadWordNet reads a WordNet dict directory, adding a sense per synset for each
// single-token lemma not already present in entries. It returns the number of new
// headwords added (lemmas an earlier source already supplied are skipped).
func loadWordNet(dir string, entries map[string]*defs.Entry, srcOf map[string]string) (int, error) {
	for _, name := range wordNetFiles {
		if err := loadWordNetFile(filepath.Join(dir, name), entries, srcOf); err != nil {
			return 0, err
		}
	}
	added := 0
	for _, src := range srcOf {
		if src == "wordnet" {
			added++
		}
	}
	return added, nil
}

// loadWordNetFile parses one WordNet data file, appending each synset's gloss to
// every single-token lemma it defines (up to the per-entry sense cap). Lemmas an
// earlier source already defined are left untouched so precedence is preserved.
func loadWordNetFile(path string, entries map[string]*defs.Entry, srcOf map[string]string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open wordnet %q: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		// The file header lines begin with two spaces; data lines never do.
		if strings.HasPrefix(line, "  ") {
			continue
		}
		lemmas, pos, gloss, ok := parseWordNetLine(line)
		if !ok {
			continue
		}
		for _, lemma := range lemmas {
			// Skip lemmas an earlier (higher-priority) source already defined,
			// but keep filling additional senses for a lemma WordNet itself owns.
			if src, seen := srcOf[lemma]; seen && src != "wordnet" {
				continue
			}
			e := entries[lemma]
			if e == nil {
				e = &defs.Entry{Word: lemma}
				entries[lemma] = e
				srcOf[lemma] = "wordnet"
			}
			if len(e.Senses) < defs.MaxSensesPerEntry {
				e.Senses = append(e.Senses, defs.Sense{POS: pos, Gloss: gloss})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read wordnet %q: %w", path, err)
	}
	return nil
}

// wordNetPOS maps a WordNet synset type code to a part-of-speech label. "s" is an
// adjective satellite, reported as an adjective like "a".
var wordNetPOS = map[string]string{"n": "noun", "v": "verb", "a": "adjective", "s": "adjective", "r": "adverb"}

// parseWordNetLine parses one WordNet data-file line into the single-token lemmas
// it defines, the part of speech, and the definition (the gloss with any usage
// examples removed). ok is false for a line that is not a parseable data record.
//
// A data line is: offset lex_file ss_type w_cnt [word lex_id]... p_cnt ... | gloss
// where w_cnt is a two-hex-digit word count and the gloss follows " | ".
func parseWordNetLine(line string) (lemmas []string, pos, gloss string, ok bool) {
	bar := strings.Index(line, " | ")
	if bar < 0 {
		return nil, "", "", false
	}
	head := strings.Fields(line[:bar])
	if len(head) < 5 {
		return nil, "", "", false
	}
	pos, ok = wordNetPOS[head[2]]
	if !ok {
		return nil, "", "", false
	}
	wcnt, err := strconv.ParseInt(head[3], 16, 0)
	if err != nil || wcnt <= 0 {
		return nil, "", "", false
	}
	// Words sit at head[4], head[6], ... (each followed by a lexical-id field).
	for i := 0; i < int(wcnt); i++ {
		idx := 4 + 2*i
		if idx >= len(head) {
			break
		}
		w := strings.ToLower(head[idx])
		// Only single-token lemmas can match a word-list entry; skip collocations
		// (WordNet joins their tokens with underscores).
		if strings.ContainsAny(w, "_ ") {
			continue
		}
		lemmas = append(lemmas, w)
	}
	if len(lemmas) == 0 {
		return nil, "", "", false
	}

	gloss = cleanGloss(wordNetDefinition(line[bar+3:]))
	if gloss == "" {
		return nil, "", "", false
	}
	return lemmas, pos, gloss, true
}

// wordNetDefinition returns the definition part of a WordNet gloss, dropping the
// usage examples that follow it. Examples are quoted and introduced by "; ", so
// the definition is the text before the first such quoted clause.
func wordNetDefinition(g string) string {
	if i := strings.Index(g, `; "`); i >= 0 {
		return g[:i]
	}
	return g
}

// cleanGloss collapses internal whitespace and truncates an over-long gloss at a
// word boundary with an ellipsis, so every source's definitions read uniformly.
func cleanGloss(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= maxGlossLen {
		return s
	}
	cut := maxGlossLen
	for cut > 0 && r[cut-1] != ' ' {
		cut--
	}
	if cut < maxGlossLen/2 { // no nearby space: hard-cut rather than lose most of it
		cut = maxGlossLen
	}
	return strings.TrimRight(string(r[:cut]), " ,;:") + "…"
}

// reportMerge prints per-list totals and the aggregate coverage gained to stderr.
func reportMerge(totals []listTotal, uniqueMisses, resolved int, perSource map[string]int, addedEntries, addedForms int) {
	fmt.Fprintln(os.Stderr)
	for _, lt := range totals {
		fmt.Fprintf(os.Stderr, "%-28s %d words, %d missing\n", lt.Name, lt.Total, lt.Missing)
	}
	fmt.Fprintf(os.Stderr, "\naggregate unique misses : %d\n", uniqueMisses)
	fmt.Fprintf(os.Stderr, "resolved by supplements : %d", resolved)
	if uniqueMisses > 0 {
		fmt.Fprintf(os.Stderr, " (%.1f%% of the gap)", 100*float64(resolved)/float64(uniqueMisses))
	}
	fmt.Fprintln(os.Stderr)
	for _, src := range sortedKeys(perSource) {
		fmt.Fprintf(os.Stderr, "  via %-12s %d\n", src, perSource[src])
	}
	fmt.Fprintf(os.Stderr, "remaining gap           : %d\n", uniqueMisses-resolved)
	fmt.Fprintf(os.Stderr, "new headword entries    : %d\n", addedEntries)
	fmt.Fprintf(os.Stderr, "new inflection edges    : %d\n", addedForms)
}

// sortedKeys returns the keys of m in sorted order for stable reporting.
func sortedKeys(m map[string]int) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// scanWords reads a word list, returning its non-blank, trimmed words in order.
func scanWords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	var words []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if w := strings.TrimSpace(sc.Text()); w != "" {
			words = append(words, w)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	return words, nil
}

// listName reduces a list path to a short label: its base name without a trailing
// ".txt", so "wordlists/atebits-letterpress.txt" prints as "atebits-letterpress".
func listName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".txt")
}

// loadDB decodes a definitions DB from a .gob(.gz) file.
func loadDB(path string) (*defs.DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open db %q: %w", path, err)
	}
	defer f.Close()
	db, err := defs.Decode(bufio.NewReaderSize(f, 1<<20))
	if err != nil {
		return nil, err
	}
	return db, nil
}

// writeDB encodes db to path as gzip-compressed gob, writing atomically via a
// temporary file so a failed encode cannot leave a truncated asset in place.
func writeDB(db *defs.DB, path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mergedefs-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %q: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	bw := bufio.NewWriterSize(tmp, 1<<20)
	if err := db.Encode(bw); err != nil {
		tmp.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		tmp.Close()
		return fmt.Errorf("flush %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp for %q: %w", path, err)
	}
	// os.CreateTemp makes the file 0600; the asset is a readable build output.
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod temp for %q: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp to %q: %w", path, err)
	}
	return nil
}
