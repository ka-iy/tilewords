// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package engine implements all game rules, board state management, tile handling,
// scoring, and the command/undo system for the TileWords crossword board game.
//
// # Architecture
//
// All GameState mutations go through a Command.Execute call.
// All reversals go through Command.Undo. This invariant is the foundation of the
// undo system (FR-09) and must not be bypassed by direct field assignment.
//
// # Thread Safety
//
// GameState is owned by the UI goroutine and is not safe for concurrent use.
// The AI goroutine receives a deep copy via GameState.Clone() before the goroutine
// is started. The clone is independent; the AI reads it without synchronisation.
//
// # Usage
//
//	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
//	state := engine.New(dictionary.DictENABLE, 5, rng)
//
//	cmd := &engine.PlayCommand{Move: move}
//	if err := cmd.Execute(state, dict, rng); err != nil {
//	    // invalid move — show error to player
//	}
//
//	over, reason := engine.IsGameOver(state)
//	if over {
//	    engine.ApplyEndgameScoring(state, reason)
//	}
package engine
