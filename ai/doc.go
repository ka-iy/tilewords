// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

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
// All three are synchronous and hold no state of their own, so a caller that must not block
// runs ChooseMove on its own goroutine and marshals the result back. This package
// deliberately provides no worker or scheduling type: the caller already owns the turn
// lifecycle — when a turn is abandoned, how long to wait, which thread applies the move — and
// a helper here could only duplicate that with less context.
package ai
