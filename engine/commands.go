// Package engine is documented in doc.go.
package engine

import (
	"fmt"
	"math/rand"

	"tilewords/dictionary"
)

// Command is the sole mechanism for mutating GameState.
// Every concrete command stores enough state at Execute time to fully reverse
// the operation via Undo (inverse-command pattern).
type Command interface {
	// Execute applies the move to state, mutating it. Returns an error if the
	// move is invalid; state is unchanged on error.
	Execute(state *GameState, dict *dictionary.Dictionary, rng *rand.Rand) error
	// Undo reverses the effect of the most recent Execute on state.
	// Undo must not fail; if state is inconsistent, it panics with a diagnostic.
	Undo(state *GameState)
}

// PlayCommand executes a PlayMove and stores the data needed to reverse it.
type PlayCommand struct {
	// Move is the play being executed. WordsFormed and Score are populated
	// by Execute via ValidatePlacement and Score.
	Move PlayMove

	// drawnTiles records the tiles drawn from the bag during Execute.
	// These are removed from the rack and returned to the bag on Undo.
	drawnTiles []Tile

	// prevPasses is ConsecutivePasses before Execute, restored on Undo.
	prevPasses int
}

// Execute validates and commits the play. Populates Move.WordsFormed and Move.Score.
func (cmd *PlayCommand) Execute(state *GameState, dict *dictionary.Dictionary, rng *rand.Rand) error {
	rack := currentRack(state)

	// Validate placement and extract formed words (populates Move.WordsFormed).
	if _, err := ValidatePlacement(state.Board, &cmd.Move, dict); err != nil {
		return fmt.Errorf("engine.PlayCommand.Execute: %w", err)
	}

	// Score the move (uses Move.WordsFormed populated above).
	if _, err := Score(state.Board, &cmd.Move); err != nil {
		return fmt.Errorf("engine.PlayCommand.Execute: %w", err)
	}

	// Remove played tiles from the rack before placement to verify ownership.
	placedTiles := make([]Tile, len(cmd.Move.Placed))
	for i, pt := range cmd.Move.Placed {
		placedTiles[i] = pt.Tile
	}
	if err := rack.Remove(placedTiles); err != nil {
		return fmt.Errorf("engine.PlayCommand.Execute: %w", err)
	}

	// Place tiles on the board.
	for _, pt := range cmd.Move.Placed {
		if err := state.Board.Place(pt.Row, pt.Col, pt.Tile); err != nil {
			// Board placement failure after rack removal indicates a bug.
			panic(fmt.Sprintf("engine.PlayCommand.Execute: board.Place failed after rack.Remove: %v", err))
		}
	}

	// Save ConsecutivePasses before updating, for Undo. A scoring play resets the
	// scoreless-turn counter; a zero-scoring play (e.g. an all-blank word on plain
	// squares, scoring 0) is itself a scoreless turn and counts toward the six-turn
	// end condition, matching official tournament rules.
	cmd.prevPasses = state.ConsecutivePasses
	if cmd.Move.Score > 0 {
		state.ConsecutivePasses = 0
	} else {
		state.ConsecutivePasses++
	}

	// Replenish the rack; record drawn tiles for Undo.
	cmd.drawnTiles = rack.Replenish(state.Bag)

	// Credit the score to the current player.
	addScore(state, cmd.Move.Score)

	// Advance turn and move counter.
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber++

	return nil
}

// Undo reverses a PlayCommand.Execute, restoring state to exactly its prior form.
func (cmd *PlayCommand) Undo(state *GameState) {
	// The player who made this move is now on the opposite turn (we flipped in Execute).
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber--

	rack := currentRack(state)

	// Deduct the score awarded by this move.
	subtractScore(state, cmd.Move.Score)

	// Return tiles drawn during Execute back to the bag (no reshuffle on undo).
	if len(cmd.drawnTiles) > 0 {
		if err := rack.Remove(cmd.drawnTiles); err != nil {
			panic(fmt.Sprintf("engine.PlayCommand.Undo: failed to remove drawn tiles from rack: %v", err))
		}
		state.Bag.Return(cmd.drawnTiles, nil)
	}

	// Remove placed tiles from the board and return them to the rack.
	// Blank tiles must have AssignedLetter and Letter cleared so they display
	// correctly as unassigned blanks in the rack after undo.
	placedTiles := make([]Tile, len(cmd.Move.Placed))
	for i, pt := range cmd.Move.Placed {
		state.Board.Remove(pt.Row, pt.Col)
		t := pt.Tile
		if t.IsBlank {
			t.Letter = 0
			t.AssignedLetter = 0
		}
		placedTiles[i] = t
	}
	if err := rack.Add(placedTiles); err != nil {
		panic(fmt.Sprintf("engine.PlayCommand.Undo: failed to return played tiles to rack: %v", err))
	}

	// Restore consecutive-pass counter.
	state.ConsecutivePasses = cmd.prevPasses
}

// ExchangeCommand executes an ExchangeMove and stores the data needed to reverse it.
type ExchangeCommand struct {
	// Move is the exchange being executed.
	Move ExchangeMove

	// drawnTiles records the tiles drawn from the bag during Execute.
	drawnTiles []Tile

	// bagSnapshot holds the complete bag state before Execute, enabling exact
	// bag restoration on Undo even after the reshuffle (Pattern 3).
	bagSnapshot []Tile

	// prevPasses is ConsecutivePasses before Execute, restored on Undo.
	prevPasses int
}

// Execute validates and commits the exchange.
func (cmd *ExchangeCommand) Execute(state *GameState, dict *dictionary.Dictionary, rng *rand.Rand) error {
	// Tile exchange requires at least 7 tiles in the bag (BR-E08).
	if state.Bag.Count() < MaxRackSize {
		return fmt.Errorf("engine.ExchangeCommand.Execute: bag has %d tile(s); need at least %d to exchange",
			state.Bag.Count(), MaxRackSize)
	}

	rack := currentRack(state)

	// Remove the tiles to be exchanged from the rack.
	if err := rack.Remove(cmd.Move.Tiles); err != nil {
		return fmt.Errorf("engine.ExchangeCommand.Execute: %w", err)
	}

	// Snapshot the bag before any changes so Undo can restore it exactly.
	cmd.bagSnapshot = make([]Tile, len(state.Bag.tiles))
	copy(cmd.bagSnapshot, state.Bag.tiles)

	// Draw replacements, then return the exchanged tiles (with reshuffle).
	cmd.drawnTiles = state.Bag.Draw(len(cmd.Move.Tiles))
	if err := rack.Add(cmd.drawnTiles); err != nil {
		panic(fmt.Sprintf("engine.ExchangeCommand.Execute: failed to add drawn tiles to rack: %v", err))
	}
	state.Bag.Return(cmd.Move.Tiles, rng) // reshuffle

	// Exchange counts as a non-play move (Q2 Option B: BR-E09).
	cmd.prevPasses = state.ConsecutivePasses
	state.ConsecutivePasses++

	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber++

	return nil
}

// Undo reverses an ExchangeCommand.Execute.
func (cmd *ExchangeCommand) Undo(state *GameState) {
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber--

	rack := currentRack(state)

	// Remove drawn tiles from rack.
	if err := rack.Remove(cmd.drawnTiles); err != nil {
		panic(fmt.Sprintf("engine.ExchangeCommand.Undo: failed to remove drawn tiles from rack: %v", err))
	}

	// Restore the exchanged tiles to the rack.
	if err := rack.Add(cmd.Move.Tiles); err != nil {
		panic(fmt.Sprintf("engine.ExchangeCommand.Undo: failed to return exchanged tiles to rack: %v", err))
	}

	// Restore bag from snapshot taken before Execute's reshuffle.
	state.Bag.restoreSnapshot(cmd.bagSnapshot)

	state.ConsecutivePasses = cmd.prevPasses
}

// PassCommand executes a PassMove and stores the data needed to reverse it.
type PassCommand struct {
	// prevPasses is ConsecutivePasses before Execute, restored on Undo.
	prevPasses int
}

// Execute commits the pass: increments the consecutive-pass counter and flips the turn.
func (cmd *PassCommand) Execute(state *GameState, dict *dictionary.Dictionary, rng *rand.Rand) error {
	cmd.prevPasses = state.ConsecutivePasses
	state.ConsecutivePasses++ // pass counts toward the 6-pass end condition (BR-E09)
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber++
	return nil
}

// Undo reverses a PassCommand.Execute.
func (cmd *PassCommand) Undo(state *GameState) {
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber--
	state.ConsecutivePasses = cmd.prevPasses
}

// UndoLastRound reverts one full human+AI round: the AI's most recent command first,
// then the human's command before that (FR-09).
// Must only be called when CurrentTurn == HumanTurn and both LastHumanCommand and
// LastAICommand are non-nil; the UI is responsible for checking preconditions.
func UndoLastRound(state *GameState) {
	// Undo AI's move first (it was the most recent), then the human's.
	if state.LastAICommand != nil {
		state.LastAICommand.Undo(state)
		state.LastAICommand = nil
	}
	if state.LastHumanCommand != nil {
		state.LastHumanCommand.Undo(state)
		state.LastHumanCommand = nil
	}
}
