package scanner_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/scanner"
)

func TestNewFilter_EdgeCases(t *testing.T) {
	filterNil, err := scanner.NewFilter(nil, false)
	if err != nil || filterNil != nil {
		t.Fatalf("expected nil filter for nil patterns, got filter=%v, err=%v", filterNil, err)
	}

	filterEmpty, err := scanner.NewFilter([]string{"", ""}, false)
	if err != nil || filterEmpty != nil {
		t.Fatalf("expected nil filter for empty patterns, got filter=%v, err=%v", filterEmpty, err)
	}
}

func TestWalkPaths_EdgeCases(t *testing.T) {
	t.Run("Single file matching ignore filter is skipped", func(t *testing.T) {
		tempDir := t.TempDir()
		file := filepath.Join(tempDir, "ignore_me.txt")
		_ = os.WriteFile(file, []byte("test"), 0644)

		filter, _ := scanner.NewFilter([]string{`ignore_me\.txt$`}, false)
		ctx := context.Background()

		var count int
		for _, err := range scanner.WalkPaths(ctx, []string{file}, filter) {
			if err == nil {
				count++
			}
		}

		if count != 0 {
			t.Errorf("expected single file matching filter to be skipped, got count %d", count)
		}
	})

	t.Run("Single file yields false breaks early", func(t *testing.T) {
		tempDir := t.TempDir()
		file := filepath.Join(tempDir, "sample.txt")
		_ = os.WriteFile(file, []byte("test"), 0644)

		ctx := context.Background()
		var count int
		for _, err := range scanner.WalkPaths(ctx, []string{file}, nil) {
			if err == nil {
				count++
				break
			}
		}

		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("Multiple single file paths walk sequentially", func(t *testing.T) {
		tempDir := t.TempDir()
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		_ = os.WriteFile(file1, []byte("1"), 0644)
		_ = os.WriteFile(file2, []byte("2"), 0644)

		ctx := context.Background()
		var pathsFound []string
		for entry, err := range scanner.WalkPaths(ctx, []string{file1, file2}, nil) {
			if err == nil {
				pathsFound = append(pathsFound, entry.Path)
			}
		}

		if len(pathsFound) != 2 {
			t.Errorf("expected 2 single file entries, got %d", len(pathsFound))
		}
	})


	t.Run("Non-existent root path yields false breaks early", func(t *testing.T) {
		ctx := context.Background()
		nonExistent := "/nonexistent/path/for/test"
		var count int
		for _, err := range scanner.WalkPaths(ctx, []string{nonExistent}, nil) {
			if err != nil {
				count++
				break
			}
		}

		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
	})

	t.Run("Handles unreadable directory walk error", func(t *testing.T) {
		tempDir := t.TempDir()
		unreadableDir := filepath.Join(tempDir, "unreadable")
		if err := os.Mkdir(unreadableDir, 0000); err != nil {
			t.Skip("skipping unreadable dir test on environment without permission restrictions")
		}
		defer func() { _ = os.Chmod(unreadableDir, 0755) }()

		ctx := context.Background()
		var errCount int
		for _, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				errCount++
			}
		}

		if errCount == 0 {
			t.Log("unreadable directory walked without error on this platform")
		}
	})

	t.Run("Handles unreadable directory walk error breaking early", func(t *testing.T) {
		tempDir := t.TempDir()
		unreadableDir := filepath.Join(tempDir, "unreadable2")
		if err := os.Mkdir(unreadableDir, 0000); err != nil {
			t.Skip("skipping test")
		}
		defer func() { _ = os.Chmod(unreadableDir, 0755) }()

		ctx := context.Background()
		for _, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				break
			}
		}
	})
}

func TestWalkPaths_Basic(t *testing.T) {
	t.Run("Walks directory structure correctly", func(t *testing.T) {
		tempDir := t.TempDir()

		subDir := filepath.Join(tempDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		file1 := filepath.Join(tempDir, "file1.txt")
		if err := os.WriteFile(file1, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to write file1: %v", err)
		}

		file2 := filepath.Join(subDir, "file2.txt")
		if err := os.WriteFile(file2, []byte("world"), 0644); err != nil {
			t.Fatalf("failed to write file2: %v", err)
		}

		ctx := context.Background()
		var entries []scanner.Entry

		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				t.Fatalf("unexpected error during walk: %v", err)
			}
			entries = append(entries, entry)
		}

		if len(entries) != 4 {
			t.Fatalf("expected 4 entries (root, subdir, file1, file2), got %d", len(entries))
		}
	})

	t.Run("Filters out files and directories matching regex patterns", func(t *testing.T) {
		tempDir := t.TempDir()

		subDir := filepath.Join(tempDir, "node_modules")
		_ = os.Mkdir(subDir, 0755)
		_ = os.WriteFile(filepath.Join(subDir, "pkg.js"), []byte("{}"), 0644)

		file1 := filepath.Join(tempDir, "main.go")
		_ = os.WriteFile(file1, []byte("package main"), 0644)

		file2 := filepath.Join(tempDir, "temp.log")
		_ = os.WriteFile(file2, []byte("logs"), 0644)

		filter, err := scanner.NewFilter([]string{"node_modules", `\.log$`}, false)
		if err != nil {
			t.Fatalf("unexpected filter creation error: %v", err)
		}

		ctx := context.Background()
		var entries []scanner.Entry

		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, filter) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			entries = append(entries, entry)
		}

		for _, e := range entries {
			if e.Path == file2 {
				t.Errorf("expected temp.log to be ignored by regex filter")
			}
			if filepath.Base(e.Path) == "node_modules" || filepath.Base(e.Path) == "pkg.js" {
				t.Errorf("expected node_modules and contents to be skipped, found %s", e.Path)
			}
		}
	})

	t.Run("Returns error for invalid regex pattern", func(t *testing.T) {
		_, err := scanner.NewFilter([]string{"[invalid"}, false)
		if err == nil {
			t.Errorf("expected error for invalid regex '[invalid', got nil")
		}
	})

	t.Run("Short-circuits on loop break", func(t *testing.T) {
		tempDir := t.TempDir()
		for i := 0; i < 10; i++ {
			f := filepath.Join(tempDir, string(rune('a'+i))+".txt")
			_ = os.WriteFile(f, []byte("test"), 0644)
		}

		ctx := context.Background()
		count := 0
		for _, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			count++
			if count == 2 {
				break
			}
		}

		if count != 2 {
			t.Errorf("expected loop to break after 2 iterations, got %d", count)
		}
	})

	t.Run("Stops on context cancellation", func(t *testing.T) {
		tempDir := t.TempDir()
		for i := 0; i < 5; i++ {
			f := filepath.Join(tempDir, string(rune('a'+i))+".txt")
			_ = os.WriteFile(f, []byte("test"), 0644)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		count := 0
		for _, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err == nil {
				count++
			}
		}

		if count > 1 {
			t.Errorf("expected context cancellation to stop iteration early, processed %d entries", count)
		}
	})

	t.Run("Yields error for non-existent path", func(t *testing.T) {
		nonExistent := "/path/to/nonexistent/file/or/dir"
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var errorSeen bool
		for entry, err := range scanner.WalkPaths(ctx, []string{nonExistent}, nil) {
			if err != nil {
				errorSeen = true
				if entry.Path != nonExistent {
					t.Errorf("expected entry path %s, got %s", nonExistent, entry.Path)
				}
			}
		}

		if !errorSeen {
			t.Errorf("expected error for non-existent path")
		}
	})

	t.Run("Filters out files and directories starting with '.' when ignoreDotfiles is true", func(t *testing.T) {
		tempDir := t.TempDir()

		dotDir := filepath.Join(tempDir, ".git")
		_ = os.Mkdir(dotDir, 0755)
		_ = os.WriteFile(filepath.Join(dotDir, "config"), []byte("git config"), 0644)

		dotFile := filepath.Join(tempDir, ".env")
		_ = os.WriteFile(dotFile, []byte("SECRET=123"), 0644)

		normalFile := filepath.Join(tempDir, "main.go")
		_ = os.WriteFile(normalFile, []byte("package main"), 0644)

		filter, err := scanner.NewFilter(nil, true)
		if err != nil {
			t.Fatalf("unexpected filter error: %v", err)
		}

		ctx := context.Background()
		var entries []scanner.Entry
		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, filter) {
			if err != nil {
				t.Fatalf("unexpected walk error: %v", err)
			}
			entries = append(entries, entry)
		}

		for _, e := range entries {
			base := filepath.Base(e.Path)
			if base == ".git" || base == "config" || base == ".env" {
				t.Errorf("expected dotfile/directory %s to be ignored", e.Path)
			}
		}
	})

	t.Run("Filters out directories matching ignore directory gitignore patterns", func(t *testing.T) {
		tempDir := t.TempDir()

		binDir := filepath.Join(tempDir, "bin")
		_ = os.Mkdir(binDir, 0755)
		_ = os.WriteFile(filepath.Join(binDir, "gomper"), []byte("binary"), 0755)

		covDir := filepath.Join(tempDir, "coverage")
		_ = os.Mkdir(covDir, 0755)
		_ = os.WriteFile(filepath.Join(covDir, "coverage.out"), []byte("coverage data"), 0644)

		srcDir := filepath.Join(tempDir, "src")
		_ = os.Mkdir(srcDir, 0755)
		subBinDir := filepath.Join(srcDir, "bin")
		_ = os.Mkdir(subBinDir, 0755)
		_ = os.WriteFile(filepath.Join(subBinDir, "helper"), []byte("helper"), 0755)
		_ = os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)

		rootBuildDir := filepath.Join(tempDir, "build")
		_ = os.Mkdir(rootBuildDir, 0755)
		_ = os.WriteFile(filepath.Join(rootBuildDir, "output.o"), []byte("obj"), 0644)

		filter, err := scanner.NewFilter(nil, false, []string{"bin", "coverage/", "/build"})
		if err != nil {
			t.Fatalf("unexpected filter error: %v", err)
		}

		ctx := context.Background()
		var paths []string
		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, filter) {
			if err != nil {
				t.Fatalf("unexpected walk error: %v", err)
			}
			paths = append(paths, entry.RelPath)
		}

		for _, p := range paths {
			base := filepath.Base(p)
			if base == "bin" || base == "gomper" || base == "coverage" || base == "coverage.out" || base == "build" || base == "output.o" || base == "helper" {
				t.Errorf("expected path %s to be ignored by directory ignore rules, got listed", p)
			}
		}

		var foundMain bool
		for _, p := range paths {
			if filepath.Base(p) == "main.go" {
				foundMain = true
			}
		}
		if !foundMain {
			t.Errorf("expected main.go to be included, paths found: %v", paths)
		}
	})

	t.Run("Returns error for invalid directory ignore regex pattern", func(t *testing.T) {
		_, err := scanner.NewFilter(nil, false, []string{"[invalid_regex"})
		if err == nil {
			t.Errorf("expected error for invalid directory ignore pattern, got nil")
		}
	})
}

