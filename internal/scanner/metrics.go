package scanner

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// Hooks for file operations to facilitate unit testing.
var (
	openFileHook = os.Open
	seekFileHook = func(f *os.File, offset int64, whence int) (int64, error) {
		return f.Seek(offset, whence)
	}
)

// CountLines reads from r and counts the number of lines.
// It counts newline ('\n') bytes and handles files without trailing newlines.
func CountLines(r io.Reader) (int, error) {
	if r == nil {
		return 0, nil
	}
	buf := make([]byte, 32*1024)
	var count int
	var hasBytes bool
	var lastByte byte = '\n'

	for {
		n, err := r.Read(buf)
		if n > 0 {
			hasBytes = true
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					count++
				}
			}
			lastByte = buf[n-1]
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}
	}
	if hasBytes && lastByte != '\n' {
		count++
	}
	return count, nil
}

// ExtractFileMetrics opens the file at path and calculates line and token counts.
func ExtractFileMetrics(path string, tokenizer Tokenizer) (lines int, tokens int, err error) {
	if tokenizer == nil {
		tokenizer = DefaultTokenizer()
	}

	f, err := openFileHook(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file %q for metrics: %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	lines, err = CountLines(f)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count lines for %q: %w", path, err)
	}

	if _, err := seekFileHook(f, 0, io.SeekStart); err != nil {
		return 0, 0, fmt.Errorf("failed to seek file %q for tokenization: %w", path, err)
	}

	tokens, err = tokenizer.CountTokens(f)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count tokens for %q: %w", path, err)
	}

	return lines, tokens, nil
}
