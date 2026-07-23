# Functional Design — Persistent Default Setup Settings (FR-15)

**Unit**: `ui`
**Implements**: FR-15 (see `inception/requirements/feature-persistent-default-settings.md`)
**Clarifying questions**: none — design decisions D1–D4 were settled and approved at the
Requirements gate; no ambiguities remained, so the functional-design question round was
skipped.

---

## 1. Domain Entities

### 1.1 `GameSettings` (value object)
The complete set of persisted setup choices. Serialised as **one JSON document**, which is
what makes the feature extensible (adding a field extends the document automatically).

| Field | Type | Meaning | Valid range |
|---|---|---|---|
| `Dict` | `dictionary.DictName` (string) | chosen word list | must be an available (bundled) dictionary |
| `Mode` | `engine.GameMode` (int enum) | board layout + tile economy | `ClassicMode` or `InterestingMode` |
| `Difficulty` | `int` | AI difficulty | 1–10 inclusive |
| `Notation` | `bool` | show move history in Scrabble notation | true/false |

- The struct maps 1:1 to the setup screen's controls (dictionary radio, mode radio,
  difficulty slider, notation check).
- **Extensibility rule**: a future setup option is added by (a) adding one field to
  `GameSettings` and (b) wiring its control's read-on-save / apply-on-load. The
  encode/decode and store code below need **no** per-field change.

### 1.2 Built-in defaults
`defaultGameSettings(avail)` returns the values the screen shows today when nothing is saved:
- `Dict` = first available dictionary (`avail[0]`),
- `Mode` = `ClassicMode`,
- `Difficulty` = 5,
- `Notation` = false.

---

## 2. Business-Logic Model

### 2.1 Pure serialisation/validation functions (technology-agnostic, headless-testable)
- `encode(gs GameSettings) (string, error)` — JSON-marshal the document.
- `decode(raw string, avail []dictionary.DictName) GameSettings` — parse then **sanitise**;
  returns fully-valid settings for **any** input string. On empty or malformed input it
  returns `defaultGameSettings(avail)`. Never returns an error and never panics.
- `sanitize(gs GameSettings, avail []dictionary.DictName) GameSettings` — coerce each field
  into range (see Business Rules), substituting the built-in default for invalid values.

These pure functions carry the PBT properties (round-trip, load-robustness) and need no Fyne
app or preferences to test.

### 2.2 Settings store (thin adapter over `fyne.Preferences`)
- `settingsStore` wraps a `fyne.Preferences` (obtained from `App.fapp.Preferences()`), keyed
  by a single string constant (e.g. `"defaultGameSettings"`).
- `store.load(avail) GameSettings` = `decode(prefs.StringWithFallback(key, ""), avail)`.
- `store.save(gs GameSettings)` = `encode(gs)` then `prefs.SetString(key, ...)`; **best
  effort** — an encode/write failure is swallowed (logged), never surfaced to the player.
- The store depends only on the `fyne.Preferences` interface, so tests can inject a fake.

### 2.3 Control flow

**Load (setup screen build — `App.buildSetup`)**
1. Compute `avail := availableDicts()`.
2. `gs := a.settings.load(avail)`.
3. Initialise the working selections and controls from `gs`:
   - dictionary radio → `SetSelected(dictDisplayName(gs.Dict))`,
   - mode radio → `SetSelected` the label for `gs.Mode`,
   - difficulty slider → `SetValue(gs.Difficulty)` (label follows via its OnChanged),
   - notation check → `Checked = gs.Notation`.
4. Add the "Save these as my defaults" check, **checked by default** (its state is not
   persisted — decision D4).

**Save (Start Game handler)**
1. On Start Game, if the save-defaults check is checked, build
   `GameSettings{Dict: selectedDict, Mode: selectedMode, Difficulty: level, Notation: notationCheck.Checked}`
   and call `a.settings.save(gs)`.
2. Proceed with `startNewGame(...)` exactly as today. Saving is independent of, and never
   blocks, starting.

Because `buildSetup` runs every time the setup screen is shown (app start, or Main Menu →
New Game), loading defaults there covers both the restart and the return-to-menu cases (FR-15.4).

---

## 3. Business Rules (validation on load — untrusted input)

- **BR-1 (dictionary)**: if `Dict` is not an available/bundled dictionary → use `avail[0]`.
- **BR-2 (difficulty)**: if `Difficulty` is not in 1–10 → use the built-in default (5).
- **BR-3 (mode)**: if `Mode` is neither `ClassicMode` nor `InterestingMode` → use `ClassicMode`.
- **BR-4 (document)**: empty, malformed, or unparseable JSON → return the full built-in
  defaults; loading never errors the screen or crashes.
- **BR-5 (save)**: persisting defaults is best-effort; a write failure must not block Start
  Game nor show an error.
- **BR-6 (no available dictionaries)**: if `avail` is empty (no bundled dictionaries), load
  returns zero-value settings and the existing "No dictionaries are available" guard in the
  setup screen still applies; nothing is persisted.

---

## 4. Frontend Components (setup screen delta)

- **New control**: `saveDefaultsCheck` — a `touchCheck` ("Save these as my defaults"),
  `Checked = true` at build, placed in the form (below the notation check, above the
  Back/Start row).
- **Changed control initialisation**: dictionary radio, mode radio, difficulty slider and
  notation check are initialised from the loaded `GameSettings` instead of hard-coded
  literals.
- **Changed interaction**: the Start Game handler additionally persists settings when
  `saveDefaultsCheck.Checked`.
- **State ownership**: the `settingsStore` lives on `App` (constructed in `Run()` from
  `fapp.Preferences()`), so `buildSetup` and the Start handler share one store.
- No change to any other screen, to the game save slot (`savegame.gob`), or to `engine`.

---

## 5. NFR / Extension Compliance (folded in per the execution plan)

- **Extensibility (FR-15.5)**: single JSON document + struct; adding a field needs no
  store/serialise change. **Compliant by design.**
- **Security Baseline (enabled)**: all persisted values are validated at the decode boundary
  (BR-1..BR-4); data is local and non-secret. **Applicable — enforced.**
- **Property-Based Testing (enabled)**: properties to implement at Code Generation —
  - *round-trip*: `decode(encode(s), avail) == s` for any valid `s` (dict drawn from `avail`,
    difficulty 1–10, mode ∈ {Classic, Interesting}).
  - *load-robustness*: for an arbitrary string, `decode` never panics and returns settings
    satisfying BR-1..BR-3.
  **Applicable — enforced.**
- **Resiliency Baseline**: graceful degradation to defaults on read failure; best-effort
  write (BR-4/BR-5). Otherwise **N/A** (no distributed/cloud components).
