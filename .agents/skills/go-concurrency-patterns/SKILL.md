---
name: go-concurrency-patterns
description: Enforces Go 1.26 concurrency standards including sync.WaitGroup.Go dispatch, generic channel pipelines, worker pools, pub/sub routers, deterministic testing with testing/synctest, and runtime leak profiling via pprof goroutineleak. Use when designing, refactoring, or testing concurrent Go code.
---

# Go Concurrency Patterns Skill (Go 1.26 Edition)

This skill provides mandatory architectural patterns, implementation logic, and verification rules for concurrent programming in Go 1.26.

## Structural Execution Matrix

| Scenario / Goal | Primary Pattern | Key Primitives | Diagnostic & Safety Hooks |
|---|---|---|---|
| Parallel processing of bounded workloads | Worker Pool | `sync.WaitGroup.Go`, channels | Context timeout, buffer sizing |
| Multi-stage stream processing | Fan-Out / Fan-In Pipeline | Generic channels, `errors.AsType` | Cancellation propagation, channel draining |
| Event fan-out to multiple subscribers | Pub/Sub Event Bus | `sync.RWMutex`, subscriber maps | Non-blocking channel dispatches |
| Deterministic unit testing | Synctest Virtual Time | `testing/synctest`, `synctest.Wait` | Virtual clock bubbles, zero flakiness |
| Production goroutine leak debugging | GC Reachability Profiling | `pprof.Lookup("goroutineleak")` | GC reachability tracing |

## Architectural Execution Rules

### 1. WaitGroup Lifecycle Management

- **Rule**: Always leverage `wg.Go(func())` for goroutine execution when using `sync.WaitGroup`.
- **Constraint**: Do not manually call `wg.Add(1)` or `defer wg.Done()` unless interfacing with legacy codebases (< Go 1.25).

### 2. Context Cancellation and Channel Draining

- **Rule**: Every spawned worker or pipeline stage MUST accept a `context.Context` as its first parameter and listen for `ctx.Done()`.
- **Constraint**: Sender goroutines must NEVER close channels unilaterally if multiple senders exist. Receiver loops must drain remaining channel items upon context cancellation to prevent buffer-blocking leaks.

### 3. Type-Safe Exception Handling

- **Rule**: Parse pipeline errors using generics via `errors.AsType[T](err)` inside concurrency failure handlers.
- **Constraint**: Do not declare uninitialized pointer variables for `errors.As` checks.

### 4. Deterministic Concurrency Verification

- **Rule**: Unit tests involving concurrent synchronization, timers, or channel exchanges MUST be isolated using `testing/synctest`.
- **Constraint**: Do not insert arbitrary `time.Sleep` calls inside test functions. Use `synctest.Wait()` to verify durable blocking states.

## Pattern Reference Blueprint

### Worker Pool Pattern (Go 1.26 Native)

```go
package pool

import (
	"context"
	"sync"
)

type Job[T any] struct {
	ID   string
	Data T
}

type Result[R any] struct {
	JobID string
	Value R
	Err   error
}

func ExecutePool[T any, R any](
	ctx context.Context,
	workers int,
	jobs <-chan Job[T],
	workerFunc func(context.Context, Job[T]) (R, error),
) <-chan Result[R] {
	results := make(chan Result[R], workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					res, err := workerFunc(ctx, job)
					select {
					case <-ctx.Done():
						return
					case results <- Result[R]{JobID: job.ID, Value: res, Err: err}:
					}
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
```

### Deterministic Test Specification (`testing/synctest`)

```go
package pool_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestWorkerPoolDeterministic(t *testing.T) {
	synctest.Run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		jobs := make(chan Job[int], 2)
		jobs <- Job[int]{ID: "1", Data: 10}
		jobs <- Job[int]{ID: "2", Data: 20}
		close(jobs)

		worker := func(ctx context.Context, j Job[int]) (int, error) {
			time.Sleep(1 * time.Second)
			return j.Data * 2, nil
		}

		results := ExecutePool(ctx, 2, jobs, worker)

		synctest.Wait()

		res1 := <-results
		res2 := <-results

		if res1.Value != 20 || res2.Value != 40 {
			t.Fatalf("unexpected pool results: got %d, %d", res1.Value, res2.Value)
		}
	})
}
```

## Detailed Resources

For procedural execution workflows, architectural anti-patterns, and live diagnostic monitoring setups, refer to `@resources/implementation-playbook.md`.
