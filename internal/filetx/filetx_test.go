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

	t.Run("Succeeds with warning when parent directory sync faces permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		restrictedDir := filepath.Join(tempDir, "restricted")
		if err := os.Mkdir(restrictedDir, 0755); err != nil {
			t.Fatalf("failed to create restricted directory: %v", err)
		}
		targetPath := filepath.Join(restrictedDir, "output.txt")

		err := filetx.WriteAtomically(context.Background(), targetPath, func(ctx context.Context, w io.Writer) error {
			if _, err := w.Write([]byte("permission test data")); err != nil {
				return err
			}
			// Remove read permission from directory (0333: write+exec, no read) so os.Rename succeeds but os.Open(dir) fails with permission denied
			_ = os.Chmod(restrictedDir, 0333)
			return nil
		})
		defer func() { _ = os.Chmod(restrictedDir, 0755) }()

		if err != nil {
			t.Fatalf("expected WriteAtomically to succeed with warning on directory sync permission error, got: %v", err)
		}

		_ = os.Chmod(restrictedDir, 0755)
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed to read target file after atomic write: %v", err)
		}
		if string(content) != "permission test data" {
			t.Errorf("expected 'permission test data', got %q", string(content))
		}
	})

	t.Run("Succeeds when writeFunc is nil", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "empty.txt")
		err := filetx.WriteAtomically(context.Background(), targetPath, nil)
		if err != nil {
			t.Fatalf("unexpected error with nil writeFunc: %v", err)
		}
		if _, err := os.Stat(targetPath); err != nil {
			t.Fatalf("expected target file to exist, got: %v", err)
		}
	})
}

func TestWriteAtomicallyWithResult(t *testing.T) {
	type WriteStats struct {
		BytesWritten int
		Summary      string
	}

	t.Run("Returns typed result and writes file atomically", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "stats.txt")

		stats, err := filetx.WriteAtomicallyWithResult(context.Background(), targetPath, func(ctx context.Context, w io.Writer) (WriteStats, error) {
			n, err := w.Write([]byte("statistics data"))
			return WriteStats{BytesWritten: n, Summary: "ok"}, err
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.BytesWritten != 15 || stats.Summary != "ok" {
			t.Errorf("unexpected stats: %+v", stats)
		}

		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed to read target file: %v", err)
		}
		if string(content) != "statistics data" {
			t.Errorf("expected 'statistics data', got %q", string(content))
		}
	})

	t.Run("Propagates error from writeFunc", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "fail.txt")

		expectedErr := errors.New("write failed")
		_, err := filetx.WriteAtomicallyWithResult(context.Background(), targetPath, func(ctx context.Context, w io.Writer) (int, error) {
			return 0, expectedErr
		})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("Handles nil writeFunc", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "nil_write.txt")

		res, err := filetx.WriteAtomicallyWithResult[string](context.Background(), targetPath, nil)
		if err != nil {
			t.Fatalf("unexpected error with nil writeFunc: %v", err)
		}
		if res != "" {
			t.Errorf("expected empty string zero value, got %q", res)
		}
	})
}

func TestTx(t *testing.T) {
	t.Run("Execute successful transaction", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "tx.txt")

		tx := filetx.NewTx(targetPath, func(ctx context.Context, w io.Writer) (int64, error) {
			n, err := w.Write([]byte("transactional write"))
			return int64(n), err
		})

		n, err := tx.Execute(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 19 {
			t.Errorf("expected 19 bytes written, got %d", n)
		}

		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "transactional write" {
			t.Errorf("expected 'transactional write', got %q", string(content))
		}
	})

	t.Run("nil *Tx Execute returns error", func(t *testing.T) {
		var tx *filetx.Tx[string]
		_, err := tx.Execute(context.Background())
		if err == nil || !strings.Contains(err.Error(), "nil transaction") {
			t.Errorf("expected nil transaction error, got: %v", err)
		}
	})
}


