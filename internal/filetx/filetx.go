package filetx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

// File and directory operation hooks for deterministic error injection in unit tests.
var (
	syncFileHook  = func(f *os.File) error { return f.Sync() }
	closeFileHook = func(f *os.File) error { return f.Close() }
	openDirHook   = func(name string) (*os.File, error) { return os.Open(name) }
	syncDirHook   = func(f *os.File) error { return f.Sync() }
)

// Tx encapsulates an atomic transactional file write returning a strongly typed result of type T.
type Tx[T any] struct {
	targetPath string
	writeFunc  func(ctx context.Context, w io.Writer) (T, error)
}

// NewTx creates a new Tx transaction instance.
func NewTx[T any](targetPath string, writeFunc func(ctx context.Context, w io.Writer) (T, error)) *Tx[T] {
	return new(Tx[T]{
		targetPath: targetPath,
		writeFunc:  writeFunc,
	})
}

// Execute performs the atomic file write operation and returns the result T.
func (tx *Tx[T]) Execute(ctx context.Context) (T, error) {
	if tx == nil {
		var zero T
		return zero, errors.New("filetx: nil transaction")
	}
	return WriteAtomicallyWithResult(ctx, tx.targetPath, tx.writeFunc)
}

// WriteAtomicallyWithResult executes writeFunc against a temporary file, atomically swaps it,
// syncs the directory, and returns writeFunc's typed result T.
func WriteAtomicallyWithResult[T any](ctx context.Context, targetPath string, writeFunc func(ctx context.Context, w io.Writer) (T, error)) (result T, err error) {
	if err = ctx.Err(); err != nil {
		return result, err
	}

	targetPath = filepath.Clean(targetPath)
	dir := filepath.Dir(targetPath)

	if err = os.MkdirAll(dir, 0755); err != nil {
		return result, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tx-*.tmp")
	if err != nil {
		return result, fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}

	defer func() {
		_ = tmpFile.Close()
		if err != nil {
			_ = os.Remove(tmpFile.Name())
		}
	}()

	if writeFunc != nil {
		result, err = writeFunc(ctx, tmpFile)
		if err != nil {
			return result, err
		}
	}

	if err = ctx.Err(); err != nil {
		return result, err
	}

	if err = syncFileHook(tmpFile); err != nil {
		return result, fmt.Errorf("failed to sync temporary file: %w", err)
	}

	if err = closeFileHook(tmpFile); err != nil {
		return result, fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = os.Rename(tmpFile.Name(), targetPath); err != nil {
		return result, fmt.Errorf("failed to rename temporary file to %s: %w", targetPath, err)
	}

	dirFile, err := openDirHook(dir)
	if err != nil {
		if isPermissionError(err) {
			slog.WarnContext(ctx, "failed to open parent directory for sync due to insufficient permissions",
				slog.String("directory", dir),
				slog.Any("error", err),
			)
			return result, nil
		}
		return result, fmt.Errorf("failed to open parent directory %s: %w", dir, err)
	}
	defer func() { _ = dirFile.Close() }()

	if err = syncDirHook(dirFile); err != nil {
		if isPermissionError(err) {
			slog.WarnContext(ctx, "failed to sync parent directory due to insufficient permissions",
				slog.String("directory", dir),
				slog.Any("error", err),
			)
			return result, nil
		}
		return result, fmt.Errorf("failed to sync parent directory %s: %w", dir, err)
	}

	return result, nil
}

// WriteAtomically writes content to targetPath in a fully atomic, transactional manner.
// It creates a temporary file in targetPath's directory, streams data via writeFunc,
// syncs the file, closes it, renames it atomically to targetPath, and syncs the parent directory.
// It respects context cancellation at all stages and cleans up temporary files on failure.
func WriteAtomically(ctx context.Context, targetPath string, writeFunc func(ctx context.Context, w io.Writer) error) error {
	_, err := WriteAtomicallyWithResult(ctx, targetPath, func(ctx context.Context, w io.Writer) (struct{}, error) {
		if writeFunc == nil {
			return struct{}{}, nil
		}
		return struct{}{}, writeFunc(ctx, w)
	})
	return err
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	return os.IsPermission(err) || errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}
