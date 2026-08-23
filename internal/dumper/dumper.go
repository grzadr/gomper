package dumper

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"io"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grzadr/gomper/internal/scanner"
)

// Hooks for streaming and staging file operations, allowing test error injection.
var (
	createTempHook    = os.CreateTemp
	formatContentHook = FormatLineNumberedContent
	seekHook          = func(f *os.File, offset int64, whence int) (int64, error) {
		return f.Seek(offset, whence)
	}
)

// EntryExtractor extracts a scanner.Entry from a generic item of type T.
type EntryExtractor[T any] func(T) scanner.Entry

// Dumper generates XML and Markdown context representations of codebases for generic items of type T.
type Dumper[T any] struct {
	logger    *slog.Logger
	extractor EntryExtractor[T]
}

// NewDumper creates a new generic Dumper instance.
func NewDumper[T any](logger *slog.Logger, extractor EntryExtractor[T]) *Dumper[T] {
	if logger == nil {
		logger = slog.Default()
	}
	if extractor == nil {
		extractor = func(t T) scanner.Entry {
			if e, ok := any(t).(scanner.Entry); ok {
				return e
			}
			return scanner.Entry{}
		}
	}
	return new(Dumper[T]{
		logger:    logger,
		extractor: extractor,
	})
}

// XMLDumper is maintained as a type alias / specialized wrapper for scanner.Entry for backward compatibility.
type XMLDumper = Dumper[scanner.Entry]

// NewXMLDumper creates a specialized Dumper for scanner.Entry.
func NewXMLDumper(logger *slog.Logger) *XMLDumper {
	return NewDumper(logger, func(e scanner.Entry) scanner.Entry { return e })
}

// FileMetadata holds extracted metadata for a single file.
type FileMetadata struct {
	Path    string `json:"path,omitzero"`
	RelPath string `json:"rel_path,omitzero"`
	Lang    string `json:"lang,omitzero"`
	Tokens  int    `json:"tokens,omitzero"`
}

// EstimateTokens calculates an approximate token count from file byte size (~4 chars per token).
func EstimateTokens(size int64) int {
	if size <= 0 {
		return 0
	}
	return int((size + 3) / 4)
}

// FormatLineNumberedContent reads from r line by line, prepends 1-indexed line numbers ("1 | ..."),
// optionally applies XML escaping, and writes formatted output directly to w.
func FormatLineNumberedContent(r io.Reader, w io.Writer, escapeXML bool) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		if escapeXML {
			line = html.EscapeString(line)
		}
		if _, err := fmt.Fprintf(w, "%d | %s\n", lineNum, line); err != nil {
			return err
		}
		lineNum++
	}
	return scanner.Err()
}

// TreeNode represents a node in the directory structure hierarchy.
type TreeNode struct {
	Name     string               `json:"name,omitzero"`
	IsDir    bool                 `json:"is_dir,omitzero"`
	Children map[string]*TreeNode `json:"children,omitzero"`
}

// BuildDirectoryTree generates an indented string representation of scanned entries.
func BuildDirectoryTree(entries []scanner.Entry) string {
	root := new(TreeNode{Children: make(map[string]*TreeNode)})

	for _, entry := range entries {
		rel := filepath.ToSlash(entry.RelPath)
		if rel == "." || rel == "" {
			continue
		}

		parts := strings.Split(rel, "/")
		curr := root
		for i, part := range parts {
			if part == "" {
				continue
			}
			isLast := (i == len(parts)-1)
			isDir := !isLast || entry.IsDir

			child, exists := curr.Children[part]
			if !exists {
				child = new(TreeNode{
					Name:     part,
					IsDir:    isDir,
					Children: make(map[string]*TreeNode),
				})
				curr.Children[part] = child
			} else if isDir {
				child.IsDir = true
			}
			curr = child
		}
	}

	var sb strings.Builder
	renderTree(root, 0, &sb)
	return sb.String()
}

func renderTree(node *TreeNode, depth int, sb *strings.Builder) {
	names := make([]string, 0, len(node.Children))
	for name := range node.Children {
		names = append(names, name)
	}
	sort.Strings(names)

	indent := strings.Repeat("  ", depth)
	for _, name := range names {
		child := node.Children[name]
		if child.IsDir {
			fmt.Fprintf(sb, "%s%s/\n", indent, child.Name)
			renderTree(child, depth+1, sb)
		} else {
			fmt.Fprintf(sb, "%s%s\n", indent, child.Name)
		}
	}
}

// GenerateXML writes the XML document for the provided scanned entries to targetWriter.
func (d *Dumper[T]) GenerateXML(ctx context.Context, seq iter.Seq2[T, error], instructions string, targetWriter io.Writer) error {
	return d.dumpStream(ctx, seq, true, instructions, targetWriter)
}

// GenerateMarkdown writes a Markdown document representation for scanned entries to targetWriter.
func (d *Dumper[T]) GenerateMarkdown(ctx context.Context, seq iter.Seq2[T, error], instructions string, targetWriter io.Writer) error {
	return d.dumpStream(ctx, seq, false, instructions, targetWriter)
}

func (d *Dumper[T]) dumpStream(ctx context.Context, seq iter.Seq2[T, error], isXML bool, instructions string, targetWriter io.Writer) error {
	stagingFile, err := createTempHook("", "gomper-stage-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create staging file: %w", err)
	}
	defer func() {
		_ = stagingFile.Close()
		_ = os.Remove(stagingFile.Name())
	}()

	var metas []FileMetadata
	var treeEntries []scanner.Entry
	totalTokens := 0

	for item, err := range seq {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		entry := d.extractor(item)
		if err != nil {
			d.logger.ErrorContext(ctx, "dump scan error encountered",
				slog.String("path", entry.Path),
				slog.Any("error", err),
			)
			continue
		}

		if entry.IsDir {
			treeEntries = append(treeEntries, scanner.Entry{
				Path:    entry.Path,
				RelPath: entry.RelPath,
				IsDir:   entry.IsDir,
			})
			continue
		}

		lang, known := scanner.LookupLanguage(entry.Path)
		stripped := scanner.StripAuxiliaryExtensions(entry.Path)
		ext := filepath.Ext(stripped)

		if ext == "" && !known {
			d.logger.WarnContext(ctx, "ignoring file without extension",
				slog.String("path", entry.Path),
			)
			continue
		}

		if !known {
			d.logger.InfoContext(ctx, "unsupported file extension encountered",
				slog.String("path", entry.Path),
				slog.String("ext", ext),
			)
		}

		treeEntries = append(treeEntries, scanner.Entry{
			Path:    entry.Path,
			RelPath: entry.RelPath,
			IsDir:   entry.IsDir,
		})

		var size int64
		if entry.Info != nil {
			size = entry.Info.Size()
		}

		tokens := EstimateTokens(size)
		totalTokens += tokens

		relPath := filepath.ToSlash(entry.RelPath)
		if relPath == "" || relPath == "." {
			relPath = filepath.Base(entry.Path)
		}

		meta := FileMetadata{
			Path:    entry.Path,
			RelPath: relPath,
			Lang:    lang,
			Tokens:  tokens,
		}
		metas = append(metas, meta)

		if isXML {
			_, _ = fmt.Fprintf(stagingFile, "  <file path=%q language=%q tokens=\"%d\">\n", meta.RelPath, meta.Lang, meta.Tokens)
			if entry.Content != nil {
				if err := formatContentHook(entry.Content, stagingFile, true); err != nil {
					return err
				}
			}
			_, _ = fmt.Fprintln(stagingFile, `  </file>`)
		} else {
			_, _ = fmt.Fprintf(stagingFile, "### File: `%s`\n", meta.RelPath)
			_, _ = fmt.Fprintf(stagingFile, "- **Language**: %s\n", meta.Lang)
			_, _ = fmt.Fprintf(stagingFile, "- **Tokens**: %d\n\n", meta.Tokens)
			_, _ = fmt.Fprintf(stagingFile, "```%s\n", meta.Lang)
			if entry.Content != nil {
				if err := formatContentHook(entry.Content, stagingFile, false); err != nil {
					return err
				}
			}
			_, _ = fmt.Fprintln(stagingFile, "```")
			_, _ = fmt.Fprintln(stagingFile)
		}
	}

	dirTree := BuildDirectoryTree(treeEntries)
	w := bufio.NewWriter(targetWriter)
	defer func() { _ = w.Flush() }()

	if isXML {
		_, _ = fmt.Fprintln(w, `<codebase_context version="1.0">`)
		_, _ = fmt.Fprintln(w, `<header>`)
		_, _ = fmt.Fprintln(w, `  <summary>`)
		_, _ = fmt.Fprintln(w, `    <generation_info>Generated by Automated Repository Packer</generation_info>`)
		_, _ = fmt.Fprintln(w, `    <purpose>Merged representation of the codebase for automated code review, refactoring, or feature implementation.</purpose>`)
		_, _ = fmt.Fprintf(w, "    <total_files>%d</total_files>\n", len(metas))
		_, _ = fmt.Fprintf(w, "    <total_tokens>%d</total_tokens>\n", totalTokens)
		_, _ = fmt.Fprintln(w, `  </summary>`)
		_, _ = fmt.Fprintln(w, `  <usage_guidelines>`)
		_, _ = fmt.Fprintln(w, `    - Treat this document as a read-only repository snapshot.`)
		_, _ = fmt.Fprintln(w, `    - Reference specific files using their full relative paths as defined in the path attribute.`)
		_, _ = fmt.Fprintln(w, `    - When generating code changes, specify line numbers or provide complete functional blocks.`)
		_, _ = fmt.Fprintln(w, `  </usage_guidelines>`)
		_, _ = fmt.Fprintln(w, `  <user_instructions>`)
		if instructions != "" {
			_, _ = fmt.Fprintf(w, "    %s\n", html.EscapeString(instructions))
		}
		_, _ = fmt.Fprintln(w, `  </user_instructions>`)
		_, _ = fmt.Fprintln(w, `</header>`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `<directory_structure>`)
		_, _ = w.WriteString(dirTree)
		_, _ = fmt.Fprintln(w, `</directory_structure>`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `<files>`)
	} else {
		_, _ = fmt.Fprintln(w, `# Codebase Context (v1.0)`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `## Header`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `### Summary`)
		_, _ = fmt.Fprintln(w, `- **Generation Info**: Generated by Automated Repository Packer`)
		_, _ = fmt.Fprintln(w, `- **Purpose**: Merged representation of the codebase for automated code review, refactoring, or feature implementation.`)
		_, _ = fmt.Fprintf(w, "- **Total Files**: %d\n", len(metas))
		_, _ = fmt.Fprintf(w, "- **Total Tokens**: %d\n", totalTokens)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `### Usage Guidelines`)
		_, _ = fmt.Fprintln(w, `- Treat this document as a read-only repository snapshot.`)
		_, _ = fmt.Fprintln(w, `- Reference specific files using their full relative paths as defined in the path attribute.`)
		_, _ = fmt.Fprintln(w, `- When generating code changes, specify line numbers or provide complete functional blocks.`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `### User Instructions`)
		if instructions != "" {
			_, _ = fmt.Fprintln(w, instructions)
		} else {
			_, _ = fmt.Fprintln(w, `None`)
		}
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `## Directory Structure`)
		_, _ = fmt.Fprintln(w, "```")
		_, _ = w.WriteString(dirTree)
		_, _ = fmt.Fprintln(w, "```")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `## Files`)
		_, _ = fmt.Fprintln(w)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	if _, err := seekHook(stagingFile, 0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek staging file: %w", err)
	}

	if _, err := io.Copy(targetWriter, stagingFile); err != nil {
		return fmt.Errorf("failed to copy staging file content: %w", err)
	}

	if isXML {
		_, _ = fmt.Fprintln(targetWriter, `</files>`)
		_, _ = fmt.Fprintln(targetWriter, `</codebase_context>`)
	}

	return nil
}
