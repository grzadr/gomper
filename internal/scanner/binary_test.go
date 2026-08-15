package scanner_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestIsBinaryReader(t *testing.T) {
	t.Run("Empty reader returns false", func(t *testing.T) {
		isBin, err := scanner.IsBinaryReader(bytes.NewReader([]byte{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected empty reader to be non-binary")
		}
	})

	t.Run("Plain ASCII text returns false", func(t *testing.T) {
		isBin, err := scanner.IsBinaryReader(strings.NewReader("Hello, world! 12345\nSecond line.\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected plain ASCII text to be non-binary")
		}
	})

	t.Run("Valid UTF-8 with multi-byte runes returns false", func(t *testing.T) {
		content := "Hello 世界! 🚀 Accents: é, à, ü, ñ. Euro: €100."
		isBin, err := scanner.IsBinaryReader(strings.NewReader(content))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected valid UTF-8 text to be non-binary")
		}
	})

	t.Run("Binary content containing NUL byte returns true", func(t *testing.T) {
		data := []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01, 0x02}
		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isBin {
			t.Errorf("expected NUL-containing content to be binary")
		}
	})

	t.Run("Binary content with invalid UTF-8 sequence returns true", func(t *testing.T) {
		data := []byte{0xff, 0xfe, 0xfd, 0x80, 0x81}
		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isBin {
			t.Errorf("expected invalid UTF-8 sequence to be binary")
		}
	})

	t.Run("Reader error is returned", func(t *testing.T) {
		_, err := scanner.IsBinaryReader(errReader{})
		if err == nil {
			t.Fatalf("expected error from errReader, got nil")
		}
	})

	t.Run("Saturated buffer with valid UTF-8 returns false", func(t *testing.T) {
		data := bytes.Repeat([]byte("a"), 8192)
		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected 8192-byte ASCII text to be non-binary")
		}
	})

	t.Run("Saturated buffer with 2-byte UTF-8 rune split at boundary returns false", func(t *testing.T) {
		// 8191 ASCII bytes followed by 1st byte of 2-byte rune (0xc3)
		data := bytes.Repeat([]byte("a"), 8191)
		data = append(data, 0xc3)
		if len(data) != 8192 {
			t.Fatalf("expected length 8192, got %d", len(data))
		}

		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected boundary-split 2-byte UTF-8 rune to be recognized as non-binary")
		}
	})

	t.Run("Saturated buffer with 3-byte UTF-8 rune split at boundary returns false", func(t *testing.T) {
		// 8190 ASCII bytes followed by first 2 bytes of 3-byte rune (0xe2, 0x82)
		data := bytes.Repeat([]byte("a"), 8190)
		data = append(data, 0xe2, 0x82)
		if len(data) != 8192 {
			t.Fatalf("expected length 8192, got %d", len(data))
		}

		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected boundary-split 3-byte UTF-8 rune to be recognized as non-binary")
		}
	})

	t.Run("Saturated buffer with 4-byte UTF-8 rune split at boundary returns false", func(t *testing.T) {
		// 8189 ASCII bytes followed by first 3 bytes of 4-byte rune (0xf0, 0x9f, 0x98)
		data := bytes.Repeat([]byte("a"), 8189)
		data = append(data, 0xf0, 0x9f, 0x98)
		if len(data) != 8192 {
			t.Fatalf("expected length 8192, got %d", len(data))
		}

		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected boundary-split 4-byte UTF-8 rune to be recognized as non-binary")
		}
	})

	t.Run("Saturated buffer with invalid UTF-8 returns true", func(t *testing.T) {
		data := bytes.Repeat([]byte{0x80}, 8192)
		isBin, err := scanner.IsBinaryReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isBin {
			t.Errorf("expected 8192-byte invalid UTF-8 buffer to be binary")
		}
	})
}

func TestIsBinaryFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Identifies text file", func(t *testing.T) {
		textFile := filepath.Join(tempDir, "sample.txt")
		if err := os.WriteFile(textFile, []byte("valid text content\n"), 0644); err != nil {
			t.Fatalf("failed to write text file: %v", err)
		}

		isBin, err := scanner.IsBinaryFile(textFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isBin {
			t.Errorf("expected sample.txt to be non-binary")
		}
	})

	t.Run("Identifies binary file with NUL byte", func(t *testing.T) {
		binFile := filepath.Join(tempDir, "app.bin")
		if err := os.WriteFile(binFile, []byte{0x00, 0x01, 0x02, 0x03}, 0755); err != nil {
			t.Fatalf("failed to write binary file: %v", err)
		}

		isBin, err := scanner.IsBinaryFile(binFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isBin {
			t.Errorf("expected app.bin to be binary")
		}
	})

	t.Run("Returns error for non-existent file", func(t *testing.T) {
		_, err := scanner.IsBinaryFile(filepath.Join(tempDir, "non_existent.bin"))
		if err == nil {
			t.Fatalf("expected error for non-existent file, got nil")
		}
	})
}

func TestWalkPaths_BinaryExclusion(t *testing.T) {
	t.Run("Single binary file path is skipped", func(t *testing.T) {
		tempDir := t.TempDir()
		binFile := filepath.Join(tempDir, "executable")
		if err := os.WriteFile(binFile, []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01}, 0755); err != nil {
			t.Fatalf("failed to create binary file: %v", err)
		}

		ctx := context.Background()
		var count int
		for _, err := range scanner.WalkPaths(ctx, []string{binFile}, nil) {
			if err == nil {
				count++
			}
		}

		if count != 0 {
			t.Errorf("expected single binary file to be skipped, got count %d", count)
		}
	})

	t.Run("Directory walk ignores binary files while keeping text files", func(t *testing.T) {
		tempDir := t.TempDir()
		textFile := filepath.Join(tempDir, "main.go")
		binFile := filepath.Join(tempDir, "main")
		subDir := filepath.Join(tempDir, "pkg")
		subTextFile := filepath.Join(subDir, "helper.go")
		subBinFile := filepath.Join(subDir, "lib.so")

		_ = os.Mkdir(subDir, 0755)
		_ = os.WriteFile(textFile, []byte("package main\n"), 0644)
		_ = os.WriteFile(binFile, []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01}, 0755)
		_ = os.WriteFile(subTextFile, []byte("package pkg\n"), 0644)
		_ = os.WriteFile(subBinFile, []byte{0x00, 0x00, 0x00, 0x00}, 0644)

		ctx := context.Background()
		var paths []string
		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				t.Fatalf("unexpected walk error: %v", err)
			}
			paths = append(paths, entry.Path)
		}

		for _, p := range paths {
			base := filepath.Base(p)
			if base == "main" || base == "lib.so" {
				t.Errorf("expected binary file %s to be implicitly ignored, got listed", p)
			}
		}

		var foundMain, foundHelper bool
		for _, p := range paths {
			if filepath.Base(p) == "main.go" {
				foundMain = true
			}
			if filepath.Base(p) == "helper.go" {
				foundHelper = true
			}
		}

		if !foundMain || !foundHelper {
			t.Errorf("expected text files to be listed, paths found: %v", paths)
		}
	})
}
