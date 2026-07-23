package ui

import (
	"strings"
	"testing"
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
