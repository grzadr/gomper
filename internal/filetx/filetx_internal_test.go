package filetx

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"generic error", errors.New("other error"), false},
		{"fs.ErrPermission", fs.ErrPermission, true},
		{"os.ErrPermission", os.ErrPermission, true},
		{"syscall.EACCES", syscall.EACCES, true},
		{"syscall.EPERM", syscall.EPERM, true},
		{"wrapped fs.ErrPermission", errors.Join(errors.New("wrap"), fs.ErrPermission), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermissionError(tt.err)
			if got != tt.expected {
				t.Errorf("isPermissionError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestWriteAtomically_HooksAndEdgeCases(t *testing.T) {
	t.Run("Fails when syncFileHook returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		target := filepath.Join(tempDir, "file.txt")

		origSync := syncFileHook
		syncFileHook = func(f *os.File) error {
			return errors.New("simulated sync error")
		}
		defer func() { syncFileHook = origSync }()

		err := WriteAtomically(context.Background(), target, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err == nil || !errors.Is(err, errors.New("simulated sync error")) && err.Error() != "failed to sync temporary file: simulated sync error" {
			t.Errorf("expected failed to sync temporary file error, got: %v", err)
		}
	})

	t.Run("Fails when closeFileHook returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		target := filepath.Join(tempDir, "file.txt")

		origClose := closeFileHook
		closeFileHook = func(f *os.File) error {
			_ = f.Close()
			return errors.New("simulated close error")
		}
		defer func() { closeFileHook = origClose }()

		err := WriteAtomically(context.Background(), target, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err == nil || err.Error() != "failed to close temporary file: simulated close error" {
			t.Errorf("expected failed to close temporary file error, got: %v", err)
		}
	})

	t.Run("Fails when openDirHook returns non-permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		target := filepath.Join(tempDir, "file.txt")

		origOpen := openDirHook
		openDirHook = func(name string) (*os.File, error) {
			return nil, errors.New("disk I/O error")
		}
		defer func() { openDirHook = origOpen }()

		err := WriteAtomically(context.Background(), target, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err == nil || err.Error() != "failed to open parent directory "+tempDir+": disk I/O error" {
			t.Errorf("expected failed to open parent directory error, got: %v", err)
		}
	})

	t.Run("Succeeds when openDirHook returns permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		target := filepath.Join(tempDir, "file.txt")

		origOpen := openDirHook
		openDirHook = func(name string) (*os.File, error) {
			return nil, os.ErrPermission
		}
		defer func() { openDirHook = origOpen }()

		err := WriteAtomically(context.Background(), target, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err != nil {
			t.Errorf("expected success with warning on open permission error, got: %v", err)
		}
	})

	t.Run("Succeeds when syncDirHook returns permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		target := filepath.Join(tempDir, "file.txt")

		origSyncDir := syncDirHook
		syncDirHook = func(f *os.File) error {
			return os.ErrPermission
		}
		defer func() { syncDirHook = origSyncDir }()

		err := WriteAtomically(context.Background(), target, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err != nil {
			t.Errorf("expected success with warning on sync permission error, got: %v", err)
		}
	})

	t.Run("Fails when syncDirHook returns non-permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		target := filepath.Join(tempDir, "file.txt")

		origSyncDir := syncDirHook
		syncDirHook = func(f *os.File) error {
			return errors.New("filesystem corruption")
		}
		defer func() { syncDirHook = origSyncDir }()

		err := WriteAtomically(context.Background(), target, func(ctx context.Context, w io.Writer) error {
			_, err := w.Write([]byte("data"))
			return err
		})

		if err == nil || err.Error() != "failed to sync parent directory "+tempDir+": filesystem corruption" {
			t.Errorf("expected failed to sync parent directory error, got: %v", err)
		}
	})
}
