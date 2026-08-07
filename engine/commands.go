// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

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
	//
	// rng reshuffles the bag once the undone tiles are back in it, so the order the next
	// draw will follow is not the order the undone move already revealed. Restoring the bag
	// exactly would otherwise let a player play a move, see the tiles they drew, undo, and
	// replay knowing what is coming. Board, racks, scores, turn and counters are still
	// restored exactly; only the unseen bag order changes. Pass nil to keep the order
	// (tests that assert exact restoration).
	Undo(state *GameState, rng *rand.Rand)
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
			// Unreachable: ValidatePlacement above has already rejected out-of-bounds
			// coordinates, cells the board already occupies, and two tiles claiming the
			// same cell — the three ways Place can fail. Kept as an assertion because the
			// rack is already debited here, so there is no error return that could honour
			// Execute's unchanged-on-error contract.
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

	// Replenish the rack; record drawn tiles for Undo. rng shuffles the bag before the
	// replacement tiles are taken. This is the draw for whichever player is on turn, so the
	// human and the AI replenish through the same path.
	cmd.drawnTiles = rack.Replenish(state.Bag, rng)

	// Credit the score to the current player.
	addScore(state, cmd.Move.Score)

	// Advance turn and move counter.
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber++

	return nil
}

// Undo reverses a PlayCommand.Execute, restoring state to exactly its prior form.
func (cmd *PlayCommand) Undo(state *GameState, rng *rand.Rand) {
	// The player who made this move is now on the opposite turn (we flipped in Execute).
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber--

	rack := currentRack(state)

	// Deduct the score awarded by this move.
	subtractScore(state, cmd.Move.Score)

	// Return tiles drawn during Execute back to the bag, reshuffling so the draw order this
	// move revealed is not the order the next draw follows (see Command.Undo).
	if len(cmd.drawnTiles) > 0 {
		if err := rack.Remove(cmd.drawnTiles); err != nil {
			panic(fmt.Sprintf("engine.PlayCommand.Undo: failed to remove drawn tiles from rack: %v", err))
		}
		state.Bag.Return(cmd.drawnTiles, rng)
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

	// An over-capacity rack cannot be exchanged: the draw below replaces exactly the tiles
	// removed, so the rack ends the turn at the size it started, and Add would reject it.
	// Rejected here, before anything is mutated, because Execute must leave state unchanged
	// when it returns an error. A rack this size can only come from a decoded save that
	// bypassed ValidateDecodedState.
	if rack.Count() > MaxRackSize {
		return fmt.Errorf("engine.ExchangeCommand.Execute: rack holds %d tile(s), more than the %d-tile capacity",
			rack.Count(), MaxRackSize)
	}

	// Remove the tiles to be exchanged from the rack.
	if err := rack.Remove(cmd.Move.Tiles); err != nil {
		return fmt.Errorf("engine.ExchangeCommand.Execute: %w", err)
	}

	// Snapshot the bag before any changes so Undo can restore it exactly.
	cmd.bagSnapshot = make([]Tile, len(state.Bag.tiles))
	copy(cmd.bagSnapshot, state.Bag.tiles)

	// Draw replacements, then return the exchanged tiles (with reshuffle). The draw shuffles
	// the bag first. It must stay ahead of the Return below: the exchanged tiles are out of
	// the bag for the whole draw, so that shuffle cannot hand a player back a tile they just
	// exchanged away.
	cmd.drawnTiles = state.Bag.Draw(len(cmd.Move.Tiles), rng)
	if err := rack.Add(cmd.drawnTiles); err != nil {
		// Unreachable: the bag holds at least MaxRackSize tiles and the rack was within
		// capacity, so the draw returns exactly as many tiles as Remove took out. Kept as
		// an assertion because the rack and bag are already mutated here, so there is no
		// error return that could honour Execute's unchanged-on-error contract.
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
func (cmd *ExchangeCommand) Undo(state *GameState, rng *rand.Rand) {
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

	// Restore the bag's contents from the snapshot taken before Execute, then reshuffle: the
	// snapshot is what makes the tile multiset exact, and the reshuffle is what stops the
	// player from previewing the replacements an exchange drew and then undoing it (see
	// Command.Undo).
	state.Bag.restoreSnapshot(cmd.bagSnapshot)
	state.Bag.Shuffle(rng)

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

// Undo reverses a PassCommand.Execute. A pass neither draws nor returns tiles, so there is
// nothing for rng to reshuffle and no draw order the move could have revealed.
func (cmd *PassCommand) Undo(state *GameState, _ *rand.Rand) {
	state.CurrentTurn = opposite(state.CurrentTurn)
	state.MoveNumber--
	state.ConsecutivePasses = cmd.prevPasses
}

// Undo is driven by the UI, which owns the stack of executed commands (see the move-history
// log in the ui package). The engine deliberately keeps no "last command" of its own: a
// single-round undo entry point here would duplicate that stack and could only ever reverse
// one round, whereas the UI's stack steps back turn after turn.
