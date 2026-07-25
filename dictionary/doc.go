// Package dictionary implements the GADDAG word-graph data structure and provides
// word validation and AI traversal support for the TileWords crossword board game.
//
// # GADDAG Algorithm
//
// The GADDAG is described in:
//
//	Appel, A. W. & Jacobson, G. J. (1988). "The World's Fastest Scrabble Program."
//	Communications of the ACM, 31(5), 572–578.
//
// For a word w₁…wₙ the GADDAG stores n strings:
//
//	k=1..n-1:  wₖ wₖ₋₁…w₁ '+' wₖ₊₁…wₙ   (enables left extensions during AI move gen)
//	k=n:       wₙ wₙ₋₁…w₁                  (full-reverse path; terminal marks a valid word)
//
// # Word List Assets
//
// Pre-built GADDAG assets (.bin files) are embedded in the binary via //go:embed and live
// in assets/dictionaries/. The raw word list .txt sources are NOT committed to the repository
// (licensing); developers must supply them and run the build tool before compiling:
//
//	make gaddag
//
// # Usage
//
//	dict, err := dictionary.Load(dictionary.DictENABLE)
//	if err != nil { ... }
//	if dict.Validate("QUIXOTIC") {
//	    fmt.Println("valid word")
//	}
//
// For AI move generation, obtain the underlying graph:
//
//	g := dict.GADDAG()
//	node, ok := g.Successor(g.Root(), 'Q')
//
package dictionary
