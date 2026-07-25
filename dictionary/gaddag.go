// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package dictionary is documented in doc.go.
package dictionary

import (
	"bufio"
	"bytes"
	"encoding/binary"
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

// The on-disk asset is a byte stream, written and read field by field in the order below.
// The graph is a compressed-sparse-row (CSR) layout of flat slices, so the stream is those
// slices with two encodings chosen to shrink the asset:
//
//	magic       assetMagic
//	counts      node count, edge count, word count, root id
//	edgeCounts  one varint per node: how many out-edges it has
//	letters     one byte per edge, the edge labels
//	targets     one varint per edge, the destination node id
//	terminal    the terminal bitset, fixed-width little-endian uint64 words
//
// A node's edge *count* is stored rather than the absolute offset the GADDAG uses at
// runtime: counts are at most 27 and so cost one varint byte each, where offsets run to the
// edge total and cost four or five. Offsets are rebuilt by prefix sum while reading, so they
// cannot drift from the arrays they index. Targets are varints because low node ids are
// common; the bitset stays fixed-width, its words being dense rather than small.
//
// Reading is streamed straight into exactly-sized slices, which the GADDAG then holds
// without a post-decode copy: peak memory during load is the graph itself.
const (
	// assetMagic identifies the asset and its layout. Change it whenever the layout
	// changes, so a stale asset is refused with a clear message instead of being
	// misparsed into a nonsensical graph.
	assetMagic = "TWGDDG\x01\n"

	// maxAssetCount is the absolute ceiling on a declared node or edge count, backing up the
	// tighter per-asset bound decodeGADDAG derives from the asset's own length. A corrupt
	// count must not be handed straight to make(), which would abort the process on an
	// absurd allocation; no legitimate asset comes close to this. The type is explicit
	// because an untyped constant this large does not fit the int of a 32-bit build
	// (GOARCH=arm, 386), which made passing it to a ...any parameter a compile error there.
	maxAssetCount uint64 = 1 << 31
)

// GADDAG is the directed acyclic word graph described in Appel & Jacobson (1998), stored
// in a compressed-sparse-row layout (see the asset layout above). It is read-only after Load
// or Build
// returns; all methods are safe for concurrent use.
type GADDAG struct {
	// edgeOffsets has length nodeCount+1. Node id's outgoing edges occupy the index range
	// [edgeOffsets[id], edgeOffsets[id+1]) in edgeLetters/edgeTargets.
	edgeOffsets []uint32
	// edgeLetters is every edge's label byte, grouped by source node and ascending within a
	// node so a node's edges can be binary-searched by letter.
	edgeLetters []byte
	// edgeTargets is the destination node for the edge at the same index in edgeLetters.
	edgeTargets []NodeID
	// terminal is a bitset over node ids: node id is terminal iff bit (id & 63) of
	// terminal[id>>6] is set.
	terminal []uint64
	// root is the root node id.
	root NodeID
	// nodeCount is one past the highest node id; valid ids are [0, nodeCount).
	nodeCount uint32
	// wordCount is the number of distinct words in the graph.
	wordCount uint32
}

// termWords is the number of bitset words needed to hold one terminal bit per node.
func termWords(nodeCount uint32) int { return int((nodeCount + 63) / 64) }

// writeGADDAG serialises a minimized graph in the layout documented above.
func writeGADDAG(w io.Writer, mg minimizedGraph, root NodeID, wordCount uint32) error {
	bw := bufio.NewWriterSize(w, 1<<16)
	scratch := make([]byte, binary.MaxVarintLen64)
	put := func(v uint64) error {
		n := binary.PutUvarint(scratch, v)
		_, err := bw.Write(scratch[:n])
		return err
	}

	err := func() error {
		if _, err := bw.WriteString(assetMagic); err != nil {
			return err
		}
		for _, v := range []uint64{
			uint64(mg.nodeCount), uint64(len(mg.edgeLetters)), uint64(wordCount), uint64(root),
		} {
			if err := put(v); err != nil {
				return err
			}
		}
		for i := 0; i+1 < len(mg.edgeOffsets); i++ {
			if err := put(uint64(mg.edgeOffsets[i+1] - mg.edgeOffsets[i])); err != nil {
				return err
			}
		}
		if _, err := bw.Write(mg.edgeLetters); err != nil {
			return err
		}
		for _, t := range mg.edgeTargets {
			if err := put(uint64(t)); err != nil {
				return err
			}
		}
		for _, word := range mg.terminal {
			binary.LittleEndian.PutUint64(scratch[:8], word)
			if _, err := bw.Write(scratch[:8]); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		return err
	}
	return bw.Flush()
}

// loadGADDAG deserialises a GADDAG from a complete asset held in memory. It is a thin
// wrapper over decodeGADDAG for callers that already have the bytes; Load streams instead,
// to avoid copying the whole asset onto the heap just to decode it.
func loadGADDAG(data []byte) (*GADDAG, error) {
	return decodeGADDAG(bytes.NewReader(data), int64(len(data)))
}

// decodeGADDAG deserialises a GADDAG from r, which must yield the bytes produced by
// tools/buildgaddag or Build. Every count, edge count and target is validated as it is read,
// so a corrupt or truncated asset is reported as an error here rather than indexing out of
// bounds during a traversal, and nothing larger than the graph itself is allocated on the way.
//
// size is the asset's length in bytes. It bounds the declared counts, and is the only thing
// that stops a corrupt count from reaching make() before any of the data that count
// describes has been read. Pass a negative size when the length is genuinely unknown, which
// leaves only the far looser maxAssetCount ceiling.
func decodeGADDAG(r io.Reader, size int64) (*GADDAG, error) {
	br := bufio.NewReaderSize(r, 1<<16)

	magic := make([]byte, len(assetMagic))
	if _, err := io.ReadFull(br, magic); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if string(magic) != assetMagic {
		return nil, fmt.Errorf("not a GADDAG asset of this version; rebuild it with 'make gaddag'")
	}

	readCount := func(what string) (uint64, error) {
		v, err := binary.ReadUvarint(br)
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", what, err)
		}
		if v > maxAssetCount {
			return 0, fmt.Errorf("%s is %d, beyond the %d maximum", what, v, maxAssetCount)
		}
		return v, nil
	}

	nodeCount, err := readCount("node count")
	if err != nil {
		return nil, err
	}
	edgeCount, err := readCount("edge count")
	if err != nil {
		return nil, err
	}
	// Bound both counts by the asset's own length before anything is allocated from them.
	// Each node costs at least one edge-count varint byte, and each edge at least a label
	// byte plus one target varint byte, so an asset of size bytes cannot describe more than
	// size nodes or size/2 edges however corrupt its header is.
	if size >= 0 {
		if nodeCount > uint64(size) {
			return nil, fmt.Errorf("node count is %d, more than a %d-byte asset can describe", nodeCount, size)
		}
		if edgeCount > uint64(size)/2 {
			return nil, fmt.Errorf("edge count is %d, more than a %d-byte asset can describe", edgeCount, size)
		}
	}
	wordCount, err := readCount("word count")
	if err != nil {
		return nil, err
	}
	root, err := readCount("root id")
	if err != nil {
		return nil, err
	}
	if NodeID(root) != RootNodeID {
		return nil, fmt.Errorf("invalid root node %d (want %d)", root, RootNodeID)
	}
	if root >= nodeCount {
		return nil, fmt.Errorf("root node %d addresses node %d of %d", root, root, nodeCount)
	}

	// Rebuild the offsets from the per-node edge counts. The counts must account for
	// exactly the declared edge total, which is what stops a malformed asset from
	// producing an offset past the end of the edge arrays.
	edgeOffsets := make([]uint32, nodeCount+1)
	var sum uint64
	for i := uint64(0); i < nodeCount; i++ {
		n, err := binary.ReadUvarint(br)
		if err != nil {
			return nil, fmt.Errorf("read edge count for node %d: %w", i, err)
		}
		sum += n
		if sum > edgeCount {
			return nil, fmt.Errorf("edge counts overrun the edge total at node %d", i)
		}
		edgeOffsets[i+1] = uint32(sum)
	}
	if sum != edgeCount {
		return nil, fmt.Errorf("edge counts cover %d of %d edges", sum, edgeCount)
	}

	edgeLetters := make([]byte, edgeCount)
	if _, err := io.ReadFull(br, edgeLetters); err != nil {
		return nil, fmt.Errorf("read edge letters: %w", err)
	}
	// Successor binary-searches a node's labels, so they must be strictly ascending within
	// the node. Corruption that leaves the counts and offsets consistent can still reorder
	// labels, and the search would then follow the wrong edge or miss a real one, answering
	// with a different set of words than the asset encodes instead of reporting a problem.
	for i := uint64(0); i < nodeCount; i++ {
		lo, hi := edgeOffsets[i], edgeOffsets[i+1]
		for j := lo + 1; j < hi; j++ {
			if edgeLetters[j] <= edgeLetters[j-1] {
				return nil, fmt.Errorf("node %d has edge labels that are not strictly ascending", i)
			}
		}
	}

	edgeTargets := make([]NodeID, edgeCount)
	for i := range edgeTargets {
		t, err := binary.ReadUvarint(br)
		if err != nil {
			return nil, fmt.Errorf("read target for edge %d: %w", i, err)
		}
		if t >= nodeCount {
			return nil, fmt.Errorf("edge %d targets node %d of %d", i, t, nodeCount)
		}
		edgeTargets[i] = NodeID(t)
	}

	terminal := make([]uint64, termWords(uint32(nodeCount)))
	buf := make([]byte, 8)
	for i := range terminal {
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, fmt.Errorf("read terminal bitset word %d: %w", i, err)
		}
		terminal[i] = binary.LittleEndian.Uint64(buf)
	}

	return &GADDAG{
		edgeOffsets: edgeOffsets,
		edgeLetters: edgeLetters,
		edgeTargets: edgeTargets,
		terminal:    terminal,
		root:        NodeID(root),
		nodeCount:   uint32(nodeCount),
		wordCount:   uint32(wordCount),
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
	// Binary-search the node's edge range, whose labels are ascending (see edgeLetters).
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

// Build constructs a GADDAG from the given word list and writes it to w in the asset layout
// documented at the top of this file.
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

	// The graph built above is a prefix trie: it shares common prefixes but never shares
	// suffixes, so it is far larger than necessary. minimizeTrie collapses all equivalent
	// sub-automata into the minimal acyclic automaton accepting the same language and emits
	// it in the compressed-sparse-row layout that GADDAG stores and writeGADDAG serialises.
	mg := minimizeTrie(edges, terminals, nextID)

	// words is already sorted+deduped at this point.
	if err := writeGADDAG(w, mg, root, uint32(len(words))); err != nil {
		return fmt.Errorf("dictionary.Build: encode failed: %w", err)
	}
	return nil
}

// minimizedGraph is the compressed-sparse-row form of a minimal acyclic automaton,
// matching the fields GADDAG holds (see the asset layout documented above).
type minimizedGraph struct {
	edgeOffsets []uint32
	edgeLetters []byte
	edgeTargets []NodeID
	terminal    []uint64
	nodeCount   uint32
}

// minimizeTrie collapses the prefix trie described by edges/terminals (node ids
// RootNodeID..nextID-1) into the minimal acyclic deterministic automaton accepting the
// same set of GADDAG strings, and returns it in CSR form with the root renumbered to
// RootNodeID.
//
// It is Revuz's linear-time algorithm (Revuz 1992): two trie nodes are equivalent iff they
// share a terminal flag and the same set of (letter -> equivalent child) edges, so merging
// equivalent nodes bottom-up yields the unique minimal automaton. The builder allocates a
// child only when descending into a new edge, so a child's id always exceeds its parent's;
// scanning ids high-to-low therefore canonicalises every child before its parents in a
// single pass.
//
// Output determinism does not depend on Go map iteration order: canonical nodes are
// renumbered by a breadth-first walk from the root that follows edges in ascending letter
// order.
func minimizeTrie(edges map[NodeID]map[byte]NodeID, terminals map[NodeID]bool, nextID NodeID) minimizedGraph {
	// A canonical (deduplicated) node: its terminal flag, its edge labels (ascending), and
	// the canonical indices its edges point to.
	type cnode struct {
		terminal bool
		letters  []byte
		children []uint32
	}
	var canonNodes []cnode
	sigToCanon := make(map[string]uint32) // exact signature bytes -> canonical index
	canon := make([]uint32, nextID)       // trie node id -> canonical index

	scratch := make([]byte, 0, 27) // A-Z plus ArcSep: the most edges any node can have
	var key []byte
	for id := nextID - 1; id >= RootNodeID; id-- {
		m := edges[id]
		scratch = scratch[:0]
		for b := range m {
			scratch = append(scratch, b)
		}
		sort.Slice(scratch, func(i, j int) bool { return scratch[i] < scratch[j] })

		letters := make([]byte, len(scratch))
		copy(letters, scratch)
		children := make([]uint32, len(letters))

		// Signature = terminal byte, then (letter, 4-byte child canonical index) per edge.
		// This is an exact key (no hashing), so distinct signatures never collide.
		key = key[:0]
		if terminals[id] {
			key = append(key, 1)
		} else {
			key = append(key, 0)
		}
		for i, l := range letters {
			c := canon[m[l]]
			children[i] = c
			key = append(key, l, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
		}

		ci, ok := sigToCanon[string(key)]
		if !ok {
			ci = uint32(len(canonNodes))
			canonNodes = append(canonNodes, cnode{terminal: terminals[id], letters: letters, children: children})
			sigToCanon[string(key)] = ci
		}
		canon[id] = ci
	}

	// Renumber canonical nodes to final ids by a BFS from the root, so the root is
	// RootNodeID and the numbering is deterministic. finalID 0 means "not yet assigned"
	// (a valid final id is never 0 — id 0 is left unused, as in the trie).
	rootCanon := canon[RootNodeID]
	finalID := make([]NodeID, len(canonNodes))
	order := make([]uint32, 0, len(canonNodes))
	finalID[rootCanon] = RootNodeID
	order = append(order, rootCanon)
	next := RootNodeID + 1
	for i := 0; i < len(order); i++ {
		for _, child := range canonNodes[order[i]].children {
			if finalID[child] == 0 {
				finalID[child] = next
				next++
				order = append(order, child)
			}
		}
	}

	// Emit CSR in final-id order. Valid ids are [0, nodeCount); id 0 is unused, root is 1.
	nodeCount := uint32(next)
	edgeCount := 0
	for _, cn := range canonNodes {
		edgeCount += len(cn.letters)
	}
	edgeOffsets := make([]uint32, nodeCount+1)
	edgeLetters := make([]byte, 0, edgeCount)
	edgeTargets := make([]NodeID, 0, edgeCount)
	termBits := make([]uint64, (nodeCount+63)/64)
	for f := NodeID(1); uint32(f) < nodeCount; f++ {
		cn := canonNodes[order[f-1]]
		edgeOffsets[f] = uint32(len(edgeLetters))
		if cn.terminal {
			termBits[uint32(f)>>6] |= 1 << (uint32(f) & 63)
		}
		for i, l := range cn.letters {
			edgeLetters = append(edgeLetters, l)
			edgeTargets = append(edgeTargets, finalID[cn.children[i]])
		}
	}
	edgeOffsets[nodeCount] = uint32(len(edgeLetters))

	return minimizedGraph{
		edgeOffsets: edgeOffsets,
		edgeLetters: edgeLetters,
		edgeTargets: edgeTargets,
		terminal:    termBits,
		nodeCount:   nodeCount,
	}
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
