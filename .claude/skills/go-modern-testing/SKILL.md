---
name: go-modern-testing
description: Write and review idiomatic Go tests and benchmarks using the current testing toolchain (Go 1.24-1.26) — table-driven subtests, testing.T.Context/T.Chdir, the testing.B.Loop benchmark style, the testing/synctest bubble for concurrent and time-based code, testing.T.ArtifactDir for output files, and errors.AsType for type-safe assertions. Use whenever the user asks to write, fix, or review Go tests, table-driven tests, benchmarks, fuzz tests, or tests for concurrent/async Go code, or mentions flaky tests, time.Sleep in tests, or slow benchmarks.
---

# Modern Go Testing (Go 1.24 - 1.26)

## Skill Overview

This skill provides design patterns and current-toolchain APIs for writing reliable, fast, idiomatic Go tests. It replaces older patterns (manual `context.WithCancel` plumbing, `os.Chdir`/`defer` pairs, `b.N` loops, real-time `time.Sleep` in concurrency tests, and manual `errors.As` type-assertion boilerplate) with the standard-library mechanisms introduced across Go 1.24, 1.25, and 1.26.

**Requires a `go.mod` with `go 1.24` or later** for `T.Context`/`T.Chdir`/`B.Loop`; **`go 1.25`** for the general-availability `testing/synctest` API; **`go 1.26`** for `T.ArtifactDir` and `errors.AsType`. Check the project's `go.mod` before relying on a given feature — see the version matrix below.

## Core Testing Principles

1. **Table-Driven by Default**: Structure test cases as a slice of structs run through `t.Run` subtests, not copy-pasted test functions. Name each case; use `t.Parallel()` inside the subtest closure when cases are independent.

2. **Context via `t.Context()`, Not Manual Plumbing**: Any code under test that takes a `context.Context` should receive `t.Context()` rather than a hand-rolled `context.WithCancel(context.Background())`. It is automatically canceled right before cleanup funcs run, so goroutines started during the test are torn down without extra bookkeeping.

3. **Deterministic Concurrency via `testing/synctest`, Not Real Sleeps**: Tests for code that spawns goroutines, waits on timers, or races against a context deadline should run inside a `synctest.Test(t, func(t *testing.T) { ... })` bubble and call `synctest.Wait()` to advance to the next blocking point, instead of `time.Sleep` + retry loops. This makes concurrency tests both instant and flake-free.

4. **`B.Loop`, Not `b.N`**: New benchmarks use `for b.Loop() { ... }` rather than `for i := 0; i < b.N; i++`. It runs setup once, keeps benchmark inputs/outputs alive so the compiler can't dead-code-eliminate the work being measured, and (as of Go 1.26) no longer blocks inlining of the loop body.

5. **Isolate Filesystem State**: Use `t.TempDir()` for scratch files and `t.Chdir()` to change the working directory for the duration of a test, instead of `os.Chdir` + a manual `defer` restore. `t.Chdir` panics if called after `t.Parallel()` — the working directory is process-global, so it can't be safely changed per-parallel-test.

6. **Durable Output via `T.ArtifactDir()`**: Tests that need to persist output files for later inspection (rendered images, protocol dumps, generated reports) should write them to `t.ArtifactDir()` rather than a hardcoded or ad hoc path. Under `go test -artifacts` the directory survives the run; otherwise it's a temp dir cleaned up automatically.

7. **Type-Safe Error Assertions via `errors.AsType`**: For extracting a typed error out of a chain in a test assertion, prefer the generic `errors.AsType[*MyError](err)` over declaring a `var target *MyError` and calling `errors.As(err, &target)`.

8. **Everything Else Stays Standard**: `t.Helper()` in shared assertion helpers, `t.Cleanup()` for teardown (composes better than `defer` across subtests), no package-level mutable test fixtures, `go test -race` in CI, and fuzz tests (`func FuzzX(f *testing.F)`) for anything parsing untrusted input.

## Feature Version Matrix

| Feature | Package.Method | Minimum Go version |
| --- | --- | --- |
| Context auto-canceled at test end | `T.Context` / `B.Context` | 1.24 |
| Scoped working-directory change | `T.Chdir` / `B.Chdir` | 1.24 |
| Inlining-friendly benchmark loop | `B.Loop` | 1.24 (inlining fixed in 1.26) |
| Deterministic concurrency bubble | `testing/synctest.Test` + `Wait` | 1.25 (GA; was experimental `Run` in 1.24, removed in 1.26) |
| Directory for persisted test output | `T.ArtifactDir` / `B.ArtifactDir` / `F.ArtifactDir` | 1.26 |
| Deterministic crypto randomness in tests | `testing/cryptotest.SetGlobalRandom` | 1.26 |
| Generic, type-safe `errors.As` | `errors.AsType[T]` | 1.26 |
| Expression operand to `new` (handy for test fixtures) | `new(expr)` | 1.26 |

## Key Technical Rules & Anti-Patterns

| Category | Anti-Pattern | Recommended Pattern |
| --- | --- | --- |
| Test structure | Separate `Test_CaseA`, `Test_CaseB` functions with duplicated setup | Table of cases run via `t.Run(tc.name, func(t *testing.T) {...})` |
| Context lifetime | `ctx, cancel := context.WithCancel(context.Background()); defer cancel()` in every test | `ctx := t.Context()` |
| Concurrency testing | `go doWork(); time.Sleep(100 * time.Millisecond); assert...` | `synctest.Test(t, func(t *testing.T) { go doWork(); synctest.Wait(); assert... })` |
| Benchmark loop | `for i := 0; i < b.N; i++ { ... }` | `for b.Loop() { ... }` |
| Working directory | `os.Chdir(dir); defer os.Chdir(orig)` | `t.Chdir(dir)` (never after `t.Parallel()`) |
| Persisted output | Writing to a hardcoded `./testdata/out/` path | Writing to `t.ArtifactDir()` |
| Error assertions | `var target *MyErr; if !errors.As(err, &target) { t.Fatal(...) }` | `target, ok := errors.AsType[*MyErr](err)` |
| Shared helpers | Assertion helper without `t.Helper()`, burying failures at the wrong line | Call `t.Helper()` as the first line of any test helper |

## Workflow & Review Checklist

1. Confirm the module's `go.mod` Go version against the feature matrix above before suggesting a given API.
2. Convert ad hoc test functions into table-driven subtests; add `t.Parallel()` where cases are independent.
3. Replace manual context/cleanup plumbing with `t.Context()` and `t.Cleanup()`.
4. For any test exercising goroutines, timers, or context deadlines, wrap it in `synctest.Test` and replace `time.Sleep`-based synchronization with `synctest.Wait()`.
5. Convert `b.N`-style benchmarks to `b.Loop()`.
6. Route any test-produced files through `t.TempDir()` (scratch, discarded) or `t.ArtifactDir()` (kept when `-artifacts` is set).
7. Add fuzz tests for parsers/decoders; run `go test -race -cover ./...` as the baseline CI command.

## Reference Implementations

* `resources/testing-playbook.md`: Runnable code for table-driven tests, `t.Context`/`t.Chdir`, `B.Loop` benchmarks, `testing/synctest` concurrency tests, `T.ArtifactDir`, `errors.AsType`, fuzzing, and the CI command set.

Read the playbook when actually generating test files rather than just discussing structure — it has copy-pasteable code for each pattern above.
