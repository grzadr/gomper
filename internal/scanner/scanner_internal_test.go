package scanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type errorDirEntry struct {
	name  string
	isDir bool
}

func (e errorDirEntry) Name() string               { return e.name }
func (e errorDirEntry) IsDir() bool                { return e.isDir }
func (e errorDirEntry) Type() fs.FileMode          { return 0644 }
func (e errorDirEntry) Info() (fs.FileInfo, error) { return nil, errors.New("simulated file info error") }

func TestWalkPaths_InternalWalkDirErrors(t *testing.T) {
	origWalk := walkDirFunc
	defer func() { walkDirFunc = origWalk }()

	tempDir := t.TempDir()

	t.Run("d.Info() error with yield returning true", func(t *testing.T) {
		walkDirFunc = func(root string, fn fs.WalkDirFunc) error {
			return fn(filepath.Join(root, "bad_info_dir"), errorDirEntry{name: "bad_info_dir", isDir: true}, nil)
		}

		ctx := context.Background()
		var errCount int
		for _, err := range WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				errCount++
			}
		}

		if errCount != 1 {
			t.Errorf("expected 1 error from bad d.Info(), got: %d", errCount)
		}
	})

	t.Run("d.Info() error with yield returning false breaks early", func(t *testing.T) {
		walkDirFunc = func(root string, fn fs.WalkDirFunc) error {
			if err := fn(filepath.Join(root, "bad_info1"), errorDirEntry{name: "bad_info1", isDir: true}, nil); err != nil {
				return err
			}
			return fn(filepath.Join(root, "bad_info2"), errorDirEntry{name: "bad_info2", isDir: true}, nil)
		}

		ctx := context.Background()
		var errCount int
		for _, err := range WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				errCount++
				break // breaks walk
			}
		}

		if errCount != 1 {
			t.Errorf("expected count 1 on early break, got: %d", errCount)
		}
	})

	t.Run("walkErr with yield returning false breaks early", func(t *testing.T) {
		walkDirFunc = func(root string, fn fs.WalkDirFunc) error {
			return fn(filepath.Join(root, "broken_walk"), nil, errors.New("simulated walk error"))
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
			t.Errorf("expected count 1 on early break, got: %d", errCount)
		}
	})

	t.Run("filepath.Rel failure falls back to path", func(t *testing.T) {
		realFile := filepath.Join(tempDir, "file.txt")
		_ = os.WriteFile(realFile, []byte("valid text"), 0644)

		walkDirFunc = func(root string, fn fs.WalkDirFunc) error {
			// root is relative ("rel/dir"), path is absolute -> filepath.Rel fails!
			return fn(realFile, dummyDirEntry{name: "file.txt"}, nil)
		}

		// Create relative dir in working directory
		relDir := filepath.Join(tempDir, "rel_root")
		_ = os.Mkdir(relDir, 0755)

		// Make stat work by passing relDir, but in WalkPaths we pass a relative path or stat passes
		ctx := context.Background()
		var seenPath string
		// Stat of relative path:
		cwd, _ := os.Getwd()
		relToCwd, _ := filepath.Rel(cwd, relDir)
		for entry, err := range WalkPaths(ctx, []string{relToCwd}, nil) {
			if err == nil {
				seenPath = entry.RelPath
			}
		}

		if seenPath != realFile {
			t.Errorf("expected fallback to path on filepath.Rel error, got: %q", seenPath)
		}
	})
}

type dummyDirEntry struct {
	name string
}

func (d dummyDirEntry) Name() string               { return d.name }
func (d dummyDirEntry) IsDir() bool                { return false }
func (d dummyDirEntry) Type() fs.FileMode          { return 0644 }
func (d dummyDirEntry) Info() (fs.FileInfo, error) { return dummyFileInfo(d), nil }

type dummyFileInfo struct {
	name string
}

func (d dummyFileInfo) Name() string       { return d.name }
func (d dummyFileInfo) Size() int64        { return 10 }
func (d dummyFileInfo) Mode() fs.FileMode  { return 0644 }
func (d dummyFileInfo) ModTime() time.Time { return time.Now() }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() any           { return nil }

func TestWalkPaths_WithComputeMetrics(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "main.go")
	_ = os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0644)

	t.Run("Computes metrics for directory traversal", func(t *testing.T) {
		ctx := context.Background()
		var found bool
		for entry, err := range WalkPaths(ctx, []string{tempDir}, nil, WithComputeMetrics(true), WithTokenizer(NewWhitespaceTokenizer())) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !entry.IsDir && entry.Path == filePath {
				found = true
				if entry.Lines != 3 {
					t.Errorf("expected 3 lines, got %d", entry.Lines)
				}
				if entry.Tokens != 5 {
					t.Errorf("expected 5 tokens, got %d", entry.Tokens)
				}
				if entry.Extension != ".go" {
					t.Errorf("expected .go extension, got %q", entry.Extension)
				}
				if entry.Size <= 0 {
					t.Errorf("expected positive size, got %d", entry.Size)
				}
			}
		}
		if !found {
			t.Fatal("expected to find main.go in scan")
		}
	})

	t.Run("Computes metrics for single file target", func(t *testing.T) {
		ctx := context.Background()
		var count int
		for entry, err := range WalkPaths(ctx, []string{filePath}, nil, WithComputeMetrics(true)) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			count++
			if entry.Lines != 3 {
				t.Errorf("expected 3 lines, got %d", entry.Lines)
			}
			if entry.Tokens != 5 {
				t.Errorf("expected 5 tokens, got %d", entry.Tokens)
			}
			if entry.Extension != ".go" {
				t.Errorf("expected .go extension, got %q", entry.Extension)
			}
		}
		if count != 1 {
			t.Errorf("expected 1 file yielded, got %d", count)
		}
	})
}
