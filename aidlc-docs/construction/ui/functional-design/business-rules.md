# Business Rules — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## BR-UI-01: Human Input Only During Human Turn

All tile-placement actions, rack interactions, and game-control buttons (Play, Exchange,
Pass, Undo, Save) are disabled while `state.CurrentTurn == AITurn` or while
`blankPicker != nil`. Input events received during these states are silently ignored.

---

## BR-UI-02: Staged Tiles Required to Commit Play

`commitPlay` must reject attempts to commit with `len(staged) == 0`. The Play button is
visually disabled in this state. The rejection message is shown in the status bar.

---

## BR-UI-03: Blank Must Be Assigned Before Commit

A staged blank tile (`Tile.IsBlank == true`) with `AssignedLetter == 0` blocks `commitPlay`.
The Play button is visually disabled until all staged blanks have been assigned letters via
the `BlankPickerOverlay`.

---

## BR-UI-04: Validation Failure Is Non-Destructive

When `engine.ValidatePlacement` rejects staged tiles, the staged tiles remain on the board
and the player is free to adjust them. The error message is shown in the status bar. No
tiles are removed and the turn does not advance.

---

## BR-UI-05: Undo Available Immediately After Human Move Only

`undoReady` is set to `true` only after the AI's response move is applied
(`applyAIMove` completes). It is cleared at the start of the next human action (any of
commitPlay, commitExchange, commitPass, doUndo). This enforces the FR-09 constraint that
undo covers exactly one human+AI round and is not available mid-turn or after multiple turns.

---

## BR-UI-06: Exchange Requires Selection and Sufficient Bag

Exchange is rejected if:
- No rack tiles are selected for exchange.
- `state.Bag.Count() < engine.MaxRackSize` (7).

The Exchange button is visually disabled when the bag has fewer than 7 tiles.

---

## BR-UI-07: AI Rack Toggle — Hidden by Default

The AI rack is hidden (`showAIRack = false`) when a `GameScreen` is first created. A
single toggle button labelled **"Show AI Rack"** / **"Hide AI Rack"** flips `showAIRack`
and its own label. This button is always enabled — available during both human and AI turns.

When `showAIRack == false`: the AI rack area renders 7 face-down tile placeholders
(coloured rectangles, no letters or point values).
When `showAIRack == true`: all AI tile letters and point values are rendered normally,
allowing the player to verify the AI holds only the tiles it plays (FR-08).

`aiRack.interactive` remains `false` regardless of `showAIRack`; the player cannot
interact with the AI rack in either state.

---

## BR-UI-08: Single Save Slot — Overwrite Without Confirmation

`doSave` overwrites `savegame.gob` unconditionally using an atomic write (write to temp
file, rename). No confirmation dialog is shown. The save button is always available during
a game, including immediately after an undo.

---

## BR-UI-09: Load Only From Main Menu

The Load Game button appears on `MainMenuScreen` only. There is no mid-game load. Starting
a new game while a save exists does not delete the save — it is only overwritten when the
player explicitly saves.

---

## BR-UI-10: Load Failure Returns to Main Menu With Error

If `SaveManager.Load` returns an error (corrupt file, missing file, incompatible gob
schema), the `MainMenuScreen` displays the sanitised error message and stays on the main
menu. No partial state is applied.

---

## BR-UI-11: Internal Resolution Is 960×640 (Q2 Answer)

`Game.Layout` always returns `(960, 640)`. All rendering uses this coordinate space.
Ebitengine handles scaling to the actual window/screen size. Board origin is at pixel
(10, 10); board occupies 480×480 (15 cells × 32 px each). Right panel (racks + score +
controls) occupies columns 500–950.

---

## BR-UI-12: Window Title Is "Squabble"

The window title set via `ebiten.SetWindowTitle("Squabble")` contains no trademark-violating
words (NFR-09 / BR-AI-15 equivalent for the ui package). All in-game text uses "Squabble",
"crossword board game", or generic terms.

---

## BR-UI-13: Premium Square Colours (Non-Hasbro Palette)

| Square Type | Fill Colour | Label |
|---|---|---|
| DoubleWord | `#FFB74D` (light orange) | `W×2` |
| TripleWord | `#E65100` (dark orange) | `W×3` |
| DoubleLetter | `#00897B` (teal) | `L×2` |
| TripleLetter | `#00695C` (deep teal) | `L×3` |
| Centre | `#CE93D8` (lavender) with ★ | `★` |
| Normal | `#228B22` (forest green) | — |

This palette is independently chosen and does not reproduce Hasbro's copyrighted board
artwork (NFR-09 / NFR-06): letter premiums use a teal family and word premiums an orange
family (lighter shade = double, darker = triple), with a lavender centre, deliberately
avoiding the classic blue-letter / red-word arrangement. Premium labels are drawn white on
dark-toned squares and dark on light-toned squares (light-orange 2×W, lavender centre) so
they stay legible.

---

## BR-UI-14: Staged Tile Visual Distinction

Staged (uncommitted) tiles are rendered with:
- Background colour: `#FFFACD` (lemon chiffon — slightly lighter than committed tiles).
- Border: 2 px `#DAA520` (goldenrod) outline.
- Committed tiles: background `#DCE0E6` (light blue-grey), 2 px `#A0783C` (brown) border.

This ensures the player can always distinguish staged from committed tiles.

---

## BR-UI-15: AI Action Description in Status Bar

After `applyAIMove`, the status bar shows:
- PlayMove: "AI played [word] for [N] points." (`[word]` is `move.WordsFormed[0]`)
- ExchangeMove: "AI exchanged [N] tile(s)."
- PassMove: "AI passed."

The AI rack is updated on screen immediately when the result arrives.

---

## BR-UI-16: End-Game Screen Content

`EndGameScreen.Draw` must show:
1. Title: "Game Over"
2. Human score and AI score (after end-game adjustments applied by `engine.ApplyEndgameScoring`)
3. Score adjustment line: "+[N] from opponent rack" (if human emptied rack) or "−[N] from remaining tiles"
4. End reason: "Tiles exhausted" or "Six consecutive passes"
5. Winner declaration: "You win!" / "AI wins!" / "It's a tie!"
6. Two buttons: "Play Again" and "Main Menu"

---

## BR-UI-17: No "Scrabble" in Any UI Text

All screen text, button labels, status messages, window title, and error messages must
use "Squabble", "crossword board game", "tile game", or generic alternatives. The word
"Scrabble" must not appear anywhere in the `ui` package source code, comments included,
except in an explicit legal-notice comment if required.

---

## BR-UI-18: Tile Letter Rendering

Each tile (committed or staged) displays:
- Uppercase letter centred in the cell (large font, dark colour).
- Point value in the upper-right corner (bold, inset slightly towards the centre for
  legibility on small mobile cells).
- Blank tiles (on board, assigned) display the assigned letter in a different colour
  (e.g. blue instead of black) to indicate blank status.
- Blank tiles (on board, unassigned — impossible after BR-UI-03, but defensive): display `?`.
