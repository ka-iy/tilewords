// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"
)

// TestAboutTextOpensWithLicenceNotice verifies the About dialog's text begins with the
// copyright line, the licence statement, and a link to the licence text, in that order and
// before anything else — no section banner above them, and nothing else before them. GPLv3
// section 4 requires the notice to be kept intact on redistribution, and this is where a
// player sees it.
func TestAboutTextOpensWithLicenceNotice(t *testing.T) {
	lines := strings.Split(strings.TrimLeft(aboutDialogText(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("about text has only %d lines", len(lines))
	}

	if !strings.Contains(lines[0], "Copyright") || !strings.Contains(lines[0], "Kartikeya IYER") {
		t.Errorf("first line = %q, want the copyright notice", lines[0])
	}
	if !strings.Contains(lines[1], "GPLv3") {
		t.Errorf("second line = %q, want the licence statement", lines[1])
	}
	if !strings.Contains(lines[2], "gnu.org/licenses/gpl-3.0") {
		t.Errorf("third line = %q, want a link to the licence text", lines[2])
	}
}

// TestAboutTextCreditsAssetSources verifies the About text still carries the attribution the
// asset licences require: CC BY-SA 4.0 for the definitions, the WordNet notice, and Joseph
// Petree by name as his permission asks. These are obligations, not decoration, so a
// regenerated about.txt that drops them is a licence violation rather than a cosmetic change.
func TestAboutTextCreditsAssetSources(t *testing.T) {
	for _, want := range []string{
		"CC BY-SA 4.0",  // Wiktionary and the Petree definitions
		"WordNet",       // Princeton, notice must be retained
		"Joseph Petree", // named at his request
		"public domain", // Webster, Jamieson, Spenser, OPTED
		"MIT License",   // Wordnik word list
	} {
		if !strings.Contains(aboutText, want) {
			t.Errorf("about text no longer mentions %q; asset attribution is a licence obligation", want)
		}
	}
}
