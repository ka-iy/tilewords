# Build and Test Summary — Persistent Default Setup Settings (FR-15)

## Build Status
- **Build Tool**: Go toolchain (module `tilewords`), Fyne v2.8.0.
- **`go build ./...`**: **Success** (exit 0, no output).
- **`go vet ./...`**: **Clean**.
- **`gofmt -l ui/`**: **Clean** (all increment files formatted).

## Test Execution Summary

### Unit Tests
- **Full `ui` suite**: **106 passed**, 0 failed.
- **This increment**: **10 passed**, 0 failed, including:
  - Property test `TestPBT_Settings_RoundTrip` — 100 generated cases pass.
  - Property test `TestPBT_Settings_LoadRobustness` — 100 generated cases pass.
- **Whole repo `go test ./...`**: all packages with tests pass (`ai`, `defs`, `dictionary`,
  `engine`, `ui`).
- **Coverage note**: persistence (`encode`/`decode`/`sanitize`/store) fully covered by unit +
  property tests; UI wiring covered by App round-trip + build-no-panic tests.

### Integration Tests
- **N/A** for this increment — the change is confined to the `ui` unit and the local
  `fyne.Preferences` store; no cross-service interactions. The App-level
  `TestDefaultsFor_RoundTripThroughApp` and `TestBuildSetup_LoadMappingAndNoPanic` exercise
  the store↔setup-screen integration within the unit.

### Performance Tests
- **N/A** — no performance-sensitive path added (a single small JSON read/write on screen
  open / game start).

### Additional Tests
- **Security**: input-validation tests at the decode boundary (`TestSanitize`,
  `TestDecodeInvalid`, `TestPBT_Settings_LoadRobustness`) satisfy the Security Baseline
  requirement for untrusted persisted input. **Pass.**
- **Property-Based Testing** (extension): round-trip + load-robustness properties. **Pass.**
- **Contract / E2E**: N/A.

## Extension Compliance
| Extension | Status | Evidence |
|---|---|---|
| Property-Based Testing | **Compliant** | `TestPBT_Settings_RoundTrip`, `TestPBT_Settings_LoadRobustness` |
| Security Baseline | **Compliant** | validation at decode boundary (BR-1..BR-4); local non-secret data |
| Resiliency Baseline | **N/A (slice compliant)** | graceful fallback to defaults on read failure; best-effort write |

## Known Nits (pre-existing, out of scope)
- `gofmt -l .` flags `dictionary/doc.go` — a pre-existing formatting nit unrelated to and
  untouched by this increment. Not addressed here to keep the change scoped.

## Overall Status
- **Build**: Success
- **All Tests**: Pass (106 `ui`, whole repo green)
- **Ready for Operations**: Yes (Operations is a placeholder stage; no deployment workflow)
