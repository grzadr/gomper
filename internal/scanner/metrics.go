package scanner

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

var newlineSlice = []byte{'\n'}

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// CountLinesAndTokens reads from r using a pooled 32KB buffer, leveraging SIMD-accelerated
// line counting (bytes.Count) and an optimized L1-hot jump-table state machine for whitespace tokens.
func CountLinesAndTokens(r io.Reader) (lines int, tokens int, err error) {
	if r == nil {
		return 0, 0, nil
	}

	bufPtr := bufferPool.Get().(*[]byte)
	defer func() {
		// Reset slice length to full capacity to prevent degradation on reuse
		*bufPtr = (*bufPtr)[:cap(*bufPtr)]
		bufferPool.Put(bufPtr)
	}()

	buf := *bufPtr
	var inToken bool
	var lastByte byte = '\n'
	var hasBytes bool

	for {
		n, err := r.Read(buf)
		if n > 0 {
			hasBytes = true
			chunk := buf[:n]

			// 1. SIMD-accelerated line count
			lines += bytes.Count(chunk, newlineSlice)

			// 2. L1-hot cache token pass using jump tables
			for i := range n {
				switch chunk[i] {
				case ' ', '\t', '\n', '\r', '\v', '\f':
					inToken = false
				default:
					if !inToken {
						inToken = true
						tokens++
					}
				}
			}
			lastByte = chunk[n-1]
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, 0, err
		}
	}
	if hasBytes && lastByte != '\n' {
		lines++
	}
	return lines, tokens, nil
}
