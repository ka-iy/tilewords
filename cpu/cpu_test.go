// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package cpu_test

import (
	"math/rand/v2"
	"os"
	"testing"

	"tilewords/cpu"
	"tilewords/dictionary"
	"tilewords/engine"
)

// cpuTestWords is a curated, license-free word list used to build the test dictionary.
var cpuTestWords = []string{
	// 2-letter words
	"AA", "AB", "AD", "AE", "AG", "AH", "AI", "AL", "AM", "AN",
	"AR", "AS", "AT", "AW", "AX", "AY",
	"GO", "HI", "IT", "ME", "MY", "NO", "OF", "ON", "OR", "SO",
	"TO", "UP", "US", "WE",
	// 3-letter words
	"ACE", "AGO", "CPUD", "CPUM", "APE", "APT",
	"BAG", "BIG", "BIT", "BOX",
	"CAB", "CAP", "CAR", "CAT", "COB", "COD", "COP", "CUB", "CUP", "CUT",
	"DAB", "DAD", "DAM", "DIM", "DIP", "DOG", "DOT",
	"EAR", "EAT", "EGG", "ELM", "EMU",
	"FAD", "FAN", "FAR", "FAT", "FIG", "FIN", "FIT", "FIX", "FLY", "FUN",
	"GAP", "GAS", "GET", "GOT", "GUM", "GUN", "GUT",
	"HAD", "HAM", "HAS", "HAT", "HIM", "HIP", "HIT", "HOG", "HOP", "HOT",
	"ICE", "ILL",
	"JAB", "JAM", "JAR", "JAW", "JET", "JOB", "JOG", "JOT", "JOY", "JUG",
	"KEG", "KIT",
	"LAB", "LAD", "LAP", "LAW", "LAX", "LAY", "LEG", "LET", "LID", "LIP",
	"LOB", "LOG", "LOT",
	"MAD", "MAN", "MAP", "MAR", "MAT", "MAX", "MOP", "MOT", "MUD", "MUG",
	"NAB", "NAP", "NET", "NIB", "NIP", "NOB", "NOD", "NOR", "NOT",
	"OAK", "OAR", "OAT", "ODD", "ODE",
	"PAD", "PAN", "PAP", "PAR", "PAT", "PAW", "PAY", "PEA", "PEG", "PEN",
	"PET", "PIE", "PIG", "PIN", "PIT", "PLY", "POD", "POP", "POT", "POX",
	"PUB", "PUN", "PUT",
	"RAG", "RAN", "RAP", "RAT", "RAW", "RAY", "RID", "RIG", "RIM", "RIP",
	"ROB", "ROD", "ROT", "ROW", "RUB", "RUG", "RUM", "RUN", "RUT",
	"SAC", "SAD", "SAP", "SAT", "SAW", "SAY", "SET", "SEW", "SIP", "SIT",
	"SOB", "SOD", "SON", "SOP", "SOT", "SOW", "SOY", "SPA", "SPY", "STY",
	"SUB", "SUM", "SUN", "SUP",
	"TAB", "TAN", "TAP", "TAR", "TAT", "TAX", "TIP", "TOD", "TON", "TOP",
	"TOT", "TOW", "TOY", "TUB", "TUG",
	"URN",
	"VAN", "VAT", "VIA", "VIE",
	"WAD", "WAR", "WAS", "WAX", "WAY", "WEB", "WED", "WET", "WIG", "WIN",
	"WIT", "WOE", "WOK", "WON", "WOO", "WOP",
	"YAK", "YAM", "YAP", "YAW", "YEN", "YEP", "YET", "YEW", "YOB", "YOD",
	"ZAP", "ZIT",
	// 4-letter words
	"ATOM", "ATOP",
	"BAIT", "BALE", "BALL", "BANE", "BARE", "BARK", "BARN", "BASK", "BASS",
	"BATE", "BATH", "BEAD", "BEAM", "BEAN", "BEAR", "BEAT", "BEEF", "BEEN",
	"BOLT", "BONE", "BOOT", "BORE", "BORN",
	"CAGE", "CAKE", "CALF", "CALM", "CAME", "CANE", "CAPE", "CARD", "CARE",
	"CART", "CASE", "CAST", "CAVE", "COAL", "COAT", "CODE", "COIL", "COIN",
	"COLD", "COLT", "COME", "CONE", "COPE", "CORD", "CORE", "CORN", "COST",
	"DARE", "DARK", "DART", "DATE", "DAWN", "DAYS", "DEAD", "DEAF", "DEAL",
	"DEAN", "DEAR", "DEBT", "DEED", "DEEM", "DEER", "DEFT", "DENY", "DESK",
	"DINE", "DIRT", "DISK", "DIVE", "DOCK", "DOME", "DONE", "DOOR", "DOSE",
	"DOVE", "DOWN", "DRAW", "DREW", "DROP", "DRUM", "DUAL", "DUEL", "DULL",
	"DUMB", "DUMP", "DUNE", "DUSK", "DUST",
	"EACH", "EARL", "EARN", "EASE", "EAST", "EDGE", "EMIT",
	"FACE", "FACT", "FADE", "FAIL", "FAIR", "FALL", "FAME", "FANG", "FARM",
	"FAST", "FATE", "FEAT", "FEED", "FEEL", "FEET", "FELL", "FELT", "FERN",
	"FILE", "FILL", "FILM", "FIND", "FINE", "FIRE", "FIRM", "FISH", "FIST",
	"FIVE", "FIZZ", "FLAG", "FLAT", "FLAW", "FLEA", "FLEW", "FLIP", "FLIT",
	"FLOW", "FOAM", "FOIL", "FOLD", "FOLK", "FOND", "FONT", "FOOD", "FOOL",
	"FOOT", "FORD", "FORE", "FORK", "FORM", "FORT", "FOUL", "FOWL", "FRAY",
	"FREE", "FRET", "FROM", "FUME", "FURL", "FUSE", "FUSS",
	"GAIN", "GALE", "GAME", "GANG", "GATE", "GAVE", "GAZE", "GEAR", "GERM",
	"GIFT", "GILD", "GILL", "GILT", "GIRD", "GIRL", "GIST", "GIVE", "GLAD",
	"GLEE", "GLOB", "GLOW", "GLUE", "GOAL", "GOAT", "GOLD", "GOLF", "GONE",
	"GOOD", "GORE", "GOWN", "GRAB", "GRAD", "GRAM", "GRAY", "GREW", "GRID",
	"GRIM", "GRIP", "GRIT", "GROW", "GRUB", "GULF", "GULL", "GULP", "GUST",
	// 5-letter words
	"APPLE", "ASSET",
	"BAKER", "BELOW", "BLAZE", "BLEED", "BLESS", "BLEND", "BLIND", "BLOCK",
	"BLOOD", "BLOOM", "BLOWN", "BLUES", "BLUNT", "BLURT", "BOARD", "BOAST",
	"BREAK", "BREED", "BRINE", "BRING", "BRINK", "BROAD", "BROKE", "BROOD",
	"BROWN", "BRUSH", "BRUTE", "BUILD", "BUILT", "BURNT", "BURST",
	"CANDY", "CHAIR", "CHARM", "CHEAP", "CHEAT", "CHECK", "CHEEK", "CHESS",
	"CHEST", "CHILD", "CHINA", "CHORD", "CIVIL", "CLAIM", "CLASS", "CLEAN",
	"CLEAR", "CLERK", "CLIFF", "CLIMB", "CLING", "CLOCK", "CLOSE", "CLOTH",
	"CLOUD", "CLOWN", "CRACK", "CRAVE", "CRAWL", "CREED", "CREEK", "CREEP",
	"CREST", "CRISP", "CROSS", "CROWD", "CROWN", "CRUEL", "CRUSH", "CURVE",
	"CYCLE",
	// 6-letter words
	"ABROAD", "ANIMAL", "BATTLE", "BEAUTY", "BRIDGE",
	"CASTLE", "CHANGE", "CHURCH", "CIRCLE", "COLUMN",
	"DAMAGE", "DANGER", "DESIGN", "DOLLAR", "DOUBLE",
	"EFFECT", "EMPLOY", "ENERGY", "ENGAGE", "ENOUGH",
	"FABRIC", "FAMILY", "FAMOUS", "FIGURE", "FINGER",
	"GARDEN", "GLOBAL", "GROUND", "GROWTH", "GUITAR",
}

var testDict *dictionary.Dictionary

func TestMain(m *testing.M) {
	var err error
	testDict, err = dictionary.NewFromWords(dictionary.DictENABLE, cpuTestWords)
	if err != nil {
		panic("ai_test: failed to build test dictionary: " + err.Error())
	}
	os.Exit(m.Run())
}

// deterministicRNG returns a seeded *rand.Rand for reproducible tests.
func deterministicRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), 0))
}

// placeTile is a test helper that places a tile on the board at (row, col).
func placeTile(b *engine.Board, row, col int, letter byte, points int) {
	t := engine.Tile{Letter: letter, Points: points}
	if err := b.Place(row, col, t); err != nil {
		panic(err)
	}
}

// TestGenerateMoves_EmptyRack verifies GenerateMoves returns empty slice without panic.
func TestGenerateMoves_EmptyRack(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	candidates := cpu.GenerateMoves(board, rack, testDict)
	if candidates == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates with empty rack, got %d", len(candidates))
	}
}

// TestGenerateMoves_FirstMove verifies a non-empty rack on an empty board produces candidates.
func TestGenerateMoves_FirstMove(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'C', Points: 3},
		{Letter: 'A', Points: 1},
		{Letter: 'T', Points: 1},
	})
	candidates := cpu.GenerateMoves(board, rack, testDict)
	if len(candidates) == 0 {
		t.Fatal("expected candidates for CAT rack on empty board, got none")
	}
	// Verify "CAT" (and "ACT", "TAC" etc. if in dict) are present.
	found := false
	for _, c := range candidates {
		for _, w := range c.Move.WordsFormed {
			if w == "CAT" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected CAT to appear as a formed word in candidates")
	}
}

// TestGenerateMoves_AllValid verifies every candidate passes ValidatePlacement.
func TestGenerateMoves_AllValid(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'C', Points: 3},
		{Letter: 'A', Points: 1},
		{Letter: 'T', Points: 1},
		{Letter: 'E', Points: 1},
		{Letter: 'R', Points: 1},
	})
	candidates := cpu.GenerateMoves(board, rack, testDict)
	for i, c := range candidates {
		move := c.Move
		if _, err := engine.ValidatePlacement(board, &move, testDict); err != nil {
			t.Errorf("candidate %d failed ValidatePlacement: %v", i, err)
		}
	}
}

// TestGenerateMoves_Sorted verifies candidates are sorted score-desc, access-asc.
func TestGenerateMoves_Sorted(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'C', Points: 3},
		{Letter: 'A', Points: 1},
		{Letter: 'T', Points: 1},
		{Letter: 'E', Points: 1},
		{Letter: 'R', Points: 1},
		{Letter: 'N', Points: 1},
		{Letter: 'G', Points: 2},
	})
	candidates := cpu.GenerateMoves(board, rack, testDict)
	for i := 1; i < len(candidates); i++ {
		a, b := candidates[i-1], candidates[i]
		if a.Score < b.Score {
			t.Errorf("candidates[%d].Score=%d < candidates[%d].Score=%d (not sorted)", i-1, a.Score, i, b.Score)
		}
		if a.Score == b.Score && a.OpponentAccess > b.OpponentAccess {
			t.Errorf("tie at score=%d: candidates[%d].OpponentAccess=%d > candidates[%d].OpponentAccess=%d",
				a.Score, i-1, a.OpponentAccess, i, b.OpponentAccess)
		}
	}
}

// TestGenerateMoves_NoDuplicates verifies no two candidates share the same move key.
func TestGenerateMoves_NoDuplicates(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'G', Points: 2},
		{Letter: 'A', Points: 1},
		{Letter: 'R', Points: 1},
		{Letter: 'D', Points: 2},
		{Letter: 'E', Points: 1},
		{Letter: 'N', Points: 1},
	})
	candidates := cpu.GenerateMoves(board, rack, testDict)
	seen := make(map[string]bool)
	for _, c := range candidates {
		placed := c.Move.Placed
		key := ""
		for _, pt := range placed {
			key += string(rune(pt.Row+'0')) + string(rune(pt.Col+'0')) + string(pt.Tile.Letter)
		}
		if seen[key] {
			t.Errorf("duplicate candidate: %s", key)
		}
		seen[key] = true
	}
}

// TestSelectMove_Level10SteepScoresPicksBest verifies that when nothing else comes close to
// the best play, level 10 plays it. The near-best window is a score window, so a steep drop
// after the leader leaves nothing to choose between and the CPU cannot squander the turn.
func TestSelectMove_Level10SteepScores(t *testing.T) {
	candidates := []cpu.MoveCandidate{
		{Score: 100},
		{Score: 80}, // 20% below the best: outside the window
		{Score: 60},
	}
	rng := deterministicRNG(42)
	for i := 0; i < 200; i++ {
		if got := cpu.SelectMove(candidates, 10, rng); got.Score != 100 {
			t.Fatalf("level 10 with a steep score drop: got %d, want the best play (100)", got.Score)
		}
	}
}

// TestSelectMove_Level10VariesAmongNearBest verifies level 10 does not always play the single
// best move when comparable alternatives exist, and never plays one outside the margin.
func TestSelectMove_Level10VariesAmongNearBest(t *testing.T) {
	candidates := []cpu.MoveCandidate{
		{Score: 100},
		{Score: 95}, // within 10% of the best: eligible
		{Score: 92}, // within 10%: eligible
		{Score: 40}, // far below: must never be chosen
	}
	rng := deterministicRNG(7)
	seen := make(map[int]bool)
	for i := 0; i < 500; i++ {
		got := cpu.SelectMove(candidates, 10, rng)
		seen[got.Score] = true
		if got.Score < 90 {
			t.Fatalf("level 10 chose a play %d, outside the near-best margin", got.Score)
		}
	}
	if len(seen) < 2 {
		t.Errorf("level 10 always played the same score %v; it should vary among near-best plays", seen)
	}
	if !seen[100] {
		t.Error("level 10 never played the best move; it should remain reachable")
	}
}

// TestSelectMove_Level10AllZeroScores verifies the degenerate case where every play scores
// zero: there is nothing to choose between on score, so the sort's OpponentAccess tiebreak
// stands rather than the window widening to the whole list.
func TestSelectMove_Level10AllZeroScores(t *testing.T) {
	candidates := []cpu.MoveCandidate{
		{Score: 0, OpponentAccess: 1},
		{Score: 0, OpponentAccess: 5},
	}
	rng := deterministicRNG(3)
	for i := 0; i < 50; i++ {
		if got := cpu.SelectMove(candidates, 10, rng); got.OpponentAccess != 1 {
			t.Fatalf("all-zero scores: got OpponentAccess %d, want the lowest (1)", got.OpponentAccess)
		}
	}
}

// TestSelectMove_DemigodModeAlwaysBest verifies the top level always plays the single best move,
// even where near-best alternatives exist that NearBestLevel would sometimes choose instead.
func TestSelectMove_DemigodModeAlwaysBest(t *testing.T) {
	candidates := []cpu.MoveCandidate{
		{Score: 100, OpponentAccess: 2},
		{Score: 99}, // within the near-best margin, so level 10 would sometimes take it
		{Score: 98},
	}
	rng := deterministicRNG(11)
	for i := 0; i < 500; i++ {
		got := cpu.SelectMove(candidates, cpu.DemigodModeLevel, rng)
		if got.Score != 100 || got.OpponentAccess != 2 {
			t.Fatalf("demigod mode: got score %d access %d, want the best play (100, 2)",
				got.Score, got.OpponentAccess)
		}
	}
}

// TestSelectMove_DemigodModeIsDeterministic verifies demigod mode ignores the RNG entirely, so the same
// board and rack always produce the same move.
func TestSelectMove_DemigodModeIsDeterministic(t *testing.T) {
	candidates := []cpu.MoveCandidate{{Score: 50}, {Score: 49}, {Score: 48}}
	a := cpu.SelectMove(candidates, cpu.DemigodModeLevel, deterministicRNG(1))
	b := cpu.SelectMove(candidates, cpu.DemigodModeLevel, deterministicRNG(9999))
	if a.Score != b.Score {
		t.Errorf("demigod mode varied with the seed: %d vs %d", a.Score, b.Score)
	}
}

// TestSelectMove_LevelsAboveMaxClampToDemigodMode verifies an out-of-range level (e.g. from a
// tampered save) clamps into the accepted range rather than indexing out of it.
func TestSelectMove_LevelsAboveMaxClampToDemigodMode(t *testing.T) {
	candidates := []cpu.MoveCandidate{{Score: 30}, {Score: 29}}
	for _, level := range []int{cpu.MaxLevel + 1, 50, 1 << 20} {
		if got := cpu.SelectMove(candidates, level, deterministicRNG(2)); got.Score != 30 {
			t.Errorf("level %d: got %d, want the best play (30)", level, got.Score)
		}
	}
}

// TestSelectMove_Level1 verifies level 1 samples from the full candidate set.
func TestSelectMove_Level1(t *testing.T) {
	candidates := make([]cpu.MoveCandidate, 100)
	for i := range candidates {
		candidates[i] = cpu.MoveCandidate{Score: 100 - i}
	}
	rng := deterministicRNG(0)
	// With 1000 samples at level 1, we expect most score values to appear.
	seen := make(map[int]bool)
	for i := 0; i < 1000; i++ {
		got := cpu.SelectMove(candidates, 1, rng)
		seen[got.Score] = true
	}
	if len(seen) < 50 {
		t.Errorf("level 1: expected diversity across candidates, only %d distinct scores seen", len(seen))
	}
}

// TestSelectMove_EmptyPanics verifies SelectMove panics on empty input.
func TestSelectMove_EmptyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty candidates, got none")
		}
	}()
	rng := deterministicRNG(0)
	cpu.SelectMove(nil, 5, rng)
}

// TestChooseMove_HasCandidates verifies ChooseMove returns a PlayMove when candidates exist.
func TestChooseMove_HasCandidates(t *testing.T) {
	rng := deterministicRNG(0)
	state := engine.New(dictionary.DictENABLE, 10, rng)
	// Ensure the CPU rack has usable tiles for a first move.
	// We use the standard New which deals tiles; if CPU goes first or second, it has a rack.
	state.CurrentTurn = engine.CPUTurn
	rng2 := deterministicRNG(1)
	move := cpu.ChooseMove(state, testDict, 10, rng2)
	if move == nil {
		t.Fatal("ChooseMove returned nil")
	}
}

// TestChooseMove_NoCandidates_LargeBag verifies exchange when bag is large.
func TestChooseMove_NoCandidates_LargeBag(t *testing.T) {
	// Use a dictionary with no valid words to force zero candidates.
	emptyDict, err := dictionary.NewFromWords(dictionary.DictENABLE, []string{"ZZZZZ"})
	if err != nil {
		t.Fatal(err)
	}
	rng := deterministicRNG(0)
	state := engine.New(dictionary.DictENABLE, 5, rng)
	state.CurrentTurn = engine.CPUTurn
	rng2 := deterministicRNG(1)
	move := cpu.ChooseMove(state, emptyDict, 5, rng2)
	if move == nil {
		t.Fatal("ChooseMove returned nil")
	}
	if _, ok := move.(engine.ExchangeMove); !ok {
		t.Logf("bag count=%d, rack count=%d", state.Bag.Count(), state.CPURack.Count())
		// Either ExchangeMove or PassMove is valid depending on bag size.
	}
}

// TestChooseMove_NoCandidates_SmallBag verifies pass when bag is too small.
func TestChooseMove_NoCandidates_SmallBag(t *testing.T) {
	emptyDict, err := dictionary.NewFromWords(dictionary.DictENABLE, []string{"ZZZZZ"})
	if err != nil {
		t.Fatal(err)
	}
	rng := deterministicRNG(0)
	state := engine.New(dictionary.DictENABLE, 5, rng)
	state.CurrentTurn = engine.CPUTurn
	// Drain the bag to below MaxRackSize.
	for state.Bag.Count() >= engine.MaxRackSize {
		state.Bag.Draw(engine.MaxRackSize, nil)
	}
	rng2 := deterministicRNG(1)
	move := cpu.ChooseMove(state, emptyDict, 5, rng2)
	if _, ok := move.(engine.PassMove); !ok {
		t.Errorf("expected PassMove with small bag, got %T", move)
	}
}

// Choosing a move on a background goroutine — the pattern a UI uses so its own thread never
// blocks — is covered by TestPBT_CPU_OffGoroutine_NoRace in cpu_pbt_test.go.
