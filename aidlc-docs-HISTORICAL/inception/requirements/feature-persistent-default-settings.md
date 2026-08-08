# Feature Requirements — Persistent Default Setup Settings

**Requirement ID**: FR-15 (see `requirements.md`)
**Increment type**: post-v1 feature addition (brownfield)
**Owning unit**: `ui` (with a small preferences-backed settings store)

---

## Intent Analysis Summary

- **User request**: Add a checkbox to the "New Game Setup" screen that saves the current
  setup choices as default settings; the box is checked by default. On app restart or when
  returning to the setup screen from the main menu, the saved defaults are loaded so the
  player can start with their previous choices in one tap.
- **Clarifications received**:
  - The **AI difficulty level** is included in the saved settings (in addition to word list,
    game mode, and notation toggle).
  - The persistence must be **extensible** so any setup option added in the future is saved
    automatically, without reworking the save/load mechanism.
- **Request type**: New feature / enhancement.
- **Scope estimate**: Single component (`ui`), plus a thin settings-persistence helper.
- **Complexity estimate**: Simple.
- **Request clarity**: Clear and complete (no blocking ambiguities).

---

## Functional Requirements

### FR-15.1 — Save-as-defaults control
- The setup screen shows a "Save these as my defaults" checkbox.
- The checkbox is **checked by default** every time the setup screen opens.

### FR-15.2 — Saved setting set
- The persisted settings model captures the full set of setup choices:
  - dictionary / word list (`dictionary.DictName`),
  - game mode (`engine.GameMode`),
  - AI difficulty level (integer 1–10),
  - move-history Scrabble-notation toggle (bool).

### FR-15.3 — Save trigger
- When the player presses **Start Game** with the checkbox checked, the setup selections in
  use at that moment are persisted as the defaults.
- When the checkbox is unchecked at Start Game, previously-saved defaults are left unchanged
  (the game still starts with the on-screen selections; nothing is persisted).

### FR-15.4 — Load / pre-population
- When the setup screen is built (app restart, or navigating Main Menu → New Game), the saved
  defaults pre-populate every control: dictionary radio, mode radio, difficulty slider, and
  notation checkbox.
- With saved defaults present, the player can press **Start Game** immediately to play with
  their previous choices.
- On first ever run (no saved settings), the screen shows the built-in defaults it shows
  today (first available dictionary, Classic mode, difficulty 5, notation off).

### FR-15.5 — Extensibility
- Settings are saved and restored via a **single settings model** serialised as one document.
- Adding a new setup option in the future requires only: (a) adding a field to the settings
  model, and (b) wiring the corresponding control's read-on-save / apply-on-load. The
  persistence (serialise/deserialise) mechanism itself needs no per-field change.

### FR-15.6 — Validation of persisted input
- On load, persisted values are treated as untrusted:
  - an unknown/unavailable dictionary falls back to the first available dictionary,
  - a difficulty outside 1–10 is clamped/reset to the built-in default,
  - an unknown game mode falls back to Classic,
  - a malformed or unreadable settings document falls back entirely to built-in defaults.
- Loading defaults MUST never error out the setup screen or crash the app.

---

## Non-Functional Requirements

### NFR — Extensibility (see FR-15.5)
- One serialised settings document; no per-field plumbing in the store. This is the primary
  design driver for the feature.

### NFR — Reliability / Graceful degradation
- A missing, empty, or corrupt settings document degrades to built-in defaults silently.
- Persisting defaults is best-effort: a failure to write MUST NOT block starting the game.

### NFR — Security (extension: Security Baseline, enabled)
- Persisted settings are local, non-secret, and untrusted on read. Enforce input validation
  on load (FR-15.6). No secrets are stored. Applicable SECURITY rules: input validation on
  the deserialise boundary; safe handling of absent/garbage data.

### NFR — Testability / Property-Based Testing (extension: PBT, enabled)
- The settings model MUST have a **round-trip property**: `decode(encode(s)) == s` for any
  valid settings value.
- The load path MUST have a **robustness property**: for an arbitrary/garbage stored string,
  load never panics and always yields a valid, in-range settings value.
- Selection round-trips for dictionary and game mode MUST be covered (the value chosen in the
  UI is the value restored).

---

## Design Decisions (committed unless changed at the approval gate)

- **D1 — Storage mechanism**: a single JSON document stored under one Fyne `Preferences`
  string key (via `App.fapp.Preferences()`). Chosen because Preferences is the idiomatic,
  per-app, platform-appropriate settings store, is independent of the single-slot game save
  (`SaveManager` / `savegame.gob`), and a single serialised document gives the required
  extensibility for free.
- **D2 — Save timing**: persist on **Start Game** (only when the box is checked), capturing
  the selections actually used to start — not on every control change.
- **D3 — Unchecked behaviour**: unchecking and starting leaves any previously-saved defaults
  intact; it does not clear them.
- **D4 — Checkbox own state**: the checkbox always opens **checked** (FR-15.1) and is not
  itself persisted. Rationale: it matches "check the box by default" literally and avoids the
  contradiction of trying to remember an "unchecked = don't save" choice. (Alternative, if
  preferred: persist the flag too so an unchecked choice sticks — flagged for the gate.)

---

## Extension Compliance Summary

| Extension | Status | Rationale |
|---|---|---|
| Property-Based Testing | **Applicable** | Round-trip + load-robustness + selection round-trip properties required (see Testability NFR). Enforced in Construction. |
| Security Baseline | **Applicable** | Input validation on the deserialise boundary; local non-secret data (see Security NFR). |
| Resiliency Baseline | **Mostly N/A** | No distributed/cloud components. The relevant slice — graceful degradation to defaults on read failure and best-effort write — is captured under Reliability. |

---

## Traceability

- Canonical requirement: `requirements.md` → FR-15.
- Related existing requirements: FR-03 (dictionaries), FR-05 (difficulty 1–10), FR-13 (game
  modes), FR-14 (notation), NFR-04 (reliability), NFR-08 (security).
- Affected code (from current tree): `ui/setup.go` (controls + Start Game), `ui/app.go`
  (`App.fapp`, screen transitions), a new `ui` settings store (e.g. `ui/settings.go`).
