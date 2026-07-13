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

// gaddagData is the gob-compatible wire format for GADDAG serialisation. It stores the
// graph in a compressed-sparse-row (CSR) layout of flat slices rather than nested maps,
// which decodes into a far smaller heap footprint and hands its slices straight into the
// in-memory GADDAG with no post-decode copy. All fields are exported for encoding/gob.
type gaddagData struct {
	// EdgeOffsets has length NodeCount+1. Node id's outgoing edges occupy the index range
	// [EdgeOffsets[id], EdgeOffsets[id+1]) in EdgeLetters/EdgeTargets.
	EdgeOffsets []uint32
	// EdgeLetters is every edge's label byte, grouped by source node and ascending within
	// a node so a node's edges can be binary-searched by letter.
	EdgeLetters []byte
	// EdgeTargets is the destination node for the edge at the same index in EdgeLetters.
	EdgeTargets []NodeID
	// Terminal is a bitset over node ids: node id is terminal iff bit (id & 63) of
	// Terminal[id>>6] is set.
	Terminal []uint64
	// Root is the root node id (always RootNodeID).
	Root NodeID
	// NodeCount is one past the highest node id; valid ids are [0, NodeCount).
	NodeCount uint32
	// WordCount is the number of distinct words stored in the graph.
	WordCount uint32
}

// GADDAG is the directed acyclic word graph described in Appel & Jacobson (1998), stored
// in a compressed-sparse-row layout (see gaddagData). It is read-only after Load or Build
// returns; all methods are safe for concurrent use.
type GADDAG struct {
	// edgeOffsets delimits each node's edge range; see gaddagData.EdgeOffsets.
	edgeOffsets []uint32
	// edgeLetters holds every edge label, sorted within each node; see gaddagData.EdgeLetters.
	edgeLetters []byte
	// edgeTargets holds each edge's destination node; see gaddagData.EdgeTargets.
	edgeTargets []NodeID
	// terminal is the terminal-node bitset; see gaddagData.Terminal.
	terminal []uint64
	// root is the root node id.
	root NodeID
	// nodeCount is one past the highest node id; valid ids are [0, nodeCount).
	nodeCount uint32
	// wordCount is the number of distinct words in the graph.
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
	// Validate the CSR arrays so Successor can index them without bounds checks or panics
	// on a malformed asset: offsets must have one entry per node plus a sentinel, the two
	// parallel edge arrays must match, and offsets must be monotonic and within bounds.
	if int(wd.NodeCount)+1 != len(wd.EdgeOffsets) {
		return nil, fmt.Errorf("dictionary.Load: corrupt GADDAG: edgeOffsets length %d, want NodeCount+1 = %d",
			len(wd.EdgeOffsets), int(wd.NodeCount)+1)
	}
	if len(wd.EdgeLetters) != len(wd.EdgeTargets) {
		return nil, fmt.Errorf("dictionary.Load: corrupt GADDAG: edgeLetters/edgeTargets length mismatch (%d vs %d)",
			len(wd.EdgeLetters), len(wd.EdgeTargets))
	}
	maxEdge := uint32(len(wd.EdgeLetters))
	prev := uint32(0)
	for i, off := range wd.EdgeOffsets {
		if off < prev || off > maxEdge {
			return nil, fmt.Errorf("dictionary.Load: corrupt GADDAG: edgeOffsets[%d]=%d out of order or out of range [0,%d]",
				i, off, maxEdge)
		}
		prev = off
	}
	return &GADDAG{
		edgeOffsets: wd.EdgeOffsets,
		edgeLetters: wd.EdgeLetters,
		edgeTargets: wd.EdgeTargets,
		terminal:    wd.Terminal,
		root:        wd.Root,
		nodeCount:   wd.NodeCount,
		wordCount:   wd.WordCount,
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
	if uint32(node) >= g.nodeCount {
		return 0, false
	}
	// Binary-search the node's edge range, whose labels are ascending (see gaddagData).
	// loadGADDAG has validated that these offsets are in range, so the slice indexing is
	// always in bounds.
	lo, hi := g.edgeOffsets[node], g.edgeOffsets[node+1]
	for lo < hi {
		mid := lo + (hi-lo)/2
		switch c := g.edgeLetters[mid]; {
		case c < letter:
			lo = mid + 1
		case c > letter:
			hi = mid
		default:
			return g.edgeTargets[mid], true
		}
	}
	return 0, false
}

// IsTerminal reports whether node is a terminal node, i.e. a valid word ends here.
func (g *GADDAG) IsTerminal(node NodeID) bool {
	w := uint32(node) >> 6
	if w >= uint32(len(g.terminal)) {
		return false
	}
	return g.terminal[w]&(1<<(uint32(node)&63)) != 0
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

	// Flatten the map-based graph into the compressed-sparse-row layout that GADDAG stores
	// and gaddagData serialises. Valid node ids are [0, nodeCount); id 0 is unused (the
	// builder assigns root=1 and counts up). Emitting nodes in id order, with each node's
	// edges sorted by letter, makes the output fully deterministic.
	nodeCount := uint32(nextID)

	totalEdges := 0
	for _, m := range edges {
		totalEdges += len(m)
	}

	edgeOffsets := make([]uint32, nodeCount+1)
	edgeLetters := make([]byte, 0, totalEdges)
	edgeTargets := make([]NodeID, 0, totalEdges)
	termBits := make([]uint64, (nodeCount+63)/64)
	scratch := make([]byte, 0, 27) // A-Z plus ArcSep: the most edges any node can have

	for id := NodeID(0); uint32(id) < nodeCount; id++ {
		edgeOffsets[id] = uint32(len(edgeLetters))
		if terminals[id] {
			termBits[uint32(id)>>6] |= 1 << (uint32(id) & 63)
		}
		m := edges[id]
		if len(m) == 0 {
			continue
		}
		scratch = scratch[:0]
		for b := range m {
			scratch = append(scratch, b)
		}
		sort.Slice(scratch, func(i, j int) bool { return scratch[i] < scratch[j] })
		for _, b := range scratch {
			edgeLetters = append(edgeLetters, b)
			edgeTargets = append(edgeTargets, m[b])
		}
	}
	edgeOffsets[nodeCount] = uint32(len(edgeLetters))

	wd := gaddagData{
		EdgeOffsets: edgeOffsets,
		EdgeLetters: edgeLetters,
		EdgeTargets: edgeTargets,
		Terminal:    termBits,
		Root:        root,
		NodeCount:   nodeCount,
		WordCount:   uint32(len(words)), // words is already sorted+deduped at this point
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
