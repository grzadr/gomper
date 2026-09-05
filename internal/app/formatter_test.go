package app_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/app"
	"github.com/grzadr/gomper/internal/scanner"
)

type dummyFileInfo struct {
	mode os.FileMode
}

func (d dummyFileInfo) Name() string       { return "dummy" }
func (d dummyFileInfo) Size() int64        { return 100 }
func (d dummyFileInfo) Mode() os.FileMode  { return d.mode }
func (d dummyFileInfo) ModTime() time.Time { return time.Now() }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() any           { return nil }

func TestStandardFormatter(t *testing.T) {
	t.Run("Standard path list output", func(t *testing.T) {
		f := app.NewStandardFormatter(false)
		if f.RequiresMetrics() {
			t.Errorf("expected RequiresMetrics to be false")
		}

		var buf bytes.Buffer
		if err := f.WriteHeader(&buf); err != nil {
			t.Fatalf("unexpected WriteHeader error: %v", err)
		}

		entries := []scanner.Entry{
			{Path: "/root/file1.go", RelPath: "file1.go"},
			{Path: "/root/file2.md", RelPath: "file2.md"},
			{Path: "/root/empty_rel.txt", RelPath: ""},
			{Path: "/root/dot_rel.txt", RelPath: "."},
		}

		for _, e := range entries {
			if err := f.FormatEntry(&buf, e); err != nil {
				t.Fatalf("unexpected FormatEntry error: %v", err)
			}
		}

		if err := f.Flush(&buf); err != nil {
			t.Fatalf("unexpected Flush error: %v", err)
		}

		expected := "file1.go\nfile2.md\n/root/empty_rel.txt\n/root/dot_rel.txt\n"
		if buf.String() != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, buf.String())
		}
	})

	t.Run("Long format output with file attributes", func(t *testing.T) {
		f := app.NewStandardFormatter(true)
		var buf bytes.Buffer
		entries := []scanner.Entry{
			{
				Path:    "/root/file1.go",
				RelPath: "file1.go",
				Size:    1024,
				Info:    dummyFileInfo{mode: 0644},
			},
			{
				Path:    "/root/no_info.go",
				RelPath: "no_info.go",
				Size:    512,
				Info:    nil,
			},
		}

		for _, e := range entries {
			if err := f.FormatEntry(&buf, e); err != nil {
				t.Fatalf("unexpected FormatEntry error: %v", err)
			}
		}

		out := buf.String()
		if !strings.Contains(out, "FILE        1024 B  -rw-r--r--  file1.go") {
			t.Errorf("expected long format line for file1.go, got:\n%s", out)
		}
		if !strings.Contains(out, "FILE         512 B    no_info.go") {
			t.Errorf("expected long format line for no_info.go, got:\n%s", out)
		}
	})
}

func TestDetailedFormatter(t *testing.T) {
	f := app.NewDetailedFormatter()
	if !f.RequiresMetrics() {
		t.Errorf("expected RequiresMetrics to be true")
	}

	t.Run("Empty entries renders header", func(t *testing.T) {
		var buf bytes.Buffer
		if err := f.WriteHeader(&buf); err != nil {
			t.Fatalf("unexpected WriteHeader error: %v", err)
		}
		if err := f.Flush(&buf); err != nil {
			t.Fatalf("unexpected Flush error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "SIZE") || !strings.Contains(out, "LINES") || !strings.Contains(out, "TOKENS") || !strings.Contains(out, "EXTENSION") || !strings.Contains(out, "LANGUAGE") || !strings.Contains(out, "FILE") {
			t.Errorf("expected fixed-width header in output, got:\n%s", out)
		}
	})

	t.Run("Formats entries in aligned tabular format with fallbacks", func(t *testing.T) {
		formatter := app.NewDetailedFormatter()
		var buf bytes.Buffer
		if err := formatter.WriteHeader(&buf); err != nil {
			t.Fatalf("unexpected WriteHeader error: %v", err)
		}

		entries := []scanner.Entry{
			{
				Path:      "/root/main.go",
				RelPath:   "main.go",
				Extension: ".go",
				Language:  "go",
				Size:      250,
				Lines:     20,
				Tokens:    45,
			},
			{
				Path:      "/root/nested/doc.md",
				RelPath:   "",
				Extension: "",
				Language:  "",
				Size:      0,
				Info:      dummyFileInfo{mode: 0644},
				Lines:     80,
				Tokens:    300,
			},
			{
				Path:      "/root/Makefile",
				RelPath:   "Makefile",
				Extension: "-",
				Language:  "makefile",
				Size:      1024,
				Lines:     50,
				Tokens:    120,
			},
		}

		for _, e := range entries {
			if err := formatter.FormatEntry(&buf, e); err != nil {
				t.Fatalf("unexpected FormatEntry error: %v", err)
			}
		}
		if err := formatter.Flush(&buf); err != nil {
			t.Fatalf("unexpected Flush error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "250 B") || !strings.Contains(out, "main.go") || !strings.Contains(out, ".go") || !strings.Contains(out, "go") {
			t.Errorf("expected main.go details in table output, got:\n%s", out)
		}
		if !strings.Contains(out, "100 B") || !strings.Contains(out, "/root/nested/doc.md") || !strings.Contains(out, "-") {
			t.Errorf("expected fallback path and info size in table output, got:\n%s", out)
		}
		if !strings.Contains(out, "1024 B") || !strings.Contains(out, "Makefile") || !strings.Contains(out, "makefile") {
			t.Errorf("expected Makefile details in table output, got:\n%s", out)
		}
	})
}

func TestJSONFormatter(t *testing.T) {
	f := app.NewJSONFormatter()
	if !f.RequiresMetrics() {
		t.Errorf("expected RequiresMetrics to be true")
	}

	t.Run("Empty entries renders empty JSON array", func(t *testing.T) {
		formatter := app.NewJSONFormatter()
		var buf bytes.Buffer
		if err := formatter.WriteHeader(&buf); err != nil {
			t.Fatalf("unexpected WriteHeader error: %v", err)
		}
		if err := formatter.Flush(&buf); err != nil {
			t.Fatalf("unexpected Flush error: %v", err)
		}
		out := buf.String()
		if out != "[\n]\n" {
			t.Errorf("expected empty array output '[\n]\n', got: %q", out)
		}
	})

	t.Run("Single entry renders valid JSON array with fallbacks", func(t *testing.T) {
		formatter := app.NewJSONFormatter()
		var buf bytes.Buffer
		if err := formatter.WriteHeader(&buf); err != nil {
			t.Fatalf("unexpected WriteHeader error: %v", err)
		}

		entry := scanner.Entry{
			Path:      "/root/nested/doc.md",
			RelPath:   "",
			Extension: ".md",
			Language:  "markdown",
			Size:      0,
			Info:      dummyFileInfo{mode: 0644},
			Lines:     80,
			Tokens:    300,
		}

		if err := formatter.FormatEntry(&buf, entry); err != nil {
			t.Fatalf("unexpected FormatEntry error: %v", err)
		}
		if err := formatter.Flush(&buf); err != nil {
			t.Fatalf("unexpected Flush error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, `"/root/nested/doc.md"`) {
			t.Errorf("expected fallback path in JSON output, got:\n%s", out)
		}
		if !strings.Contains(out, `"size": 100`) {
			t.Errorf("expected fallback size in JSON output, got:\n%s", out)
		}
	})

	t.Run("Multiple entries render valid JSON array", func(t *testing.T) {
		formatter := app.NewJSONFormatter()
		var buf bytes.Buffer
		if err := formatter.WriteHeader(&buf); err != nil {
			t.Fatalf("unexpected WriteHeader error: %v", err)
		}

		entries := []scanner.Entry{
			{
				Path:      "/root/main.go",
				RelPath:   "main.go",
				Extension: ".go",
				Language:  "go",
				Size:      250,
				Lines:     20,
				Tokens:    45,
			},
			{
				Path:      "/root/Makefile",
				RelPath:   ".",
				Extension: "-",
				Language:  "makefile",
				Size:      1024,
				Lines:     50,
				Tokens:    120,
			},
		}

		for _, e := range entries {
			if err := formatter.FormatEntry(&buf, e); err != nil {
				t.Fatalf("unexpected FormatEntry error: %v", err)
			}
		}
		if err := formatter.Flush(&buf); err != nil {
			t.Fatalf("unexpected Flush error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, `"rel_path": "main.go"`) || !strings.Contains(out, `"rel_path": "/root/Makefile"`) {
			t.Errorf("expected both entries in JSON output, got:\n%s", out)
		}
	})

	t.Run("I/O error handling", func(t *testing.T) {
		// WriteHeader error
		f1 := app.NewJSONFormatter()
		if err := f1.WriteHeader(new(errWriter{failOnWrite: 1})); err == nil {
			t.Errorf("expected WriteHeader error with failing writer, got nil")
		}

		// FormatEntry first entry error
		f2 := app.NewJSONFormatter()
		_ = f2.WriteHeader(new(bytes.Buffer))
		if err := f2.FormatEntry(new(errWriter{failOnWrite: 1}), scanner.Entry{Path: "test"}); err == nil {
			t.Errorf("expected FormatEntry error on first entry with failing writer, got nil")
		}

		// FormatEntry second entry comma error
		f3 := app.NewJSONFormatter()
		var buf3 bytes.Buffer
		_ = f3.WriteHeader(&buf3)
		_ = f3.FormatEntry(&buf3, scanner.Entry{Path: "test1"})
		if err := f3.FormatEntry(new(errWriter{failOnWrite: 1}), scanner.Entry{Path: "test2"}); err == nil {
			t.Errorf("expected FormatEntry error on second entry comma write, got nil")
		}

		// Flush with count > 0 error
		f4 := app.NewJSONFormatter()
		var buf4 bytes.Buffer
		_ = f4.WriteHeader(&buf4)
		_ = f4.FormatEntry(&buf4, scanner.Entry{Path: "test1"})
		if err := f4.Flush(new(errWriter{failOnWrite: 1})); err == nil {
			t.Errorf("expected Flush error on non-empty array with failing writer, got nil")
		}

		// Flush with count == 0 error
		f5 := app.NewJSONFormatter()
		_ = f5.WriteHeader(new(bytes.Buffer))
		if err := f5.Flush(new(errWriter{failOnWrite: 1})); err == nil {
			t.Errorf("expected Flush error on empty array with failing writer, got nil")
		}
	})
}

type errWriter struct {
	failOnWrite int
	writes      int
}

func (e *errWriter) Write(p []byte) (int, error) {
	e.writes++
	if e.failOnWrite == 0 || e.writes >= e.failOnWrite {
		return 0, os.ErrPermission
	}
	return len(p), nil
}

func TestNewListFormatter(t *testing.T) {
	tests := []struct {
		format   string
		long     bool
		wantType string
		wantErr  bool
	}{
		{"", false, "*app.StandardFormatter", false},
		{"standard", false, "*app.StandardFormatter", false},
		{" STANDARD ", true, "*app.StandardFormatter", false},
		{"detailed", false, "*app.DetailedFormatter", false},
		{" Detailed ", false, "*app.DetailedFormatter", false},
		{"json", false, "*app.JSONFormatter", false},
		{" JSON ", true, "*app.JSONFormatter", false},
		{"invalid", false, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter, err := app.NewListFormatter(tt.format, tt.long)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewListFormatter(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
			if !tt.wantErr && formatter == nil {
				t.Fatalf("expected non-nil formatter for format %q", tt.format)
			}
		})
	}
}

var benchEntry = scanner.Entry{
	Path:      "/root/main.go",
	RelPath:   "main.go",
	Extension: ".go",
	Language:  "go",
	Size:      4096,
	Lines:     200,
	Tokens:    800,
}

func BenchmarkJSONFormatter_FormatEntry(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		f := app.NewJSONFormatter()
		_ = f.WriteHeader(&buf)
		_ = f.FormatEntry(&buf, benchEntry)
		_ = f.Flush(&buf)
	}
}

func BenchmarkDetailedFormatter_FormatEntry(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		f := app.NewDetailedFormatter()
		_ = f.WriteHeader(&buf)
		_ = f.FormatEntry(&buf, benchEntry)
		_ = f.Flush(&buf)
	}
}
