package ui

import (
	"strings"
	"testing"

	"tilewords/buildinfo"
)

// TestAboutTextGenerated guards the build-time generation + embed wiring: the About
// dialog text must carry both file-named sections and their content, so a missing or
// stale ui/about.txt (which would otherwise fail silently at runtime) is caught.
func TestAboutTextGenerated(t *testing.T) {
	for _, want := range []string{
		"ABOUT",                               // section header from ABOUT.txt
		"FEATURES",                            // section header from FEATURES.txt
		"LEXICON",                             // section header from LEXICON.txt
		"github.com/ka-iy/tilewords",          // an ABOUT.txt URL
		"Selectable AI difficulty",            // a FEATURES.txt bullet
		"https://github.com/wordnik/wordlist", // a LEXICON.txt source URL
	} {
		if !strings.Contains(aboutText, want) {
			t.Errorf("aboutText is missing %q; regenerate ui/about.txt (make ui/about.txt)", want)
		}
	}

	// Sections must appear in assembly order: ABOUT, then FEATURES, then LEXICON.
	iAbout := strings.Index(aboutText, "\n  ABOUT")
	iFeatures := strings.Index(aboutText, "\n  FEATURES")
	iLexicon := strings.Index(aboutText, "\n  LEXICON")
	if !(iAbout < iFeatures && iFeatures < iLexicon) {
		t.Errorf("sections out of order: ABOUT@%d FEATURES@%d LEXICON@%d", iAbout, iFeatures, iLexicon)
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
		// The section must carry the metadata, and lead the text so it reads first.
		if !strings.Contains(text, buildinfo.BuildVersion()) {
			t.Errorf("BUILD INFO section is missing the build version %q", buildinfo.BuildVersion())
		}
		if i := strings.Index(text, "BUILD INFO"); i > strings.Index(text, "\n  ABOUT") {
			t.Errorf("BUILD INFO@%d must precede the embedded ABOUT section@%d",
				i, strings.Index(text, "\n  ABOUT"))
		}
	}

	// The embedded text is always present, section skipped or not.
	if !strings.Contains(text, "\n  ABOUT") {
		t.Error("aboutDialogText is missing the embedded ABOUT section")
	}
}
