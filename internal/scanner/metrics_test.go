package scanner_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

type metricErrorReader struct {
	err error
}

func (e *metricErrorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestCountLinesAndTokens(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLines  int
		wantTokens int
		wantErr    bool
	}{
		{
			name:       "Empty input",
			input:      "",
			wantLines:  0,
			wantTokens: 0,
		},
		{
			name:       "Whitespace only",
			input:      "   \t\n\r\v\f  ",
			wantLines:  2,
			wantTokens: 0,
		},
		{
			name:       "Single line with newline",
			input:      "hello world\n",
			wantLines:  1,
			wantTokens: 2,
		},
		{
			name:       "Single line without newline",
			input:      "hello world",
			wantLines:  1,
			wantTokens: 2,
		},
		{
			name:       "Multiple lines with trailing newline",
			input:      "line 1: quick brown fox\nline 2: jumps over lazy dog\n",
			wantLines:  2,
			wantTokens: 11,
		},
		{
			name:       "Multiple lines without trailing newline",
			input:      "line 1: alpha\nline 2: beta\nline 3: gamma",
			wantLines:  3,
			wantTokens: 9,
		},
		{
			name:       "Empty lines with newlines",
			input:      "\n\n\n",
			wantLines:  3,
			wantTokens: 0,
		},
		{
			name:       "Large input across buffer boundary",
			input:      strings.Repeat("word1 word2 word3\n", 5000),
			wantLines:  5000,
			wantTokens: 15000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, tokens, err := scanner.CountLinesAndTokens(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CountLinesAndTokens() error = %v, wantErr %v", err, tt.wantErr)
			}
			if lines != tt.wantLines {
				t.Errorf("CountLinesAndTokens() lines = %d, want %d", lines, tt.wantLines)
			}
			if tokens != tt.wantTokens {
				t.Errorf("CountLinesAndTokens() tokens = %d, want %d", tokens, tt.wantTokens)
			}
		})
	}

	t.Run("Nil reader", func(t *testing.T) {
		lines, tokens, err := scanner.CountLinesAndTokens(nil)
		if err != nil {
			t.Fatalf("unexpected error for nil reader: %v", err)
		}
		if lines != 0 || tokens != 0 {
			t.Errorf("expected 0 lines and 0 tokens for nil reader, got lines=%d tokens=%d", lines, tokens)
		}
	})

	t.Run("Reader error propagation", func(t *testing.T) {
		expectedErr := errors.New("stream read failure")
		_, _, err := scanner.CountLinesAndTokens(&metricErrorReader{err: expectedErr})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
