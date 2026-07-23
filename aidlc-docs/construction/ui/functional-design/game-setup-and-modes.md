# Functional Design Addendum — Game Setup & Mode Selection (`ui`)

> Retroactively documented. The New Game Setup screen gained a game-mode chooser (with a
> per-mode preview dialog) and a move-history-notation toggle, alongside the existing
> dictionary and difficulty selectors.

## Setup Screen (`ui/setup.go`)

The New Game Setup screen collects, then constructs a game from:
- **Dictionary** — radio choice among the bundled word lists (each shown with its word count).
- **Game Mode** — radio choice: **Classic** / **Interesting** (see `engine` game-modes
  addendum), each with an **Info** button opening the mode preview dialog.
- **Difficulty** — a 1–10 slider (AI level).
- **Show move history in Scrabble notation** — a checkbox setting the persisted
  `GameState.ScrabbleNotation` preference (plain word list when off).
- **Start Game** — builds the `GameState` for the selected dictionary, mode, difficulty, and
  notation preference, then shows the game screen.

## Mode Preview Dialog (`ui/modeinfo.go`)

`showModeInfo(mode)` pops a scrollable dialog titled "<Mode> Mode" describing the mode so the
player can choose informed:
- **Board layout** — a compact coloured 15×15 grid preview of the mode's premium squares
  (`boardPreview`), reusing the board's own colours/labels; the 15-wide grid gets its own
  horizontal scroll so it is not clipped on a phone.
- A short note for Interesting mode (pinwheel premiums; a distinct tile economy).
- **Premium legend** — the multiplier colours.
- **Tile economy** — the mode's letter counts and per-tile point values (`economyView`).

## Business Rules
- **BR-UI-SET1**: Mode, dictionary, difficulty, and notation preference are chosen once at
  start and passed into the constructed `GameState`.
- **BR-UI-SET2**: The mode preview is read-only information; it never mutates game state.
- **BR-UI-SET3**: The board preview and economy view are derived from the same engine
  mode tables the game uses, so the preview always matches actual play.
