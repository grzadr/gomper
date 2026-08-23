package scanner_test

import (
	"context"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

func BenchmarkGenericPipeline_ProcessAndFilter(b *testing.B) {
	const count = 1000
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		seq := func(yield func(scanner.Entry, error) bool) {
			for i := range count {
				if !yield(scanner.Entry{Path: "main.go", Size: int64(i)}, nil) {
					return
				}
			}
		}

		processed := scanner.ProcessEntries(ctx, seq, func(e scanner.Entry) (int64, error) {
			return e.Size * 2, nil
		})

		filtered := scanner.FilterEntries(processed, func(s int64) bool {
			return s%2 == 0
		})

		for _, err := range filtered {
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}

func BenchmarkGenericPipeline_BatchEntries(b *testing.B) {
	const count = 1000
	const batchSize = 50

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		seq := func(yield func(int, error) bool) {
			for i := range count {
				if !yield(i, nil) {
					return
				}
			}
		}

		batched := scanner.BatchEntries(seq, batchSize)
		for _, err := range batched {
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}
