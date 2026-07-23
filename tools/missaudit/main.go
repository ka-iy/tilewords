// Command missaudit reports the words that have no definition in a definitions
// DB, aggregated and deduplicated across one or more word lists. It is the
// counterpart to defslookup's -audit (which inspects words that DO resolve): its
// job is to surface the coverage gap so it can be measured and closed.
//
// For each list it prints a coverage line (matched / total). Across all the lists
// it emits a single deduplicated, sorted list of the missing words — the words a
// player could form for which the game would show "no definition".
//
// The miss list goes to stdout (or -out) so it can be fed to a definition-sourcing
// step; the per-list and aggregate stats go to stderr so they don't pollute it.
//
// Usage:
//
//	# Deduplicated misses across every bundled list, to a file:
//	go run ./tools/missaudit -db defs/assets/definitions/definitions.gob.gz \
//	  -out missing-words.txt \
//	  wordlists/enable.txt wordlists/wordnik.txt wordlists/atebits-letterpress.txt
//
//	# Same, annotating each miss with the lists it came from:
//	go run ./tools/missaudit -db defs/assets/definitions/definitions.gob.gz -tag \
//	  wordlists/enable.txt wordlists/wordnik.txt wordlists/atebits-letterpress.txt
//
// missaudit is a developer diagnostic and is not part of the shipped app.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"tilewords/defs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "missaudit: %v\n", err)
		os.Exit(1)
	}
}

// listStat holds one word list's coverage against the definitions DB.
type listStat struct {
	// Path is the word-list file this stat describes.
	Path string
	// Total is the number of non-blank words read from the list.
	Total int
	// Missing is how many of those words resolved to no definition.
	Missing int
}

// run performs the audit and returns the first error encountered.
func run() error {
	dbFlag := flag.String("db", "", "Path to the definitions .gob(.gz) (required)")
	outFlag := flag.String("out", "", "Write the deduplicated miss list here (default: stdout)")
	tagFlag := flag.Bool("tag", false, "Annotate each miss with the tab-separated lists it appears in")
	flag.Parse()

	if *dbFlag == "" {
		return fmt.Errorf("-db is required")
	}
	lists := flag.Args()
	if len(lists) == 0 {
		return fmt.Errorf("no word lists given; pass one or more word-list paths as arguments")
	}

	db, err := loadDB(*dbFlag)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "loaded %d headwords, %d inflection edges from %s\n", db.Len(), db.FormCount(), *dbFlag)

	// missIn maps a canonical (lowercase) missing word to the set of list indices
	// it was missing in, so a miss can be attributed back to its source lists.
	missIn := make(map[string]map[int]bool)
	stats := make([]listStat, len(lists))

	for i, path := range lists {
		words, err := scanWords(path)
		if err != nil {
			return err
		}
		st := listStat{Path: path, Total: len(words)}
		for _, w := range words {
			if _, ok := db.Lookup(w); ok {
				continue
			}
			st.Missing++
			key := strings.ToLower(w)
			if missIn[key] == nil {
				missIn[key] = make(map[int]bool)
			}
			missIn[key][i] = true
		}
		stats[i] = st
	}

	misses := make([]string, 0, len(missIn))
	for w := range missIn {
		misses = append(misses, w)
	}
	sort.Strings(misses)

	if err := writeMisses(*outFlag, misses, missIn, lists, *tagFlag); err != nil {
		return err
	}
	reportStats(stats, len(misses))
	return nil
}

// scanWords reads a word list, returning its non-blank, trimmed words in file
// order. Blank lines are skipped so a trailing newline does not count as a word.
func scanWords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	var words []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		w := strings.TrimSpace(sc.Text())
		if w == "" {
			continue
		}
		words = append(words, w)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	return words, nil
}

// writeMisses emits the deduplicated, sorted miss list to dest ("" means stdout).
// With tag set, each line is the word followed by the tab-separated base names of
// the lists it was missing in; otherwise each line is just the word.
func writeMisses(dest string, misses []string, missIn map[string]map[int]bool, lists []string, tag bool) error {
	out := os.Stdout
	if dest != "" {
		f, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("create %q: %w", dest, err)
		}
		defer f.Close()
		out = f
	}

	w := bufio.NewWriter(out)
	for _, word := range misses {
		if tag {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", word, strings.Join(missListNames(missIn[word], lists), "\t")); err != nil {
				return fmt.Errorf("write miss list: %w", err)
			}
			continue
		}
		if _, err := fmt.Fprintln(w, word); err != nil {
			return fmt.Errorf("write miss list: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush miss list: %w", err)
	}
	return nil
}

// missListNames returns the base names of the lists a word was missing in, in the
// order the lists were given on the command line.
func missListNames(idx map[int]bool, lists []string) []string {
	names := make([]string, 0, len(idx))
	for i, path := range lists {
		if idx[i] {
			names = append(names, listName(path))
		}
	}
	return names
}

// listName reduces a list path to a short label: its base name without any
// trailing ".txt", so "wordlists/atebits-letterpress.txt" prints as
// "atebits-letterpress".
func listName(path string) string {
	base := path
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".txt")
}

// reportStats prints per-list coverage and the aggregate unique-miss count to
// stderr. uniqueMisses is the deduplicated miss count across all lists.
func reportStats(stats []listStat, uniqueMisses int) {
	fmt.Fprintln(os.Stderr)
	for _, st := range stats {
		matched := st.Total - st.Missing
		fmt.Fprintf(os.Stderr, "%-28s %7d / %-7d covered (%6.2f%%), %d missing\n",
			listName(st.Path), matched, st.Total, coverage(matched, st.Total), st.Missing)
	}
	fmt.Fprintf(os.Stderr, "\naggregate: %d unique words missing across %d list(s)\n", uniqueMisses, len(stats))
}

// coverage returns matched/total as a percentage, or 0 for an empty list so an
// empty input never divides by zero.
func coverage(matched, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(matched) / float64(total)
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
