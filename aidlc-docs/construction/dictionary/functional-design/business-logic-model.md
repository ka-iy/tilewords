# Business Logic Model — Unit 1: `dictionary`

## BL-01: GADDAG Construction (Build Tool — `tools/buildgaddag`)

**Input**: One or more raw word list `.txt` files (one word per line, any case).
**Output**: One `.gob` file per input, plus one combined `.gob` for the `all` dictionary.

### Algorithm

```
1. Read all words from input file(s), normalise each to uppercase.
2. Filter: discard any word containing a byte outside A–Z; log each discard to stderr.
3. Filter: discard any word shorter than 2 or longer than 15 characters.
4. If building the combined dictionary: merge all word slices, sort, deduplicate (remove consecutive duplicates).
5. Sort the word list lexicographically (required for deterministic GADDAG output).
6. For each word w = w₁w₂…wₙ:
   a. For k = 1 to n−1:
      - Build string: wₖwₖ₋₁…w₁ + '+' + wₖ₊₁…wₙ
      - Insert this string into the GADDAG character-by-character, creating nodes as needed.
      - Mark the node reached after '+' as terminal.
   b. For k = n:
      - Build string: wₙwₙ₋₁…w₁ (full reverse, no separator)
      - Insert into GADDAG; mark final node terminal.
7. Serialise the completed GADDAG to gob bytes and write to the output `.gob` file.
```

**Reference**: Appel, A. W. & Jacobson, G. J. (1988). "The World's Fastest Scrabble Program." *CACM 31(5)*, §3 (GADDAG definition) and §4 (construction).

---

## BL-02: GADDAG Deserialisation (`GADDAG.Load`)

**Input**: `[]byte` — gob-encoded GADDAG (from `//go:embed` asset).
**Output**: `*GADDAG` ready for read-only use, or `error`.

### Algorithm
```
1. gob-decode the byte slice into the internal GADDAG struct.
2. Verify root NodeID == 1; return error if malformed.
3. Return the GADDAG pointer.
```
No GADDAG construction occurs at runtime; all graphs are pre-built by the build tool.

---

## BL-03: Word Validation (`GADDAG.Contains`, `Dictionary.Validate`)

**Input**: A string (any case).
**Output**: `bool` — true if and only if the string (uppercased) is in the dictionary.

### Algorithm
```
1. Normalise input: convert to uppercase byte-by-byte; return false immediately if any byte is outside A–Z after normalisation.
2. Traverse the GADDAG from root following the reversed full word path:
   - For word w = w₁…wₙ (after uppercasing), traverse: wₙ, wₙ₋₁, …, w₁
   - At each step call Successor(currentNode, letter); if edge absent, return false.
3. After consuming all letters, return IsTerminal(currentNode).
```

**Rationale**: The full-reverse path (k=n case from BL-01) encodes the complete word as a reversed string ending in a terminal node. This provides O(n) validation without traversing the separator, and without allocating the reversed string (traverse letters in reverse index order).

---

## BL-04: GADDAG Traversal for AI Move Generation (`GADDAG.Successor`, `GADDAG.IsTerminal`, `GADDAG.Root`)

**Purpose**: Expose the raw GADDAG graph to the `ai.Generator` for left-extension move enumeration.

**Contract**:
- `Root() NodeID` — always returns 1
- `Successor(node NodeID, letter byte) (NodeID, bool)` — looks up `edges[node][letter]`; returns (target, true) or (0, false)
- `IsTerminal(node NodeID) bool` — returns `terminals[node]`

The AI generator traverses the GADDAG following the Appel-Jacobson left-extension algorithm, using the arc-separator byte `'+'` to switch from reversed-prefix traversal to forward-suffix traversal.

---

## BL-05: Dictionary Loading (`Loader.Load`)

**Input**: One or more `DictName` values.
**Output**: `*Dictionary` backed by the appropriate GADDAG, or `error`.

### Algorithm
```
1. If exactly one name provided (not "all"):
   - Embed asset path: "assets/dictionaries/{name}.gob"
   - Call GADDAG.Load on the embedded bytes.
   - Wrap in Dictionary{name, gaddag, wordCount}.
2. If "all" (or multiple names) provided:
   - Use the pre-built "assets/dictionaries/all.gob" asset.
   - This GADDAG was built from the union of all 5 word lists, deduplicated offline.
   - Wrap in Dictionary{name:"all", gaddag, wordCount}.
3. Return the Dictionary.
```

---

## Testable Properties (PBT-01)

| Property | Category | Description |
|---|---|---|
| **Round-trip serialisation** | Round-trip | Serialise a GADDAG to gob → deserialise → for every word in original list, Contains returns true; for every known non-word, Contains returns false. Maps to PBT-02. |
| **Deduplication invariant** | Invariant | Word count of the combined "all" dictionary ≤ sum of individual dictionary word counts. Maps to PBT-03. |
| **Contains oracle** | Oracle | For any word w in the source word list, `Contains(w) == true`; for a randomly generated string not in the list, `Contains(str)` matches brute-force list membership. Maps to PBT-05. |
| **Case invariance** | Invariant | `Contains(word.ToUpper()) == Contains(word.ToLower()) == Contains(word)` for all alphabetic inputs. Maps to PBT-03. |
| **Invalid input rejection** | Invariant | `Contains(s) == false` for any string s with non-A-Z characters. Maps to PBT-03. |
| **Word length bounds** | Invariant | No word shorter than 2 or longer than 15 letters returns true from Contains. Maps to PBT-03. |
| **Idempotent Load** | Idempotence | Loading the same `.gob` file twice produces two GADDAGs with identical Contains results for all queries. Maps to PBT-04. |
