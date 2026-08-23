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
	filterNil, err := scanner.NewFilter(scanner.FilterOptions{})
	if err != nil || filterNil != nil {
		t.Fatalf("expected nil filter for nil patterns, got filter=%v, err=%v", filterNil, err)
	}

	filterEmpty, err := scanner.NewFilter(scanner.FilterOptions{IgnorePatterns: []string{"", ""}})
	if err != nil || filterEmpty != nil {
		t.Fatalf("expected nil filter for empty patterns, got filter=%v, err=%v", filterEmpty, err)
	}

	filterEmptyDirs, err := scanner.NewFilter(scanner.FilterOptions{
		IgnoreDirs:   []string{"", "# comment"},
		NamePatterns: []string{""},
	})
	if err != nil || filterEmptyDirs != nil {
		t.Fatalf("expected nil filter for comments/empty dirs, got filter=%v, err=%v", filterEmptyDirs, err)
	}

	_, err = scanner.NewFilter(scanner.FilterOptions{IgnorePatterns: []string{"(?<invalid"}})
	if err == nil {
		t.Errorf("expected error for invalid ignore pattern, got nil")
	}

	_, err = scanner.NewFilter(scanner.FilterOptions{NamePatterns: []string{"(?<invalid"}})
	if err == nil {
		t.Errorf("expected error for invalid name pattern, got nil")
	}
}

func TestWalkPaths_EdgeCases(t *testing.T) {
	t.Run("Single file matching ignore filter is skipped", func(t *testing.T) {
		tempDir := t.TempDir()
		file := filepath.Join(tempDir, "ignore_me.txt")
		_ = os.WriteFile(file, []byte("test"), 0644)

		filter, _ := scanner.NewFilter(scanner.FilterOptions{IgnorePatterns: []string{`ignore_me\.txt$`}})
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

	t.Run("Aborts across multiple root paths on canceled context", func(t *testing.T) {
		tempDir := t.TempDir()
		d1 := filepath.Join(tempDir, "d1")
		d2 := filepath.Join(tempDir, "d2")
		_ = os.Mkdir(d1, 0755)
		_ = os.Mkdir(d2, 0755)
		_ = os.WriteFile(filepath.Join(d1, "a.txt"), []byte("a"), 0644)
		_ = os.WriteFile(filepath.Join(d2, "b.txt"), []byte("b"), 0644)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var seen []string
		for entry, err := range scanner.WalkPaths(ctx, []string{d1, d2}, nil) {
			if err == nil {
				seen = append(seen, entry.RelPath)
				cancel()
			}
		}

		if len(seen) == 0 {
			t.Errorf("expected at least 1 entry seen before cancel")
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

		filter, err := scanner.NewFilter(scanner.FilterOptions{
			IgnorePatterns: []string{"node_modules", `\.log$`},
		})
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
		_, err := scanner.NewFilter(scanner.FilterOptions{IgnorePatterns: []string{"[invalid"}})
		if err == nil {
			t.Errorf("expected error for invalid regex '[invalid', got nil")
		}
	})

	t.Run("Short-circuits on loop break", func(t *testing.T) {
		tempDir := t.TempDir()
		for i := range 10 {
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
		for i := range 5 {
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

		filter, err := scanner.NewFilter(scanner.FilterOptions{IgnoreDotfiles: true})
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

		filter, err := scanner.NewFilter(scanner.FilterOptions{
			IgnoreDirs: []string{"bin", "coverage/", "/build"},
		})
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
		_, err := scanner.NewFilter(scanner.FilterOptions{IgnoreDirs: []string{"[invalid_regex"}})
		if err == nil {
			t.Errorf("expected error for invalid directory ignore pattern, got nil")
		}
	})
}

func TestNewFilter_PerlRegex(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		testName   string
		testRel    string
		wantIgnore bool
	}{
		{
			name:       "Positive Lookahead matches",
			pattern:    `temp(?=\.txt$)`,
			testName:   "temp.txt",
			testRel:    "temp.txt",
			wantIgnore: true,
		},
		{
			name:       "Positive Lookahead does not match",
			pattern:    `temp(?=\.txt$)`,
			testName:   "temp.go",
			testRel:    "temp.go",
			wantIgnore: false,
		},
		{
			name:       "Negative Lookahead matches non-matching suffix",
			pattern:    `temp(?!\.txt$)`,
			testName:   "temp.go",
			testRel:    "temp.go",
			wantIgnore: true,
		},
		{
			name:       "Negative Lookahead excludes matching suffix",
			pattern:    `temp(?!\.txt$)`,
			testName:   "temp.txt",
			testRel:    "temp.txt",
			wantIgnore: false,
		},
		{
			name:       "Positive Lookbehind matches prefix",
			pattern:    `(?<=ignore_)file\.txt$`,
			testName:   "ignore_file.txt",
			testRel:    "ignore_file.txt",
			wantIgnore: true,
		},
		{
			name:       "Positive Lookbehind excludes non-matching prefix",
			pattern:    `(?<=ignore_)file\.txt$`,
			testName:   "keep_file.txt",
			testRel:    "keep_file.txt",
			wantIgnore: false,
		},
		{
			name:       "Negative Lookbehind matches without prefix",
			pattern:    `(?<!keep_)file\.txt$`,
			testName:   "ignore_file.txt",
			testRel:    "ignore_file.txt",
			wantIgnore: true,
		},
		{
			name:       "Negative Lookbehind excludes with prefix",
			pattern:    `(?<!keep_)file\.txt$`,
			testName:   "keep_file.txt",
			testRel:    "keep_file.txt",
			wantIgnore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := scanner.NewFilter(scanner.FilterOptions{IgnorePatterns: []string{tt.pattern}})
			if err != nil {
				t.Fatalf("failed to compile Perl regex %q: %v", tt.pattern, err)
			}
			got := filter.ShouldIgnore(tt.testName, tt.testRel)
			if got != tt.wantIgnore {
				t.Errorf("ShouldIgnore(%q, %q) with pattern %q = %v, want %v", tt.testName, tt.testRel, tt.pattern, got, tt.wantIgnore)
			}
		})
	}
}

func TestNewFilter_NameFilter_WholeNameMatch(t *testing.T) {
	t.Run("Matches whole file name anchor", func(t *testing.T) {
		filter, err := scanner.NewFilter(scanner.FilterOptions{
			NamePatterns: []string{"main"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if filter.ShouldIgnore("main", "main", false) {
			t.Errorf("expected 'main' to match name filter 'main'")
		}
		if !filter.ShouldIgnore("main.go", "main.go", false) {
			t.Errorf("expected 'main.go' NOT to match whole name filter 'main'")
		}
	})

	t.Run("Multiple name filter patterns (OR logic)", func(t *testing.T) {
		filter, err := scanner.NewFilter(scanner.FilterOptions{
			NamePatterns: []string{".*\\.go", ".*\\.md"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if filter.ShouldIgnore("main.go", "main.go", false) {
			t.Errorf("expected main.go to match name filter")
		}
		if filter.ShouldIgnore("README.md", "README.md", false) {
			t.Errorf("expected README.md to match name filter")
		}
		if !filter.ShouldIgnore("app.exe", "app.exe", false) {
			t.Errorf("expected app.exe NOT to match name filter")
		}
	})

	t.Run("Invalid regex in name filter returns error", func(t *testing.T) {
		_, err := scanner.NewFilter(scanner.FilterOptions{
			NamePatterns: []string{"[invalid"},
		})
		if err == nil {
			t.Errorf("expected error for invalid name filter regex, got nil")
		}
	})
}

func TestFilter_EvaluationSequence(t *testing.T) {
	// The evaluation sequence strictly goes:
	// 1. evaluate ignore dot files flag
	// 2. ignore directories
	// 3. name filter
	// 4. ignore flag

	opts := scanner.FilterOptions{
		IgnoreDotfiles: true,
		IgnoreDirs:     []string{"bin"},
		NamePatterns:   []string{".*\\.go"},
		IgnorePatterns: []string{".*_test\\.go"},
	}

	filter, err := scanner.NewFilter(opts)
	if err != nil {
		t.Fatalf("failed to create filter: %v", err)
	}

	// Step 1: Dotfiles rejected first
	if !filter.ShouldIgnore(".hidden", ".hidden", true) {
		t.Errorf("expected dotfile directory to be ignored at step 1")
	}
	if !filter.ShouldIgnore(".env", ".env", false) {
		t.Errorf("expected dotfile to be ignored at step 1")
	}

	// Step 2: Ignored directories rejected second
	if !filter.ShouldIgnore("bin", "bin", true) {
		t.Errorf("expected bin directory to be ignored at step 2")
	}

	// Step 3: Name filter evaluated third
	// Non-ignored directories pass step 3 so subtrees can be traversed
	if filter.ShouldIgnore("src", "src", true) {
		t.Errorf("expected directory 'src' NOT to be ignored by name filter at step 3")
	}
	// utils.js fails name filter match (.*\.go) at step 3 -> ignored
	if !filter.ShouldIgnore("utils.js", "src/utils.js", false) {
		t.Errorf("expected utils.js to be ignored at step 3 by name filter")
	}

	// Step 4: Ignore patterns evaluated fourth
	// main_test.go matches name filter at step 3, but fails step 4 ignore pattern (.*_test\.go)
	if !filter.ShouldIgnore("main_test.go", "src/main_test.go", false) {
		t.Errorf("expected main_test.go to be ignored at step 4 by ignore pattern")
	}

	// main.go passes step 1, step 2, step 3 (matches name filter), step 4 (not ignored) -> kept
	if filter.ShouldIgnore("main.go", "src/main.go", false) {
		t.Errorf("expected main.go to NOT be ignored")
	}
}

func TestFilter_ShouldIgnore_Direct(t *testing.T) {
	var nilFilter *scanner.Filter
	if nilFilter.ShouldIgnore("foo.go", "foo.go") {
		t.Errorf("expected nil filter to return false")
	}

	filter, err := scanner.NewFilter(scanner.FilterOptions{
		IgnoreDirs:     []string{"sub/nested"},
		IgnorePatterns: []string{"dir/.*\\.txt$"},
		IgnoreDotfiles: true,
	})
	if err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}

	// Match on slashRel for directory
	if !filter.ShouldIgnore("nested", "sub/nested", true) {
		t.Errorf("expected ShouldIgnore to match dirRegexes against slashRel")
	}

	// Match on slashRel for ignore pattern
	if !filter.ShouldIgnore("notes.txt", "dir/notes.txt", false) {
		t.Errorf("expected ShouldIgnore to match regexes against slashRel")
	}

	// Dotfile matching on baseName of slashRel
	if !filter.ShouldIgnore("file", "parent/.hiddenfile", false) {
		t.Errorf("expected ShouldIgnore to match dotfile on baseName of slashRel")
	}
}

func TestWalkPaths_Metrics(t *testing.T) {
	tempDir := t.TempDir()
	docFile := filepath.Join(tempDir, "README.md")
	docContent := "# Readme Title\n\nSome introductory content here.\n"
	_ = os.WriteFile(docFile, []byte(docContent), 0644)

	t.Run("Without ComputeMetrics, metrics remain zero", func(t *testing.T) {
		ctx := context.Background()
		var found bool
		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !entry.IsDir && entry.Path == docFile {
				found = true
				if entry.Extension != ".md" {
					t.Errorf("expected extension .md, got %q", entry.Extension)
				}
				if entry.Lines != 0 || entry.Tokens != 0 {
					t.Errorf("expected 0 lines and 0 tokens without metrics option, got lines=%d tokens=%d", entry.Lines, entry.Tokens)
				}
			}
		}
		if !found {
			t.Fatal("expected to find README.md in scan")
		}
	})

	t.Run("With ComputeMetrics, metrics are accurately populated", func(t *testing.T) {
		ctx := context.Background()
		var found bool
		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil, scanner.WithComputeMetrics(true)) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !entry.IsDir && entry.Path == docFile {
				found = true
				if entry.Extension != ".md" {
					t.Errorf("expected extension .md, got %q", entry.Extension)
				}
				if entry.Language != "markdown" {
					t.Errorf("expected language markdown, got %q", entry.Language)
				}
				if entry.Lines != 3 {
					t.Errorf("expected 3 lines, got %d", entry.Lines)
				}
				if entry.Tokens != 7 {
					t.Errorf("expected 7 tokens, got %d", entry.Tokens)
				}
			}
		}
		if !found {
			t.Fatal("expected to find README.md in scan")
		}
	})

	t.Run("Resolves language, auxiliary extensions and placeholders for special files", func(t *testing.T) {
		makeFile := filepath.Join(tempDir, "Makefile")
		yamlExFile := filepath.Join(tempDir, "config.yaml.example")
		unknownFile := filepath.Join(tempDir, "data.unknownext")
		_ = os.WriteFile(makeFile, []byte("all:\n\t@echo hi\n"), 0644)
		_ = os.WriteFile(yamlExFile, []byte("key: value\n"), 0644)
		_ = os.WriteFile(unknownFile, []byte("custom content\n"), 0644)

		ctx := context.Background()
		results := make(map[string]scanner.Entry)
		for entry, err := range scanner.WalkPaths(ctx, []string{tempDir}, nil) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !entry.IsDir {
				results[filepath.Base(entry.Path)] = entry
			}
		}

		if e, ok := results["Makefile"]; !ok {
			t.Errorf("expected Makefile in results")
		} else {
			if e.Extension != "-" {
				t.Errorf("expected '-' extension for Makefile, got %q", e.Extension)
			}
			if e.Language != "makefile" {
				t.Errorf("expected 'makefile' language for Makefile, got %q", e.Language)
			}
		}

		if e, ok := results["config.yaml.example"]; !ok {
			t.Errorf("expected config.yaml.example in results")
		} else {
			if e.Extension != ".yaml" {
				t.Errorf("expected '.yaml' extension for config.yaml.example, got %q", e.Extension)
			}
			if e.Language != "yaml" {
				t.Errorf("expected 'yaml' language for config.yaml.example, got %q", e.Language)
			}
		}

		if e, ok := results["data.unknownext"]; !ok {
			t.Errorf("expected data.unknownext in results")
		} else {
			if e.Extension != ".unknownext" {
				t.Errorf("expected '.unknownext' extension, got %q", e.Extension)
			}
			if e.Language != "-" {
				t.Errorf("expected '-' language for unknown extension, got %q", e.Language)
			}
		}
	})
}
