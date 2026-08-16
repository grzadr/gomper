package scanner_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

type lineErrorReader struct {
	err error
}

func (e *lineErrorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLines int
		wantErr   bool
	}{
		{
			name:      "Empty input",
			input:     "",
			wantLines: 0,
		},
		{
			name:      "Single line with newline",
			input:     "hello\n",
			wantLines: 1,
		},
		{
			name:      "Single line without newline",
			input:     "hello",
			wantLines: 1,
		},
		{
			name:      "Multiple lines with trailing newline",
			input:     "line 1\nline 2\nline 3\n",
			wantLines: 3,
		},
		{
			name:      "Multiple lines without trailing newline",
			input:     "line 1\nline 2\nline 3",
			wantLines: 3,
		},
		{
			name:      "Empty lines with newlines",
			input:     "\n\n\n",
			wantLines: 3,
		},
		{
			name:      "Large input across buffer boundary",
			input:     strings.Repeat("line with some text\n", 5000),
			wantLines: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scanner.CountLines(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CountLines() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantLines {
				t.Errorf("CountLines() = %d, want %d", got, tt.wantLines)
			}
		})
	}

	t.Run("Nil reader", func(t *testing.T) {
		got, err := scanner.CountLines(nil)
		if err != nil {
			t.Fatalf("unexpected error for nil reader: %v", err)
		}
		if got != 0 {
			t.Errorf("expected 0 lines for nil reader, got %d", got)
		}
	})

	t.Run("Reader error propagation", func(t *testing.T) {
		expectedErr := errors.New("read failed")
		_, err := scanner.CountLines(&lineErrorReader{err: expectedErr})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestExtractFileMetrics(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	content := "Line 1: quick brown fox\nLine 2: jumps over lazy dog\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create sample file: %v", err)
	}

	t.Run("Extracts lines and tokens correctly with default tokenizer", func(t *testing.T) {
		lines, tokens, err := scanner.ExtractFileMetrics(filePath, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lines != 2 {
			t.Errorf("expected 2 lines, got %d", lines)
		}
		// "Line" "1:" "quick" "brown" "fox" (5) + "Line" "2:" "jumps" "over" "lazy" "dog" (6) = 11
		if tokens != 11 {
			t.Errorf("expected 11 tokens, got %d", tokens)
		}
	})

	t.Run("Fails on non-existent file", func(t *testing.T) {
		_, _, err := scanner.ExtractFileMetrics(filepath.Join(tempDir, "non_existent.txt"), nil)
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})
}
