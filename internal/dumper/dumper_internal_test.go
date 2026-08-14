package dumper

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/scanner"
)

type internalDummyFileInfo struct {
	name string
}

func (d internalDummyFileInfo) Name() string       { return d.name }
func (d internalDummyFileInfo) Size() int64        { return 100 }
func (d internalDummyFileInfo) Mode() os.FileMode  { return 0644 }
func (d internalDummyFileInfo) ModTime() time.Time { return time.Now() }
func (d internalDummyFileInfo) IsDir() bool        { return false }
func (d internalDummyFileInfo) Sys() any           { return nil }

func TestDumper_InternalHooks(t *testing.T) {
	t.Run("Logs warning when openFileHook returns error in XML and Markdown", func(t *testing.T) {
		tempDir := t.TempDir()
		f1 := filepath.Join(tempDir, "f1.go")
		_ = os.WriteFile(f1, []byte("package main"), 0644)

		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, nil))
		d := NewXMLDumper(logger)

		entries := []scanner.Entry{
			{Path: f1, RelPath: "f1.go", IsDir: false, Info: internalDummyFileInfo{"f1.go"}},
		}

		origOpen := openFileHook
		openFileHook = func(name string) (*os.File, error) {
			return nil, errors.New("simulated open stream error")
		}
		defer func() { openFileHook = origOpen }()

		var xmlBuf bytes.Buffer
		_ = d.GenerateXML(context.Background(), entries, "", &xmlBuf)
		if !strings.Contains(logBuf.String(), "unable to stream file content for dump") {
			t.Errorf("expected XML dump stream error warning in log, got: %s", logBuf.String())
		}

		logBuf.Reset()
		var mdBuf bytes.Buffer
		_ = d.GenerateMarkdown(context.Background(), entries, "", &mdBuf)
		if !strings.Contains(logBuf.String(), "unable to stream file content for dump") {
			t.Errorf("expected Markdown dump stream error warning in log, got: %s", logBuf.String())
		}
	})

	t.Run("Aborts file loop on context cancellation in XML and Markdown", func(t *testing.T) {
		tempDir := t.TempDir()
		f1 := filepath.Join(tempDir, "f1.go")
		f2 := filepath.Join(tempDir, "f2.go")
		_ = os.WriteFile(f1, []byte("package main"), 0644)
		_ = os.WriteFile(f2, []byte("package main"), 0644)

		entries := []scanner.Entry{
			{Path: f1, RelPath: "f1.go", IsDir: false, Info: internalDummyFileInfo{"f1.go"}},
			{Path: f2, RelPath: "f2.go", IsDir: false, Info: internalDummyFileInfo{"f2.go"}},
		}

		origOpen := openFileHook
		defer func() { openFileHook = origOpen }()

		// XML test
		ctx, cancel := context.WithCancel(context.Background())
		d := NewXMLDumper(nil)

		openFileHook = func(name string) (*os.File, error) {
			cancel() // Cancel context during first file opening
			return origOpen(name)
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(ctx, entries, "", &xmlBuf)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled from GenerateXML, got: %v", err)
		}

		// Markdown test
		ctx2, cancel2 := context.WithCancel(context.Background())
		openFileHook = func(name string) (*os.File, error) {
			cancel2() // Cancel context during first file opening
			return origOpen(name)
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(ctx2, entries, "", &mdBuf)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled from GenerateMarkdown, got: %v", err)
		}
	})
}
