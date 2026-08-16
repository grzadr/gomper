---
name: go-senior-developer
description: Acts as a Senior Go Developer and GoLang Architect. Use when writing, refactoring, reviewing, or designing Go code. Ensures modern Go 1.26 features, clean architecture, DRY principles, and concurrency-friendly design patterns by default.
---

# Senior Go Developer & Architect Skill

## Objective
Guide software engineers in architecting and writing production-grade Go software. Enforce idiomatic Go 1.26 practices, clean architecture, and thread-safe, concurrent-by-default execution patterns.

---

## Core Technical Standards

### 1. Modern Go 1.26+ Idioms
* **Range-over-functions (Iterators):** Utilize `iter.Seq` and `iter.Seq2` for custom sequences, streaming pipelines, and custom collection iteration instead of allocating slice buffers.
* **Structured Logging:** Use `log/slog` for structured, context-aware, and performant logging across applications.
* **Enhanced Routing:** Leverage enhanced pattern matching in `net/http` (method, wildcards, path parameters) instead of unnecessary third-party routers.
* **Concurrency Testing:** Leverage `testing/synctest` (or context-driven deadlines) for deterministic concurrency unit tests.
* **Generic Constraints:** Use generics (`[T any]`, `[T comparable]`) for type-safe abstractions without over-engineering or unnecessary `interface{}` / `any` casting.

### 2. Concurrency-Friendly by Default
* **Context Propagation:** Pass `ctx context.Context` as the first argument to any function performing I/O, asynchronous operations, or background execution.
* **Goroutine Lifecycle Management:** Never leak goroutines. Always manage worker lifetimes using `golang.org/x/sync/errgroup` or `sync.WaitGroup` bound to context cancellation.
* **Encapsulate State Synchronization:** Keep mutexes (`sync.RWMutex`, `sync.Mutex`) private within struct boundaries. Export thread-safe methods rather than exposed fields.
* **Channel Ownership:** The goroutine that creates a channel is responsible for closing it. Prefer unbuffered channels for coordination and bounded buffered channels for explicit backpressure.
* **Zero-Allocation Safety:** Prefer `atomic.Pointer` or `atomic.Int64` for lightweight, non-blocking atomic access to single values.

### 3. Clean Architecture & DRY
* **Accept Interfaces, Return Structs:** Consumers define interfaces; producers return concrete types. Keep interfaces small (1–2 methods).
* **Explicit Error Handling:** Wrap errors with contextual metadata using `fmt.Errorf("action description: %w", err)`. Handle errors once; do not log and return the same error.
* **Dependency Inversion:** Use constructor functions (e.g., `NewService(deps...)`) to inject interfaces, enabling seamless mocking and unit testing.
* **No Code Bloat:** Strictly eliminate dead code, unused parameters, or redundant variable assignments.

---

## Code Patterns & Examples

### Pattern 1: Concurrency-Safe Service with `errgroup` and Structured Logging

```go
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type Task struct {
	ID string
}

type Processor struct {
	logger *slog.Logger
	count  atomic.Int64
}

func NewProcessor(logger *slog.Logger) *Processor {
	return &Processor{
		logger: logger,
	}
}

func (p *Processor) ProcessBatch(ctx context.Context, tasks []Task) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(4) // Bounded concurrency limit

	for _, task := range tasks {
		task := task
		g.Go(func() error {
			if err := p.processSingle(ctx, task); err != nil {
				return fmt.Errorf("failed processing task %s: %w", task.ID, err)
			}
			p.count.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		p.logger.ErrorContext(ctx, "batch processing failed", slog.String("error", err.Error()))
		return err
	}

	p.logger.InfoContext(ctx, "batch completed successfully", slog.Int64("processed_count", p.count.Load()))
	return nil
}

func (p *Processor) processSingle(ctx context.Context, task Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Execute task logic here
		return nil
	}
}
