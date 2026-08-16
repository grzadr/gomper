# Implementation Playbook: Modern Go Testing (Go 1.24 - 1.26)

Copy-pasteable examples for each pattern in `SKILL.md`. Each section notes the minimum Go version it needs.

---

## 1. Table-Driven Tests with Subtests

Baseline structure for nearly every unit test — no version requirement.

```go
package calc_test

import (
	"testing"

	"myapp/internal/calc"
)

func TestDivide(t *testing.T) {
	tests := []struct {
		name    string
		a, b    int
		want    int
		wantErr bool
	}{
		{name: "even division", a: 10, b: 2, want: 5},
		{name: "truncates", a: 7, b: 2, want: 3},
		{name: "division by zero", a: 1, b: 0, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // safe: each case only touches its own local state

			got, err := calc.Divide(tc.a, tc.b)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Divide(%d, %d) error = %v, wantErr %v", tc.a, tc.b, err, tc.wantErr)
			}
			if err == nil && got != tc.want {
				t.Errorf("Divide(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
```

---

## 2. `T.Context()` — Auto-Canceled Test Context (Go 1.24+)

```go
package fetch_test

import (
	"testing"

	"myapp/internal/fetch"
)

func TestFetchUser(t *testing.T) {
	// Canceled automatically right before t's Cleanup funcs run — no
	// manual context.WithCancel/defer cancel() needed, and goroutines
	// started with this context are guaranteed to unwind before the
	// next test starts.
	ctx := t.Context()

	user, err := fetch.User(ctx, "alice")
	if err != nil {
		t.Fatalf("fetch.User() error = %v", err)
	}
	if user.Name != "alice" {
		t.Errorf("user.Name = %q, want %q", user.Name, "alice")
	}
}
```

Benchmarks get the same thing via `b.Context()`.

---

## 3. `T.Chdir()` — Scoped Working Directory (Go 1.24+)

```go
package config_test

import (
	"os"
	"testing"

	"myapp/internal/config"
)

func TestLoadFromCWD(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/config.yaml", []byte("port: 8080\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// Changes cwd for this test only; Cleanup restores the original
	// directory automatically. Do NOT call this after t.Parallel() —
	// the working directory is process-global, so Chdir panics if the
	// test has already opted into running in parallel with others.
	t.Chdir(dir)

	cfg, err := config.LoadFromCWD()
	if err != nil {
		t.Fatalf("LoadFromCWD() error = %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("cfg.Port = %d, want 8080", cfg.Port)
	}
}
```

---

## 4. `B.Loop()` — Modern Benchmark Loop (Go 1.24+, inlining fixed in 1.26)

```go
package calc_test

import (
	"testing"

	"myapp/internal/calc"
)

func BenchmarkFib(b *testing.B) {
	for b.Loop() {
		// Setup done before the loop runs once, not once per b.N
		// iteration. Inputs/outputs referenced in the loop body are
		// kept alive, so the compiler can't optimize the call away —
		// the classic silent-benchmark-that-measures-nothing bug from
		// the old `for i := 0; i < b.N; i++` style.
		calc.Fib(20)
	}
}

func BenchmarkParseConfig(b *testing.B) {
	data := loadFixture(b) // runs once, before the timed loop

	for b.Loop() {
		if _, err := calc.ParseConfig(data); err != nil {
			b.Fatal(err)
		}
	}
}
```

---

## 5. `testing/synctest` — Deterministic Concurrency & Time (GA in Go 1.25+)

Requires `go 1.25` or later in `go.mod`. `synctest.Test` replaces the Go 1.24 experimental `synctest.Run` (which required `GOEXPERIMENT=synctest` and was removed in Go 1.26).

```go
package ratelimit_test

import (
	"testing"
	"testing/synctest"
	"time"

	"myapp/internal/ratelimit"
)

func TestLimiterRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rl := ratelimit.New(3, 100*time.Millisecond) // 3 tokens, refills every 100ms

		// Drain the bucket.
		for i := 0; i < 3; i++ {
			if !rl.Allow() {
				t.Fatalf("request %d: expected allowed, got denied", i+1)
			}
		}
		if rl.Allow() {
			t.Fatal("expected 4th request to be denied")
		}

		// Advance the bubble's fake clock past one refill interval.
		// Wait blocks until every goroutine in the bubble is "durably
		// blocked" (e.g. sleeping, waiting on a channel/cond); time
		// then jumps straight to the next thing that unblocks it —
		// this test runs in microseconds, not 100ms+.
		time.Sleep(100 * time.Millisecond)
		synctest.Wait()

		if !rl.Allow() {
			t.Fatal("expected a token to be available after refill")
		}
	})
}
```

Testing a context timeout without a real 5-second wait:

```go
func TestFetchTimesOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		errCh := make(chan error, 1)
		go func() { errCh <- slowFetch(ctx) }()

		synctest.Wait() // let the goroutine block on its next event
		if err := ctx.Err(); err != nil {
			t.Fatalf("context canceled early: %v", err)
		}

		time.Sleep(5 * time.Second) // instant inside the bubble
		synctest.Wait()

		if err := <-errCh; !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("slowFetch() error = %v, want DeadlineExceeded", err)
		}
	})
}
```

Rules of thumb inside a bubble: avoid touching goroutines started outside the bubble, avoid real network calls (fake them), and don't let background goroutines leak past the end of the test — an unresolved durable block with nothing left to wait on is a deadlock and `synctest.Test` panics.

---

## 6. `T.ArtifactDir()` — Persisted Test Output (Go 1.26+)

```go
package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"myapp/internal/render"
)

func TestRenderThumbnail(t *testing.T) {
	img, err := render.Thumbnail("testdata/source.png")
	if err != nil {
		t.Fatalf("Thumbnail() error = %v", err)
	}

	// With `go test -artifacts`, this directory is kept under the
	// output dir (or -outputdir) so a human can inspect the rendered
	// file afterward. Without the flag, it's a temp dir cleaned up
	// automatically — the test doesn't need to know which.
	out := filepath.Join(t.ArtifactDir(), "thumbnail.png")
	if err := os.WriteFile(out, img, 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
}
```

Run with `go test -artifacts -outputdir=./out ./...` to keep the rendered files for review.

---

## 7. `errors.AsType` — Generic Type-Safe Error Assertion (Go 1.26+)

```go
package validate_test

import (
	"errors"
	"testing"

	"myapp/internal/validate"
)

type FieldError struct {
	Field string
	Msg   string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Msg }

func TestValidate_ReturnsFieldError(t *testing.T) {
	err := validate.Check(map[string]string{"email": "not-an-email"})

	// Old style:
	//   var fe *FieldError
	//   if !errors.As(err, &fe) { t.Fatalf(...) }
	//
	// New style — generic, so the target type is inferred at the call
	// site with no throwaway variable:
	fe, ok := errors.AsType[*FieldError](err)
	if !ok {
		t.Fatalf("Check() error = %v, want *FieldError", err)
	}
	if fe.Field != "email" {
		t.Errorf("fe.Field = %q, want %q", fe.Field, "email")
	}
}
```

---

## 8. Fuzz Testing (stable since Go 1.18, still underused)

Use for any parser, decoder, or other function that handles untrusted input.

```go
package parse_test

import (
	"testing"

	"myapp/internal/parse"
)

func FuzzParseQuery(f *testing.F) {
	f.Add("key=value&other=1")
	f.Add("")
	f.Add("=====")

	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic, regardless of input.
		_, _ = parse.Query(input)
	})
}
```

Run interactively during development: `go test -fuzz=FuzzParseQuery -fuzztime=30s`.

---

## 9. Deterministic Crypto Randomness in Tests (Go 1.26+)

As of Go 1.26, the `random` parameter to functions like `rsa.GenerateKey` and `rand.Prime` is ignored in favor of a secure source — so tests that previously passed a seeded `io.Reader` for reproducibility need `testing/cryptotest` instead.

```go
package keys_test

import (
	"testing"
	"testing/cryptotest"

	"myapp/internal/keys"
)

func TestGenerateDeterministic(t *testing.T) {
	cryptotest.SetGlobalRandom(t, deterministicReader(42))

	k1, err := keys.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	// k1 is now reproducible for the lifetime of this test.
}
```

---

## 10. CI Baseline Commands

| Purpose | Command |
| --- | --- |
| Run tests with the race detector | `go test -race ./...` |
| Run tests with coverage | `go test -race -cover ./...` |
| Keep artifacts for inspection | `go test -artifacts -outputdir=./out ./...` |
| Run a specific fuzz target for a fixed budget | `go test -fuzz=FuzzX -fuzztime=30s ./...` |
| Modernize old test idioms automatically | `go fix ./...` (Go 1.26+; rewrites e.g. `b.N` loops and old error-assertion patterns where a safe fixer exists) |
