# Code Generation Plan — Unit 1: `dictionary`

## Unit Context
- **Workspace root**: `/home/kartik/PROGS/SQUABBLE-Scrabble_Vibe_coded`
- **Package path**: `dictionary/`
- **Build tool path**: `tools/buildgaddag/`
- **Asset path**: `assets/dictionaries/`
- **Stories**: Supporting unit — contributes to US-02, US-08
- **Dependencies**: None (leaf unit)
- **Project type**: Greenfield single-module monolith

## Stories Implemented
- [x] US-02 (Select dictionary) — Loader.Load provides the 6 named dictionaries
- [x] US-08 (Word validation) — Dictionary.Validate enforces no-bluffing rule

---

## Step 1: Project Bootstrap — `go.mod` and `.gitignore`
- [ ] Create `go.mod` at workspace root:
  - Module name: `squabble`
  - Go version: `go 1.22`
  - Dependency: `pgregory.net/rapid` (latest stable) for PBT
- [ ] Create `.gitignore` at workspace root:
  - `assets/dictionaries/` raw word list `.txt` sources (not committed — licensing)
  - Standard Go ignores (`*.exe`, `*.test`, `*.out`, `vendor/`)
- [ ] Create `assets/dictionaries/.gitkeep` so directory is committed empty

## Step 2: Dictionary Names — `dictionary/names.go`
- [ ] Package `dictionary`
- [ ] Package-level GoDoc comment
- [ ] `DictName` type (`string`)
- [ ] Constants: `DictCSW`, `DictSOWPODS`, `DictOSPD`, `DictNASPA`, `DictOTCWL`, `DictAll`
- [ ] `AllDictNames []DictName` — ordered slice of the 5 individual names (used by UI)
- [ ] `DisplayName(DictName) string` — human-readable label for each dict (e.g. "Collins Scrabble Words")

## Step 3: GADDAG Core — `dictionary/gaddag.go`
- [ ] Package doc comment referencing Appel-Jacobson paper
- [ ] Named constants: `ArcSep`, `RootNodeID`, `MinWordLen`, `MaxWordLen` (each with GoDoc)
- [ ] `NodeID` type (`uint32`) with GoDoc
- [ ] `GADDAG` struct (unexported fields: `edges`, `terminals`, `root`, `size`)
- [ ] `Load(data []byte) (*GADDAG, error)` — gob decode + root validation + error wrapping
- [ ] `Root() NodeID`
- [ ] `Successor(node NodeID, letter byte) (NodeID, bool)` — with paper section reference in comment
- [ ] `IsTerminal(node NodeID) bool`
- [ ] `contains(word string) bool` (unexported; called by `Dictionary.Validate` after normalisation)
  - Implements 3-tier fast-path: length check → byte validity → full-reverse GADDAG traversal

## Step 4: Dictionary Wrapper — `dictionary/dictionary.go`
- [ ] `Dictionary` struct (unexported fields: `name`, `gaddag`, `wordCount`)
- [ ] GoDoc on all exported methods
- [ ] `Name() DictName`
- [ ] `WordCount() int`
- [ ] `Validate(word string) bool` — normalises case, delegates to `gaddag.contains`
- [ ] `GADDAG() *GADDAG` — exposes graph for AI traversal

## Step 5: Loader — `dictionary/loader.go`
- [ ] `//go:embed assets/dictionaries/*.gob` directive
- [ ] `embeddedAssets embed.FS` (package-level, unexported)
- [ ] `assetPath(name DictName) string` — maps DictName to file path
- [ ] `Load(names ...DictName) (*Dictionary, error)` — selects asset, calls `GADDAG.Load`, wraps errors with context

## Step 6: Package Doc — `dictionary/doc.go`
- [ ] Package declaration with full multi-line GoDoc comment:
  - Purpose of the package
  - Reference to Appel-Jacobson 1998 paper (full citation)
  - Usage example (load → validate)
  - Note that word lists are not included in source; only pre-built GADDAG assets

## Step 7: Build Tool — `tools/buildgaddag/main.go`
- [ ] Package `main`; standalone CLI
- [ ] Flags: `-input` (comma-separated word list paths), `-output` (path to write .gob), `-name` (dict name string)
- [ ] Input parsing: read lines, `strings.ToUpper`, skip non-A-Z with `fmt.Fprintf(os.Stderr, ...)` warning, skip length < MinWordLen or > MaxWordLen
- [ ] Sort + deduplicate word slice (for combined dict)
- [ ] GADDAG construction function: `buildGADDAG(words []string) *gaddag` — inline package-private struct (build tool has its own unexported `gaddag` type mirroring the structure gob-compatible with `dictionary.GADDAG`)
  - Inner type fields must be exported for gob to encode them correctly (gob only encodes exported fields)
  - Comment each loop body with reference to Appel-Jacobson §3 string-insertion algorithm
- [ ] Atomic write: encode to temp file → `os.Rename` on success
- [ ] Exit 0 on success, exit 1 on any error (with message to stderr)
- [ ] `//go:generate` comment at top of `dictionary/doc.go` documenting how to run the tool

## Step 8: Unit Tests — `dictionary/dictionary_test.go`
- [ ] Example-based tests:
  - `TestLoad_ValidAsset` — loads a real `.gob` fixture, checks no error
  - `TestLoad_CorruptData` — passes garbage bytes, expects descriptive error
  - `TestValidate_KnownWords` — table-driven: known valid and invalid words per dict
  - `TestValidate_CaseInsensitive` — "WORD" == "word" == "Word"
  - `TestValidate_Boundaries` — single-letter (false), 2-letter (depends), 15-letter (depends), 16-letter (false)
  - `TestValidate_NonAlpha` — words with hyphens, digits, spaces → always false
  - `TestSuccessor_ArcSep` — traversal through arc-separator node works correctly
- [ ] PBT tests using `rapid`:
  - `TestPBT_RoundTrip` — serialise GADDAG → deserialise → Contains results identical (PBT-02)
  - `TestPBT_ContainsOracle` — Contains(word from source list) == true; Contains(random non-word) matches brute-force set (PBT-05)
  - `TestPBT_CaseInvariance` — Contains(upper) == Contains(lower) for all alpha strings (PBT-03)
  - `TestPBT_InvalidRejection` — Contains(string with non-A-Z) always false (PBT-03)
  - `TestPBT_DedupInvariant` — WordCount(all) ≤ sum of individual WordCounts (PBT-03)
  - `TestPBT_IdempotentLoad` — Load(bytes) twice → identical Contains results (PBT-04)
- [ ] Test helpers in `dictionary/testhelpers_test.go`:
  - `wordFromDictGen` — rapid generator sampling from embedded fixture word list
  - `randomAlphaGen(minLen, maxLen int)` — rapid generator for A-Z strings
  - `invalidStringGen` — rapid generator with guaranteed non-A-Z byte

## Step 9: Code Documentation Summary
- [ ] Create `aidlc-docs/construction/dictionary/code/code-summary.md`:
  - List all files created with one-line description each
  - Note build tool usage (`go generate ./dictionary/...`)
  - Note test fixture requirement (`.gob` files must exist for integration tests)

---

## File Manifest

| File | Type | Step |
|---|---|---|
| `go.mod` | Project config | 1 |
| `.gitignore` | Project config | 1 |
| `assets/dictionaries/.gitkeep` | Asset placeholder | 1 |
| `dictionary/doc.go` | Go source | 6 |
| `dictionary/names.go` | Go source | 2 |
| `dictionary/gaddag.go` | Go source | 3 |
| `dictionary/dictionary.go` | Go source | 4 |
| `dictionary/loader.go` | Go source | 5 |
| `dictionary/testhelpers_test.go` | Go test | 8 |
| `dictionary/dictionary_test.go` | Go test | 8 |
| `tools/buildgaddag/main.go` | Go source | 7 |
| `aidlc-docs/construction/dictionary/code/code-summary.md` | Documentation | 9 |
