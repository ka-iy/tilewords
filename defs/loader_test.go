package defs

import "testing"

// TestLoadEmbedded exercises the embedded-asset path. It is skipped when the asset
// has not been built (CI without 'make defs'), and otherwise checks that the DB
// decodes and resolves a common word.
func TestLoadEmbedded(t *testing.T) {
	if !Available() {
		t.Skip("definitions asset not embedded; run 'make defs'")
	}
	db, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if db.Len() == 0 {
		t.Fatal("Load returned an empty DB")
	}
	// Load caches: a second call must return the same instance.
	if db2, _ := Load(); db2 != db {
		t.Error("Load did not return the cached instance")
	}
	if res, ok := db.Lookup("cat"); !ok || res.Entry == nil || len(res.Entry.Senses) == 0 {
		t.Errorf("Lookup(cat) = %+v,%v; want a definition", res, ok)
	}
}
