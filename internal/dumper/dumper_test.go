package dumper_test

import (
	"bytes"
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/dumper"
	"github.com/grzadr/gomper/internal/scanner"
)

type dummyFileInfo struct {
	name string
}

func (d dummyFileInfo) Name() string       { return d.name }
func (d dummyFileInfo) Size() int64        { return 100 }
func (d dummyFileInfo) Mode() fs.FileMode  { return 0644 }
func (d dummyFileInfo) ModTime() time.Time { return time.Now() }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() any           { return nil }

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input []byte
		want  int
	}{
		{nil, 0},
		{[]byte{}, 0},
		{[]byte("a"), 1},
		{[]byte("ab"), 1},
		{[]byte("abc"), 1},
		{[]byte("abcd"), 1},
		{[]byte("abcde"), 2},
		{[]byte("12345678"), 2},
	}

	for _, tt := range tests {
		if got := dumper.EstimateTokens(tt.input); got != tt.want {
			t.Errorf("EstimateTokens(%q) = %d, want %d", string(tt.input), got, tt.want)
		}
	}
}

func TestFormatLineNumberedContent(t *testing.T) {
	t.Run("Empty content", func(t *testing.T) {
		if got := dumper.FormatLineNumberedContent(nil); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("Single line with special characters", func(t *testing.T) {
		content := []byte(`fmt.Println("<hello & world>")`)
		formatted := dumper.FormatLineNumberedContent(content)
		expected := "1 | fmt.Println(&#34;&lt;hello &amp; world&gt;&#34;)\n"
		if formatted != expected {
			t.Errorf("expected %q, got %q", expected, formatted)
		}
	})

	t.Run("Multiple lines with newline at end", func(t *testing.T) {
		content := []byte("line1\nline2\n")
		formatted := dumper.FormatLineNumberedContent(content)
		expected := "1 | line1\n2 | line2\n"
		if formatted != expected {
			t.Errorf("expected %q, got %q", expected, formatted)
		}
	})
}

func TestBuildDirectoryTree(t *testing.T) {
	entries := []scanner.Entry{
		{RelPath: "src", IsDir: true},
		{RelPath: "src/auth", IsDir: true},
		{RelPath: "/src/auth//jwt.ts", IsDir: false},
		{RelPath: "src/auth/middleware.ts", IsDir: false},
		{RelPath: "src/user", IsDir: true},
		{RelPath: "src/user/controller.ts", IsDir: false},
		{RelPath: "src/index.ts", IsDir: false},
	}

	tree := dumper.BuildDirectoryTree(entries)
	expectedLines := []string{
		"src/",
		"  auth/",
		"    jwt.ts",
		"    middleware.ts",
		"  user/",
		"    controller.ts",
		"  index.ts",
	}

	for _, line := range expectedLines {
		if !strings.Contains(tree, line) {
			t.Errorf("expected directory tree to contain line %q, got tree:\n%s", line, tree)
		}
	}
}

func TestGenerateXML(t *testing.T) {
	tempDir := t.TempDir()

	goFile := filepath.Join(tempDir, "main.go")
	_ = os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	customFile := filepath.Join(tempDir, "data.unknownext")
	_ = os.WriteFile(customFile, []byte("custom content\n"), 0644)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	xmlDumper := dumper.NewXMLDumper(logger)

	entries := []scanner.Entry{
		{Path: tempDir, RelPath: ".", IsDir: true, Info: dummyFileInfo{"."}},
		{Path: goFile, RelPath: "main.go", IsDir: false, Info: dummyFileInfo{"main.go"}},
		{Path: customFile, RelPath: "data.unknownext", IsDir: false, Info: dummyFileInfo{"data.unknownext"}},
	}

	var xmlBuf bytes.Buffer
	err := xmlDumper.GenerateXML(context.Background(), entries, "Refactor authentication", &xmlBuf)
	if err != nil {
		t.Fatalf("unexpected GenerateXML error: %v", err)
	}

	xmlStr := xmlBuf.String()

	// Assert XML structure
	if !strings.Contains(xmlStr, `<codebase_context version="1.0">`) {
		t.Errorf("expected XML root element codebase_context")
	}
	if !strings.Contains(xmlStr, `<total_files>2</total_files>`) {
		t.Errorf("expected total_files to be 2, got output:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<user_instructions>`) || !strings.Contains(xmlStr, `Refactor authentication`) {
		t.Errorf("expected user instructions in XML header")
	}
	if !strings.Contains(xmlStr, `<file path="main.go" language="go"`) {
		t.Errorf("expected file entry for main.go with language=go")
	}
	if !strings.Contains(xmlStr, `1 | package main`) {
		t.Errorf("expected line-numbered content in file block")
	}

	// Assert unknown extension was logged at info level
	logStr := logBuf.String()
	if !strings.Contains(logStr, "unsupported file extension encountered") || !strings.Contains(logStr, ".unknownext") {
		t.Errorf("expected log output to record unsupported extension .unknownext, got: %q", logStr)
	}
}

func TestGenerateMarkdown(t *testing.T) {
	tempDir := t.TempDir()

	goFile := filepath.Join(tempDir, "main.go")
	_ = os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	xmlDumper := dumper.NewXMLDumper(logger)

	entries := []scanner.Entry{
		{Path: tempDir, RelPath: ".", IsDir: true, Info: dummyFileInfo{"."}},
		{Path: goFile, RelPath: "main.go", IsDir: false, Info: dummyFileInfo{"main.go"}},
	}

	var mdBuf bytes.Buffer
	err := xmlDumper.GenerateMarkdown(context.Background(), entries, "Update dependencies", &mdBuf)
	if err != nil {
		t.Fatalf("unexpected GenerateMarkdown error: %v", err)
	}

	mdStr := mdBuf.String()

	if !strings.Contains(mdStr, "# Codebase Context (v1.0)") {
		t.Errorf("expected header '# Codebase Context (v1.0)', got: %s", mdStr)
	}
	if !strings.Contains(mdStr, "- **Total Files**: 1") {
		t.Errorf("expected Total Files 1")
	}
	if !strings.Contains(mdStr, "Update dependencies") {
		t.Errorf("expected user instructions in markdown header")
	}
	if !strings.Contains(mdStr, "### File: `main.go`") || !strings.Contains(mdStr, "- **Language**: go") {
		t.Errorf("expected file section with main.go and language go")
	}
	if !strings.Contains(mdStr, "1 | package main") {
		t.Errorf("expected line-numbered code content")
	}
}

func TestNewXMLDumper_NilLogger(t *testing.T) {
	d := dumper.NewXMLDumper(nil)
	if d == nil {
		t.Fatalf("expected non-nil dumper when initialized with nil logger")
	}
}

func TestDumper_EdgeCases(t *testing.T) {
	t.Run("Skips non-UTF8 binary file and unreadable file in XML and Markdown", func(t *testing.T) {
		tempDir := t.TempDir()

		// Non-UTF8 binary file
		binaryFile := filepath.Join(tempDir, "bin.dat")
		_ = os.WriteFile(binaryFile, []byte{0xff, 0xfe, 0xfd, 0x00}, 0644)

		// Unreadable file
		unreadableFile := filepath.Join(tempDir, "unreadable.txt")
		if err := os.WriteFile(unreadableFile, []byte("secret"), 0000); err != nil {
			t.Skip("skipping unreadable file test")
		}
		defer func() { _ = os.Chmod(unreadableFile, 0644) }()

		// Unknown extension file for Markdown
		unknownFile := filepath.Join(tempDir, "data.unknown")
		_ = os.WriteFile(unknownFile, []byte("data"), 0644)

		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, nil))
		d := dumper.NewXMLDumper(logger)

		entries := []scanner.Entry{
			{Path: binaryFile, RelPath: "", IsDir: false, Info: dummyFileInfo{"bin.dat"}},
			{Path: unreadableFile, RelPath: "unreadable.txt", IsDir: false, Info: dummyFileInfo{"unreadable.txt"}},
			{Path: unknownFile, RelPath: "data.unknown", IsDir: false, Info: dummyFileInfo{"data.unknown"}},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entries, "", &xmlBuf)
		if err != nil {
			t.Fatalf("unexpected GenerateXML error: %v", err)
		}
		if strings.Contains(xmlBuf.String(), "<file path=\"bin.dat\"") || strings.Contains(xmlBuf.String(), "<file path=\"unreadable.txt\"") {
			t.Errorf("expected binary and unreadable file content blocks to be omitted in XML dump, got: %s", xmlBuf.String())
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entries, "", &mdBuf)
		if err != nil {
			t.Fatalf("unexpected GenerateMarkdown error: %v", err)
		}
		if strings.Contains(mdBuf.String(), "### File: `bin.dat`") || strings.Contains(mdBuf.String(), "### File: `unreadable.txt`") {
			t.Errorf("expected binary and unreadable file content blocks to be omitted in Markdown dump, got: %s", mdBuf.String())
		}
		if !strings.Contains(mdBuf.String(), "User Instructions\nNone") {
			t.Errorf("expected 'None' under User Instructions when empty")
		}
	})

	t.Run("Aborts XML and Markdown dump on canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		d := dumper.NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "test.go", RelPath: "test.go", IsDir: false, Info: dummyFileInfo{"test.go"}},
		}

		var buf bytes.Buffer
		if err := d.GenerateXML(ctx, entries, "", &buf); err == nil {
			t.Errorf("expected error on canceled context in GenerateXML")
		}
		if err := d.GenerateMarkdown(ctx, entries, "", &buf); err == nil {
			t.Errorf("expected error on canceled context in GenerateMarkdown")
		}
	})

	t.Run("Falls back to filepath.Base when RelPath is empty in XML and Markdown", func(t *testing.T) {
		tempDir := t.TempDir()
		sampleFile := filepath.Join(tempDir, "standalone.go")
		_ = os.WriteFile(sampleFile, []byte("package main"), 0644)

		d := dumper.NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: sampleFile, RelPath: "", IsDir: false, Info: dummyFileInfo{"standalone.go"}},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entries, "", &xmlBuf)
		if err != nil {
			t.Fatalf("unexpected GenerateXML error: %v", err)
		}
		if !strings.Contains(xmlBuf.String(), `path="standalone.go"`) {
			t.Errorf("expected XML dump to use filepath.Base when RelPath is empty, got: %s", xmlBuf.String())
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entries, "", &mdBuf)
		if err != nil {
			t.Fatalf("unexpected GenerateMarkdown error: %v", err)
		}
		if !strings.Contains(mdBuf.String(), "### File: `standalone.go`") {
			t.Errorf("expected Markdown dump to use filepath.Base when RelPath is empty, got: %s", mdBuf.String())
		}
	})
}

