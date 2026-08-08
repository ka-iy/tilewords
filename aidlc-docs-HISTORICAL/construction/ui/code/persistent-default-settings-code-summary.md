# Code Summary — Persistent Default Setup Settings (FR-15)

**Unit**: `ui` · **Plan**: `construction/plans/ui-persistent-default-settings-code-generation-plan.md`

## Files

### Created
- **`ui/settings.go`** — the settings subsystem:
  - `GameSettings` — persisted setup choices (`Dict`, `Mode`, `Difficulty`, `Notation`),
    serialised as one JSON document (extensible: new field ⇒ auto-persisted).
  - `defaultGameSettings(avail)` — built-in defaults (first available dict, Classic, 5, off).
  - `sanitize` / `dictInList` — load-time validation (BR-1..BR-3).
  - `encode` / `decode` — JSON marshal / safe unmarshal (empty or malformed ⇒ defaults, BR-4;
    never errors, never panics).
  - `settingsStore` over `fyne.Preferences` (single key `defaultGameSettings`) with
    `load` and best-effort `save` (BR-5).
  - `(*App).defaultsFor(avail)` — store-or-built-in defaults, nil-safe for headless tests.
- **`ui/settings_test.go`** — unit tests (defaults, `sanitize` BR-1..BR-3, `decode` BR-4,
  store round-trip via `test.NewApp()` preferences) + property-based tests (`pgregory.net/rapid`):
  round-trip `decode(encode(s))==s`, and load-robustness (arbitrary string ⇒ no panic + valid).
- **`ui/setup_settings_test.go`** — `defaultsFor` nil-guard, App-level save→load round-trip,
  and a load-mapping + no-panic build test for `buildSetup`.

### Modified
- **`ui/app.go`** — added `settings *settingsStore` to `App`, constructed in `Run()` from
  `fapp.Preferences()`.
- **`ui/setup.go`** — `buildSetup` now:
  - loads defaults (`gs := a.defaultsFor(avail)`) and initialises the dictionary radio, mode
    radio, difficulty slider, and notation check from them;
  - adds a "Save these as my defaults" `touchCheck`, checked by default (not itself persisted);
  - on Start Game, when that box is checked, persists the current selections via
    `a.settings.save(...)` before starting.

## Requirement Coverage
- FR-15.1 checkbox (checked by default) · FR-15.2 setting set (dict/mode/difficulty/notation)
  · FR-15.3 save on Start · FR-15.4 load on setup build (restart + Main Menu → New Game) ·
  FR-15.5 single-document extensibility · FR-15.6 validation of untrusted persisted input.

## Extension Compliance
- **PBT** — round-trip + load-robustness properties pass (100 cases each).
- **Security** — all persisted values validated at the decode boundary; local non-secret data.
- **Resiliency** — graceful fallback to defaults on read failure; best-effort write.

## Verification (at generation time; formal run in Build & Test)
- `go build ./...` clean · `go vet ./ui/` clean · `gofmt -l` clean · full `ui` suite green.

## Notes
- No changes to the game save slot (`savegame.gob`), `engine`, or any public API. Settings
  live in Fyne `Preferences`, independent of the single-slot game save.
