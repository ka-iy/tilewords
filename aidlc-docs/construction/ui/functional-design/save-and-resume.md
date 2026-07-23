# Functional Design Addendum — Save & Resume (`ui`)

> Retroactively documented to reflect the shipped save feature. The initial design named a
> `SaveManager`; this captures the actual single-slot implementation and the extended
> `GameState` fields it persists.

## SaveManager (`ui/save.go`)

Persists and restores `engine.GameState` to a **single save slot**.
- **Path**: `<os.UserConfigDir>/tilewords/savegame.gob`. `NewSaveManager(configRoot)` allows
  injecting a directory in tests (and using the app's per-app storage root on mobile, where
  `UserConfigDir` is unavailable).
- **`Save`** — atomic write (write to `savegame.gob.tmp`, then `rename`), so a crash mid-write
  never corrupts an existing save. Directory `0700`, file `0600`. The undo commands
  (`LastHumanCommand`/`LastAICommand`) are zeroed before encoding (undo is not persisted).
- **`Load`** — gob-decodes; a defensive guard rejects a decoded state with nil core fields
  (Board/racks/bag) as corrupt.
- **`Exists`** — enables/disables the main-menu Load button.
- **`Delete`** — idempotent removal of the slot (missing file is not an error).
- **`sanitiseError`** — strips function-name prefixes and truncates to 120 chars so internal
  paths/types never reach the UI.

## Persisted State (`engine.GameState`)

The save round-trips the full game: board (with its baked premium layout), racks, bag,
scores, pass counter, turn, move number, dictionary name, AI level, and — added after the
initial design — `Mode` (game mode), `EndgameScored`, `ScrabbleNotation` (notation
preference), `History` (`[]MoveRecord` — the move log, incl. each play's `Words`), and
`OpeningDraw`. All the newer fields are **optional in the gob format**: older saves decode
them as their zero value (Mode→Classic, ScrabbleNotation→false, History→empty), so old saves
still load.

## Main-Menu & Resume Flow (`ui/mainmenu.go`, `ui/app.go`)

- The main menu offers **New Game**, **Load Game** (enabled only when a save exists), and
  **Delete Save**.
- Loading decodes the save and calls the same `App.showGame` path as a new game, so a resumed
  game restores its board/economy (via `Mode`), move history (via `History`, in the persisted
  notation), and — because each play's `Words` are stored — repopulates the Definitions tab.

## Business Rules
- **BR-UI-SAVE1**: Exactly one save slot; `Save` overwrites it atomically.
- **BR-UI-SAVE2**: Undo state is discarded across save/load; restored moves are not undoable.
- **BR-UI-SAVE3**: A corrupt or partial save fails the load with a sanitised message rather
  than crashing or loading a half-initialised game.
- **BR-UI-SAVE4**: The save format is forward/backward tolerant — new optional fields never
  break older saves, and missing fields decode to sensible defaults.
