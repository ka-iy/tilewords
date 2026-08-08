# Tech Stack Decisions — Unit 1: `dictionary`

## Language
| Decision | Choice | Rationale |
|---|---|---|
| Implementation language | Go 1.22+ | Project-wide requirement; cross-platform compilation |

## Serialisation
| Decision | Choice | Rationale |
|---|---|---|
| GADDAG wire format | `encoding/gob` | stdlib; no external dependency; compact binary; integrates with `//go:embed` |
| Asset embedding | `//go:embed assets/dictionaries/*.gob` | Stdlib; single-binary deployment; no runtime file I/O |

## Data Structures
| Decision | Choice | Rationale |
|---|---|---|
| Edge storage per node | `map[byte]NodeID` | Idiomatic Go; flexible; sufficient for 27-symbol alphabet; small alphabet means maps stay small (avg ~5-10 edges per node) |
| Terminal flag storage | `map[NodeID]bool` | Same map pattern; space-proportional to terminal node count |
| NodeID type | `uint32` | Sufficient for graphs up to 4B nodes; all known Scrabble-dictionary GADDAGs have <2M nodes |

## Build Tool
| Decision | Choice | Rationale |
|---|---|---|
| Build trigger | `go generate ./dictionary/...` | Standard Go toolchain convention |
| Word list format | One word per line, UTF-8 text | Simple; compatible with all known public word list distributions |
| Output format | gob-encoded struct | Matches runtime deserialisation; no separate parsing step |

## Testing
| Decision | Choice | Rationale |
|---|---|---|
| Unit test framework | `testing` (stdlib) | Standard Go; no additional dependency |
| Property-based testing | `pgregory.net/rapid` | Selected in NFR-07; supports custom generators, shrinking, seed reproducibility; idiomatic Go |
| Race detection | `go test -race ./dictionary/...` | Required by SECURITY-10 and unit completion criteria |

## Logging (Build Tool Only)
| Decision | Choice | Rationale |
|---|---|---|
| Warning output | `fmt.Fprintf(os.Stderr, ...)` | CLI tool; structured logging unnecessary; direct stderr write for simplicity |

## Dependencies
| Package | Version | Purpose |
|---|---|---|
| `pgregory.net/rapid` | latest stable (pinned in go.sum) | Property-based testing |
| All other dependencies | stdlib only | No external runtime dependencies for this unit |
