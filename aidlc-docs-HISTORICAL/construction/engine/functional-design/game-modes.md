# Functional Design Addendum — Game Modes (`engine`)

> Retroactively documented. Added after the initial engine design. A **game mode** bundles a
> board premium-square layout with a tile economy (letter distribution + point values),
> chosen once at game start and persisted with the save so a resumed game keeps the same
> board and economy.

## Entity: `GameMode`

**Type**: `int` (typed constant), in `engine/mode.go`.
| Value | Name | Description |
|---|---|---|
| `0` | `ClassicMode` | Standard 15×15 premium-square layout + standard 100-tile English economy. Zero value, so pre-mode saves decode as Classic. |
| `1` | `InterestingMode` | An independently-designed 4-fold-rotational ("pinwheel") premium layout + a 110-tile frequency-derived economy. |

`String()` → `"Classic"` / `"Interesting"`.

## BL: Mode-Parameterised Board (`engine/board.go`)

- A `Board` carries its `mode` and exposes it as the persisted `Mode` field.
- Premium-square layout is **baked into the cells at construction** from the mode's table:
  - `premiumSquares` — the Classic layout.
  - `premiumSquaresInteresting` — the Interesting "pinwheel" layout: 4-fold rotational,
    even coverage (every cell within two steps of a premium), no orthogonally adjacent word
    multipliers, empty corners; distinct from Classic.
- The board is constructed for a mode at game start; scoring reads point values through the
  mode's letter-point table (below).

## BL: Mode-Parameterised Tile Economy (`engine/tile.go`, `engine/bag.go`)

- `distributionForMode(mode)` returns the tile distribution:
  - Classic: `tileDistribution` — 100-tile North American English set.
  - Interesting: `tileDistributionInteresting` — 110 tiles (106 letters + 4 blanks), counts
    derived from letter-occurrence frequencies over the bundled public-domain / open word
    lists.
- `letterPointsForMode(mode)` returns the A–Z point table; Interesting's points
  (`letterPointsInteresting`) are **derived from its distribution** (`deriveLetterPoints`),
  so rarer letters score more, consistent with the mode's own economy.
- `NewBagForMode(rng, mode)` fills a shuffled bag from the mode's distribution (`NewBag`
  remains the Classic convenience wrapper).

## Business Rules

- **BR-MODE-1**: A game's mode is fixed at start; it never changes mid-game.
- **BR-MODE-2**: The mode is persisted (`GameState.Mode`); a resumed game uses the same
  board layout, tile distribution, and point values.
- **BR-MODE-3**: Backward compatibility — a save written before modes existed has no `Mode`
  field and decodes as `ClassicMode`, matching the layout/economy it was played with.
- **BR-MODE-4**: The Interesting layout and economy are independently designed (not copied
  from any trademarked board), consistent with NFR-09.

## Testable Properties

| Property | Description |
|---|---|
| Distribution totals | Classic distribution sums to 100; Interesting to 110 (verified in tests). |
| Points derivation | `letterPointsInteresting` is a pure function of `tileDistributionInteresting`. |
| Mode round-trips | `GameState.Mode` survives save/load; board layout matches the persisted mode. |

## UI

Selection and the per-mode preview dialog are covered in
`construction/ui/functional-design/game-setup-and-modes.md`.
