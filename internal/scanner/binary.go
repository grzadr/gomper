package scanner

import (
	"bytes"
	"io"
	"os"
	"unicode/utf8"
)

const sniffBufferSize = 8192

// Hook for opening files during binary detection to allow test error injection.
var openBinaryFileHook = func(name string) (*os.File, error) {
	return os.Open(name)
}

// IsBinaryReader inspects up to 8192 bytes from r to determine if content is binary.
// Returns true if a NUL byte or invalid UTF-8 sequence is encountered.
func IsBinaryReader(r io.Reader) (bool, error) {
	buf := make([]byte, sniffBufferSize)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	if n == 0 {
		return false, nil
	}

	data := buf[:n]
	if bytes.IndexByte(data, 0) != -1 {
		return true, nil
	}

	// If buffer was fully saturated, handle potential truncated multi-byte UTF-8 rune at boundary
	if n == sniffBufferSize {
		for i := 1; i <= 3 && i <= n; i++ {
			if (data[n-i] & 0xC0) == 0xC0 {
				expectedLen := 2
				if (data[n-i] & 0xE0) == 0xE0 {
					expectedLen = 3
				}
				if (data[n-i] & 0xF0) == 0xF0 {
					expectedLen = 4
				}
				if i < expectedLen {
					data = data[:n-i]
				}
				break
			}
		}
	}

	return !utf8.Valid(data), nil
}

// OpenAndSniff opens a file, reads up to 8KB to detect binary data.
// If it is text, it returns a reconstituted io.ReadCloser combining sniffed bytes and remaining file content.
func OpenAndSniff(path string) (bool, io.ReadCloser, error) {
	file, err := openBinaryFileHook(path)
	if err != nil {
		return false, nil, err
	}

	buf := make([]byte, sniffBufferSize)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		_ = file.Close()
		return false, nil, err
	}

	if n == 0 {
		return false, file, nil
	}

	data := buf[:n]
	if bytes.IndexByte(data, 0) != -1 {
		_ = file.Close()
		return true, nil, nil
	}

	validData := data
	if n == sniffBufferSize {
		for i := 1; i <= 3 && i <= n; i++ {
			if (validData[n-i] & 0xC0) == 0xC0 {
				expectedLen := 2
				if (validData[n-i] & 0xE0) == 0xE0 {
					expectedLen = 3
				}
				if (validData[n-i] & 0xF0) == 0xF0 {
					expectedLen = 4
				}
				if i < expectedLen {
					validData = validData[:n-i]
				}
				break
			}
		}
	}

	if !utf8.Valid(validData) {
		_ = file.Close()
		return true, nil, nil
	}

	// Reconstitute the file stream: sniffed bytes + remaining unread file bytes
	mr := io.MultiReader(bytes.NewReader(data), file)
	rc := struct {
		io.Reader
		io.Closer
	}{mr, file}

	return false, rc, nil
}
