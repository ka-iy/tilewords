# AI-DLC State Tracking

## Project Information
- **Project Name**: Squabble — Cross-Platform Scrabble Game in Go
- **Project Type**: Greenfield
- **Start Date**: 2026-04-17T21:30:00Z
- **Current Stage**: INCEPTION - Requirements Analysis

## Workspace State
- **Existing Code**: No
- **Reverse Engineering Needed**: No
- **Workspace Root**: /home/kartik/PROGS/SQUABBLE-Scrabble_Vibe_coded

## Code Location Rules
- **Application Code**: Workspace root (NEVER in aidlc-docs/)
- **Documentation**: aidlc-docs/ only
- **Structure patterns**: See code-generation.md Critical Rules

## Extension Configuration
| Extension | Enabled | Decided At |
|---|---|---|
| Property-Based Testing | Yes — Full Enforcement (PBT-01 through PBT-10, all blocking) | Requirements Analysis |
| Security Baseline | Yes — Full Enforcement (SECURITY-01 through SECURITY-15, all blocking) | Requirements Analysis |

## Stage Progress
### INCEPTION PHASE
- [x] Workspace Detection
- [-] Reverse Engineering — SKIPPED (greenfield)
- [x] Requirements Analysis
- [x] User Stories
- [x] Workflow Planning
- [x] Application Design — COMPLETE
- [x] Units Generation — COMPLETE (5 units: dictionary, engine, ai, ui, cmd)

### CONSTRUCTION PHASE (4 units: dictionary, engine, ai, ui)
- [ ] Functional Design (per unit) — EXECUTE
- [ ] NFR Requirements (per unit) — EXECUTE
- [ ] NFR Design (per unit) — EXECUTE
- [-] Infrastructure Design — SKIPPED (no cloud infrastructure)
- [ ] Code Generation (per unit) — EXECUTE
- [ ] Build and Test — EXECUTE

### OPERATIONS PHASE
- [ ] Operations — PLACEHOLDER

## Current Status
- **Lifecycle Phase**: INCEPTION
- **Current Stage**: Workflow Planning — awaiting user approval
- **Next Stage**: Application Design
- **Units Planned**: dictionary, engine, ai, ui (dependency order)

## Post-v1 Retroactive Documentation
A pass was run to bring the AIDLC docs up to the current code. Documented the following
previously-uncaptured work (application code was already implemented and committed):

- **Unit 6: `defs`** (NEW) — word definitions shown during gameplay. Full construction doc set
  under `construction/defs/` (functional-design, nfr-requirements, nfr-design, code) + tools
  (`builddefs`, `defslookup`, `memcheck`). Registered in `unit-of-work.md` / `components.md`.
- **`engine` additions** — game modes (Classic/Interesting), Scrabble-notation formatting,
  persisted move records / opening draw. See `construction/engine/functional-design/{game-modes,
  notation-and-move-records}.md`.
- **`ui` additions** — Move history / Definitions two-tab panel (copy + touch scrolling),
  game-mode selection with preview dialog, notation toggle, single-slot atomic save/resume.
  See `construction/ui/functional-design/{move-history-and-definitions, game-setup-and-modes,
  save-and-resume}.md`.
- **Requirements/Stories** — added FR-12 (definitions), FR-13 (game modes), FR-14 (notation);
  US-20–US-23.

**Note**: the shipped UI toolkit is **Fyne** (the initial design named Ebitengine), and the
bundled word lists are `enable`, `wordnik`, `atebits-letterpress` (not the placeholder
`csw/sowpods/…` names in the original design docs). These divergences are noted where relevant
but the original v1 design docs were left intact as the historical record.
