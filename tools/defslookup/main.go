// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Command defslookup inspects a definitions .gob: it resolves individual words to
// their definitions and can audit the fuzzy-matched words in a word list, so the
// quality of the weakest resolution layer can be checked by eye.
//
// Usage:
//
//	go run ./tools/defslookup -db defs/assets/definitions/definitions.bin.gz cats baking qat
//	go run ./tools/defslookup -db definitions.bin.gz -audit wordlists/atebits-letterpress.txt -kind fuzzy
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"tilewords/defs"
)

func main() {
	dbFlag := flag.String("db", "", "Path to the definitions .gob (required)")
	auditFlag := flag.String("audit", "", "Word list to audit; prints each word resolved by -kind")
	kindFlag := flag.String("kind", "fuzzy", "Match kind to audit: exact|formof|stem|fuzzy")
	limitFlag := flag.Int("limit", 200, "Maximum audit lines to print")
	mergeFlag := flag.String("mergeaudit", "", "Word list to scan for words that are both a headword and an inflected form")
	flag.Parse()

	if *dbFlag == "" {
		fmt.Fprintln(os.Stderr, "defslookup: -db is required")
		os.Exit(1)
	}

	db, err := loadDB(*dbFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "defslookup: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "loaded %d headwords, %d inflection edges\n", db.Len(), db.FormCount())

	if *auditFlag != "" {
		if err := audit(db, *auditFlag, *kindFlag, *limitFlag); err != nil {
			fmt.Fprintf(os.Stderr, "defslookup: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *mergeFlag != "" {
		if err := mergeAudit(db, *mergeFlag, *limitFlag); err != nil {
			fmt.Fprintf(os.Stderr, "defslookup: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "defslookup: nothing to do; pass words to look up, or use -audit / -mergeaudit")
		flag.Usage()
		os.Exit(2)
	}

	for _, w := range flag.Args() {
		res, ok := db.Lookup(w)
		if !ok {
			fmt.Printf("%-24s  (no definition)\n", w)
			continue
		}
		gloss := ""
		if len(res.Entry.Senses) > 0 {
			s := res.Entry.Senses[0]
			gloss = strings.TrimSpace(s.POS + " " + s.Gloss)
		}
		fmt.Printf("%-24s  [%-6s -> %-20s] %s\n", w, res.Kind, res.Headword, gloss)
		if res.AlsoForm != nil && len(res.AlsoForm.Senses) > 0 {
			s := res.AlsoForm.Senses[0]
			fmt.Printf("%-24s     also form of %-13s %s\n", "", res.AlsoFormWord, strings.TrimSpace(s.POS+" "+s.Gloss))
		}
	}
}

// audit prints every word in the list that resolves via the named match kind,
// pairing it with the headword it reached so the mapping can be judged by eye.
func audit(db *defs.DB, path, kindName string, limit int) error {
	want, ok := parseKind(kindName)
	if !ok {
		return fmt.Errorf("unknown -kind %q; valid: exact, formof, stem, fuzzy", kindName)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	printed := 0
	var total, matched int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		w := strings.TrimSpace(sc.Text())
		if w == "" {
			continue
		}
		total++
		res, ok := db.Lookup(w)
		if !ok || res.Kind != want {
			continue
		}
		matched++
		if printed < limit {
			gloss := ""
			if len(res.Entry.Senses) > 0 {
				gloss = res.Entry.Senses[0].Gloss
			}
			fmt.Printf("%-28s -> %-24s %s\n", w, res.Headword, gloss)
			printed++
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	fmt.Fprintf(os.Stderr, "\n%s: %d of %d words matched via %s (showing %d)\n", path, matched, total, want, printed)
	return nil
}

// mergeAudit reports how many words in the list are both a headword (exact match)
// and a recorded inflected form of a different lemma, so the scope and asset-size
// impact of a lemma merge can be judged. The form-to-lemma edge these words need
// is already in the DB, so surfacing the lemma sense costs no extra asset bytes.
func mergeAudit(db *defs.DB, path string, limit int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	var total, eligible, edgePresent, printed int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		w := strings.TrimSpace(sc.Text())
		if w == "" {
			continue
		}
		total++
		res, ok := db.Lookup(w)
		if !ok || res.Kind != defs.MatchExact {
			continue
		}
		lemma, hasEdge := db.FormLemma(w)
		if !hasEdge || lemma == strings.ToLower(w) {
			continue
		}
		eligible++
		edgePresent++
		if printed < limit {
			fmt.Printf("%-20s headword + form of %-20s\n", w, lemma)
			printed++
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	fmt.Fprintf(os.Stderr, "\n%s: %d of %d words are both a headword and an inflected form (edge already in asset: %d)\n",
		path, eligible, total, edgePresent)
	return nil
}

// parseKind maps a kind name to its MatchKind, reporting ok=false for an
// unrecognised name so the caller can reject it rather than silently defaulting.
func parseKind(name string) (defs.MatchKind, bool) {
	switch strings.ToLower(name) {
	case "exact":
		return defs.MatchExact, true
	case "formof":
		return defs.MatchFormOf, true
	case "stem":
		return defs.MatchStem, true
	case "fuzzy":
		return defs.MatchFuzzy, true
	default:
		return defs.MatchNone, false
	}
}

// loadDB decodes a definitions DB from a .gob file.
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
