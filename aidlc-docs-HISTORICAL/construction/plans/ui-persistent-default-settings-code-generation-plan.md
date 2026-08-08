# Code Generation Plan — Persistent Default Setup Settings (FR-15)

**Unit**: `ui`
**Implements**: FR-15 — see `construction/ui/functional-design/persistent-default-settings.md`
**Project type**: Brownfield (Go + Fyne). Workspace root: repo root. Application code in
`ui/`; documentation summaries in `aidlc-docs/construction/ui/code/`.
**Single source of truth**: this plan governs generation; steps are executed in order and
checked off as completed.

## Unit Context
- **Story/requirement traceability**: FR-15 (FR-15.1 checkbox, FR-15.2 setting set, FR-15.3
  save trigger, FR-15.4 load, FR-15.5 extensibility, FR-15.6 validation).
- **Dependencies**: `dictionary` (available-dict validation), `engine` (`GameMode`),
  `fyne.Preferences` (persistence). No new external dependencies.
- **Interfaces/contracts**: no public API change; internal `ui` only. The game save slot
  (`savegame.gob`) is untouched.
- **Automation-friendly UI note**: N/A — this is a native Fyne desktop/mobile app, not web;
  `data-testid` attributes do not apply. Testability is provided via pure functions + an
  injectable `fyne.Preferences`.

---

## Steps

### Step 1 — Business Logic: settings model + store  `[x]`
- **File (create)**: `ui/settings.go`
- Contents:
  - `GameSettings` struct with JSON tags: `Dict dictionary.DictName`, `Mode engine.GameMode`,
    `Difficulty int`, `Notation bool`.
  - `settingsPrefKey` const (single preferences key, e.g. `"defaultGameSettings"`).
  - `defaultGameSettings(avail []dictionary.DictName) GameSettings` — first available dict,
    `ClassicMode`, difficulty 5, notation false.
  - `sanitize(gs GameSettings, avail []dictionary.DictName) GameSettings` — BR-1..BR-3
    (dict∈avail else avail[0]; difficulty∈1..10 else 5; mode∈{Classic,Interesting} else Classic).
  - `encode(gs GameSettings) (string, error)` — JSON marshal.
  - `decode(raw string, avail []dictionary.DictName) GameSettings` — empty/malformed → defaults
    (BR-4); else unmarshal then `sanitize`. Never errors, never panics.
  - `settingsStore struct { prefs fyne.Preferences }` + `newSettingsStore(fyne.Preferences)`.
  - `(*settingsStore) load(avail) GameSettings` = `decode(prefs.StringWithFallback(key,""), avail)`.
  - `(*settingsStore) save(gs GameSettings)` — best-effort: `encode` then `prefs.SetString`;
    on encode error, log and return (BR-5).
- **Traceability**: FR-15.2, FR-15.5, FR-15.6.

### Step 2 — Business Logic: unit + property-based tests  `[x]`
- **File (create)**: `ui/settings_test.go`
- Contents:
  - Unit tests: `defaultGameSettings`; `sanitize` for BR-1 (unknown dict), BR-2 (difficulty
    0/11/negative), BR-3 (unknown mode); `decode("")` and `decode("{garbage")` → defaults (BR-4).
  - A fake `fyne.Preferences` (in-memory map) to test `settingsStore.save`→`load` round-trips.
  - **PBT (pgregory.net/rapid)**:
    - `PBT-UI: settings round-trip` — for valid `GameSettings` (dict drawn from a fixed avail
      set, difficulty 1..10, mode∈{Classic,Interesting}), `decode(encode(s), avail) == s`.
    - `PBT-UI: load robustness` — for an arbitrary drawn string, `decode` never panics and the
      result satisfies BR-1..BR-3 (dict∈avail, difficulty∈1..10, mode valid).
- **Traceability**: FR-15.5, FR-15.6; Security (validation), PBT extension.

### Step 3 — App wiring  `[x]`
- **File (modify)**: `ui/app.go`
- Add `settings *settingsStore` field to `App`; construct it in `Run()` from
  `fapp.Preferences()` and assign to the `App` value.
- **Traceability**: FR-15.3, FR-15.4.

### Step 4 — Frontend wiring: setup screen  `[x]`
- **File (modify)**: `ui/setup.go` (`buildSetup`)
- Load defaults at build: `gs := defaults` where `gs = a.settings.load(avail)` when
  `a.settings != nil`, else `defaultGameSettings(avail)` (nil-guard for headless construction).
- Initialise controls from `gs`: dict radio `SetSelected(dictDisplayName(gs.Dict))`, mode
  radio `SetSelected(label(gs.Mode))`, difficulty `level=gs.Difficulty` +
  `levelSlider.SetValue`, `notationCheck.Checked = gs.Notation`.
- Add `saveDefaultsCheck := newTouchCheck("Save these as my defaults", nil)` with
  `SetChecked(true)`, placed below the notation check (inside the scrolled form).
- In the Start Game handler, before/after starting: if `saveDefaultsCheck.Checked && a.settings != nil`,
  `a.settings.save(GameSettings{Dict: selectedDict, Mode: selectedMode, Difficulty: level, Notation: notationCheck.Checked})`.
- **Traceability**: FR-15.1, FR-15.2, FR-15.3, FR-15.4.

### Step 5 — Frontend test: load-path + mapping  `[x]`
- **File (create)**: `ui/setup_settings_test.go`
- Contents:
  - Guard that every available dictionary's `dictDisplayName` is a valid radio label, so a
    loaded `gs.Dict` always maps to a selectable row (load→control mapping is total).
  - Smoke test: build an `App` on `test.NewApp()` with its `settings` seeded from a fake
    preferences holding a known settings JSON; call `a.buildSetup()`; assert it returns
    non-nil content without panicking (exercises the load-on-build wiring).
- **Traceability**: FR-15.4.

### Step 6 — Documentation summary  `[x]`
- **File (create)**: `aidlc-docs/construction/ui/code/persistent-default-settings-code-summary.md`
- Markdown summary of created/modified files, the model, and the test coverage.

---

## Verification (performed in Build & Test stage)
- `go build ./...`, `go vet ./ui/`, `gofmt -l` clean.
- Full `ui` test suite green, including the new unit + rapid property tests.

## Scope
- **Total steps**: 6 (2 create source, 1 modify source, 1 modify source, 2 create test/doc).
- **Files created**: `ui/settings.go`, `ui/settings_test.go`, `ui/setup_settings_test.go`,
  `aidlc-docs/construction/ui/code/persistent-default-settings-code-summary.md`.
- **Files modified**: `ui/app.go`, `ui/setup.go`.
