# Gomper Architecture & Engineering Invariants

## Toolchain & Quality Gates
- Go Version: 1.25+
- Enforce formatting and linting via project targets before committing changes:
  - `make lint` (runs `golangci-lint run ./...` with project `.golangci.yaml`)
  - `make test` (runs race-detector enabled test suite: `go test -race -shuffle=on ./...`)
- Unit tests must be executed with `-race` enabled on both root and internal packages.

## Architectural Boundaries
- Command Dispatch (`cmd/`):
  - Cobra commands must strictly parse flags, read Viper config, and immediately delegate to `internal/app/`.
  - No business logic, file walking, or formatting belongs inside `cmd/`.
- Core Scanning & Tokenization (`internal/scanner/`):
  - Tokenization routines sit on the critical execution path; avoid dynamic heap allocations inside loops.
  - Rely on standard profile definitions located in `internal/scanner/profiles/`.
- Atomic Operations (`internal/filetx/`):
  - File generation and mutation must be atomic and reversible. Keep transaction rollbacks isolated from formatters.
- Output Generation (`internal/dumper/`):
  - Expose streaming interfaces (`io.Writer`) rather than buffering large aggregations entirely in memory.

## Code Standards & Idioms
- Wrap errors idiomatically: `fmt.Errorf("scanner: failed to parse profile: %w", err)`.
- Use table-driven testing for both `*_test.go` (black-box) and `*_internal_test.go` (white-box).
- Enforce the DRY principle; keep functions focused with zero unused return values or parameters.

## Fundamental Rules
- Think before coding. State assumptions, surface tradeoffs, push back when warranted.
- Simplicity first. Minimum code that solves the problem. Nothing speculative.
- Surgical changes. Touch only what you must. Clean up only your own mess.
- Goal-driven execution. Define success criteria. Loop until verified.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (grzadr/gomper), via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context layout (root `CONTEXT.md` + `docs/adr/`, created lazily as concepts get resolved). See `docs/agents/domain.md`.
