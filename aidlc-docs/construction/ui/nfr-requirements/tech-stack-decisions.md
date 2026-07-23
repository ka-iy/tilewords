# Tech Stack Decisions — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## Language & Framework

| Decision | Choice | Rationale |
|---|---|---|
| Implementation language | Go 1.22+ | Project-wide requirement |
| Game loop framework | `github.com/hajimehoshi/ebiten/v2` | FR-01; cross-platform (desktop + Android); pure Go |
| Text rendering | `golang.org/x/image/font/basicfont` + `golang.org/x/image/font` | Zero external dependency; embedded bitmap font; adequate for tile letters and labels |
| Vector drawing | `github.com/hajimehoshi/ebiten/v2/vector` | Ebitengine built-in; fills and strokes for board, tiles, buttons |
| Debug text | `github.com/hajimehoshi/ebiten/v2/ebitenutil` | `DebugPrint` for status bar and score panel in v1 |

## Rendering

| Decision | Choice | Rationale |
|---|---|---|
| Rendering model | Programmatic (no external image assets) | Avoids Hasbro trade dress; simpler v1; no asset pipeline |
| Logical resolution | 960×640 | Q2 answer; landscape; Ebitengine auto-scales to physical display |
| Cell size | 32 px (480 / 15) | Board occupies 480×480 within 960×640 canvas |
| Tile size (rack) | 44 px | Meets NFR-UI-U6 touch target minimum; fits 7 tiles in right panel |
| Board origin | (10, 10) | 10 px margin from top-left; leaves room for score panel above |
| Draw options reuse | Pre-allocated `*ebiten.DrawImageOptions` | NFR-UI-M2: zero per-frame heap allocations |

## Persistence

| Decision | Choice | Rationale |
|---|---|---|
| Save format | `encoding/gob` | Go-native; compact; already used by engine types |
| Save path | `os.UserConfigDir()/squabble/savegame.gob` | Platform-standard app data directory; SECURITY-UI-1 |
| Write strategy | Atomic (temp file + `os.Rename`) | NFR-UI-R3: prevents torn saves on crash |
| File permissions | 0600 (file), 0700 (directory) | SECURITY-UI-1 |

## Concurrency

| Decision | Choice | Rationale |
|---|---|---|
| AI computation | `ai.AIWorker` (Unit 3 goroutine + buffered channels) | Inherited; non-blocking Poll in Update |
| AI timeout | 10-second deadline checked in Update | NFR-UI-R4: prevents indefinite hang |
| UI goroutine | Main goroutine only (Ebitengine requirement) | All ebiten calls must be on main goroutine |

## Dependencies

| Package | Version | Purpose |
|---|---|---|
| `github.com/hajimehoshi/ebiten/v2` | v2.7+ | Game loop, input, rendering |
| `github.com/hajimehoshi/ebiten/v2/vector` | (included in ebiten) | Shape drawing |
| `github.com/hajimehoshi/ebiten/v2/ebitenutil` | (included in ebiten) | DebugPrint for text |
| `golang.org/x/image` | latest | basicfont for tile letter rendering |
| `squabble/engine` | internal | GameState, commands, rules |
| `squabble/ai` | internal | AIWorker, ChooseMove |
| `squabble/dictionary` | internal | Dictionary, Load |
| `encoding/gob`, `os`, `path/filepath`, `fmt`, `time`, `image`, `image/color` | stdlib | Persistence, path, colour |

## Testing

| Decision | Choice | Rationale |
|---|---|---|
| Unit test framework | `testing` (stdlib) | Standard Go |
| Property-based testing | `pgregory.net/rapid` | Project-wide selection |
| Ebitengine isolation | Non-rendering logic in plain functions; test without display | NFR-UI-TEST-5: headless testability |
| SaveManager test root | `os.TempDir()` injected at construction | NFR-UI-TEST-1: no real config dir in tests |
| Race detection | `go test -race ./ui/...` | No shared mutable state expected, but verified |
