// Package dictionary is documented in doc.go.
package dictionary

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ArcSep is the arc-separator byte used in GADDAG strings to delimit
// the reversed prefix from the forward suffix (Appel-Jacobson §3).
const ArcSep byte = '+'

// RootNodeID is the fixed NodeID of the GADDAG root node.
// The build tool always assigns 1 to the root.
const RootNodeID NodeID = 1

// MinWordLen is the minimum valid word length for a 15×15 board.
const MinWordLen = 2

// MaxWordLen is the maximum valid word length for a 15×15 board.
const MaxWordLen = 15

// NodeID identifies a node in the GADDAG graph.
type NodeID uint32

// gaddagData is the gob-compatible wire format for GADDAG serialisation.
// All fields are exported so encoding/gob can encode and decode them.
type gaddagData struct {
	Edges     map[NodeID]map[byte]NodeID
	Terminals map[NodeID]bool
	Root      NodeID
	Size      uint32
	WordCount uint32 // number of distinct words (stored explicitly; terminal count is no longer a proxy)
}

// GADDAG is the directed acyclic word graph described in Appel & Jacobson (1998).
// It is read-only after Load or Build returns; all methods are safe for concurrent use.
type GADDAG struct {
	edges     map[NodeID]map[byte]NodeID
	terminals map[NodeID]bool
	root      NodeID
	size      uint32
	wordCount uint32
}

// loadGADDAG deserialises a GADDAG from gob-encoded bytes produced by tools/buildgaddag or Build.
// Returns a descriptive error if the data is malformed or the root node is invalid.
func loadGADDAG(data []byte) (*GADDAG, error) {
	var wd gaddagData
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&wd); err != nil {
		return nil, fmt.Errorf("dictionary.Load: gob decode failed: %w", err)
	}
	if wd.Root != RootNodeID {
		return nil, fmt.Errorf("dictionary.Load: invalid root node %d (want %d)", wd.Root, RootNodeID)
	}
	return &GADDAG{
		edges:     wd.Edges,
		terminals: wd.Terminals,
		root:      wd.Root,
		size:      wd.Size,
		wordCount: wd.WordCount,
	}, nil
}

// Root returns the NodeID of the GADDAG root node.
func (g *GADDAG) Root() NodeID { return g.root }

// words returns the number of distinct words stored in this GADDAG.
func (g *GADDAG) words() int { return int(g.wordCount) }

// Successor returns the NodeID reached by following the edge labelled letter from node,
// and whether such an edge exists.
// letter must be an uppercase A-Z byte or the arc-separator ArcSep ('+').
// Used by the AI move generator during left-extension traversal (Appel-Jacobson §5, GenerateMoves).
func (g *GADDAG) Successor(node NodeID, letter byte) (NodeID, bool) {
	m, ok := g.edges[node]
	if !ok {
		return 0, false
	}
	next, ok := m[letter]
	return next, ok
}

// IsTerminal reports whether node is a terminal node, i.e. a valid word ends here.
func (g *GADDAG) IsTerminal(node NodeID) bool {
	return g.terminals[node]
}

// contains reports whether word (already normalised to uppercase) is in the GADDAG.
// Implements 3-tier fast rejection before GADDAG traversal:
//  1. Length check
//  2. Byte validity
//  3. Full-reverse path traversal (Appel-Jacobson §3, k=n string)
func (g *GADDAG) contains(word string) bool {
	n := len(word)
	// Tier 1: length check
	if n < MinWordLen || n > MaxWordLen {
		return false
	}
	// Tier 2: byte validity (A-Z only)
	for i := 0; i < n; i++ {
		b := word[i]
		if b < 'A' || b > 'Z' {
			return false
		}
	}
	// Tier 3: full-reverse GADDAG traversal (the k=n string: wₙwₙ₋₁…w₁)
	node := g.root
	for i := n - 1; i >= 0; i-- {
		next, ok := g.Successor(node, word[i])
		if !ok {
			return false
		}
		node = next
	}
	return g.IsTerminal(node)
}

// Build constructs a GADDAG from the given word list and writes gob-encoded bytes to w.
// words must already be normalised to uppercase and contain only A-Z characters.
// The input is sorted and deduplicated before construction.
// This function is the single source of the GADDAG construction algorithm and is used
// by both tools/buildgaddag and the test suite.
func Build(words []string, w io.Writer) error {
	// Sort and deduplicate
	sort.Strings(words)
	words = dedup(words)

	nextID := NodeID(RootNodeID + 1)
	edges := make(map[NodeID]map[byte]NodeID)
	terminals := make(map[NodeID]bool)
	root := RootNodeID
	edges[root] = make(map[byte]NodeID)

	alloc := func() NodeID {
		id := nextID
		nextID++
		edges[id] = make(map[byte]NodeID)
		return id
	}

	// addString inserts a single GADDAG string into the graph.
	// The string is encoded as a sequence of bytes; the caller provides the exact sequence.
	addString := func(seq []byte, isTerminal bool) {
		node := root
		for _, b := range seq {
			next, ok := edges[node][b]
			if !ok {
				next = alloc()
				edges[node][b] = next
			}
			node = next
		}
		if isTerminal {
			terminals[node] = true
		}
	}

	// For each word w₁…wₙ, insert n GADDAG strings (Appel-Jacobson §3–4).
	// All paths are marked terminal at their final node:
	//   k=1..n-1: wₖ wₖ₋₁…w₁ '+' wₖ₊₁…wₙ  (terminal = true — end of a valid word path)
	//   k=n:      wₙ wₙ₋₁…w₁               (terminal = true — full reverse, used by Validate)
	//
	// Marking k<n paths as terminal is required by the Appel-Jacobson §5 move generator:
	// AfterAnchor (extendRight) records a play when IsTerminal(node) is true at an empty
	// or board-edge position. Without terminal marks on k<n paths, no '+'-based play is
	// ever recorded. The k=n terminal is used by contains (Validate) and is unaffected.
	for _, word := range words {
		n := len(word)
		for k := 1; k <= n; k++ {
			// Build the GADDAG string for position k.
			seq := make([]byte, 0, n+1)
			// Reversed prefix: wₖ wₖ₋₁ … w₁
			for i := k; i >= 1; i-- {
				seq = append(seq, word[i-1])
			}
			if k < n {
				// Arc separator + forward suffix: '+' wₖ₊₁…wₙ
				seq = append(seq, ArcSep)
				seq = append(seq, []byte(word[k:])...)
			}
			// All paths are terminal: k=n for validation, k<n for move generation.
			addString(seq, true)
		}
	}

	wd := gaddagData{
		Edges:     edges,
		Terminals: terminals,
		Root:      root,
		Size:      uint32(nextID - RootNodeID - 1),
		WordCount: uint32(len(words)), // words is already sorted+deduped at this point
	}
	enc := gob.NewEncoder(w)
	if err := enc.Encode(wd); err != nil {
		return fmt.Errorf("dictionary.Build: gob encode failed: %w", err)
	}
	return nil
}

// dedup removes adjacent duplicate strings from a sorted slice.
func dedup(sorted []string) []string {
	if len(sorted) == 0 {
		return sorted
	}
	out := sorted[:1]
	for _, s := range sorted[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}

// toUpper returns word normalised to uppercase.
// Avoids allocation when word is already uppercase.
func toUpper(word string) string {
	for i := 0; i < len(word); i++ {
		if word[i] >= 'a' && word[i] <= 'z' {
			return strings.ToUpper(word)
		}
	}
	return word
}
