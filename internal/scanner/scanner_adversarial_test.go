package scanner_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

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
}
