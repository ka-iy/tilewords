# Domain Entities — Unit 1: `dictionary`

> **Correction addendum** — The shipped dictionaries are **enable**, **wordnik**, and
> **atebits-letterpress** (public-domain / open lists), not the placeholder
> `csw/sowpods/ospd/naspa/otcwl/all` names used below, and there is no combined "all" GADDAG.
> The project name is **TileWords**. See `aidlc-docs/corrections.md`.

## Entity: `NodeID`

**Type**: `uint32`
**Description**: Opaque identifier for a node within a GADDAG graph. Value 0 is reserved as the null/invalid node. Value 1 is always the root node.
**Constraints**:
- Must not be interpreted as an index or memory address
- Assigned sequentially by the build tool during GADDAG construction
- Stable across serialise/deserialise round-trips (gob preserves values)

---

## Entity: `GADDAG`

**Description**: A Directed Acyclic Word Graph variant encoding all hook positions of every word in a dictionary. Enables efficient enumeration of all words placeable through any board position given a set of rack letters.

**Internal structure**:
```
GADDAG {
    edges     map[NodeID]map[byte]NodeID   // outgoing edges: node → letter → successor
    terminals map[NodeID]bool              // nodes that mark end of a valid word path
    root      NodeID                       // always 1
    nodeCount uint32                       // total nodes allocated
}
```

**Edge alphabet**: 27 valid byte values:
- Letters `A` (0x41) through `Z` (0x5A) — the 26 English letters
- Arc-separator `+` (0x2B) — marks the boundary between reversed prefix and forward suffix during traversal

**Encoding** (Appel-Jacobson 1998): For a word w = w₁w₂…wₙ, the GADDAG contains n strings:
- For k = 1 to n−1: `wₖwₖ₋₁…w₁ + wₖ₊₁…wₙ` (reversed prefix + separator + forward suffix)
- For k = n: `wₙwₙ₋₁…w₁` (full reverse, no separator; final node marked terminal)
- The node reached after traversing the separator in each string is also marked terminal (it represents a complete left-extension anchor)

**Lifecycle**: Constructed once by the build tool offline; loaded read-only at runtime via `Load`.

---

## Entity: `Dictionary`

**Description**: A named, loaded dictionary backed by a `GADDAG`. Provides the public API for word validation and GADDAG traversal.

**Attributes**:
```
Dictionary {
    name      DictName
    gaddag    *GADDAG
    wordCount int      // stored in gob; avoids recount at load time
}
```

---

## Entity: `DictName`

**Type**: `string` (typed constant)
**Valid values**: `"csw"`, `"sowpods"`, `"ospd"`, `"naspa"`, `"otcwl"`, `"all"`
**Description**: Identifies a dictionary by its canonical short name. Used to select the `.gob` asset file and for display in the UI.

---

## Entity relationships

```
DictName  ----selects----> Dictionary
                                |
                           owns (1:1)
                                |
                             GADDAG
                           /       \
                  has-many           has-many
                  edges              terminals
               (NodeID→letter→NodeID)  (NodeID→bool)
```
