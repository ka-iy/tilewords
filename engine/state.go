// Package engine is documented in doc.go.
package engine

import (
	"math/rand"

	"squabble/dictionary"
)

// Turn identifies whose turn it is.
type Turn int

const (
	// HumanTurn means the human player is to move.
	HumanTurn Turn = iota
	// AITurn means the AI player is to move.
	AITurn
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
type GameState struct {
	Board      *Board
	HumanRack  *Rack
	AIRack     *Rack
	Bag        *Bag
	HumanScore int
	AIScore    int
	// ConsecutivePasses counts consecutive scoreless turns: PassMove, ExchangeMove,
	// and zero-scoring PlayMove executions. Resets to 0 on a scoring play. The game
	// ends when this reaches 6 (BR-E09/BR-E10).
	ConsecutivePasses int
	CurrentTurn       Turn
	// MoveNumber increments on each Command.Execute and decrements on Undo.
	MoveNumber int
	DictName   dictionary.DictName
	// AILevel is the difficulty level (1–10) selected at game start.
	AILevel int

	// LastHumanCommand and LastAICommand store the most recent commands for undo.
	// These fields are intentionally excluded from save files: undo state is
	// discarded on save/load (consistent with FR-09: undo is available only
	// immediately after a move, not after resuming a saved game).
	LastHumanCommand Command
	LastAICommand    Command

	// EndgameScored guards ApplyEndgameScoring against double application, including
	// across a save/load: it is exported so gob persists it, ensuring a finished
	// game that is saved and reloaded is never scored a second time.
	EndgameScored bool

	// ScrabbleNotation records the player's move-history display preference (Scrabble
	// coordinate notation when true, otherwise the plain word list). It is a UI preference
	// rather than rules state, but is persisted here so a saved game resumes in the same
	// format. Older save files without this field decode as false (plain).
	ScrabbleNotation bool

	// History is the move log shown in the UI's move-history panel, persisted so a resumed
	// game keeps its record. It holds already-rendered display data rather than executable
	// commands: undo is intentionally not restored across a save/load (consistent with the
	// zeroed LastHumanCommand/LastAICommand above). Empty for a fresh game and for older
	// save files without this field.
	History []MoveRecord

	// OpeningDraw records how the first turn was decided (BR-E19). It is set by New
	// and is nil for GameState values constructed directly (e.g. in tests).
	OpeningDraw *OpeningDraw
}

// MoveRecord is one entry of the persisted move-history log. It stores rendered display
// data (not an executable command), so it is gob-serialisable and independent of undo.
type MoveRecord struct {
	// Player is "You" or "AI".
	Player string
	// Line is the already-formatted move-history display line.
	Line string
	// Points is the score this move contributed (a play's score; 0 for a pass or exchange),
	// used to restore the status summary.
	Points int
	// Cells are the board cells this move placed, used to restore the AI's last-word
	// highlight; nil for a pass or exchange.
	Cells [][2]int
}

// OpeningDraw records the single tile each player drew to decide who plays first
// under the standard opening-draw rule (BR-E19): the player whose letter is nearest
// the start of the alphabet plays first, with a blank counting as earlier than 'A'.
// The drawn tiles are returned to the bag and reshuffled before the racks are dealt,
// so this is purely an informational record of how the first turn was decided.
type OpeningDraw struct {
	HumanLetter byte // letter drawn by the human; 0 = blank
	AILetter    byte // letter drawn by the AI; 0 = blank
	First       Turn // the player who won the draw and plays first
}

// New initialises a fresh GameState: creates a shuffled bag, decides who plays
// first via the standard opening draw (BR-E19), then deals 7 tiles to each rack.
func New(dictName dictionary.DictName, aiLevel int, rng *rand.Rand) *GameState {
	board := NewBoard()
	bag := NewBag(rng)

	// Decide the first player by the standard opening draw, then deal the racks.
	// drawForFirstTurn returns its tiles to the bag and reshuffles, so the racks
	// are dealt from the full 100-tile bag.
	firstTurn, humanLetter, aiLetter := drawForFirstTurn(bag, rng)

	humanRack := &Rack{}
	humanRack.Replenish(bag)

	aiRack := &Rack{}
	aiRack.Replenish(bag)

	return &GameState{
		Board:       board,
		HumanRack:   humanRack,
		AIRack:      aiRack,
		Bag:         bag,
		CurrentTurn: firstTurn,
		DictName:    dictName,
		AILevel:     aiLevel,
		OpeningDraw: &OpeningDraw{
			HumanLetter: humanLetter,
			AILetter:    aiLetter,
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
func drawForFirstTurn(bag *Bag, rng *rand.Rand) (first Turn, humanLetter, aiLetter byte) {
	for {
		drawn := bag.Draw(2)
		humanLetter, aiLetter = drawn[0].Letter, drawn[1].Letter
		bag.Return(drawn, rng)
		if humanLetter == aiLetter {
			continue // tie — draw again
		}
		// A smaller letter byte is nearer the start of the alphabet; the blank
		// sentinel (0) is smallest and therefore plays first.
		if humanLetter < aiLetter {
			return HumanTurn, humanLetter, aiLetter
		}
		return AITurn, humanLetter, aiLetter
	}
}

// Clone returns a deep copy of the GameState suitable for use by the AI goroutine.
// The clone is fully independent: mutations to the original do not affect it.
// LastHumanCommand and LastAICommand are intentionally omitted; the AI operates
// on a read-only snapshot and has no need for undo state.
func (s *GameState) Clone() *GameState {
	return &GameState{
		Board:             s.Board.Clone(),
		HumanRack:         s.HumanRack.Clone(),
		AIRack:            s.AIRack.Clone(),
		Bag:               s.Bag.Clone(),
		HumanScore:        s.HumanScore,
		AIScore:           s.AIScore,
		ConsecutivePasses: s.ConsecutivePasses,
		CurrentTurn:       s.CurrentTurn,
		MoveNumber:        s.MoveNumber,
		DictName:          s.DictName,
		AILevel:           s.AILevel,
		// LastHumanCommand and LastAICommand are deliberately omitted.
	}
}

// currentRack returns the rack belonging to the player whose turn it is.
func currentRack(state *GameState) *Rack {
	if state.CurrentTurn == HumanTurn {
		return state.HumanRack
	}
	return state.AIRack
}

// addScore adds points to the current player's score.
func addScore(state *GameState, points int) {
	if state.CurrentTurn == HumanTurn {
		state.HumanScore += points
	} else {
		state.AIScore += points
	}
}

// subtractScore deducts points from the current player's score.
func subtractScore(state *GameState, points int) {
	if state.CurrentTurn == HumanTurn {
		state.HumanScore -= points
	} else {
		state.AIScore -= points
	}
}

// opposite returns the other Turn value.
func opposite(t Turn) Turn {
	if t == HumanTurn {
		return AITurn
	}
	return HumanTurn
}
