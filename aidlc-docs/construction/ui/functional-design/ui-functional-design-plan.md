# Functional Design Plan — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## Pre-Answered Design Decisions

The following decisions are derived from requirements, prior units, and standard Ebitengine
practice. No user input is needed.

| Decision | Choice | Rationale |
|---|---|---|
| Rendering approach | Programmatic (Ebitengine drawing primitives only) | No external image dependency; avoids Hasbro trade dress; simpler for v1 |
| Screen flow | MainMenu → Setup → Game → EndGame (→ MainMenu) | Standard menu flow matching user stories US-01–US-19 |
| Tile placement model | Click-to-select then click-to-place; drag-and-drop as enhancement | NFR-05; touch-friendly for Android |
| Staged tile display | Distinct visual state (semi-transparent / yellow outline) | Player must distinguish placed-but-uncommitted from committed tiles |
| Tile recall | Click a staged tile on the board to return it to the rack | Standard digital board game UX |
| Exchange UI | Click rack tiles to toggle selection; dedicated "Exchange" button | Clear selection model; matches FR-07 |
| AI thinking state | Disable human input; show "Thinking…" text in status bar | Prevents double-action; clear feedback |
| Undo button | Shown during human turn; hidden/disabled once AI has responded | FR-09: undo only available immediately after human move |
| Save/load | Single slot; `os.UserConfigDir()/squabble/savegame.gob` | NFR-08 app-data directory; one slot sufficient for v1 |
| End-game trigger | Engine signals via `IsGameOver`; UI transitions automatically | Engine owns the rule; UI reacts |
| Premium square colours | Orange (DW), Red (TW), Light-blue (DL), Dark-blue (TL), Gold star (Centre) | Open custom palette; not Hasbro's scheme |

---

## Open Questions

### Q1: Blank Tile Letter Assignment

When the human player places a blank tile on the board, the game must ask which letter the
blank should represent. Two interaction models:

**Option A — On-board overlay (recommended)**: When a blank is placed, a translucent overlay
appears showing a 4×7 grid of letter buttons (A–Z + one Cancel button). The player taps the
desired letter. The overlay disappears and the blank tile shows the chosen letter in a smaller
font (with a visual indicator that it is a blank, e.g. a dot or different colour).

**Option B — Keyboard input**: The player types a letter. Works on desktop but is awkward on
Android (requires soft keyboard). Not recommended for mobile.

**Recommended**: Option A — the letter-picker overlay works on both desktop (mouse click)
and Android (touch). It is the industry-standard approach for digital crossword board games.

### Q2: Internal Logical Resolution

Ebitengine's `Layout(w, h int) (int, int)` returns the logical (internal) resolution
independent of the physical window size. This determines the coordinate system for all
rendering. Two options:

**Option A — 960×640 landscape (recommended)**: Gives a 3:2 aspect ratio comfortable on
both desktop monitors and Android landscape orientation. The 15×15 board occupies a ~480×480
square; the remaining horizontal space holds rack, score panel, and control buttons.
On portrait Android, Ebitengine scales automatically.

**Option B — 720×1280 portrait**: Optimised for phone portrait mode. Desktop window is
awkward (tall and narrow). Board would be ~600×600 with controls below.

**Recommended**: Option A — 960×640. Desktop-first layout with Ebitengine auto-scaling
for Android. Matches the landscape layout of most digital board games on tablets.

---

## Summary of Your Input Needed

Please answer both questions (A or B for each):

- **Q1** (blank tile overlay): A or B?
- **Q2** (internal resolution): A or B?
