package dumper_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/dumper"
	"github.com/grzadr/gomper/internal/scanner"
)

type dummyFileInfo struct {
	name string
	size int64
}

func (d dummyFileInfo) Name() string       { return d.name }
func (d dummyFileInfo) Size() int64        { return d.size }
func (d dummyFileInfo) Mode() fs.FileMode  { return 0644 }
func (d dummyFileInfo) ModTime() time.Time { return time.Now() }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() any           { return nil }

func entriesToSeq(entries []scanner.Entry) iter.Seq2[scanner.Entry, error] {
	return func(yield func(scanner.Entry, error) bool) {
		for _, e := range entries {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		size int64
		want int
	}{
		{-10, 0},
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 1},
		{4, 1},
		{5, 2},
		{8, 2},
		{100, 25},
	}

	for _, tt := range tests {
		if got := dumper.EstimateTokens(tt.size); got != tt.want {
			t.Errorf("EstimateTokens(%d) = %d, want %d", tt.size, got, tt.want)
		}
	}
}

func TestFormatLineNumberedContent(t *testing.T) {
	t.Run("Empty content", func(t *testing.T) {
		var buf bytes.Buffer
		err := dumper.FormatLineNumberedContent(bytes.NewReader(nil), &buf, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := buf.String(); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("Single line with special characters", func(t *testing.T) {
		content := []byte(`fmt.Println("<hello & world>")`)
		var buf bytes.Buffer
		err := dumper.FormatLineNumberedContent(bytes.NewReader(content), &buf, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "1 | fmt.Println(&#34;&lt;hello &amp; world&gt;&#34;)\n"
		if got := buf.String(); got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("Multiple lines with newline at end", func(t *testing.T) {
		content := []byte("line1\nline2\n")
		var buf bytes.Buffer
		err := dumper.FormatLineNumberedContent(bytes.NewReader(content), &buf, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "1 | line1\n2 | line2\n"
		if got := buf.String(); got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("Without XML escaping", func(t *testing.T) {
		content := []byte(`fmt.Println("<hello & world>")`)
		var buf bytes.Buffer
		err := dumper.FormatLineNumberedContent(bytes.NewReader(content), &buf, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "1 | fmt.Println(\"<hello & world>\")\n"
		if got := buf.String(); got != expected {
			t.Errorf("expected %q, got %q", expected, got)
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
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	xmlDumper := dumper.NewXMLDumper(logger)

	mainContent := "package main\n\nfunc main() {}\n"
	customContent := "custom content\n"

	entries := []scanner.Entry{
		{Path: "root", RelPath: ".", IsDir: true, Info: dummyFileInfo{name: ".", size: 0}},
		{Path: "main.go", RelPath: "main.go", IsDir: false, Info: dummyFileInfo{name: "main.go", size: int64(len(mainContent))}, Content: io.NopCloser(strings.NewReader(mainContent))},
		{Path: "data.unknownext", RelPath: "data.unknownext", IsDir: false, Info: dummyFileInfo{name: "data.unknownext", size: int64(len(customContent))}, Content: io.NopCloser(strings.NewReader(customContent))},
	}

	var xmlBuf bytes.Buffer
	err := xmlDumper.GenerateXML(context.Background(), entriesToSeq(entries), "Refactor authentication", &xmlBuf)
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
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	xmlDumper := dumper.NewXMLDumper(logger)

	mainContent := "package main\n\nfunc main() {}\n"

	entries := []scanner.Entry{
		{Path: "root", RelPath: ".", IsDir: true, Info: dummyFileInfo{name: ".", size: 0}},
		{Path: "main.go", RelPath: "main.go", IsDir: false, Info: dummyFileInfo{name: "main.go", size: int64(len(mainContent))}, Content: io.NopCloser(strings.NewReader(mainContent))},
	}

	var mdBuf bytes.Buffer
	err := xmlDumper.GenerateMarkdown(context.Background(), entriesToSeq(entries), "Update dependencies", &mdBuf)
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

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func (errReadCloser) Close() error {
	return nil
}

func TestDumper_EdgeCases(t *testing.T) {
	t.Run("Returns error when entry.Content returns read error during streaming", func(t *testing.T) {
		d := dumper.NewXMLDumper(nil)

		entries := []scanner.Entry{
			{Path: "bad.txt", RelPath: "bad.txt", IsDir: false, Info: dummyFileInfo{name: "bad.txt", size: 100}, Content: errReadCloser{}},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", &xmlBuf)
		if err == nil || !strings.Contains(err.Error(), "simulated read error") {
			t.Errorf("expected streaming read error in GenerateXML, got: %v", err)
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entriesToSeq(entries), "", &mdBuf)
		if err == nil || !strings.Contains(err.Error(), "simulated read error") {
			t.Errorf("expected streaming read error in GenerateMarkdown, got: %v", err)
		}
	})

	t.Run("Handles nil Content gracefully in XML and Markdown", func(t *testing.T) {
		d := dumper.NewXMLDumper(nil)

		entries := []scanner.Entry{
			{Path: "empty.txt", RelPath: "empty.txt", IsDir: false, Info: nil, Content: nil},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", &xmlBuf)
		if err != nil {
			t.Fatalf("unexpected error with nil Content: %v", err)
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entriesToSeq(entries), "", &mdBuf)
		if err != nil {
			t.Fatalf("unexpected error with nil Content: %v", err)
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
			{Path: "test.go", RelPath: "test.go", IsDir: false, Info: dummyFileInfo{name: "test.go", size: 10}, Content: io.NopCloser(strings.NewReader("content"))},
		}

		var buf bytes.Buffer
		if err := d.GenerateXML(ctx, entriesToSeq(entries), "", &buf); err == nil {
			t.Errorf("expected error on canceled context in GenerateXML")
		}
		if err := d.GenerateMarkdown(ctx, entriesToSeq(entries), "", &buf); err == nil {
			t.Errorf("expected error on canceled context in GenerateMarkdown")
		}
	})

	t.Run("Falls back to filepath.Base when RelPath is empty in XML and Markdown", func(t *testing.T) {
		d := dumper.NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "/some/path/standalone.go", RelPath: "", IsDir: false, Info: dummyFileInfo{name: "standalone.go", size: 12}, Content: io.NopCloser(strings.NewReader("package main"))},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", &xmlBuf)
		if err != nil {
			t.Fatalf("unexpected GenerateXML error: %v", err)
		}
		if !strings.Contains(xmlBuf.String(), `path="standalone.go"`) {
			t.Errorf("expected XML dump to use filepath.Base when RelPath is empty, got: %s", xmlBuf.String())
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entriesToSeq(entries), "", &mdBuf)
		if err != nil {
			t.Fatalf("unexpected GenerateMarkdown error: %v", err)
		}
		if !strings.Contains(mdBuf.String(), "### File: `standalone.go`") {
			t.Errorf("expected Markdown dump to use filepath.Base when RelPath is empty, got: %s", mdBuf.String())
		}
	})

	t.Run("Logs scan errors encountered during sequence traversal", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, nil))
		d := dumper.NewXMLDumper(logger)

		seqWithErr := func(yield func(scanner.Entry, error) bool) {
			_ = yield(scanner.Entry{Path: "err_path"}, errors.New("simulated scan error"))
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), seqWithErr, "", &xmlBuf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(logBuf.String(), "dump scan error encountered") {
			t.Errorf("expected log output to record scan error, got: %s", logBuf.String())
		}
	})

	t.Run("BuildDirectoryTree handles root and empty rel paths", func(t *testing.T) {
		entries := []scanner.Entry{
			{RelPath: ".", IsDir: true},
			{RelPath: "", IsDir: true},
			{RelPath: "file.go", IsDir: false},
		}
		tree := dumper.BuildDirectoryTree(entries)
		if !strings.Contains(tree, "file.go") {
			t.Errorf("expected directory tree to contain file.go, got:\n%s", tree)
		}
	})

	t.Run("Returns error when writer fails in FormatLineNumberedContent", func(t *testing.T) {
		err := dumper.FormatLineNumberedContent(strings.NewReader("line1\n"), &failWriter{failOnWrite: true}, false)
		if err == nil {
			t.Errorf("expected write error from FormatLineNumberedContent, got nil")
		}
	})
}

type failWriter struct {
	failOnWrite bool
}

func (f *failWriter) Write(p []byte) (n int, err error) {
	if f.failOnWrite {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}
