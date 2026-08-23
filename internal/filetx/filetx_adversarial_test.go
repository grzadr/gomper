package filetx_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/grzadr/gomper/internal/filetx"
)

func TestAdversarial_WriteAtomicallyWithResult_ErrorAndRollback(t *testing.T) {
	t.Run("Error in writeFunc removes tmp file and leaves target untouched", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "target_file.txt")

		// Create existing target file
		initialContent := "original pristine content"
		if err := os.WriteFile(targetPath, []byte(initialContent), 0644); err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		expectedErr := errors.New("deliberate writeFunc catastrophic failure")
		type ResultMeta struct {
			Records int
		}

		res, err := filetx.WriteAtomicallyWithResult(context.Background(), targetPath, func(ctx context.Context, w io.Writer) (ResultMeta, error) {
			_, _ = w.Write([]byte("corrupted partial payload"))
			return ResultMeta{Records: -1}, expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if res.Records != -1 {
			t.Errorf("expected zero/error result, got %+v", res)
		}

		// Verify existing target file content was preserved
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("target file was deleted or unreadable: %v", err)
		}
		if string(content) != initialContent {
			t.Errorf("target file content was corrupted! got %q, want %q", string(content), initialContent)
		}

		// Verify no temporary .tx-*.tmp files exist in tempDir
		entries, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("failed to read temp dir: %v", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tx-") {
				t.Errorf("leaked temporary file on error: %s", entry.Name())
			}
		}
	})

	t.Run("Concurrent atomic writes to distinct files", func(t *testing.T) {
		tempDir := t.TempDir()
		const numWriters = 30
		var wg sync.WaitGroup
		errChan := make(chan error, numWriters)

		for i := range numWriters {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent_%02d.txt", idx))
				expectedContent := fmt.Sprintf("content of writer %d", idx)

				tx := filetx.NewTx(filePath, func(ctx context.Context, w io.Writer) (int, error) {
					return w.Write([]byte(expectedContent))
				})

				n, err := tx.Execute(context.Background())
				if err != nil {
					errChan <- fmt.Errorf("writer %d failed: %w", idx, err)
					return
				}
				if n != len(expectedContent) {
					errChan <- fmt.Errorf("writer %d wrote %d bytes, want %d", idx, n, len(expectedContent))
					return
				}

				readBack, err := os.ReadFile(filePath)
				if err != nil {
					errChan <- fmt.Errorf("writer %d read error: %w", idx, err)
					return
				}
				if string(readBack) != expectedContent {
					errChan <- fmt.Errorf("writer %d data mismatch: got %q, want %q", idx, string(readBack), expectedContent)
					return
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			t.Errorf("concurrent write failure: %v", err)
		}

		// Ensure no leaked temp files
		entries, _ := os.ReadDir(tempDir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".tx-") {
				t.Errorf("leaked temporary file after concurrent writes: %s", e.Name())
			}
		}
	})

	t.Run("Context cancellation during write aborts transaction cleanly", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "cancel_test.txt")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := filetx.WriteAtomicallyWithResult(ctx, targetPath, func(ctx context.Context, w io.Writer) (string, error) {
			_, _ = w.Write([]byte("some data"))
			cancel() // cancel mid-write
			return "done", nil
		})

		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}

		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Errorf("target file should not exist on canceled context")
		}

		entries, _ := os.ReadDir(tempDir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".tx-") {
				t.Errorf("leaked temporary file on context cancel: %s", e.Name())
			}
		}
	})
}
