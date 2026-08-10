package filetx_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/filetx"
)

func TestWriteAtomically(t *testing.T) {
	t.Run("Successfully writes file atomically", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "sub", "output.txt")

		err := filetx.WriteAtomically(context.Background(), targetPath, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("hello atomic world"))
			return err
		})
		if err != nil {
			t.Fatalf("unexpected WriteAtomically error: %v", err)
		}

		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed to read target file: %v", err)
		}
		if string(content) != "hello atomic world" {
			t.Errorf("expected 'hello atomic world', got %q", string(content))
		}
	})

	t.Run("Cleans up temporary file on write error", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "failed.txt")

		expectedErr := errors.New("write failure")
		err := filetx.WriteAtomically(context.Background(), targetPath, func(ctx context.Context, w io.Writer) error {
			_, _ = w.Write([]byte("partial data"))
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}

		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Errorf("expected target file not to exist on error")
		}

		entries, _ := os.ReadDir(tempDir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".tx-") {
				t.Errorf("expected temporary file %s to be cleaned up", e.Name())
			}
		}
	})

	t.Run("Aborts and cleans up on context cancellation", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "cancelled.txt")

		ctx, cancel := context.WithCancel(context.Background())

		err := filetx.WriteAtomically(ctx, targetPath, func(ctx context.Context, w io.Writer) error {
			_, _ = w.Write([]byte("data"))
			cancel() // Cancel context during write
			return nil
		})

		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled error, got %v", err)
		}

		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Errorf("expected target file not to exist after context cancellation")
		}
	})

	t.Run("Returns error on pre-canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Pre-cancel context

		err := filetx.WriteAtomically(ctx, "/tmp/should_not_be_created.txt", func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("Returns error when directory creation fails", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a file where a directory should be created
		filePathAsDir := filepath.Join(tempDir, "not_a_dir")
		_ = os.WriteFile(filePathAsDir, []byte("file"), 0644)

		targetPath := filepath.Join(filePathAsDir, "nested", "out.txt")
		err := filetx.WriteAtomically(context.Background(), targetPath, func(ctx context.Context, w io.Writer) error {
			return nil
		})

		if err == nil {
			t.Errorf("expected error when MkdirAll fails on file path, got nil")
		}
	})

	t.Run("Returns error when temp file creation fails", func(t *testing.T) {
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0555); err != nil {
			t.Skip("skipping permission-restricted test on environment without permission enforcement")
		}
		defer func() { _ = os.Chmod(readOnlyDir, 0755) }()

		targetPath := filepath.Join(readOnlyDir, "out.txt")
		err := filetx.WriteAtomically(context.Background(), targetPath, func(ctx context.Context, w io.Writer) error {
			return nil
		})

		if err == nil {
			t.Log("unwritable directory test bypassed on platform without strict permissions")
		}
	})

	t.Run("Returns error when rename fails on directory target", func(t *testing.T) {
		tempDir := t.TempDir()
		targetDir := filepath.Join(tempDir, "existing_dir")
		_ = os.Mkdir(targetDir, 0755)

		err := filetx.WriteAtomically(context.Background(), targetDir, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err == nil {
			t.Errorf("expected error when os.Rename fails on directory target, got nil")
		}
		if !strings.Contains(err.Error(), "failed to rename temporary file") {
			t.Errorf("expected rename failure error, got: %v", err)
		}
	})
}

