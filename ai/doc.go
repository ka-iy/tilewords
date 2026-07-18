// Package ai implements the computer player for TileWords.
//
// Move generation uses the GADDAG left-extension algorithm described in:
//
//	Appel, A. W. & Jacobson, G. J. (1988/1998).
//	"The World's Fastest Crossword-Board Game Program."
//	Communications of the ACM, 31(5), 572–578.
//
// The three public entry points are:
//
//	GenerateMoves — enumerate every legal play for a given board, rack, and dictionary.
//	SelectMove    — choose one candidate from the sorted result according to difficulty level.
//	ChooseMove    — orchestrate one full AI turn (generate → select → exchange/pass fallback).
//
// For integration with a polling UI loop, use AIWorker so that move generation
// runs on a dedicated goroutine and the UI goroutine never blocks:
//
//	worker := ai.NewAIWorker(ai.ChooseMove)
//
//	// On the AI's turn (Request clones state internally):
//	worker.Request(state, dict, level)
//
//	// Each Update() frame:
//	if move, ok := worker.Poll(); ok {
//	    // Apply move to game state.
//	}
package ai
