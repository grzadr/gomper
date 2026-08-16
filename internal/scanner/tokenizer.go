package scanner

import (
	"errors"
	"io"
)

// Tokenizer defines the contract for counting tokens in file content.
type Tokenizer interface {
	CountTokens(r io.Reader) (int, error)
}

// WhitespaceTokenizer implements Tokenizer by counting whitespace-delimited tokens.
type WhitespaceTokenizer struct{}

// NewWhitespaceTokenizer creates a new WhitespaceTokenizer instance.
func NewWhitespaceTokenizer() *WhitespaceTokenizer {
	return &WhitespaceTokenizer{}
}

// CountTokens counts whitespace-delimited tokens in the provided reader.
func (t *WhitespaceTokenizer) CountTokens(r io.Reader) (int, error) {
	if r == nil {
		return 0, nil
	}
	buf := make([]byte, 32*1024)
	count := 0
	inWord := false

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := buf[i]
				isSpace := b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f'
				if !isSpace {
					if !inWord {
						inWord = true
						count++
					}
				} else {
					inWord = false
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}
	}
	return count, nil
}

// DefaultTokenizer returns the default Tokenizer instance.
func DefaultTokenizer() Tokenizer {
	return NewWhitespaceTokenizer()
}
