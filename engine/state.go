// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package engine is documented in doc.go.
package engine

import (
	"fmt"
	"math/rand/v2"

	"tilewords/dictionary"
)

// Turn identifies whose turn it is.
type Turn int

const (
	// HumanTurn means the human player is to move.
	HumanTurn Turn = iota
	// CPUTurn means the CPU player is to move.
	CPUTurn
)

// EndReason describes why the game ended.
type EndReason int

const (
	// NotOver means the game is still in progress.
	NotOver EndReason = iota
	// RackExhausted means a player emptied their rack while the bag was empty.
	RackExhausted
	// SixConsecutivePasses means both players accumulated 6 consecutive non-play moves.
	SixConsecutivePasses
)

// GameState is the canonical, single source of truth for all mutable game data.
// All fields are exported so encoding/gob can serialise a save file.
// All mutations must go through Command.Execute; all reversals through Command.Undo.
//
// It holds no undo stack of its own: the executed commands live with the UI's move-history
// log, which is what steps an undo back through the turns. Undo is therefore not restored
// across a save/load (FR-09 — undo is available immediately after a move, not after resuming
// a saved game), which is why History below stores rendered lines rather than commands.
type GameState struct {
	Board      *Board
	HumanRack  *Rack
	CPURack    *Rack
	Bag        *Bag
	HumanScore int
	CPUScore   int
	// ConsecutivePasses counts consecutive scoreless turns: PassMove, ExchangeMove,
	// and zero-scoring PlayMove executions. Resets to 0 on a scoring play. The game
	// ends when this reaches 6 (BR-E09/BR-E10).
	ConsecutivePasses int
	CurrentTurn       Turn
	// MoveNumber increments on each Command.Execute and decrements on Undo.
	MoveNumber int
	DictName   dictionary.DictName
	// CPULevel is the difficulty level selected at game start, in the cpu package's level range
	// (see cpu.MinLevel and cpu.MaxLevel). The engine stores it without interpreting it; the
	// range is the cpu package's to define and it clamps anything outside it.
	CPULevel int
	// Mode is the game mode (board layout + tile economy) chosen at game start. It is
	// persisted so a resumed game keeps the same board and economy. Older save files
	// without this field decode as the zero value, ClassicMode.
	Mode GameMode

	// EndgameScored guards ApplyEndgameScoring against double application, including
	// across a save/load: it is exported so gob persists it, ensuring a finished
	// game that is saved and reloaded is never scored a second time.
	EndgameScored bool

	// OfficialNotation records the player's move-history display preference (official
	// coordinate notation when true, otherwise the plain word list). It is a UI preference
	// rather than rules state, but is persisted here so a saved game resumes in the same
	// format. Older save files without this field decode as false (plain).
	OfficialNotation bool

	// BoardHeaders records whether the board shows its row and column headers (A-O across
	// the top, 1-15 down the left) — the strips a square's coordinate is read off. Like
	// OfficialNotation it is a UI preference rather than rules state, persisted here so a
	// resumed game looks the way the player left it. Older save files without this field
	// decode as false (no headers).
	BoardHeaders bool

	// History is the move log shown in the UI's move-history panel, persisted so a resumed
	// game keeps its record. It holds already-rendered display data rather than executable
	// commands: undo is intentionally not restored across a save/load. Empty for a fresh
	// game and for older save files without this field.
	History []MoveRecord

	// OpeningDraw records how the first turn was decided (BR-E19). It is set by New
	// and is nil for GameState values constructed directly (e.g. in tests).
	OpeningDraw *OpeningDraw
}

// MoveRecord is one entry of the persisted move-history log. It stores rendered display
// data (not an executable command), so it is gob-serialisable and independent of undo.
type MoveRecord struct {
	// Player is "You" or "CPU".
	Player string
	// Line is the already-formatted move-history display line.
	Line string
	// Points is the score this move contributed (a play's score; 0 for a pass or exchange),
	// used to restore the status summary.
	Points int
	// Cells are the board cells this move placed, used to restore the CPU's last-word
	// highlight; nil for a pass or exchange.
	Cells [][2]int
	// Words are the words this play formed (main word + cross-words), used to repopulate the
	// definitions panel when a save is loaded; nil for a pass or exchange. Absent from saves
	// written before this field existed, in which case a resumed game's definitions stay empty.
	Words []string
}

// OpeningDraw records the single tile each player drew to decide who plays first
// under the standard opening-draw rule (BR-E19): the player whose letter is nearest
// the start of the alphabet plays first, with a blank counting as earlier than 'A'.
// The drawn tiles are returned to the bag and reshuffled before the racks are dealt,
// so this is purely an informational record of how the first turn was decided.
type OpeningDraw struct {
	HumanLetter byte // letter drawn by the human; 0 = blank
	CPULetter   byte // letter drawn by the CPU; 0 = blank
	First       Turn // the player who won the draw and plays first
}

// New initialises a fresh ClassicMode GameState.
func New(dictName dictionary.DictName, cpuLevel int, rng *rand.Rand) *GameState {
	return NewWithMode(dictName, cpuLevel, ClassicMode, rng)
}

// NewWithMode initialises a fresh GameState for mode: creates a shuffled bag and board
// for that mode, decides who plays first via the standard opening draw (BR-E19), then
// deals 7 tiles to each rack.
func NewWithMode(dictName dictionary.DictName, cpuLevel int, mode GameMode, rng *rand.Rand) *GameState {
	board := NewBoardForMode(mode)
	bag := NewBagForMode(rng, mode)

	// Decide the first player by the standard opening draw, then deal the racks.
	// drawForFirstTurn returns its tiles to the bag and reshuffles, so the racks
	// are dealt from the full bag.
	firstTurn, humanLetter, cpuLetter := drawForFirstTurn(bag, rng)

	humanRack := &Rack{}
	humanRack.Replenish(bag, rng)

	cpuRack := &Rack{}
	cpuRack.Replenish(bag, rng)

	return &GameState{
		Board:       board,
		HumanRack:   humanRack,
		CPURack:     cpuRack,
		Bag:         bag,
		CurrentTurn: firstTurn,
		DictName:    dictName,
		CPULevel:    cpuLevel,
		Mode:        mode,
		OpeningDraw: &OpeningDraw{
			HumanLetter: humanLetter,
			CPULetter:   cpuLetter,
			First:       firstTurn,
		},
	}
}

// drawForFirstTurn implements the standard opening draw (BR-E19): each player draws
// one tile from the bag, and the player whose letter is nearest the start of the
// alphabet plays first. A blank (letter 0) sorts before 'A', so it wins any draw.
// A tie (both players draw the same letter, or both draw blanks) is re-drawn. The
// drawn tiles are always returned to the bag and reshuffled with rng before this
// returns, so the caller deals the racks from the full bag.
func drawForFirstTurn(bag *Bag, rng *rand.Rand) (first Turn, humanLetter, cpuLetter byte) {
	for {
		drawn := bag.Draw(2, rng)
		humanLetter, cpuLetter = drawn[0].Letter, drawn[1].Letter
		bag.Return(drawn, rng)
		if humanLetter == cpuLetter {
			continue // tie — draw again
		}
		// A smaller letter byte is nearer the start of the alphabet; the blank
		// sentinel (0) is smallest and therefore plays first.
		if humanLetter < cpuLetter {
			return HumanTurn, humanLetter, cpuLetter
		}
		return CPUTurn, humanLetter, cpuLetter
	}
}

// ValidateDecodedState checks a GameState that came from outside the engine — a decoded
// save file — for the structural invariants every state the engine builds already
// satisfies. It returns an error describing the first problem found.
//
// Callers must run this on any decoded state before playing it. gob reports only the
// problems it can see in the encoding, so a file whose bytes decode cleanly can still
// carry values no game could have reached, and code downstream is written against the
// invariants rather than re-checking them. A single flipped bit is enough: it can turn a
// rack blank into a tile with letter 0, which the CPU's move generator would then index a
// letter-keyed array with.
//
// It deliberately checks only what would otherwise crash or corrupt play. Implausible but
// harmless values — a negative score, an odd turn number — are left alone rather than
// second-guessed, so a save is refused only when playing it could not work.
func ValidateDecodedState(s *GameState) error {
	if s == nil {
		return fmt.Errorf("engine.ValidateDecodedState: nil state")
	}
	if s.Board == nil || s.HumanRack == nil || s.CPURack == nil || s.Bag == nil {
		return fmt.Errorf("engine.ValidateDecodedState: missing board, rack, or bag")
	}

	// Tiles must be ones the game could have produced, wherever they are held. The bag
	// matters as much as the racks: a malformed tile there surfaces turns later, when
	// Replenish draws it onto a rack.
	for _, holder := range []struct {
		name string
		bad  func() (Tile, bool)
	}{
		{"human rack", s.HumanRack.malformedTile},
		{"CPU rack", s.CPURack.malformedTile},
		{"bag", s.Bag.malformedTile},
		{"board", s.Board.malformedTile},
	} {
		if t, found := holder.bad(); found {
			return fmt.Errorf("engine.ValidateDecodedState: %s holds an unplayable tile {Letter:%d IsBlank:%v}",
				holder.name, t.Letter, t.IsBlank)
		}
	}

	// A rack over capacity cannot be played: an exchange removes the selected tiles, draws
	// that many back, and then cannot fit them.
	if n := s.HumanRack.Count(); n > MaxRackSize {
		return fmt.Errorf("engine.ValidateDecodedState: human rack holds %d tiles, more than the %d-tile capacity", n, MaxRackSize)
	}
	if n := s.CPURack.Count(); n > MaxRackSize {
		return fmt.Errorf("engine.ValidateDecodedState: CPU rack holds %d tiles, more than the %d-tile capacity", n, MaxRackSize)
	}
	return nil
}

// Clone returns a deep copy of the GameState suitable for use by the CPU goroutine.
// The clone is fully independent: mutations to the original do not affect it.
//
// It copies the struct wholesale and then replaces every field that would otherwise be
// shared, so a field added to GameState is carried over by default rather than silently
// dropped from clones until someone notices.
func (s *GameState) Clone() *GameState {
	c := *s
	c.Board = s.Board.Clone()
	c.HumanRack = s.HumanRack.Clone()
	c.CPURack = s.CPURack.Clone()
	c.Bag = s.Bag.Clone()
	// History is rendered display data the CPU never reads; sharing the slice would let the
	// UI append to it while the CPU goroutine holds the clone.
	c.History = nil
	// OpeningDraw is written once by New and never mutated, so the pointer is safe to share.
	return &c
}

// currentRack returns the rack belonging to the player whose turn it is.
func currentRack(state *GameState) *Rack {
	if state.CurrentTurn == HumanTurn {
		return state.HumanRack
	}
	return state.CPURack
}

// addScore adds points to the current player's score.
func addScore(state *GameState, points int) {
	if state.CurrentTurn == HumanTurn {
		state.HumanScore += points
	} else {
		state.CPUScore += points
	}
}

// subtractScore deducts points from the current player's score.
func subtractScore(state *GameState, points int) {
	if state.CurrentTurn == HumanTurn {
		state.HumanScore -= points
	} else {
		state.CPUScore -= points
	}
}

// opposite returns the other Turn value.
func opposite(t Turn) Turn {
	if t == HumanTurn {
		return CPUTurn
	}
	return HumanTurn
}
