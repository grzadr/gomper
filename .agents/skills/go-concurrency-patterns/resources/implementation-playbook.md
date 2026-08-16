# Go Concurrency Implementation Playbook (Go 1.26 Standard)

This playbook provides operational guidelines, structural recipes, and diagnostic procedures for building concurrent software with Go 1.26.

## Section 1: Advanced Concurrency Blueprints

### 1.1 Fan-Out / Fan-In with Generic Pipeline Mechanics

The Fan-Out / Fan-In architecture processes streaming data in parallel stages across multiple goroutines, merging outputs into a consolidated downstream channel.

```go
package pipeline

import (
	"context"
	"sync"
)

type PipelineError struct {
	Stage string
	Err   error
}

func (e *PipelineError) Error() string {
	return "pipeline failure at " + e.Stage + ": " + e.Err.Error()
}

func FanIn[R any](ctx context.Context, channels ...<-chan R) <-chan R {
	var wg sync.WaitGroup
	out := make(chan R)

	output := func(c <-chan R) {
		for val := range c {
			select {
			case <-ctx.Done():
				return
			case out <- val:
			}
		}
	}

	for _, c := range channels {
		ch := c
		wg.Go(func() {
			output(ch)
		})
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
```

### 1.2 Generic Pub/Sub Event Router

A thread-safe, non-blocking publish/subscribe system leveraging `sync.RWMutex` and context-aware dispatches.

```go
package pubsub

import (
	"context"
	"sync"
)

type EventBus[T any] struct {
	mu          sync.RWMutex
	subscribers map[string][]chan T
	capacity    int
}

func NewEventBus[T any](channelCapacity int) *EventBus[T] {
	return &EventBus[T]{
		subscribers: make(map[string][]chan T),
		capacity:    channelCapacity,
	}
}

func (b *EventBus[T]) Subscribe(topic string) (<-chan T, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan T, b.capacity)
	b.subscribers[topic] = append(b.subscribers[topic], ch)

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subscribers[topic]
		for i, sub := range subs {
			if sub == ch {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsubscribe
}

func (b *EventBus[T]) Publish(ctx context.Context, topic string, event T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers[topic] {
		select {
		case <-ctx.Done():
			return
		case ch <- event:
		default:
			// Non-blocking drop or overflow telemetry handling
		}
	}
}
```

### 1.3 Bounded Worker Pool with Dynamic Resizing & `sync.WaitGroup.Go`

A production worker pool pattern leveraging `sync.WaitGroup.Go` for clean task worker lifecycle management.

```go
package pool

import (
	"context"
	"errors"
	"sync"
)

var ErrPoolClosed = errors.New("worker pool is closed")

type Task[T any, R any] struct {
	Input T
	Exec  func(context.Context, T) (R, error)
}

type TaskResult[R any] struct {
	Output R
	Err    error
}

type WorkerPool[T any, R any] struct {
	workers int
	tasks   chan Task[T, R]
	results chan TaskResult[R]
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorkerPool[T any, R any](parentCtx context.Context, workers int, bufferSize int) *WorkerPool[T, R] {
	ctx, cancel := context.WithCancel(parentCtx)
	p := &WorkerPool[T, R]{
		workers: workers,
		tasks:   make(chan Task[T, R], bufferSize),
		results: make(chan TaskResult[R], bufferSize),
		ctx:     ctx,
		cancel:  cancel,
	}

	p.start()
	return p
}

func (p *WorkerPool[T, R]) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Go(func() {
			for {
				select {
				case <-p.ctx.Done():
					return
				case task, ok := <-p.tasks:
					if !ok {
						return
					}
					res, err := task.Exec(p.ctx, task.Input)
					select {
					case <-p.ctx.Done():
						return
					case p.results <- TaskResult[R]{Output: res, Err: err}:
					}
				}
			}
		})
	}

	go func() {
		p.wg.Wait()
		close(p.results)
	}()
}

func (p *WorkerPool[T, R]) Submit(task Task[T, R]) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.tasks <- task:
		return nil
	}
}

func (p *WorkerPool[T, R]) Results() <-chan TaskResult[R] {
	return p.results
}

func (p *WorkerPool[T, R]) Shutdown() {
	close(p.tasks)
	p.wg.Wait()
	p.cancel()
}
```

### 1.4 Rate-Limited Concurrent Batch Processor

Combines ticker-based rate limiting with context-driven timeout management to prevent downstream API exhaustion during bulk operations.

```go
package ratelimit

import (
	"context"
	"sync"
	"time"
)

type BatchItem[T any] struct {
	Data T
}

func ProcessBatchInParallel[T any](
	ctx context.Context,
	items []BatchItem[T],
	maxOpsPerSecond int,
	processor func(context.Context, BatchItem[T]) error,
) []error {
	ticker := time.NewTicker(time.Second / time.Duration(maxOpsPerSecond))
	defer ticker.Stop()

	errs := make([]error, len(items))
	var wg sync.WaitGroup

	for i, item := range items {
		select {
		case <-ctx.Done():
			errs[i] = ctx.Err()
			continue
		case <-ticker.C:
		}

		idx := i
		itm := item
		wg.Go(func() {
			errs[idx] = processor(ctx, itm)
		})
	}

	wg.Wait()
	return errs
}
```

### 1.5 Short-Circuiting Task Execution with `errgroup` and Concurrency Limits

Utilizes `golang.org/x/sync/errgroup` with strict concurrency limits (`SetLimit`) to bound resource allocation while allowing early error termination.

```go
package taskrunner

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type WorkUnit struct {
	ID   int
	Path string
}

func ExecuteBoundedTasks(ctx context.Context, tasks []WorkUnit, maxConcurrency int) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	for _, task := range tasks {
		unit := task
		g.Go(func() error {
			if err := processUnit(gCtx, unit); err != nil {
				return fmt.Errorf("task %d failed on %s: %w", unit.ID, unit.Path, err)
			}
			return nil
		})
	}

	return g.Wait()
}

func processUnit(ctx context.Context, u WorkUnit) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Execute task operation
		return nil
	}
}
```

### 1.6 Singleflight Thundering Herd Protection for Concurrent Caches

Prevents cache stampedes under high concurrency by coalescing duplicate reads through singleflight execution.

```go
package cache

import (
	"context"
	"fmt"

	"golang.org/x/sync/singleflight"
)

type ItemFetcher[K comparable, V any] struct {
	group singleflight.Group
	store map[K]V
}

func NewItemFetcher[K comparable, V any]() *ItemFetcher[K, V] {
	return &ItemFetcher[K, V]{
		store: make(map[K]V),
	}
}

func (f *ItemFetcher[K, V]) FetchCoalesced(
	ctx context.Context,
	key K,
	fallback func(context.Context, K) (V, error),
) (V, error) {
	keyStr := fmt.Sprintf("%v", key)

	v, err, _ := f.group.Do(keyStr, func() (any, error) {
		return fallback(ctx, key)
	})

	if err != nil {
		var zero V
		return zero, err
	}

	return v.(V), nil
}
```

## Section 2: Production Leak Prevention and Diagnostics

### 2.1 The Green Tea GC Reachability Profiler

Go 1.26 uses Green Tea GC span scanning to detect blocked goroutines that are no longer accessible via active program roots.

#### Diagnostic HTTP Endpoint Integration

To expose production leak metrics via standard HTTP routers:

```go
package telemetry

import (
	"net/http"
	"net/http/pprof"
)

func RegisterDiagnosticHandlers(mux *http.ServeMux) {
	mux.Handle("/debug/pprof/goroutineleak", pprof.Handler("goroutineleak"))
}
```

#### Runtime Leak Verification Script

An engineer or agent can programmatically query the leak engine within operational maintenance runs:

```go
package telemetry_test

import (
	"bytes"
	"runtime/pprof"
	"testing"
)

func VerifyNoGoroutineLeaks(t *testing.T) {
	profile := pprof.Lookup("goroutineleak")
	if profile == nil {
		t.Skip("goroutineleak profile unavailable")
	}

	var buf bytes.Buffer
	if err := profile.WriteTo(&buf, 0); err != nil {
		t.Fatalf("failed to query goroutineleak profile: %v", err)
	}

	if buf.Len() > 0 && bytes.Contains(buf.Bytes(), []byte("goroutine")) {
		t.Fatalf("Leaked goroutines detected by runtime GC reachability engine:\n%s", buf.String())
	}
}
```

### 2.2 Atomic Pointer and Concurrent State Management (`sync/atomic`)

Thread-safe config updates without lock contention using `atomic.Pointer[T]`.

```go
package config

import (
	"sync/atomic"
)

type ServiceConfig struct {
	TimeoutMs int
	MaxConns  int
}

type ConfigManager struct {
	current atomic.Pointer[ServiceConfig]
}

func NewConfigManager(initial *ServiceConfig) *ConfigManager {
	m := &ConfigManager{}
	m.current.Store(initial)
	return m
}

func (m *ConfigManager) Get() *ServiceConfig {
	return m.current.Load()
}

func (m *ConfigManager) Swap(next *ServiceConfig) {
	m.current.Store(next)
}
```

## Section 3: Generic Error Propagation in Concurrent Networks

### 3.1 Type-Safe Error Handling with `errors.AsType`

When errors occur across distributed pipeline worker pools, extracting custom application errors must be done without allocating typed pointers.

```go
package processing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type DatabaseError struct {
	Query string
	Err   error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("db execution failure on query [%s]: %v", e.Query, e.Err)
}

func HandlePipelineErrors(ctx context.Context, errChan <-chan error) {
	for err := range errChan {
		if err == nil {
			continue
		}

		if dbErr, ok := errors.AsType[*DatabaseError](err); ok {
			slog.ErrorContext(ctx, "Database failure isolated",
				"query", dbErr.Query,
				"internal_err", dbErr.Err,
			)
			continue
		}

		slog.ErrorContext(ctx, "Unclassified pipeline execution failure", "error", err)
	}
}
```



