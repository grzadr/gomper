package scanner_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestWhitespaceTokenizer_CountTokens(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "Empty input",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "Whitespace only",
			input:     "   \t\n\r\v\f  ",
			wantCount: 0,
		},
		{
			name:      "Single token",
			input:     "hello",
			wantCount: 1,
		},
		{
			name:      "Multiple tokens separated by various whitespace",
			input:     "  hello \t world \n foo \r\n bar \v baz \f qux  ",
			wantCount: 6,
		},
		{
			name:      "Large text with repeating words",
			input:     strings.Repeat("word1 word2 word3\n", 1000),
			wantCount: 3000,
		},
	}

	tokenizer := scanner.NewWhitespaceTokenizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenizer.CountTokens(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CountTokens() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantCount {
				t.Errorf("CountTokens() = %d, want %d", got, tt.wantCount)
			}
		})
	}

	t.Run("Nil reader", func(t *testing.T) {
		got, err := tokenizer.CountTokens(nil)
		if err != nil {
			t.Fatalf("unexpected error for nil reader: %v", err)
		}
		if got != 0 {
			t.Errorf("expected 0 for nil reader, got %d", got)
		}
	})

	t.Run("Reader error propagation", func(t *testing.T) {
		expectedErr := errors.New("read failure")
		_, err := tokenizer.CountTokens(&errorReader{err: expectedErr})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestDefaultTokenizer(t *testing.T) {
	tok := scanner.DefaultTokenizer()
	if tok == nil {
		t.Fatal("expected non-nil default tokenizer")
	}

	count, err := tok.CountTokens(strings.NewReader("one two three"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 tokens, got %d", count)
	}
}
