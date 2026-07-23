# Build Instructions

Scope: whole project; the Persistent Default Setup Settings increment (FR-15) adds only
`ui`-package Go files, so the standard build applies unchanged.

## Prerequisites
- **Build Tool**: Go toolchain (module `tilewords`, see `go.mod`). Fyne v2.8.0 UI.
- **Dependencies**: resolved via Go modules (`pgregory.net/rapid` is a test-only dep).
- **System Requirements**: a working Fyne build environment (C toolchain + OpenGL/X11 headers
  on Linux) for the GUI binary; no such requirement for `go test ./ui/` headless tests.
- **Environment Variables**: none required for this increment.

## Build Steps

### 1. Resolve dependencies
```bash
go mod download
```

### 2. Build all packages
```bash
go build ./...
```

### 3. (Optional) Build the app binary
```bash
go build -o tilewords ./cmd/tilewords
```

### 4. Verify build success
- **Expected**: `go build ./...` exits 0 with no output.
- **Artifacts**: package objects in the build cache; optional `tilewords` binary.

## Static checks
```bash
go vet ./...
gofmt -l ui/
```
- `go vet ./...` must be clean.
- `gofmt -l ui/` must list no files.

## Troubleshooting
- **Fyne/OpenGL link errors on Linux**: install the GUI dev headers (X11/GL) required by
  Fyne. Not needed to run the headless `ui` tests.
