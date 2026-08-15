package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkPaths_BinaryHookErrors(t *testing.T) {
	origHook := openBinaryFileHook
	defer func() { openBinaryFileHook = origHook }()

	tempDir := t.TempDir()
	singleFile := filepath.Join(tempDir, "target.txt")
	if err := os.WriteFile(singleFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("Single file open error with yield returning true", func(t *testing.T) {
		openBinaryFileHook = func(name string) (*os.File, error) {
			return nil, errors.New("simulated single file open error")
		}

		ctx := context.Background()
		var errCount int
		for _, err := range WalkPaths(ctx, []string{singleFile}, nil) {
			if err != nil {
				errCount++
			}
		}

		if errCount != 1 {
			t.Errorf("expected 1 error yielded for single file open error, got %d", errCount)
		}
	})

	t.Run("Single file open error with yield returning false breaks early", func(t *testing.T) {
		openBinaryFileHook = func(name string) (*os.File, error) {
			return nil, errors.New("simulated single file open error")
		}

		ctx := context.Background()
		var errCount int
		for _, err := range WalkPaths(ctx, []string{singleFile, singleFile}, nil) {
			if err != nil {
				errCount++
				break
			}
		}

		if errCount != 1 {
			t.Errorf("expected 1 error on early break, got %d", errCount)
		}
	})

	t.Run("Directory walk file open error with yield returning true continues", func(t *testing.T) {
		openBinaryFileHook = func(name string) (*os.File, error) {
			return nil, errors.New("simulated dir walk file open error")
		}

		ctx := context.Background()
		var errCount int
		for _, err := range WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				errCount++
			}
		}

		if errCount < 1 {
			t.Errorf("expected at least 1 error yielded for dir walk file open error, got %d", errCount)
		}
	})

	t.Run("Directory walk file open error with yield returning false breaks walk (fs.SkipAll)", func(t *testing.T) {
		openBinaryFileHook = func(name string) (*os.File, error) {
			return nil, errors.New("simulated dir walk file open error")
		}

		ctx := context.Background()
		var errCount int
		for _, err := range WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				errCount++
				break
			}
		}

		if errCount != 1 {
			t.Errorf("expected exactly 1 error on early break, got %d", errCount)
		}
	})
}
