// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Command builddefs filters a Wiktionary extract down to the definitions reachable
// from one or more word lists and writes the gob-encoded result.
//
// Usage:
//
//	go run ./tools/builddefs \
//	    -kaikki path/to/kaikki-en.jsonl \
//	    -input wordlists/enable.txt,wordlists/wordnik.txt,wordlists/atebits-letterpress.txt \
//	    -output defs/assets/definitions/definitions.bin.gz
//
// Each -input file is measured for coverage independently (its name is the file
// stem). The shipped DB is the union of definitions all lists can reach, resolved
// via exact, form-of, stem, and fuzzy matching. A coverage report is printed to
// stdout.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tilewords/defs"
)

func main() {
	kaikkiFlag := flag.String("kaikki", "", "Path to the Wiktionary extract JSONL (required)")
	inputFlag := flag.String("input", "", "Comma-separated word list file path(s) (required)")
	outputFlag := flag.String("output", "", "Output .gob file path (required)")
	flag.Parse()

	if *kaikkiFlag == "" || *inputFlag == "" || *outputFlag == "" {
		fmt.Fprintln(os.Stderr, "builddefs: -kaikki, -input, and -output are all required")
		flag.Usage()
		os.Exit(1)
	}

	var lists []defs.WordList
	for _, p := range strings.Split(*inputFlag, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		lists = append(lists, defs.WordList{Name: name, Path: p})
	}
	if len(lists) == 0 {
		fmt.Fprintln(os.Stderr, "builddefs: -input contained no word list paths")
		os.Exit(1)
	}

	db, report, err := defs.BuildFilteredDB(*kaikkiFlag, lists)
	if err != nil {
		fmt.Fprintf(os.Stderr, "builddefs: %v\n", err)
		os.Exit(1)
	}

	if err := writeGOB(*outputFlag, db); err != nil {
		fmt.Fprintf(os.Stderr, "builddefs: %v\n", err)
		os.Exit(1)
	}

	printReport(report, *outputFlag)
}

// writeGOB encodes db to outPath atomically via a temp file and rename, so a
// failed encode never leaves a partial .gob behind.
func writeGOB(outPath string, db *defs.DB) error {
	if dir := filepath.Dir(outPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output dir %q: %w", dir, err)
		}
	}
	tmpPath := outPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file %q: %w", tmpPath, err)
	}
	w := bufio.NewWriterSize(f, 1<<20)

	encErr := db.Encode(w)
	flushErr := w.Flush()
	closeErr := f.Close()

	if err := firstErr(encErr, flushErr, closeErr); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("encode DB: %w", err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %q -> %q: %w", tmpPath, outPath, err)
	}
	return nil
}

// printReport writes the coverage breakdown for every list plus the shipped-asset
// summary to stdout.
func printReport(r *defs.Report, outPath string) {
	fmt.Printf("\nbuilddefs coverage report\n")
	fmt.Printf("  parsed %d English headwords, %d inflection edges\n\n", r.FullHeadwords, r.FullForms)

	fmt.Printf("  %-22s %9s %8s %8s %8s %8s %8s %9s\n",
		"list", "total", "exact", "formof", "stem", "fuzzy", "miss", "covered")
	for _, c := range r.Lists {
		pct := 0.0
		if c.Total > 0 {
			pct = float64(c.Covered()) / float64(c.Total) * 100
		}
		fmt.Printf("  %-22s %9d %8d %8d %8d %8d %8d %8.1f%%\n",
			c.Name, c.Total, c.Exact, c.FormOf, c.Stem, c.Fuzzy, c.Miss, pct)
	}

	fmt.Printf("\n  shipped asset: %d headwords, %d inflection edges -> %s\n",
		r.ShippedHeadwords, r.ShippedForms, outPath)

	for _, c := range r.Lists {
		if len(c.SampleMisses) == 0 {
			continue
		}
		fmt.Printf("\n  sample misses (%s): %s\n", c.Name, strings.Join(c.SampleMisses, " "))
	}
}

// firstErr returns the first non-nil error among errs, or nil.
func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
