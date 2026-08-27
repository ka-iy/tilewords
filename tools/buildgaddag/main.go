// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Command buildgaddag constructs a GADDAG from one or more word list files and
// writes the serialised result to an output file.
//
// Usage:
//
//	go run ./tools/buildgaddag \
//	    -input path/to/wordlist.txt[,path/to/other.txt] \
//	    -output dictionary/assets/dictionaries/<name>.bin \
//	    -name <name> \
//	    [-exclude defs/possibly_invalid_words.txt]
//
// The build is deterministic: same input words always produce identical output bytes.
// Input words are normalised to uppercase; non-A-Z words are skipped with a warning.
// When multiple -input files are supplied, the word lists are merged and deduplicated.
//
// -exclude names a file of words to leave out of the asset, so a word a shipped list
// carries but no dictionary defines is not accepted as a play. Its format is one entry
// per line, the word first and anything after the first field (such as the source word
// lists) ignored, with '#' comments and blank lines skipped.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"tilewords/dictionary"
)

func main() {
	inputFlag := flag.String("input", "", "Comma-separated word list file path(s) (required)")
	outputFlag := flag.String("output", "", "Output asset file path (required)")
	nameFlag := flag.String("name", "", "Dictionary name (e.g. csw, naspa, all) (required)")
	excludeFlag := flag.String("exclude", "", "File of words to omit from the asset (optional)")
	flag.Parse()

	if *inputFlag == "" || *outputFlag == "" || *nameFlag == "" {
		fmt.Fprintln(os.Stderr, "buildgaddag: -input, -output, and -name are all required")
		flag.Usage()
		os.Exit(1)
	}

	excluded, err := readExclusions(*excludeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildgaddag: %v\n", err)
		os.Exit(1)
	}

	inputPaths := strings.Split(*inputFlag, ",")
	words, dropped, err := readWords(inputPaths, excluded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildgaddag: %v\n", err)
		os.Exit(1)
	}
	if len(dropped) > 0 {
		fmt.Fprintf(os.Stderr, "buildgaddag: omitted %d excluded word(s): %s\n",
			len(dropped), strings.Join(dropped, ", "))
	}

	if len(words) == 0 {
		fmt.Fprintln(os.Stderr, "buildgaddag: no valid words found in input file(s)")
		os.Exit(1)
	}

	if err := writeAsset(*outputFlag, words); err != nil {
		fmt.Fprintf(os.Stderr, "buildgaddag: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "buildgaddag: wrote %d words to %s (dict=%s)\n", len(words), *outputFlag, *nameFlag)
}

// readExclusions reads the -exclude file into a set of uppercased words. An empty path
// yields an empty set, which excludes nothing. Only the first whitespace-separated field
// of a line is taken as the word, so an entry may carry trailing columns (the source word
// lists, say) without them being mistaken for part of it.
func readExclusions(path string) (map[string]struct{}, error) {
	excluded := make(map[string]struct{})
	if strings.TrimSpace(path) == "" {
		return excluded, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open exclusion file %q: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		excluded[strings.ToUpper(strings.Fields(line)[0])] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan exclusion file %q: %w", path, err)
	}
	return excluded, nil
}

// readWords reads word lists from all given file paths, normalises to uppercase, skips
// invalid entries with a warning, and returns a sorted deduplicated slice. Words present
// in excluded are left out; the second return value lists those actually dropped, sorted,
// so a stale exclusion entry that matches nothing is visible as an absence in the report.
func readWords(paths []string, excluded map[string]struct{}) ([]string, []string, error) {
	seen := make(map[string]struct{})
	var words []string
	droppedSet := make(map[string]struct{})

	for _, path := range paths {
		path = strings.TrimSpace(path)
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, fmt.Errorf("open %q: %w", path, err)
		}

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			raw := strings.TrimSpace(scanner.Text())
			if raw == "" {
				continue
			}
			upper := strings.ToUpper(raw)

			// Validate length (Appel-Jacobson §3 requires 2 ≤ len ≤ 15 for a 15×15 board)
			if len(upper) < dictionary.MinWordLen || len(upper) > dictionary.MaxWordLen {
				fmt.Fprintf(os.Stderr, "buildgaddag: %s:%d: skipping %q (length %d out of range [%d,%d])\n",
					path, lineNum, raw, len(upper), dictionary.MinWordLen, dictionary.MaxWordLen)
				continue
			}

			// Validate bytes: A-Z only
			valid := true
			for _, b := range []byte(upper) {
				if b < 'A' || b > 'Z' {
					valid = false
					break
				}
			}
			if !valid {
				fmt.Fprintf(os.Stderr, "buildgaddag: %s:%d: skipping %q (contains non-A-Z character)\n",
					path, lineNum, raw)
				continue
			}

			if _, skip := excluded[upper]; skip {
				droppedSet[upper] = struct{}{}
				continue
			}

			if _, dup := seen[upper]; !dup {
				seen[upper] = struct{}{}
				words = append(words, upper)
			}
		}
		if err := scanner.Err(); err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("scan %q: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return nil, nil, fmt.Errorf("close %q: %w", path, err)
		}
	}

	dropped := make([]string, 0, len(droppedSet))
	for w := range droppedSet {
		dropped = append(dropped, w)
	}
	sort.Strings(dropped)

	sort.Strings(words)
	return words, dropped, nil
}

// writeAsset constructs a GADDAG from words and writes it to outPath in the layout
// documented at the top of dictionary/gaddag.go. The write is atomic: it writes to a temp
// file first and renames on success, so no partial asset is ever left on disk for a later
// build to treat as complete.
func writeAsset(outPath string, words []string) error {
	// Write to a temporary file alongside the output to enable atomic rename.
	tmpPath := outPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file %q: %w", tmpPath, err)
	}

	buildErr := dictionary.Build(words, f)
	closeErr := f.Close()

	if buildErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("build GADDAG: %w", buildErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", closeErr)
	}

	if err := os.Rename(tmpPath, outPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %q → %q: %w", tmpPath, outPath, err)
	}
	return nil
}
