package engine

import (
	"math/rand"
	"os"
	"testing"

	"squabble/dictionary"
)

// engineTestWords is a curated, license-free word list used to build the test dictionary.
var engineTestWords = []string{
	// 2-letter words
	"AA", "AB", "AD", "AE", "AG", "AH", "AI", "AL", "AM", "AN",
	"AR", "AS", "AT", "AW", "AX", "AY",
	"GO", "HI", "IT", "ME", "MY", "NO", "OF", "ON", "OR", "SO",
	"TO", "UP", "US", "WE",
	// 3-letter words
	"ACE", "AGO", "AID", "AIM", "APE", "APT",
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
	"PEP", "PET", "PIE", "PIG", "PIN", "PIP", "PIT", "PLY", "POD", "POP",
	"POT", "POX", "PRY", "PUB", "PUN", "PUP", "PUS", "PUT",
	"RAG", "RAN", "RAP", "RAT", "RAW", "RAY", "RID", "RIG", "RIM", "RIP",
	"ROB", "ROD", "ROT", "ROW", "RUB", "RUG", "RUM", "RUN", "RUT",
	"SAC", "SAD", "SAP", "SAT", "SAW", "SAY", "SET", "SEW", "SIP", "SIT",
	"SIX", "SKI", "SKY", "SLY", "SOB", "SOD", "SOP", "SOT", "SOW", "SOY",
	"SPA", "SPY", "STY", "SUB", "SUM", "SUN", "SUP",
	"TAB", "TAD", "TAN", "TAP", "TAR", "TAT", "TAX", "TEN", "THE", "TIE",
	"TIN", "TIP", "TOD", "TOE", "TON", "TOP", "TOT", "TOW", "TOY",
	"TUB", "TUG", "TUN",
	"URN",
	"VAT", "VIA", "VIM",
	"WAD", "WAR", "WAS", "WAX", "WAY", "WEB", "WED", "WET", "WHO", "WHY",
	"WIG", "WIN", "WIT", "WOE", "WOK", "WON", "WOO",
	"YAK", "YAM", "YAP", "YAW", "YEA", "YEP", "YET", "YEW",
	"ZAP", "ZAX", "ZIT",
	// 4-letter words needed for specific tests
	"BRAG", "CATS", "COAT", "CORD", "CORN", "COST", "COTS", "DOGS",
	"EACH", "EDGE", "EPIC",
	"FAKE", "FATE", "FLAT", "FLAW", "FLED", "FLOG", "FLOW", "FOAM",
	"GAIT", "GAVE", "GAZE", "GILD", "GLAM", "GLAD", "GLOW",
	"HATS", "HAZE", "HEAL", "HEAP", "HEAT", "HIVE", "HOLD", "HOLE",
	"JABS", "JACK", "JADE", "JAIL", "JAKE", "JAMB", "JAPE", "JAVA",
	"MATS", "MAZE", "MEAL", "MEAN",
	"NAIL", "NAME", "NAPE", "NARC", "NARD", "NARK",
	"OAST", "OATH",
	"PACK", "PAGE", "PAID", "PAIL", "PAIR", "PALE", "PALM",
	"RACK", "RACE", "RAFT", "RAGE", "RAID", "RAIL", "RAIN", "RAKE",
	"RATS", "RATE",
	"SACK", "SAFE", "SAGE", "SAID", "SAIL", "SAKE", "SALE", "SAME",
	"SAND", "SANE", "SANG", "SAP",
	"TABS", "TACK", "TACT", "TAIL", "TAKE", "TALE", "TALK", "TALL",
	"TAME", "TANG", "TAPE", "TAPS", "TARE", "TARN", "TARP", "TARS",
	"VANE", "VANG", "VARS", "VATS",
	"WAND", "WANE", "WANT", "WARD", "WARE", "WARN", "WARP", "WARS",
	"WART", "WARY", "WAVE", "WAVY",
	// 5+ letter words
	"BRACE", "BRAND", "BRAVE", "BRAWL",
	"CARGO", "CHAIN", "CHAIR", "CHARM",
	"DRAFT", "DRAIN", "DRAKE", "DRAMA",
	"GRACE", "GRADE", "GRAIN", "GRASP",
	"PLACE", "PLAID", "PLAIN", "PLAIT", "PLANE", "PLANK",
	"TRACE", "TRACK", "TRADE", "TRAIL", "TRAIN", "TRAIT",
	"SQUAB", "SQUAD", "SQUAT",
}

// newFlatBoard returns a board with all Normal squares and no placed tiles.
// Used in tests to produce predictable face-value-only scoring.
func newFlatBoard() *Board {
	return &Board{} // zero value: all Normal, all nil Tile
}

// newTestBag returns a bag with tiles in exactly the provided order, without
// shuffling. The first Draw will return tiles from the end of the slice.
func newTestBag(tiles []Tile) *Bag {
	cp := make([]Tile, len(tiles))
	copy(cp, tiles)
	return &Bag{tiles: cp}
}

// testDict is constructed once in TestMain and shared across all engine tests.
var testDict *dictionary.Dictionary

// deterministicRNG returns a seeded *rand.Rand for reproducible tests.
func deterministicRNG() *rand.Rand {
	return rand.New(rand.NewSource(42))
}

// newGameState returns a fresh GameState using the shared testDict and fixed RNG seed.
func newGameState() *GameState {
	return New(testDict.Name(), 5, deterministicRNG())
}

func TestMain(m *testing.M) {
	var err error
	testDict, err = dictionary.NewFromWords("test", engineTestWords)
	if err != nil {
		panic("TestMain: failed to build test dictionary: " + err.Error())
	}
	os.Exit(m.Run())
}
