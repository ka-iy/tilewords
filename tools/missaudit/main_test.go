// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestReportStatsShowsCountsAndPercentages verifies each list's coverage is reported as both a
// count and a share of that list, so the gap can be read either way.
func TestReportStatsShowsCountsAndPercentages(t *testing.T) {
	var buf bytes.Buffer
	stats := []listStat{
		{Path: "wordlists/listA.txt", Total: 4, Missing: 2},
		{Path: "wordlists/listB.txt", Total: 3, Missing: 1},
	}
	reportStats(&buf, stats, 6, 3)
	got := buf.String()

	for _, want := range []string{
		"listA",
		"2 / 4",
		"( 50.00%)", // covered
		"(50.00%)",  // missing
		"listB",
		"2 / 3",
		"( 66.67%)",
		"(33.33%)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report is missing %q; got:\n%s", want, got)
		}
	}
}

// TestReportStatsAggregatesOverTheUnion verifies the aggregate percentages are measured against
// the distinct words across the lists, not their concatenation: a word carried by two lists is
// one word a player can form, and the miss list it is compared against is deduplicated too.
func TestReportStatsAggregatesOverTheUnion(t *testing.T) {
	var buf bytes.Buffer
	// Two lists of 4 and 3 words sharing one word: 6 distinct, of which 3 have no definition.
	stats := []listStat{
		{Path: "listA.txt", Total: 4, Missing: 2},
		{Path: "listB.txt", Total: 3, Missing: 1},
	}
	reportStats(&buf, stats, 6, 3)
	got := buf.String()

	if !strings.Contains(got, "aggregate across 2 list(s):") {
		t.Errorf("report does not name the number of lists aggregated; got:\n%s", got)
	}
	// 3 of 6 covered, not 4 of 7 (which the summed list totals would give).
	if !strings.Contains(got, "3 / 6") {
		t.Errorf("aggregate is not measured over the union of the lists; got:\n%s", got)
	}
	if strings.Contains(got, "/ 7") {
		t.Errorf("aggregate counted a shared word twice; got:\n%s", got)
	}
}

// TestPercentEmptyTotal verifies an empty list reports 0% rather than dividing by zero.
func TestPercentEmptyTotal(t *testing.T) {
	if got := percent(0, 0); got != 0 {
		t.Errorf("percent(0, 0) = %v, want 0", got)
	}
}

// TestReportStatsNoLists verifies a run with no lists still prints a well-formed aggregate
// rather than panicking or reporting a nonsense percentage.
func TestReportStatsNoLists(t *testing.T) {
	var buf bytes.Buffer
	reportStats(&buf, nil, 0, 0)

	if !strings.Contains(buf.String(), "0 / 0") {
		t.Errorf("empty run did not report an empty aggregate; got:\n%s", buf.String())
	}
}
