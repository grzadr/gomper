package scanner

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/dlclark/regexp2"
)

// Entry encapsulates metadata and fs.FileInfo for a scanned file or directory.
type Entry struct {
	Path      string        `json:"path,omitzero"`
	RelPath   string        `json:"rel_path,omitzero"`
	Root      string        `json:"root,omitzero"`
	Info      fs.FileInfo   `json:"-"`
	IsDir     bool          `json:"is_dir,omitzero"`
	Content   io.ReadCloser `json:"-"`
	Extension string        `json:"extension,omitzero"`
	Language  string        `json:"language,omitzero"`
	Size      int64         `json:"size,omitzero"`
	Lines     int           `json:"lines,omitzero"`
	Tokens    int           `json:"tokens,omitzero"`
}

// FileResult is an alias for Entry representing a scanned file result.
type FileResult = Entry

// FilterOptions holds configuration options for building a Filter.
type FilterOptions struct {
	IgnorePatterns []string
	IgnoreDotfiles bool
	NamePatterns   []string
	IgnoreDirs     []string
}

// Filter holds compiled regular expression patterns and options used to exclude files and directories during scanning.
type Filter struct {
	nameRegexes    []*regexp2.Regexp
	regexes        []*regexp2.Regexp
	dirRegexes     []*regexp2.Regexp
	ignoreDotfiles bool
}

// NewFilter parses and compiles regex pattern strings, directory ignore patterns, and name filter patterns into a Filter.
func NewFilter(opts FilterOptions) (*Filter, error) {
	if len(opts.IgnorePatterns) == 0 && len(opts.IgnoreDirs) == 0 && len(opts.NamePatterns) == 0 && !opts.IgnoreDotfiles {
		return nil, nil
	}

	var regexes []*regexp2.Regexp
	for _, p := range opts.IgnorePatterns {
		if p == "" {
			continue
		}
		re, err := regexp2.Compile(p, regexp2.None)
		if err != nil {
			return nil, fmt.Errorf("invalid ignore regex pattern %q: %w", p, err)
		}
		regexes = append(regexes, re)
	}

	var dirRegexes []*regexp2.Regexp
	for _, dp := range opts.IgnoreDirs {
		if dp == "" {
			continue
		}
		pattern := GitignoreToRegex(dp)
		if pattern == "" {
			continue
		}
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			return nil, fmt.Errorf("invalid ignore directory pattern %q: %w", dp, err)
		}
		dirRegexes = append(dirRegexes, re)
	}

	var nameRegexes []*regexp2.Regexp
	for _, np := range opts.NamePatterns {
		if np == "" {
			continue
		}
		anchoredPattern := fmt.Sprintf("^(?:%s)$", np)
		re, err := regexp2.Compile(anchoredPattern, regexp2.None)
		if err != nil {
			return nil, fmt.Errorf("invalid name filter regex pattern %q: %w", np, err)
		}
		nameRegexes = append(nameRegexes, re)
	}

	if len(regexes) == 0 && len(dirRegexes) == 0 && len(nameRegexes) == 0 && !opts.IgnoreDotfiles {
		return nil, nil
	}
	return &Filter{
		nameRegexes:    nameRegexes,
		regexes:        regexes,
		dirRegexes:     dirRegexes,
		ignoreDotfiles: opts.IgnoreDotfiles,
	}, nil
}

// ShouldIgnore checks if an entry should be ignored following the strict evaluation order:
// 1. evaluate ignore dot files flag
// 2. ignore directories
// 3. name filter (matches whole name of the file)
// 4. ignore flag
func (f *Filter) ShouldIgnore(name string, relPath string, isDir ...bool) bool {
	if f == nil {
		return false
	}
	var isDirectory bool
	if len(isDir) > 0 {
		isDirectory = isDir[0]
	}
	slashRel := filepath.ToSlash(relPath)

	// Step 1: Evaluate ignore dot files flag
	if f.ignoreDotfiles {
		baseName := filepath.Base(slashRel)
		if strings.HasPrefix(name, ".") || strings.HasPrefix(baseName, ".") {
			return true
		}
	}

	// Step 2: Evaluate ignore directories
	for _, re := range f.dirRegexes {
		if match, _ := re.MatchString(name); match {
			return true
		}
		if match, _ := re.MatchString(slashRel); match {
			return true
		}
	}

	// Step 3: Evaluate name filter (matches whole name of the file)
	// Only files that would match expression should be further analyzed.
	if !isDirectory && len(f.nameRegexes) > 0 {
		matched := false
		for _, re := range f.nameRegexes {
			if match, _ := re.MatchString(name); match {
				matched = true
				break
			}
		}
		if !matched {
			return true
		}
	}

	// Step 4: Evaluate ignore flag
	for _, re := range f.regexes {
		if match, _ := re.MatchString(name); match {
			return true
		}
		if match, _ := re.MatchString(slashRel); match {
			return true
		}
	}

	return false
}


// ScanOptions configures options for directory and file traversal.
type ScanOptions struct {
	ComputeMetrics bool
	Tokenizer      Tokenizer
}

// ScanOption represents a functional option for customizing path scanning.
type ScanOption func(*ScanOptions)

// WithComputeMetrics configures whether line and token metrics should be computed for files.
func WithComputeMetrics(compute bool) ScanOption {
	return func(o *ScanOptions) {
		o.ComputeMetrics = compute
	}
}

// WithTokenizer sets a custom tokenizer for calculating file tokens during scanning.
func WithTokenizer(t Tokenizer) ScanOption {
	return func(o *ScanOptions) {
		o.Tokenizer = t
	}
}

// Hooks for file operations and traversal, allowing unit test error injection.
var (
	walkDirFunc             = filepath.WalkDir
	countLinesAndTokensFunc = CountLinesAndTokens
)

// WalkPaths returns an iter.Seq2[Entry, error] iterator (Go range-over-function)
// that traverses all files and directories specified by paths, filtering out entries
// that match any active ignore patterns in filter.
func WalkPaths(ctx context.Context, paths []string, filter *Filter, opts ...ScanOption) iter.Seq2[Entry, error] {
	var scanOpts ScanOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&scanOpts)
		}
	}

	return func(yield func(Entry, error) bool) {
		for _, root := range paths {
			cleanedRoot := filepath.Clean(root)

			info, err := os.Stat(cleanedRoot)
			if err != nil {
				if !yield(Entry{Path: cleanedRoot, Root: cleanedRoot}, err) {
					return
				}
				continue
			}

			// If root is a single file, test filter and yield if not ignored
			if !info.IsDir() {
				if filter.ShouldIgnore(info.Name(), filepath.Base(cleanedRoot), false) {
					continue
				}

				isBin, rc, binErr := OpenAndSniff(cleanedRoot)
				if binErr != nil {
					if !yield(Entry{Path: cleanedRoot, Root: cleanedRoot}, binErr) {
						return
					}
					continue
				}
				if isBin {
					continue
				}

				var lines, tokens int
				if scanOpts.ComputeMetrics {
					var metricErr error
					lines, tokens, metricErr = countLinesAndTokensFunc(rc)
					_ = rc.Close()
					if metricErr != nil {
						if !yield(Entry{Path: cleanedRoot, Root: cleanedRoot}, metricErr) {
							return
						}
						continue
					}
					rc = nil
				}

				lang, known := LookupLanguage(cleanedRoot)
				stripped := StripAuxiliaryExtensions(cleanedRoot)
				ext := filepath.Ext(stripped)

				displayExt := ext
				if displayExt == "" {
					displayExt = "-"
				}

				displayLang := lang
				if !known || displayLang == "" {
					displayLang = "-"
				}

				entry := Entry{
					Path:      cleanedRoot,
					RelPath:   filepath.Base(cleanedRoot),
					Root:      cleanedRoot,
					Info:      info,
					IsDir:     false,
					Content:   rc,
					Extension: displayExt,
					Language:  displayLang,
					Size:      info.Size(),
					Lines:     lines,
					Tokens:    tokens,
				}
				if !yield(entry, nil) {
					if rc != nil {
						_ = rc.Close()
					}
					return
				}
				if rc != nil {
					_ = rc.Close()
				}
				continue
			}

			// Traverse directory tree
			_ = walkDirFunc(cleanedRoot, func(path string, d fs.DirEntry, walkErr error) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if walkErr != nil {
					if !yield(Entry{Path: path, Root: cleanedRoot}, walkErr) {
						return fs.SkipAll
					}
					return nil
				}

				relPath, err := filepath.Rel(cleanedRoot, path)
				if err != nil {
					relPath = path
				}

				// Check ignore filter for children (skip directory trees via filepath.SkipDir)
				if path != cleanedRoot && filter.ShouldIgnore(d.Name(), relPath, d.IsDir()) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				fileInfo, err := d.Info()
				if err != nil {
					if !yield(Entry{Path: path, Root: cleanedRoot}, err) {
						return fs.SkipAll
					}
					return nil
				}

				if d.IsDir() {
					entry := Entry{
						Path:    path,
						RelPath: relPath,
						Root:    cleanedRoot,
						Info:    fileInfo,
						IsDir:   true,
					}
					if !yield(entry, nil) {
						return fs.SkipAll
					}
					return nil
				}

				isBin, rc, binErr := OpenAndSniff(path)
				if binErr != nil {
					if !yield(Entry{Path: path, Root: cleanedRoot}, binErr) {
						return fs.SkipAll
					}
					return nil
				}
				if isBin {
					return nil
				}

				var lines, tokens int
				if scanOpts.ComputeMetrics {
					var metricErr error
					lines, tokens, metricErr = countLinesAndTokensFunc(rc)
					_ = rc.Close()
					if metricErr != nil {
						if !yield(Entry{Path: path, Root: cleanedRoot}, metricErr) {
							return fs.SkipAll
						}
						return nil
					}
					rc = nil
				}

				lang, known := LookupLanguage(path)
				stripped := StripAuxiliaryExtensions(path)
				ext := filepath.Ext(stripped)

				displayExt := ext
				if displayExt == "" {
					displayExt = "-"
				}

				displayLang := lang
				if !known || displayLang == "" {
					displayLang = "-"
				}

				entry := Entry{
					Path:      path,
					RelPath:   relPath,
					Root:      cleanedRoot,
					Info:      fileInfo,
					IsDir:     false,
					Content:   rc,
					Extension: displayExt,
					Language:  displayLang,
					Size:      fileInfo.Size(),
					Lines:     lines,
					Tokens:    tokens,
				}

				if !yield(entry, nil) {
					if rc != nil {
						_ = rc.Close()
					}
					return fs.SkipAll
				}
				if rc != nil {
					_ = rc.Close()
				}
				return nil
			})

			if ctx.Err() != nil {
				return
			}
		}
	}
}

