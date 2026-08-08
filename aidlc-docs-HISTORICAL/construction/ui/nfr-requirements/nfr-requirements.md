# NFR Requirements — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## Performance (NFR-01)

| ID | Requirement | Threshold | Rationale |
|---|---|---|---|
| NFR-UI-P1 | `Game.Update()` execution time (excluding AI computation) | ≤ 2 ms | Must not drop below 30 fps; Ebitengine calls Update 60×/s on desktop |
| NFR-UI-P2 | `Game.Draw()` execution time | ≤ 8 ms | Allows ≥30 fps render budget on mid-range Android (2020+) |
| NFR-UI-P3 | `SaveManager.Save()` wall-clock time | ≤ 200 ms | Acceptable pause; write is atomic (temp→rename) so no torn state |
| NFR-UI-P4 | `SaveManager.Load()` wall-clock time | ≤ 500 ms | Happens only at startup from MainMenuScreen; one-time cost |
| NFR-UI-P5 | Dictionary load time (in SetupScreen before GameScreen transition) | ≤ 3 s | NFR-01 GADDAG construction budget; shown with progress indication |
| NFR-UI-P6 | Screen transition latency (button click to new screen rendered) | ≤ 1 frame (≤ 17 ms) | Transition is a pointer swap; no blocking I/O on transition path |

---

## Memory (NFR-03)

| ID | Requirement | Threshold | Rationale |
|---|---|---|---|
| NFR-UI-M1 | Peak heap during GameScreen | ≤ 50 MB (excluding embedded dictionary) | Board, racks, staged tiles are small; rendering uses no texture atlas |
| NFR-UI-M2 | No per-frame allocations in `Draw()` | Zero heap allocs in steady-state Draw | Prevents GC pauses that cause frame drops; use pre-allocated draw options |

---

## Thread Safety (NFR-UI-T1)

All Ebitengine callbacks (`Update`, `Draw`, `Layout`) run on the main goroutine. The
`ai.AIWorker` goroutine communicates results via a buffered channel polled in `Update`.
No mutex is needed in the `ui` package; the channel protocol (inherited from Unit 3)
enforces all happens-before guarantees.

`SaveManager.Save` and `SaveManager.Load` must NOT be called from within `Draw`; they
are called from `Update` only.

---

## Reliability (NFR-04)

| ID | Requirement |
|---|---|
| NFR-UI-R1 | `Game.Update()` must never panic on any valid `GameState`; panics from engine or ai packages must be recovered and shown as an error screen |
| NFR-UI-R2 | `SaveManager.Load()` must return a typed error on corrupt/missing file; it must never panic or partially apply a corrupt state |
| NFR-UI-R3 | An atomic write pattern (write temp file → rename) must be used for save to prevent half-written save files |
| NFR-UI-R4 | If `ai.AIWorker` takes longer than 10 seconds to respond, `GameScreen.Update()` must time out, show an error, and transition to MainMenuScreen |
| NFR-UI-R5 | `SaveManager.Save` must create the config directory if it does not exist (first run) |

---

## Usability (NFR-05)

| ID | Requirement |
|---|---|
| NFR-UI-U1 | The board must visually distinguish all five premium square types (DL, TL, DW, TW, Centre) with distinct colours and short labels |
| NFR-UI-U2 | Staged (uncommitted) tiles must be visually distinct from committed board tiles at all times |
| NFR-UI-U3 | The AI rack toggle button label must change immediately on click ("Show AI Rack" ↔ "Hide AI Rack") |
| NFR-UI-U4 | Status bar messages must be human-readable and contain no Go error type names or stack traces |
| NFR-UI-U5 | All interactive buttons must show a hover state on desktop (colour change on mouse-over) |
| NFR-UI-U6 | Touch targets for Android must be ≥ 44×44 logical pixels (tile size = 44 px, button min height = 44 px) |

---

## Security (NFR-08 / SECURITY-15)

| ID | Requirement |
|---|---|
| SECURITY-UI-1 | Save file written with permissions 0600 (user-read/write only); parent directory 0700 |
| SECURITY-UI-2 | Error messages shown in the UI must never include internal file paths, Go type names, or stack traces — sanitise with a one-line message only |
| SECURITY-UI-3 | `SaveManager` must validate the decoded `GameState` is non-nil before returning it; malformed gob data must be caught by gob's own error return, not cause a nil-pointer panic |
| SECURITY-UI-4 | No user-supplied input is eval'd, passed to a shell, or used to construct file paths; the save path is fully resolved at construction time from `os.UserConfigDir()` |

---

## Testability (NFR-07)

| ID | Requirement |
|---|---|
| NFR-UI-TEST-1 | `SaveManager` is fully testable without a running Ebitengine loop; tests use `os.TempDir()` as the config root |
| NFR-UI-TEST-2 | `cellAt` (board hit-test) is a pure function testable without any Ebitengine state |
| NFR-UI-TEST-3 | Screen transition logic (which screen follows which) is exercisable via direct `Update` calls with a stub `*Game` |
| NFR-UI-TEST-4 | `pgregory.net/rapid` is used for PBT-UI-01 through PBT-UI-04 |
| NFR-UI-TEST-5 | The `ui` package must compile and its non-Ebitengine logic must be testable with `go test ./ui/...` without a display (headless); Ebitengine rendering tests are excluded with a `//go:build ignore` tag or equivalent |

---

## Code Commentary (NFR-10)

| ID | Requirement |
|---|---|
| NFR-UI-C1 | Every exported type and function has a GoDoc comment |
| NFR-UI-C2 | `GameScreen.Update` has inline comments delimiting the five state phases: BlankPicker input, AI poll, HumanTurn input dispatch, button handling, end-game check |
| NFR-UI-C3 | `BoardRenderer.Draw` has comments identifying each of the five draw passes (background, premium squares, committed tiles, staged tiles, grid lines) |
| NFR-UI-C4 | `SaveManager` comments explain the atomic write pattern and the OS path resolution strategy |
| NFR-UI-C5 | `button.IsClicked` comment explains why bounds check precedes enabled check (avoids spurious hover state on disabled buttons) |
