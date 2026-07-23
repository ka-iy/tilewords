# Unit Test Execution

Scope: the Persistent Default Setup Settings increment (FR-15) and the full `ui` suite.

## Run the tests

### 1. All packages
```bash
go test ./...
```

### 2. The ui unit (where this feature lives)
```bash
go test ./ui/
```

### 3. Just this feature's tests
```bash
go test ./ui/ -run 'Settings|Setup|Defaults|Sanitize|DecodeInvalid|PBT_Settings' -v
```

## Feature test inventory (`ui/settings_test.go`, `ui/setup_settings_test.go`)
- `TestDefaultGameSettings` — built-in defaults, incl. empty-avail case.
- `TestSanitize` — load-time validation BR-1 (dict), BR-2 (difficulty), BR-3 (mode).
- `TestDecodeInvalid` — BR-4: empty / malformed / non-object JSON → defaults.
- `TestSettingsStoreRoundTrip` — save→load through in-memory `fyne.Preferences`.
- `TestPBT_Settings_RoundTrip` — property: `decode(encode(s)) == s` for any valid settings.
- `TestPBT_Settings_LoadRobustness` — property: arbitrary stored string never panics and
  always yields a valid, in-range settings value.
- `TestDefaultsFor_NilStore` — headless nil-guard returns built-in defaults.
- `TestDefaultsFor_RoundTripThroughApp` — App-level save→load.
- `TestBuildSetup_LoadMappingAndNoPanic` — loaded dict maps to a real radio label; the setup
  screen builds without panicking with saved defaults present.

## Expected results
- **Full `ui` suite**: 106 tests pass, 0 failures.
- **This feature**: 10 tests pass (incl. 2 property tests at 100 generated cases each).
- **Coverage note**: the persistence logic is fully covered by pure-function unit + property
  tests; the UI wiring is covered by the App round-trip and build-no-panic tests.

## On failure
1. Read the failing test output (rapid prints a minimal counterexample for property failures).
2. Fix the code and re-run until green.
