# Implementation Corrections & Divergences

> **Authoritative correction record.** The original v1 AIDLC docs (inception + the four
> initial construction units) were written up front and, in places, describe choices that
> changed during implementation. Those docs are preserved as the historical record; **this
> file is the authoritative source of truth wherever it contradicts them.** Any reference in
> the historical docs to the items in the left column below should be read as the right column.

## Systematic corrections (apply repo-wide)

| Historical docs say | Actual implementation | Notes |
|---|---|---|
| **Project name: "Squabble"** | **"TileWords"** | Go module is `tilewords`; app ID `fyi.tilewords.game`; save dir `<UserConfigDir>/tilewords/`. |
| **UI toolkit: Ebitengine (`ebiten` v2)** | **Fyne v2** (`fyne.io/fyne/v2`) | See "UI architecture" below — this is more than a rename. |
| **Entry point: `cmd/squabble/`** | **`cmd/tilewords/`** | |
| **Dictionaries: `csw`, `sowpods`, `ospd`, `naspa`, `otcwl`, `all`** | **`enable`, `wordnik`, `atebits-letterpress`** | Only public-domain / open word lists are shipped. Collins (CSW) / NASPA (NWL/OSPD) are copyrighted and deliberately **not** bundled. There is no combined/"all" GADDAG. |
| **`DictName` constants `DictCSW…DictAll`** | **`DictENABLE`, `DictWordnik`, `DictAtebits`** | `dictionary/names.go`; each maps to `wordlists/<name>.txt` → `<name>.gob`. |
| **Save path `…/squabble/`** | **`…/tilewords/savegame.gob`** | Single atomic slot (see `construction/ui/functional-design/save-and-resume.md`). |

## UI architecture correction (Ebitengine → Fyne)

The `ui` unit was implemented on **Fyne**, not Ebitengine. This is an architectural change, so
the original `ui` construction docs (game loop, screens, renderers, input) describe a design
that was not built. Key differences:

| Historical design (Ebitengine) | Actual (Fyne) |
|---|---|
| `ebiten.Game` with `Update`/`Draw`/`Layout`; fixed 960×640 logical resolution | Fyne widget tree; **responsive** layout that adapts to phone vs desktop (`ui/responsive.go`) |
| Screens as structs implementing a `Screen` interface, swapped in the game loop | Screens built by `App` and installed via `win.SetContent` (`ui/mainmenu.go`, `ui/setup.go`, `ui/game.go`) |
| Programmatic pixel rendering (`BoardRenderer`, `RackRenderer`, basicfont, `vector`) | Fyne widgets/canvas (`ui/board_widget.go`, `ui/rack_widget.go`, `ui/tile.go`, theme fonts) |
| Ebiten mouse/touch polling in `Update()`; `AIWorker` channel poll in `Update()` | Fyne tap/drag event handlers; AI runs on a goroutine and marshals results back with `fyne.Do` |
| Mobile via `ebitenmobile` + `AndroidManifest.xml` | Mobile via `fyne package`/`fyne release`; `FyneApp.toml`; `make android-*` targets |

The **actual, current** UI design is documented in the post-v1 addenda under
`construction/ui/functional-design/` (move-history-and-definitions, game-setup-and-modes,
save-and-resume) and in the source under `ui/`.

## What is unchanged / correctly documented

- The **engine** (Appel-Jacobson GADDAG, rules, scoring, command/undo), **ai** (move
  generator, difficulty model, worker), and **dictionary** GADDAG design are as documented
  (aside from the dictionary-name correction above and the post-v1 engine additions).
- The command/undo pattern, gob save format, `//go:embed` asset strategy, PBT and security
  extension enforcement, and the Hasbro-trademark constraint (NFR-09) all hold as written.

## Related

- New/added work is documented separately (not a correction): see the post-v1 additions in
  `aidlc-state.md`, Unit 6 `defs` under `construction/defs/`, and the engine/ui addenda.
