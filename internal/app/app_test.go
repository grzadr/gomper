package app_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/app"
	"github.com/grzadr/gomper/internal/setup"
)

func TestService_NewServiceAndLogger(t *testing.T) {
	appInst := setup.NewApp(slog.LevelDebug)
	svc := app.NewService(appInst)
	if svc == nil {
		t.Fatalf("expected non-nil Service")
	}

	svcNil := app.NewService(nil)
	if svcNil == nil {
		t.Fatalf("expected non-nil Service when initialized with nil app")
	}

	tempDir := t.TempDir()
	file1 := filepath.Join(tempDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("hello"), 0644)

	outBuf := new(bytes.Buffer)
	opts := app.ListOptions{}
	err := svcNil.List(context.Background(), outBuf, []string{tempDir}, opts)
	if err != nil {
		t.Fatalf("unexpected error running List with nil app: %v", err)
	}

	dumpOpts := app.DumpOptions{Format: app.FormatMarkdown}
	err = svcNil.Dump(context.Background(), outBuf, []string{tempDir}, dumpOpts)
	if err != nil {
		t.Fatalf("unexpected error running Dump with nil app: %v", err)
	}
}

func TestService_List(t *testing.T) {
	appInst := setup.NewApp(slog.LevelDebug)
	svc := app.NewService(appInst)

	t.Run("Standard list execution excludes directories", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")
		_ = os.Mkdir(subDir, 0755)
		file1 := filepath.Join(tempDir, "sample.txt")
		_ = os.WriteFile(file1, []byte("data"), 0644)

		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{}

		err := svc.List(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "sample.txt") {
			t.Errorf("expected output to contain sample.txt, got: %q", output)
		}
		if strings.Contains(output, "subdir") {
			t.Errorf("expected directory 'subdir' to be excluded by default from list output, got: %q", output)
		}
	})

	t.Run("List with LongFormat option", func(t *testing.T) {
		tempDir := t.TempDir()
		file1 := filepath.Join(tempDir, "data.bin")
		_ = os.WriteFile(file1, []byte("12345"), 0644)

		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{LongFormat: true}

		err := svc.List(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "FILE") || !strings.Contains(output, "5 B") {
			t.Errorf("expected long format attributes in output, got: %q", output)
		}
	})

	t.Run("List with DetailedFormatter option", func(t *testing.T) {
		tempDir := t.TempDir()
		file1 := filepath.Join(tempDir, "main.go")
		_ = os.WriteFile(file1, []byte("package main\n\nfunc main() {}\n"), 0644)

		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{
			Formatter: app.NewDetailedFormatter(),
		}

		err := svc.List(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "FILE") || !strings.Contains(output, "EXTENSION") || !strings.Contains(output, "LANGUAGE") || !strings.Contains(output, "SIZE") || !strings.Contains(output, "LINES") || !strings.Contains(output, "TOKENS") {
			t.Errorf("expected table headers with LANGUAGE in detailed format output, got: %q", output)
		}
		if !strings.Contains(output, "main.go") || !strings.Contains(output, ".go") || !strings.Contains(output, "go") {
			t.Errorf("expected main.go row with language in detailed format output, got: %q", output)
		}
	})

	t.Run("List with Profiles option", func(t *testing.T) {
		tempDir := t.TempDir()
		goFile := filepath.Join(tempDir, "main.go")
		exeFile := filepath.Join(tempDir, "app.exe")
		_ = os.WriteFile(goFile, []byte("package main"), 0644)
		_ = os.WriteFile(exeFile, []byte("binary"), 0755)

		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{Profiles: []string{"go"}}

		err := svc.List(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "app.exe") {
			t.Errorf("expected app.exe to be filtered by 'go' profile, got: %q", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be listed, got: %q", output)
		}
	})

	t.Run("Returns error for invalid profile", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{Profiles: []string{"invalid_profile"}}

		err := svc.List(context.Background(), outBuf, []string{"."}, opts)
		if err == nil {
			t.Errorf("expected error for invalid profile, got nil")
		}
	})

	t.Run("Returns error for invalid ignore pattern", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{IgnorePatterns: []string{"[invalid"}}

		err := svc.List(context.Background(), outBuf, []string{"."}, opts)
		if err == nil {
			t.Errorf("expected error for invalid ignore pattern, got nil")
		}
	})

	t.Run("List with IgnoreDirs option", func(t *testing.T) {
		tempDir := t.TempDir()
		binDir := filepath.Join(tempDir, "bin")
		_ = os.Mkdir(binDir, 0755)
		_ = os.WriteFile(filepath.Join(binDir, "app"), []byte("exec"), 0755)

		srcDir := filepath.Join(tempDir, "src")
		_ = os.Mkdir(srcDir, 0755)
		_ = os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)

		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{IgnoreDirs: []string{"bin"}}

		err := svc.List(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "bin") || strings.Contains(output, "app") {
			t.Errorf("expected bin directory and contents to be ignored, got output: %q", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be included, got output: %q", output)
		}
	})

	t.Run("Handles scan error for non-existent path", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.ListOptions{}

		err := svc.List(context.Background(), outBuf, []string{"/nonexistent/file/path"}, opts)
		if err != nil {
			t.Fatalf("unexpected method error for non-existent path: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "[ERROR]") {
			t.Errorf("expected output to contain [ERROR] block for non-existent path, got: %q", output)
		}
	})
}

func TestService_Dump(t *testing.T) {
	appInst := setup.NewApp(slog.LevelDebug)
	svc := app.NewService(appInst)

	t.Run("Standard dump execution in Markdown format", func(t *testing.T) {
		tempDir := t.TempDir()
		file1 := filepath.Join(tempDir, "doc.md")
		_ = os.WriteFile(file1, []byte("content"), 0644)

		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format:     app.FormatMarkdown,
			OutputPath: "stdout",
		}

		err := svc.Dump(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "# Codebase Context") || !strings.Contains(output, "doc.md") {
			t.Errorf("expected output to reflect markdown dump, got: %q", output)
		}
	})

	t.Run("Dump execution in XML format to output file", func(t *testing.T) {
		tempDir := t.TempDir()
		file1 := filepath.Join(tempDir, "data.xml")
		_ = os.WriteFile(file1, []byte("<xml/>"), 0644)

		outFile := filepath.Join(tempDir, "output.xml")
		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format:     app.FormatXML,
			OutputPath: outFile,
		}

		err := svc.Dump(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "Dumping 1 root path(s) in xml format to") {
			t.Errorf("expected info output to stdout when outputting to file, got: %q", output)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected output file to exist and be readable: %v", err)
		}
		if !strings.Contains(string(content), `<codebase_context version="1.0">`) {
			t.Errorf("expected generated XML file content, got: %s", string(content))
		}
	})

	t.Run("Returns error for invalid OutputFormat", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format: app.OutputFormat("invalid_format"),
		}

		err := svc.Dump(context.Background(), outBuf, []string{"."}, opts)
		if err == nil {
			t.Errorf("expected error for invalid OutputFormat, got nil")
		}
	})

	t.Run("Returns error for invalid profile in dump", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format:   app.FormatMarkdown,
			Profiles: []string{"invalid_profile"},
		}

		err := svc.Dump(context.Background(), outBuf, []string{"."}, opts)
		if err == nil {
			t.Errorf("expected error for invalid profile in dump, got nil")
		}
	})

	t.Run("Returns error for invalid ignore pattern in dump", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format:         app.FormatMarkdown,
			IgnorePatterns: []string{"[invalid"},
		}

		err := svc.Dump(context.Background(), outBuf, []string{"."}, opts)
		if err == nil {
			t.Errorf("expected error for invalid ignore pattern in dump, got nil")
		}
	})

	t.Run("Dump with IgnoreDirs option", func(t *testing.T) {
		tempDir := t.TempDir()
		covDir := filepath.Join(tempDir, "coverage")
		_ = os.Mkdir(covDir, 0755)
		_ = os.WriteFile(filepath.Join(covDir, "coverage.out"), []byte("data"), 0644)
		_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format:     app.FormatMarkdown,
			IgnoreDirs: []string{"coverage"},
		}

		err := svc.Dump(context.Background(), outBuf, []string{tempDir}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if strings.Contains(output, "coverage.out") {
			t.Errorf("expected coverage directory and contents to be excluded from dump, got: %q", output)
		}
		if !strings.Contains(output, "main.go") {
			t.Errorf("expected main.go to be present in dump, got: %q", output)
		}
	})

	t.Run("Handles scan error for non-existent path in dump", func(t *testing.T) {
		outBuf := new(bytes.Buffer)
		opts := app.DumpOptions{
			Format: app.FormatMarkdown,
		}

		err := svc.Dump(context.Background(), outBuf, []string{"/nonexistent/file/path"}, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := outBuf.String()
		if !strings.Contains(output, "# Codebase Context") || !strings.Contains(output, "Total Files**: 0") {
			t.Errorf("expected output to contain valid empty dump structure, got: %q", output)
		}
	})
}

func TestOutputFormat_String(t *testing.T) {
	if app.FormatMarkdown.String() != "markdown" {
		t.Errorf("expected 'markdown', got %q", app.FormatMarkdown.String())
	}
	if app.FormatXML.String() != "xml" {
		t.Errorf("expected 'xml', got %q", app.FormatXML.String())
	}
}
