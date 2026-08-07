// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ai is documented in doc.go.
package ai

import (
	"math/rand/v2"

	"tilewords/dictionary"
	"tilewords/engine"
)

// ChooseMove selects and returns the AI's move for the current turn.
// It always returns a non-nil engine.Move (NFR-AI-R5).
//
// Algorithm (BR-AI-10):
//  1. Generate all legal moves for the AI rack.
//  2. If any legal moves exist, select one via SelectMove and return a PlayMove.
//  3. If no legal moves exist and the bag has enough tiles, exchange all rack tiles.
//  4. Otherwise pass.
//
// ChooseMove always operates on state.AIRack — the human rack is never consulted (BR-AI-08).
func ChooseMove(
	state *engine.GameState,
	dict *dictionary.Dictionary,
	level int,
	rng *rand.Rand,
) engine.Move {
	candidates := GenerateMoves(state.Board, state.AIRack, dict)

	if len(candidates) > 0 {
		selected := SelectMove(candidates, level, rng)
		return selected.Move
	}

	// No legal plays available — exchange all tiles if the rack is non-empty and the
	// bag can replenish a full rack (BR-AI-10). An empty rack means the game is
	// effectively over, so pass rather than emit a zero-tile exchange.
	if state.AIRack.Count() > 0 && state.Bag.Count() >= engine.MaxRackSize {
		return engine.ExchangeMove{Tiles: state.AIRack.Tiles()}
	}

	// Bag too small (or rack empty) to warrant exchange — pass.
	return engine.PassMove{}
}
