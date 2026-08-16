package scanner

import (
	"errors"
	"io"
)

// CountLinesAndTokens reads from r in a single pass using a 32KB buffer,
// counting both the number of lines and whitespace-delimited tokens simultaneously.
func CountLinesAndTokens(r io.Reader) (lines int, tokens int, err error) {
	if r == nil {
		return 0, 0, nil
	}
	buf := make([]byte, 32*1024)
	var inToken bool
	var lastByte byte = '\n'
	var hasBytes bool

	for {
		n, err := r.Read(buf)
		if n > 0 {
			hasBytes = true
			for i := 0; i < n; i++ {
				b := buf[i]
				if b == '\n' {
					lines++
				}
				if b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f' {
					inToken = false
				} else if !inToken {
					inToken = true
					tokens++
				}
			}
			lastByte = buf[n-1]
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
