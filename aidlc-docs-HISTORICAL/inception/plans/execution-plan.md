# Execution Plan — Squabble (Cross-Platform Scrabble in Go)

## Detailed Analysis Summary

### Change Impact Assessment
- **User-facing changes**: Yes — entire application is new and user-facing (graphical game on 4 platforms)
- **Structural changes**: Yes — greenfield; full system architecture must be designed
- **Data model changes**: Yes — GADDAG word graph, board state, tile bag, rack, game save format
- **API changes**: N/A — no external API; internal Go package interfaces only
- **NFR impact**: Yes — performance (≤500 ms AI, ≥30 fps UI), binary size (≤50 MB embedded dictionaries), security (file permissions, error handling), PBT (full enforcement), code commentary (NFR-10), trademark compliance (NFR-09)

### Risk Assessment
- **Risk Level**: High
- **Rollback Complexity**: N/A (greenfield — no existing system to roll back to)
- **Testing Complexity**: Complex — algorithmic engine requires oracle-based PBT; cross-platform build matrix; Android mobile testing

---

## Workflow Visualization

```
INCEPTION PHASE
================
[DONE] Workspace Detection
[SKIP] Reverse Engineering      <- Greenfield, no existing code
[DONE] Requirements Analysis
[DONE] User Stories
[NOW ] Workflow Planning
[EXEC] Application Design       <- New components, service layer, complex dependencies
[EXEC] Units Generation         <- 4 distinct units identified; decomposition needed

CONSTRUCTION PHASE (per-unit loop x4 units)
===========================================
For each of 4 units:
  [EXEC] Functional Design      <- Complex business logic in every unit
  [EXEC] NFR Requirements       <- Performance, security, PBT, commentary requirements
  [EXEC] NFR Design             <- Incorporate PBT, security, performance patterns
  [SKIP] Infrastructure Design  <- No cloud infrastructure; local desktop/mobile app
  [EXEC] Code Generation        <- Always

[EXEC] Build and Test           <- Always; cross-platform build matrix required

OPERATIONS PHASE
================
[PLACEHOLDER] Operations
```

---

## Proposed Units of Work

The system decomposes into **4 units**, each a distinct Go package with a clean interface boundary:

| Unit | Name | Description |
|---|---|---|
| U1 | `dictionary` | Word list loading, deduplication, GADDAG construction, word validation |
| U2 | `engine` | Board state, move validation, scoring, tile bag, rack, game flow |
| U3 | `ai` | Move enumeration (via GADDAG + engine), difficulty model (levels 1–10) |
| U4 | `ui` | Ebitengine graphics, board/tile rendering, input handling, screens, save/load |

**Unit dependency order** (build sequence):
```
dictionary  -->  engine  -->  ai  -->  ui
```
Each unit depends only on units to its left. No circular dependencies.

---

## Phases to Execute

### INCEPTION PHASE
- [x] Workspace Detection — COMPLETED
- [-] Reverse Engineering — SKIPPED (greenfield project, no existing code)
- [x] Requirements Analysis — COMPLETED
- [x] User Stories — COMPLETED
- [x] Workflow Planning — IN PROGRESS
- [ ] Application Design — EXECUTE
  - **Rationale**: New system with 4 distinct components, complex service boundaries, and non-obvious inter-unit interfaces (especially GADDAG-to-AI and engine-to-UI contracts).
- [ ] Units Generation — EXECUTE
  - **Rationale**: 4 distinct units with dependency ordering; story-to-unit mapping needed; Go package structure must be decided before code generation.

### CONSTRUCTION PHASE (per-unit, 4 iterations)
- [ ] Functional Design (each unit) — EXECUTE
  - **Rationale**: Complex business logic in every unit — GADDAG traversal algorithm, scoring rules (multipliers, bingo), AI interpolation model, Ebitengine game loop structure. PBT-01 (property identification) is a blocking requirement.
- [ ] NFR Requirements (each unit) — EXECUTE
  - **Rationale**: Performance targets (≤500 ms, ≥30 fps, ≤3 s startup, ≤50 MB binary), PBT framework selection (rapid), security baseline, and code commentary (NFR-10) all require explicit NFR design per unit.
- [ ] NFR Design (each unit) — EXECUTE
  - **Rationale**: NFR patterns must be incorporated: PBT test structure, structured logging (SECURITY-03), fail-safe error handling (SECURITY-15), compressed embedding for dictionaries (NFR-03).
- [-] Infrastructure Design (each unit) — SKIPPED
  - **Rationale**: No cloud infrastructure. The game is a local desktop/mobile application. File I/O (save/load) uses the OS standard app data directory — no IaC, no network infrastructure, no cloud services.
- [ ] Code Generation (each unit) — EXECUTE (always)
- [ ] Build and Test — EXECUTE (always)
  - **Rationale**: Cross-platform build matrix (Windows/macOS/Linux/Android) and comprehensive test suite required. PBT-08 (CI seed logging) and SECURITY-10 (dependency scanning) are blocking requirements.

### OPERATIONS PHASE
- [ ] Operations — PLACEHOLDER

---

## Success Criteria

- **Primary Goal**: A playable, graphical crossword board game in Go running on Windows, macOS, Linux, and Android with a GADDAG-based AI engine.
- **Key Deliverables**:
  1. `dictionary` package: GADDAG built from 5 embedded word lists, O(1) word validation
  2. `engine` package: Full rules engine (placement, scoring, bag, game flow)
  3. `ai` package: Move generator using Appel-Jacobson algorithm, 10 difficulty levels
  4. `ui` package: Ebitengine-based game with all screens and controls
  5. Cross-platform builds and a comprehensive test suite with PBT
- **Quality Gates**:
  - All 10 PBT rules compliant (blocking)
  - All 15 Security Baseline rules compliant where applicable (blocking)
  - AI move generation ≤500 ms on all platforms
  - Binary ≤50 MB
  - All exported symbols have GoDoc comments
  - No Hasbro trademark violations in code, assets, or text
