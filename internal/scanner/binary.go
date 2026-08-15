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

	if utf8.Valid(data) {
		return false, nil
	}

	// If buffer was fully saturated, handle potential truncated multi-byte UTF-8 rune at boundary
	if n == sniffBufferSize {
		for i := 1; i <= 3 && i <= n; i++ {
			if utf8.Valid(data[:n-i]) {
				return false, nil
			}
		}
	}

	return true, nil
}

// IsBinaryFile checks if the target file at path contains binary data.
func IsBinaryFile(path string) (bool, error) {
	file, err := openBinaryFileHook(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = file.Close() }()

	return IsBinaryReader(file)
}
