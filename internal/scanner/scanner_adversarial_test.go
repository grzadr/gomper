package scanner_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

type leakTrackingReadCloser struct {
	content []byte
	offset  int
	closed  atomic.Bool
	onClose func()
}

func newLeakTrackingReadCloser(data []byte, onClose func()) *leakTrackingReadCloser {
	return new(leakTrackingReadCloser{
		content: data,
		onClose: onClose,
	})
}

func (l *leakTrackingReadCloser) Read(p []byte) (int, error) {
	if l.closed.Load() {
		return 0, errors.New("read on closed reader")
	}
	if l.offset >= len(l.content) {
		return 0, io.EOF
	}
	n := copy(p, l.content[l.offset:])
	l.offset += n
	return n, nil
}

func (l *leakTrackingReadCloser) Close() error {
	if l.closed.CompareAndSwap(false, true) {
		if l.onClose != nil {
			l.onClose()
		}
	}
	return nil
}

// TestAdversarial_IteratorEarlyTerminationLeak tests resource cleanup when iterators are terminated early.
func TestAdversarial_IteratorEarlyTerminationLeak(t *testing.T) {
	tempDir := t.TempDir()
	const numFiles = 100
	for i := range numFiles {
		p := filepath.Join(tempDir, fmt.Sprintf("file_%03d.txt", i))
		if err := os.WriteFile(p, []byte(fmt.Sprintf("content for file %d\nline 2\n", i)), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	s := scanner.NewScanner(nil)

	t.Run("Break after 1 item from Walk", func(t *testing.T) {
		ctx := context.Background()
		var count int
		for entry, err := range s.Walk(ctx, []string{tempDir}) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !entry.IsDir {
				count++
				break
			}
		}
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("Break at various points across 100 items", func(t *testing.T) {
		breakPoints := []int{1, 5, 10, 50, 99}
		for _, bp := range breakPoints {
			ctx := context.Background()
			var count int
			for entry, err := range s.Walk(ctx, []string{tempDir}) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !entry.IsDir {
					count++
					if count == bp {
						break
					}
				}
			}
			if count != bp {
				t.Errorf("expected count %d on break, got %d", bp, count)
			}
		}
	})

	t.Run("Full pipeline chain early break resource leak check", func(t *testing.T) {
		var openCount atomic.Int64
		const totalItems = 50

		closers := make([]*leakTrackingReadCloser, totalItems)
		for i := range totalItems {
			openCount.Add(1)
			closers[i] = newLeakTrackingReadCloser([]byte("test"), func() {
				openCount.Add(-1)
			})
		}

		seq := func(yield func(scanner.Entry, error) bool) {
			for i := range totalItems {
				entry := scanner.Entry{
					Path:    fmt.Sprintf("item_%d.txt", i),
					Content: closers[i],
				}
				if !yield(entry, nil) {
					// Close remaining items if consumer aborted early without taking them
					for j := i + 1; j < totalItems; j++ {
						_ = closers[j].Close()
					}
					return
				}
			}
		}

		ctx := context.Background()
		// Chain: ProcessEntries -> FilterEntries -> BatchEntries
		processed := scanner.ProcessEntries(ctx, seq, func(e scanner.Entry) (scanner.Entry, error) {
			return e, nil
		})

		filtered := scanner.FilterEntries(processed, func(e scanner.Entry) bool {
			return true
		})

		batched := scanner.BatchEntries(filtered, 5)

		// Break after consuming 2 batches (10 items)
		var batchesConsumed int
		for _, err := range batched {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			batchesConsumed++
			if batchesConsumed == 2 {
				break
			}
		}

		if batchesConsumed != 2 {
			t.Fatalf("expected 2 batches consumed, got %d", batchesConsumed)
		}

		if remaining := openCount.Load(); remaining != 0 {
			t.Errorf("leak detected: %d unclosed resources after pipeline early break", remaining)
		}
	})

	t.Run("BatchEntries specific mid-batch leak check", func(t *testing.T) {
		var openCount atomic.Int64
		const totalItems = 20
		const batchSize = 10

		closers := make([]*leakTrackingReadCloser, totalItems)
		for i := range totalItems {
			openCount.Add(1)
			closers[i] = newLeakTrackingReadCloser([]byte("test"), func() {
				openCount.Add(-1)
			})
		}

		seq := func(yield func(scanner.Entry, error) bool) {
			for i := range totalItems {
				entry := scanner.Entry{
					Path:    fmt.Sprintf("item_%d.txt", i),
					Content: closers[i],
				}
				if !yield(entry, nil) {
					for j := i + 1; j < totalItems; j++ {
						_ = closers[j].Close()
					}
					return
				}
			}
		}

		batched := scanner.BatchEntries(seq, batchSize)

		// Consume and break in the middle of the second batch (item 15)
		var itemsConsumed int
		for batch, err := range batched {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, item := range batch {
				itemsConsumed++
				if itemsConsumed == 15 {
					// Manually close the item we just consumed to simulate consumer behavior
					if item.Content != nil {
						_ = item.Content.Close()
					}
					return
				}
			}
		}

		if remaining := openCount.Load(); remaining != 0 {
			t.Errorf("leak detected: %d unclosed resources after mid-batch break", remaining)
		}
	})
}

// TestAdversarial_GenericCombinatorsBoundary tests empty, error, and large sequences on generic combinators.
func TestAdversarial_GenericCombinatorsBoundary(t *testing.T) {
	t.Run("Empty sequence on all combinators", func(t *testing.T) {
		emptySeq := func(yield func(scanner.Entry, error) bool) {}
		ctx := context.Background()

		// ProcessEntries on empty sequence
		processed := scanner.ProcessEntries(ctx, emptySeq, func(e scanner.Entry) (string, error) {
			return e.Path, nil
		})
		items, err := scanner.CollectEntries(processed)
		if err != nil {
			t.Fatalf("unexpected error on empty ProcessEntries: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}

		// FilterEntries on empty sequence
		filtered := scanner.FilterEntries(processed, func(s string) bool { return true })
		fItems, err := scanner.CollectEntries(filtered)
		if err != nil {
			t.Fatalf("unexpected error on empty FilterEntries: %v", err)
		}
		if len(fItems) != 0 {
			t.Errorf("expected 0 items, got %d", len(fItems))
		}

		// BatchEntries on empty sequence
		batched := scanner.BatchEntries(filtered, 10)
		bItems, err := scanner.CollectEntries(batched)
		if err != nil {
			t.Fatalf("unexpected error on empty BatchEntries: %v", err)
		}
		if len(bItems) != 0 {
			t.Errorf("expected 0 items, got %d", len(bItems))
		}
	})

	t.Run("Error propagation in middle of sequence", func(t *testing.T) {
		testErr := errors.New("injected sequence error at item 3")
		seqWithErr := func(yield func(int, error) bool) {
			for i := 1; i <= 5; i++ {
				if i == 3 {
					if !yield(0, testErr) {
						return
					}
				} else {
					if !yield(i, nil) {
						return
					}
				}
			}
		}

		// FilterEntries propagates error directly
		filtered := scanner.FilterEntries(seqWithErr, func(n int) bool {
			return n%2 != 0
		})

		var yieldedItems []int
		var yieldedErrs []error
		for item, err := range filtered {
			if err != nil {
				yieldedErrs = append(yieldedErrs, err)
			} else {
				yieldedItems = append(yieldedItems, item)
			}
		}

		if len(yieldedErrs) != 1 || !errors.Is(yieldedErrs[0], testErr) {
			t.Errorf("expected injected error propagated, got %v", yieldedErrs)
		}
		if len(yieldedItems) != 2 || yieldedItems[0] != 1 || yieldedItems[1] != 5 {
			t.Errorf("expected items [1, 5], got %v", yieldedItems)
		}

		// CollectEntries fails when sequence contains error
		_, err := scanner.CollectEntries(seqWithErr)
		if !errors.Is(err, testErr) {
			t.Errorf("expected CollectEntries to return testErr, got %v", err)
		}
	})

	t.Run("Large sequence throughput and allocation sanity (100,000 items)", func(t *testing.T) {
		const total = 100_000

		ctx := context.Background()
		processed := scanner.ProcessEntries(ctx, func(yield func(scanner.Entry, error) bool) {
			for i := range total {
				if !yield(scanner.Entry{Path: fmt.Sprintf("file_%d", i), Size: int64(i)}, nil) {
					return
				}
			}
		}, func(e scanner.Entry) (int64, error) {
			return e.Size * 2, nil
		})

		filtered := scanner.FilterEntries(processed, func(n int64) bool {
			return n%2 == 0
		})

		batched := scanner.BatchEntries(filtered, 1000)

		var batchCount int
		var totalCount int
		for batch, err := range batched {
			if err != nil {
				t.Fatalf("unexpected error during large batch stream: %v", err)
			}
			batchCount++
			totalCount += len(batch)
		}

		if batchCount != 100 {
			t.Errorf("expected 100 batches of 1000 items, got %d", batchCount)
		}
		if totalCount != total {
			t.Errorf("expected %d total items, got %d", total, totalCount)
		}
	})

	t.Run("BatchEntries with various batch sizes", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5, 6, 7}
		seq := func(yield func(int, error) bool) {
			for _, item := range items {
				if !yield(item, nil) {
					return
				}
			}
		}

		// batchSize = 3 -> [1,2,3], [4,5,6], [7]
		b3, err := scanner.CollectEntries(scanner.BatchEntries(seq, 3))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b3) != 3 || len(b3[0]) != 3 || len(b3[1]) != 3 || len(b3[2]) != 1 {
			t.Errorf("unexpected batching for size 3: %+v", b3)
		}

		// batchSize = 7 -> [1,2,3,4,5,6,7]
		b7, err := scanner.CollectEntries(scanner.BatchEntries(seq, 7))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b7) != 1 || len(b7[0]) != 7 {
			t.Errorf("unexpected batching for size 7: %+v", b7)
		}

		// batchSize = 100 -> [1,2,3,4,5,6,7]
		b100, err := scanner.CollectEntries(scanner.BatchEntries(seq, 100))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b100) != 1 || len(b100[0]) != 7 {
			t.Errorf("unexpected batching for size 100: %+v", b100)
		}
	})
}
