// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"

	"tilewords/buildinfo"
)

// TestAboutTextGenerated guards the build-time generation + embed wiring: the About
// dialog text must carry each source file's content under its own section header, so a
// missing or stale ui/about.txt (which would otherwise fail silently at runtime) is caught.
func TestAboutTextGenerated(t *testing.T) {
	// Each source contributes a section header and a line of its own body, so a generation
	// that dropped or truncated one of the three is caught. The body strings are matched
	// against ABOUT.txt, FEATURES.txt and LEXICON.txt respectively — the repository URL is
	// deliberately not among them: it lives in COPYRIGHT.txt, which is a separate asset
	// (ui/copyright.txt) reaching the dialog by its own path, so asserting it here would pass
	// on text this file has nothing to do with.
	for _, want := range []string{
		"ABOUT",                               // section header from ABOUT.txt
		"FEATURES",                            // section header from FEATURES.txt
		"LEXICON",                             // section header from LEXICON.txt
		"crossword tile game",                 // ABOUT.txt's description of the game
		"Selectable CPU difficulty",           // a FEATURES.txt bullet
		"https://github.com/wordnik/wordlist", // a LEXICON.txt source URL
	} {
		if !strings.Contains(aboutText, want) {
			t.Errorf("aboutText is missing %q; regenerate ui/about.txt (make ui/about.txt)", want)
		}
	}

	// Sources must appear in assembly order: ABOUT.txt, then FEATURES, then LEXICON.
	iAbout := strings.Index(aboutText, "\n  ABOUT")
	iFeatures := strings.Index(aboutText, "\n  FEATURES")
	iLexicon := strings.Index(aboutText, "\n  LEXICON")
	if !(iAbout < iFeatures && iFeatures < iLexicon) {
		t.Errorf("sources out of order: ABOUT@%d FEATURES@%d LEXICON@%d", iAbout, iFeatures, iLexicon)
	}
}

// TestAboutDialogTextBuildInfo guards the rule that the About dialog carries a BUILD INFO
// section exactly when the binary has a build version injected: a build without one must
// not show the section at all, rather than showing placeholder metadata. The assertion is
// written as that equivalence so it holds however the test binary was linked — run it with
// -ldflags "-X 'tilewords/buildinfo.buildVersion=<v>'" to exercise the injected case.
func TestAboutDialogTextBuildInfo(t *testing.T) {
	text := aboutDialogText()

	if got, want := strings.Contains(text, "BUILD INFO"), buildinfo.BuildVersion() != ""; got != want {
		t.Errorf("BUILD INFO section present = %t, want %t (build version %q)",
			got, want, buildinfo.BuildVersion())
	}

	if buildinfo.BuildVersion() != "" {
		// The section must carry the metadata.
		if !strings.Contains(text, buildinfo.BuildVersion()) {
			t.Errorf("BUILD INFO section is missing the build version %q", buildinfo.BuildVersion())
		}
		// It sits between the licence notice, which leads the dialog, and the sectioned
		// text that the notice must not be buried inside.
		iNotice := strings.Index(text, "Copyright ©")
		iBuild := strings.Index(text, "BUILD INFO")
		iAbout := strings.Index(text, "\n  ABOUT")
		if !(iNotice < iBuild && iBuild < iAbout) {
			t.Errorf("sections out of order: notice@%d BUILD INFO@%d ABOUT@%d", iNotice, iBuild, iAbout)
		}
	}

	// The embedded text is always present, section skipped or not. The string is one of
	// ABOUT.txt's own, not the repository URL: that arrives with the copyright notice, so it
	// would be found here even if the sectioned text were missing entirely.
	if !strings.Contains(text, "crossword tile game") {
		t.Error("aboutDialogText is missing the embedded ABOUT.txt content")
	}
}
