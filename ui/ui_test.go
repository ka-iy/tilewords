// Package ui is documented in doc.go.
package ui

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"tilewords/engine"
)

// approxEq reports whether two float32 geometry values are equal within a small
// tolerance, tolerating the rounding of fractional gutter/offset arithmetic.
func approxEq(a, b float32) bool {
	d := a - b
	return d < 1e-3 && d > -1e-3
}

// ---------- boardGeometry ----------

func TestBoardGeometry_SquareReservesGutter(t *testing.T) {
	// On a square area the cell is the largest that fits boardDim cells plus the label
	// gutter, and the gutter-plus-grid block is centred with the grid past the gutter.
	const side = 480
	cell, offX, offY := boardGeometry(side, side)

	// The cell is maximal: one pixel larger would overflow the area.
	if (cell+1)*(boardDim+labelGutterFactor) <= side {
		t.Fatalf("cell %v is not maximal for side %d", cell, side)
	}
	gutter := cell * labelGutterFactor
	block := gutter + cell*boardDim
	if block > side+1e-3 {
		t.Fatalf("block %v does not fit in side %d", block, side)
	}
	// The grid sits one gutter past the left/top of a centred block.
	wantOff := (side-block)/2 + gutter
	if !approxEq(offX, wantOff) || !approxEq(offY, wantOff) {
		t.Fatalf("offsets: got (%v,%v) want (%v,%v)", offX, offY, wantOff, wantOff)
	}
}

func TestBoardGeometry_WideAreaCentres(t *testing.T) {
	// 600 wide × 480 tall → the square is bounded by the height (480); the block is
	// centred within the full width while fitting exactly in the height.
	const w, h = 600, 480
	cell, offX, offY := boardGeometry(w, h)
	gutter := cell * labelGutterFactor
	block := gutter + cell*boardDim
	if !approxEq(offX, (w-block)/2+gutter) {
		t.Fatalf("offX: got %v want %v (centred in width)", offX, (w-block)/2+gutter)
	}
	if !approxEq(offY, (h-block)/2+gutter) {
		t.Fatalf("offY: got %v want %v (fits height)", offY, (h-block)/2+gutter)
	}
	// The wider axis leaves a strictly larger margin than the tighter one.
	if offX <= offY {
		t.Fatalf("offX %v should exceed offY %v when the area is wider than tall", offX, offY)
	}
}

func TestBoardGeometry_ZeroAreaSafe(t *testing.T) {
	cell, _, _ := boardGeometry(0, 0)
	if cell != 0 {
		t.Fatalf("cell for zero area: got %v want 0", cell)
	}
}

func TestBoardGeometry_FitsAndNonNegativeOffsets(t *testing.T) {
	cell, offX, offY := boardGeometry(333, 1000)
	if cell*boardDim > 333+1e-3 {
		t.Fatalf("board (%v) wider than area 333", cell*boardDim)
	}
	if offX < 0 || offY < 0 {
		t.Fatalf("offsets must be non-negative: got (%v,%v)", offX, offY)
	}
}

// ---------- rackGeometry ----------

func TestRackGeometry_SevenSlotsTightFit(t *testing.T) {
	// Width exactly fits 7 minimum slots plus gaps; height equals slot.
	w := float32(minRackSlotPx*7 + rackGapPx*6)
	slot, offX := rackGeometry(w, minRackSlotPx, 7)
	if slot != minRackSlotPx {
		t.Fatalf("slot: got %v want %v", slot, float32(minRackSlotPx))
	}
	if offX != 0 {
		t.Fatalf("offX: got %v want 0", offX)
	}
}

func TestRackGeometry_HeightConstrains(t *testing.T) {
	// Plenty of width but a short height — slot is limited by height.
	slot, _ := rackGeometry(2000, 30, 7)
	if slot != 30 {
		t.Fatalf("slot constrained by height: got %v want 30", slot)
	}
}

func TestRackGeometry_ZeroSlotsSafe(t *testing.T) {
	slot, offX := rackGeometry(500, 50, 0)
	if slot != 0 || offX != 0 {
		t.Fatalf("zero slots: got (%v,%v) want (0,0)", slot, offX)
	}
}

// ---------- sanitiseError ----------

func TestSanitiseError_Nil(t *testing.T) {
	if got := sanitiseError(nil); got != "" {
		t.Fatalf("nil: got %q want empty", got)
	}
}

func TestSanitiseError_StripPrefix(t *testing.T) {
	err := errors.New("ui.SaveManager.Load: open: no such file")
	got := sanitiseError(err)
	if got != "no such file" {
		t.Fatalf("strip prefix: got %q want %q", got, "no such file")
	}
}

func TestSanitiseError_Truncated(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	err := errors.New(long)
	got := sanitiseError(err)
	if len(got) > 124 {
		t.Fatalf("truncation: result too long (%d chars)", len(got))
	}
}

func TestSanitiseError_NoPrefix(t *testing.T) {
	err := errors.New("disk full")
	got := sanitiseError(err)
	if got != "disk full" {
		t.Fatalf("no prefix: got %q want %q", got, "disk full")
	}
}

// ---------- premium helpers ----------

func TestPremiumLabel(t *testing.T) {
	cases := []struct {
		sq   engine.SquareType
		want string
	}{
		{engine.DoubleWord, "W×2"},
		{engine.TripleWord, "W×3"},
		{engine.DoubleLetter, "L×2"},
		{engine.TripleLetter, "L×3"},
		{engine.Centre, "★"},
		{engine.Normal, ""},
	}
	for _, c := range cases {
		if got := premiumLabel(c.sq); got != c.want {
			t.Fatalf("premiumLabel(%v): got %q want %q", c.sq, got, c.want)
		}
	}
}

func TestColorForSquare_Distinct(t *testing.T) {
	// The centre and a normal square must not share a colour.
	if colorForSquare(engine.Centre) == colorForSquare(engine.Normal) {
		t.Fatal("centre and normal squares should be visually distinct")
	}
}

// ---------- SaveManager ----------

func TestSaveManager_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSaveManager(dir)
	if err != nil {
		t.Fatalf("NewSaveManager: %v", err)
	}
	if sm.Exists() {
		t.Fatal("Exists() should be false before first save")
	}

	state := engine.New("csw", 5, rand.New(rand.NewSource(42)))
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !sm.Exists() {
		t.Fatal("Exists() should be true after save")
	}

	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AILevel != state.AILevel {
		t.Fatalf("AILevel: got %d want %d", loaded.AILevel, state.AILevel)
	}
	if loaded.HumanScore != state.HumanScore || loaded.AIScore != state.AIScore {
		t.Fatalf("Scores: got human=%d ai=%d want human=%d ai=%d",
			loaded.HumanScore, loaded.AIScore, state.HumanScore, state.AIScore)
	}
	if loaded.Board == nil {
		t.Fatal("Board is nil after load")
	}
	if loaded.HumanRack.Count() != state.HumanRack.Count() {
		t.Fatalf("HumanRack.Count: got %d want %d", loaded.HumanRack.Count(), state.HumanRack.Count())
	}
	if loaded.Bag.Count() != state.Bag.Count() {
		t.Fatalf("Bag.Count: got %d want %d", loaded.Bag.Count(), state.Bag.Count())
	}
}

// TestSaveManager_RoundTrip_WithCommands verifies Save succeeds even when
// LastHumanCommand and LastAICommand are non-nil (mid-game state).
func TestSaveManager_RoundTrip_WithCommands(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewSaveManager(dir)

	state := engine.New("csw", 5, rand.New(rand.NewSource(42)))
	state.LastHumanCommand = &engine.PassCommand{}
	state.LastAICommand = &engine.PassCommand{}

	if err := sm.Save(state); err != nil {
		t.Fatalf("Save with non-nil commands: %v", err)
	}
	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.LastHumanCommand != nil || loaded.LastAICommand != nil {
		t.Fatal("loaded state should have nil command fields")
	}
	if loaded.AILevel != state.AILevel {
		t.Fatalf("AILevel: got %d want %d", loaded.AILevel, state.AILevel)
	}
}

func TestSaveManager_LoadMissing(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewSaveManager(dir)
	_, err := sm.Load()
	if err == nil {
		t.Fatal("Load of missing file: want error")
	}
}

func TestSaveManager_LoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewSaveManager(dir)
	savePath := filepath.Join(dir, "tilewords", "savegame.gob")
	if err := os.MkdirAll(filepath.Dir(savePath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(savePath, []byte("not a gob"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := sm.Load()
	if err == nil {
		t.Fatal("Load of corrupt file: want error")
	}
}

func TestSaveManager_MkdirOnFirstSave(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewSaveManager(filepath.Join(dir, "nested"))
	state := engine.New("csw", 1, rand.New(rand.NewSource(42)))
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save to new directory: %v", err)
	}
	if !sm.Exists() {
		t.Fatal("save file should exist")
	}
}

// TestSaveManager_Delete: Delete removes an existing save so Exists() is false
// afterwards, and deleting again (no file present) is a no-op that succeeds.
func TestSaveManager_Delete(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewSaveManager(dir)

	state := engine.New("csw", 5, rand.New(rand.NewSource(42)))
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !sm.Exists() {
		t.Fatal("Exists() should be true after save")
	}

	if err := sm.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if sm.Exists() {
		t.Fatal("Exists() should be false after delete")
	}

	// Deleting an already-absent save is idempotent, not an error.
	if err := sm.Delete(); err != nil {
		t.Fatalf("Delete of missing save should be a no-op, got: %v", err)
	}
}
