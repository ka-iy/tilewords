// Package ui is documented in doc.go.
package ui

import (
	"testing"

	"squabble/dictionary"
)

// TestDictDisplayNameCoversAllDicts guards against a dictionary being registered in
// dictionary.AllDictNames without a matching case in dictDisplayName. Such a list
// would silently fall through to the default branch and be shown by its raw internal
// name (e.g. "wordnik" instead of "Wordnik (crowd-sourced)") in the setup menu.
func TestDictDisplayNameCoversAllDicts(t *testing.T) {
	for _, name := range dictionary.AllDictNames {
		label := dictDisplayName(name)
		if label == string(name) {
			t.Errorf("dictDisplayName(%q) returned the raw name %q — add an explicit case for this dictionary", name, label)
		}
		if label == "" {
			t.Errorf("dictDisplayName(%q) returned an empty label", name)
		}
	}
}
