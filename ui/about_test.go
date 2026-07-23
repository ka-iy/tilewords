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
		"LEXICON",                             // section header from LEXICON.txt
		"github.com/ka-iy/tilewords",          // an ABOUT.txt URL
		"https://github.com/wordnik/wordlist", // a LEXICON.txt source URL
	} {
		if !strings.Contains(aboutText, want) {
			t.Errorf("aboutText is missing %q; regenerate ui/about.txt (make ui/about.txt)", want)
		}
	}

	// ABOUT must precede LEXICON, the order the sections are assembled in.
	if strings.Index(aboutText, "\n  ABOUT") >= strings.Index(aboutText, "\n  LEXICON") {
		t.Error("ABOUT section must come before LEXICON section")
	}
}
