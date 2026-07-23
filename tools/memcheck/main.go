// Command memcheck reports the retained heap of the game's loadable data
// structures — the definitions DB and each dictionary GADDAG — so their runtime
// memory cost can be tracked as the word lists and definitions grow.
//
// It loads each structure, forces a garbage collection, and reports the live-heap
// delta. Because dictionary.Load caches a single GADDAG at a time, dictionaries
// are measured one after another (each load drops the previous), which mirrors the
// game holding at most one dictionary resident.
//
// Usage:
//
//	go run ./tools/memcheck                 # defs + every dictionary
//	go run ./tools/memcheck -dict enable    # defs + one dictionary
//	go run ./tools/memcheck -defs=false      # dictionaries only
//
// memcheck is a developer diagnostic and is not part of the shipped app.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"tilewords/defs"
	"tilewords/dictionary"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "memcheck: %v\n", err)
		os.Exit(1)
	}
}

// run performs the measurement and returns the first error encountered.
func run() error {
	dictFlag := flag.String("dict", "all", `Dictionary to measure: "all" or a name from dictionary.AllDictNames`)
	defsFlag := flag.Bool("defs", true, "Measure the definitions DB")
	flag.Parse()

	names, err := selectDicts(*dictFlag)
	if err != nil {
		return err
	}

	base := liveHeap()
	report("baseline", "", base, base)

	// afterDefs is the live heap once the definitions DB (if requested) is resident;
	// it is the reference point against which each dictionary's cost is measured.
	afterDefs := base
	if *defsFlag {
		if !defs.Available() {
			return fmt.Errorf("definitions asset not embedded (run 'make defs')")
		}
		db, err := defs.Load()
		if err != nil {
			return err
		}
		afterDefs = liveHeap()
		report("defs DB", fmt.Sprintf("%d headwords", db.Len()), afterDefs, afterDefs-base)
		runtime.KeepAlive(db)
	}

	var lastDict dictionary.DictName
	for _, name := range names {
		d, err := dictionary.Load(name)
		if err != nil {
			return err
		}
		h := liveHeap()
		report("GADDAG "+string(name), fmt.Sprintf("%d words", d.WordCount()), h, h-afterDefs)
		lastDict = name
		runtime.KeepAlive(d)
	}

	printTotals(*defsFlag, lastDict)
	return nil
}

// selectDicts resolves the -dict flag to the dictionaries to measure, validating
// any explicit name against the registered set.
func selectDicts(sel string) ([]dictionary.DictName, error) {
	if sel == "all" {
		return dictionary.AllDictNames, nil
	}
	for _, n := range dictionary.AllDictNames {
		if string(n) == sel {
			return []dictionary.DictName{n}, nil
		}
	}
	var valid []string
	for _, n := range dictionary.AllDictNames {
		valid = append(valid, string(n))
	}
	return nil, fmt.Errorf("unknown dictionary %q; valid: all, %s", sel, strings.Join(valid, ", "))
}

// liveHeap returns the live heap size in bytes after two garbage collections. The
// second GC frees objects whose finalizers the first queued, giving a stable
// retained figure.
func liveHeap() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// report prints one measurement row: a label, an optional detail, the cumulative
// live heap, and the delta attributable to the just-loaded structure.
func report(label, detail string, cumulative, delta uint64) {
	if detail != "" {
		detail = "(" + detail + ")"
	}
	fmt.Printf("%-22s %-18s cumulative %7.1f MB   +%6.1f MB\n", label, detail, mb(cumulative), mb(delta))
}

// printTotals prints the process-wide heap figures with the last-loaded structures
// resident, including the high-water reservation that reflects transient decode cost.
func printTotals(withDefs bool, lastDict dictionary.DictName) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	resident := "defs + " + string(lastDict)
	if !withDefs {
		resident = string(lastDict)
	}
	fmt.Printf("\nprocess heap with %s resident:\n", resident)
	fmt.Printf("  live objects (HeapAlloc):    %7.1f MB\n", mb(m.HeapAlloc))
	fmt.Printf("  heap in use  (HeapInuse):    %7.1f MB\n", mb(m.HeapInuse))
	fmt.Printf("  heap reserved (HeapSys):     %7.1f MB   (high-water, includes decode churn)\n", mb(m.HeapSys))
	fmt.Printf("  total from OS (Sys):         %7.1f MB\n", mb(m.Sys))
}

// mb converts bytes to mebibytes.
func mb(b uint64) float64 { return float64(b) / (1024 * 1024) }
