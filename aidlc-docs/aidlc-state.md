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

## Feature Increment — Persistent Default Setup Settings (FR-15)
- **Type**: post-v1 brownfield feature addition
- **Owning unit**: `ui` (+ a preferences-backed settings store)
- **Started**: 2026-07-23T14:30:00Z
- **Extensions**: Property-Based Testing (enabled, applicable), Security Baseline (enabled,
  applicable), Resiliency (mostly N/A)

### Stage Progress (increment)
- [x] Workspace Detection — brownfield; Reverse Engineering skipped (artifacts current)
- [x] Requirements Analysis — FR-15 added; `feature-persistent-default-settings.md` created — APPROVED 2026-07-23T14:45:00Z
- [x] Workflow Planning — `feature-persistent-default-settings-execution-plan.md` created — APPROVED 2026-07-23T14:45:00Z
- [-] User Stories — SKIP (simple single-touchpoint enhancement; FR-15 criteria are clear)
- [-] Application Design — SKIP (within `ui` boundary; one small settings store, no new cross-component services/APIs)
- [-] Units Generation — SKIP (single unit: `ui`)
- [x] Construction: Functional Design (ui) — `construction/ui/functional-design/persistent-default-settings.md` created — APPROVED 2026-07-23T15:00:00Z
- [-] Construction: NFR Requirements/Design (ui) — SKIP as separate stages (folded into Functional Design; extensions still enforced at Code Generation)
- [-] Construction: Infrastructure Design — SKIP (no infrastructure)
- [x] Construction: Code Generation (ui) — Part 1 (Planning) APPROVED 2026-07-23T15:15:00Z; Part 2 (Generation) APPROVED 2026-07-23T15:30:00Z (all 6 plan steps [x])
- [x] Build and Test — build/vet/fmt clean; full `ui` suite 106 pass (10 new); whole repo green — `construction/build-and-test/` docs created — **awaiting user approval (next: Operations placeholder)**
