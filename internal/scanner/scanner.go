package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Entry encapsulates metadata and fs.FileInfo for a scanned file or directory.
type Entry struct {
	Path    string
	RelPath string
	Root    string
	Info    fs.FileInfo
	IsDir   bool
}

// Filter holds compiled regular expression patterns and options used to exclude files and directories during scanning.
type Filter struct {
	regexes        []*regexp.Regexp
	ignoreDotfiles bool
}

// NewFilter parses and compiles regex pattern strings into a Filter with optional dotfile exclusion.
func NewFilter(patterns []string, ignoreDotfiles bool) (*Filter, error) {
	if len(patterns) == 0 && !ignoreDotfiles {
		return nil, nil
	}

	var regexes []*regexp.Regexp
	for _, p := range patterns {
		if p == "" {
			continue
		}
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid ignore regex pattern %q: %w", p, err)
		}
		regexes = append(regexes, re)
	}

	if len(regexes) == 0 && !ignoreDotfiles {
		return nil, nil
	}
	return &Filter{
		regexes:        regexes,
		ignoreDotfiles: ignoreDotfiles,
	}, nil
}

// ShouldIgnore checks if the given entry name or relative path matches any ignore regex or dotfile rule.
func (f *Filter) ShouldIgnore(name string, relPath string) bool {
	if f == nil {
		return false
	}
	if f.ignoreDotfiles {
		baseName := filepath.Base(relPath)
		if strings.HasPrefix(name, ".") || strings.HasPrefix(baseName, ".") {
			return true
		}
	}
	for _, re := range f.regexes {
		if re.MatchString(name) || re.MatchString(relPath) {
			return true
		}
	}
	return false
}

// WalkPaths returns an iter.Seq2[Entry, error] iterator (Go range-over-function)
// that traverses all files and directories specified by paths, filtering out entries
// that match any active ignore patterns in filter.
func WalkPaths(ctx context.Context, paths []string, filter *Filter) iter.Seq2[Entry, error] {
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
				if filter.ShouldIgnore(info.Name(), filepath.Base(cleanedRoot)) {
					continue
				}
				entry := Entry{
					Path:    cleanedRoot,
					RelPath: filepath.Base(cleanedRoot),
					Root:    cleanedRoot,
					Info:    info,
					IsDir:   false,
				}
				if !yield(entry, nil) {
					return
				}
				continue
			}

			// Traverse directory tree
			_ = filepath.WalkDir(cleanedRoot, func(path string, d fs.DirEntry, walkErr error) error {
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
				if path != cleanedRoot && filter.ShouldIgnore(d.Name(), relPath) {
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

				entry := Entry{
					Path:    path,
					RelPath: relPath,
					Root:    cleanedRoot,
					Info:    fileInfo,
					IsDir:   d.IsDir(),
				}

				if !yield(entry, nil) {
					return fs.SkipAll
				}
				return nil
			})

			if ctx.Err() != nil {
				return
			}
		}
	}
}
