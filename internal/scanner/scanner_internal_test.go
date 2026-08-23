package scanner

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
			if entry.Language != "go" {
				t.Errorf("expected go language, got %q", entry.Language)
			}
		}
		if count != 1 {
			t.Errorf("expected 1 file yielded, got %d", count)
		}
	})

	t.Run("Single file target with special filename and unknown extension", func(t *testing.T) {
		makefilePath := filepath.Join(tempDir, "Makefile")
		unknownExtPath := filepath.Join(tempDir, "config.customext")
		_ = os.WriteFile(makefilePath, []byte("all:\n\t@echo hi\n"), 0644)
		_ = os.WriteFile(unknownExtPath, []byte("key: value\n"), 0644)

		ctx := context.Background()
		var makefileEntry, unknownEntry Entry
		for entry, err := range WalkPaths(ctx, []string{makefilePath}, nil) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			makefileEntry = entry
		}
		if makefileEntry.Extension != "-" || makefileEntry.Language != "makefile" {
			t.Errorf("expected '-' extension and 'makefile' language for single Makefile, got ext=%q lang=%q", makefileEntry.Extension, makefileEntry.Language)
		}

		for entry, err := range WalkPaths(ctx, []string{unknownExtPath}, nil) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			unknownEntry = entry
		}
		if unknownEntry.Extension != ".customext" || unknownEntry.Language != "-" {
			t.Errorf("expected '.customext' extension and '-' language for single custom file, got ext=%q lang=%q", unknownEntry.Extension, unknownEntry.Language)
		}
	})

	t.Run("Single file metric computation error handling", func(t *testing.T) {
		origCount := countLinesAndTokensFunc
		defer func() { countLinesAndTokensFunc = origCount }()

		expectedErr := errors.New("simulated metric error")
		countLinesAndTokensFunc = func(r io.Reader) (int, int, error) {
			return 0, 0, expectedErr
		}

		ctx := context.Background()

		// Yield returning false breaks early
		var errCount int
		for _, err := range WalkPaths(ctx, []string{filePath}, nil, WithComputeMetrics(true)) {
			if err != nil {
				errCount++
				break
			}
		}
		if errCount != 1 {
			t.Errorf("expected 1 error on single file metric failure with early break, got: %d", errCount)
		}

		// Yield returning true continues
		errCount = 0
		for _, err := range WalkPaths(ctx, []string{filePath}, nil, WithComputeMetrics(true)) {
			if err != nil {
				errCount++
			}
		}
		if errCount != 1 {
			t.Errorf("expected 1 error on single file metric failure with continue, got: %d", errCount)
		}
	})

	t.Run("Directory walk metric computation error handling", func(t *testing.T) {
		origCount := countLinesAndTokensFunc
		defer func() { countLinesAndTokensFunc = origCount }()

		expectedErr := errors.New("simulated dir metric error")
		countLinesAndTokensFunc = func(r io.Reader) (int, int, error) {
			return 0, 0, expectedErr
		}

		ctx := context.Background()

		// Yield returning false breaks early
		var errCount int
		for _, err := range WalkPaths(ctx, []string{tempDir}, nil, WithComputeMetrics(true)) {
			if err != nil {
				errCount++
				break
			}
		}
		if errCount != 1 {
			t.Errorf("expected 1 error on dir metric failure with early break, got: %d", errCount)
		}

		// Yield returning true continues
		errCount = 0
		for _, err := range WalkPaths(ctx, []string{tempDir}, nil, WithComputeMetrics(true)) {
			if err != nil {
				errCount++
			}
		}
		if errCount == 0 {
			t.Errorf("expected at least 1 error on dir metric failure with continue, got: %d", errCount)
		}
	})
}

// closerItem implements io.ReadCloser to verify the io.Closer cleanup branches.
type closerItem struct{ closed bool }

func (c *closerItem) Read(p []byte) (int, error) { return 0, io.EOF }
func (c *closerItem) Close() error                { c.closed = true; return nil }

// entryWithContent returns an Entry with a non-nil Content for testing Entry.Content.Close().
func entryWithContent() Entry {
	return Entry{Content: io.NopCloser(strings.NewReader("data"))}
}

// TestCollectEntries_CleanupOnError covers the two type-assertion cleanup paths
// in CollectEntries when a downstream error fires after items have been collected.
func TestCollectEntries_CleanupOnError(t *testing.T) {
	sentinel := errors.New("seq error")

	t.Run("io.Closer items are closed on error", func(t *testing.T) {
		item := &closerItem{}
		seq := func(yield func(*closerItem, error) bool) {
			if !yield(item, nil) {
				return
			}
			_ = yield(nil, sentinel)
		}
		_, err := (&Scanner{}).CollectEntries(seq)
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel, got %v", err)
		}
		if !item.closed {
			t.Error("expected io.Closer item to be closed on error")
		}
	})

	t.Run("Entry items with Content are closed on error", func(t *testing.T) {
		rc := &closerItem{}
		entry := Entry{Content: rc}
		seq := func(yield func(Entry, error) bool) {
			if !yield(entry, nil) {
				return
			}
			_ = yield(Entry{}, sentinel)
		}
		_, err := (&Scanner{}).CollectEntries(seq)
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel, got %v", err)
		}
		if !rc.closed {
			t.Error("expected Entry.Content to be closed on error")
		}
	})
}

// TestBatchEntries_CloserCleanup covers the io.Closer and Entry.Content.Close paths
// in the three cleanup sites inside BatchEntries.
func TestBatchEntries_CloserCleanup(t *testing.T) {
	sentinel := errors.New("batch seq error")

	// Site 1: pending batch flushed before error, yield(batch) returns false → cleanup batch items.
	t.Run("io.Closer cleanup in pending-batch-error path on early break", func(t *testing.T) {
		item := &closerItem{}
		seq := func(yield func(*closerItem, error) bool) {
			if !yield(item, nil) {
				return
			}
			_ = yield(nil, sentinel)
		}
		for batch, err := range (&Scanner{}).BatchEntries(seq, 10) {
			if err == nil && len(batch) > 0 {
				break // yield returns false: triggers io.Closer cleanup in site 1
			}
		}
		if !item.closed {
			t.Error("expected io.Closer item closed when breaking on pending-batch yield")
		}
	})

	t.Run("Entry.Content cleanup in pending-batch-error path on early break", func(t *testing.T) {
		rc := &closerItem{}
		entry := Entry{Content: rc}
		seq := func(yield func(Entry, error) bool) {
			if !yield(entry, nil) {
				return
			}
			_ = yield(Entry{}, sentinel)
		}
		for batch, err := range (&Scanner{}).BatchEntries(seq, 10) {
			if err == nil && len(batch) > 0 {
				break
			}
		}
		if !rc.closed {
			t.Error("expected Entry.Content closed when breaking on pending-batch yield")
		}
	})

	// Site 2: batch fills up (len == batchSize), yield(batch) returns false → cleanup full batch.
	t.Run("io.Closer cleanup in full-batch early-break path", func(t *testing.T) {
		item := &closerItem{}
		seq := func(yield func(*closerItem, error) bool) {
			_ = yield(item, nil) // fills the batch of size 1 → triggers cleanup on break
		}
		for _, err := range (&Scanner{}).BatchEntries(seq, 1) {
			if err == nil {
				break // break when full batch is yielded
			}
		}
		if !item.closed {
			t.Error("expected io.Closer item closed in full-batch break cleanup")
		}
	})

	t.Run("Entry.Content cleanup in full-batch early-break path", func(t *testing.T) {
		rc := &closerItem{}
		entry := Entry{Content: rc}
		seq := func(yield func(Entry, error) bool) {
			_ = yield(entry, nil)
		}
		for _, err := range (&Scanner{}).BatchEntries(seq, 1) {
			if err == nil {
				break
			}
		}
		if !rc.closed {
			t.Error("expected Entry.Content closed in full-batch break cleanup")
		}
	})

	// Site 3: trailing batch (after loop ends), yield(batch) returns false → cleanup.
	t.Run("io.Closer cleanup in trailing-batch early-break path", func(t *testing.T) {
		item := &closerItem{}
		seq := func(yield func(*closerItem, error) bool) {
			_ = yield(item, nil) // single item: goes to trailing batch
		}
		// batchSize larger than items so it ends up in the trailing-batch path
		for _, err := range (&Scanner{}).BatchEntries(seq, 10) {
			if err == nil {
				break
			}
		}
		if !item.closed {
			t.Error("expected io.Closer item closed in trailing-batch break cleanup")
		}
	})

	t.Run("Entry.Content cleanup in trailing-batch early-break path", func(t *testing.T) {
		rc := &closerItem{}
		entry := Entry{Content: rc}
		seq := func(yield func(Entry, error) bool) {
			_ = yield(entry, nil)
		}
		for _, err := range (&Scanner{}).BatchEntries(seq, 10) {
			if err == nil {
				break
			}
		}
		if !rc.closed {
			t.Error("expected Entry.Content closed in trailing-batch break cleanup")
		}
	})
}
