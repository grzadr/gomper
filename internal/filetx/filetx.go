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

// WriteAtomically writes content to targetPath in a fully atomic, transactional manner.
// It creates a temporary file in targetPath's directory, streams data via writeFunc,
// syncs the file, closes it, renames it atomically to targetPath, and syncs the parent directory.
// It respects context cancellation at all stages and cleans up temporary files on failure.
func WriteAtomically(ctx context.Context, targetPath string, writeFunc func(ctx context.Context, w io.Writer) error) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}

	targetPath = filepath.Clean(targetPath)
	dir := filepath.Dir(targetPath)

	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tx-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}

	defer func() {
		_ = tmpFile.Close()
		if err != nil {
			_ = os.Remove(tmpFile.Name())
		}
	}()

	if err = writeFunc(ctx, tmpFile); err != nil {
		return err
	}

	if err = ctx.Err(); err != nil {
		return err
	}

	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = os.Rename(tmpFile.Name(), targetPath); err != nil {
		return fmt.Errorf("failed to rename temporary file to %s: %w", targetPath, err)
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		if isPermissionError(err) {
			slog.WarnContext(ctx, "failed to open parent directory for sync due to insufficient permissions",
				slog.String("directory", dir),
				slog.Any("error", err),
			)
			return nil
		}
		return fmt.Errorf("failed to open parent directory %s: %w", dir, err)
	}
	defer func() { _ = dirFile.Close() }()

	if err = dirFile.Sync(); err != nil {
		if isPermissionError(err) {
			slog.WarnContext(ctx, "failed to sync parent directory due to insufficient permissions",
				slog.String("directory", dir),
				slog.Any("error", err),
			)
			return nil
		}
		return fmt.Errorf("failed to sync parent directory %s: %w", dir, err)
	}

	return nil
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) || errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrPermission) {
		return true
	}
	if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}
