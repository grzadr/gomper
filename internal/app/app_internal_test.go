package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/scanner"
)

type appDummyFileInfo struct {
	name string
}

func (d appDummyFileInfo) Name() string       { return d.name }
func (d appDummyFileInfo) Size() int64        { return 42 }
func (d appDummyFileInfo) Mode() os.FileMode  { return 0644 }
func (d appDummyFileInfo) ModTime() time.Time { return time.Now() }
func (d appDummyFileInfo) IsDir() bool        { return false }
func (d appDummyFileInfo) Sys() any           { return nil }

func TestService_List_DisplayPathFallback(t *testing.T) {
	origWalk := walkPathsFunc
	defer func() { walkPathsFunc = origWalk }()

	walkPathsFunc = func(ctx context.Context, paths []string, filter *scanner.Filter, opts ...scanner.ScanOption) iter.Seq2[scanner.Entry, error] {
		return func(yield func(scanner.Entry, error) bool) {
			yield(scanner.Entry{
				Path:    "/root/empty_rel.go",
				RelPath: "",
				IsDir:   false,
				Info:    appDummyFileInfo{"empty_rel.go"},
			}, nil)
			yield(scanner.Entry{
				Path:    "/root/dot_rel.go",
				RelPath: ".",
				IsDir:   false,
				Info:    appDummyFileInfo{"dot_rel.go"},
			}, nil)
		}
	}

	svc := NewService(nil)
	outBuf := new(bytes.Buffer)

	// Short format
	err := svc.List(context.Background(), outBuf, []string{"/root"}, ListOptions{LongFormat: false})
	if err != nil {
		t.Fatalf("unexpected error in List: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "/root/empty_rel.go") || !strings.Contains(output, "/root/dot_rel.go") {
		t.Errorf("expected full Path fallback when RelPath is empty or dot, got output:\n%s", output)
	}

	// Long format
	outBuf.Reset()
	err = svc.List(context.Background(), outBuf, []string{"/root"}, ListOptions{LongFormat: true})
	if err != nil {
		t.Fatalf("unexpected error in List long format: %v", err)
	}

	outputLong := outBuf.String()
	if !strings.Contains(outputLong, "/root/empty_rel.go") || !strings.Contains(outputLong, "42 B") {
		t.Errorf("expected long format line with full Path fallback, got output:\n%s", outputLong)
	}
}

type failingFormatter struct {
	writeHeaderErr error
	formatEntryErr error
	flushErr       error
}

func (f *failingFormatter) WriteHeader(w io.Writer) error {
	return f.writeHeaderErr
}

func (f *failingFormatter) FormatEntry(w io.Writer, entry scanner.Entry) error {
	return f.formatEntryErr
}

func (f *failingFormatter) Flush(w io.Writer) error {
	return f.flushErr
}

func (f *failingFormatter) RequiresMetrics() bool {
	return false
}

func TestService_List_FormatterErrors(t *testing.T) {
	svc := NewService(nil)

	t.Run("WriteHeader error", func(t *testing.T) {
		expectedErr := errors.New("write header failure")
		opts := ListOptions{
			Formatter: &failingFormatter{writeHeaderErr: expectedErr},
		}

		err := svc.List(context.Background(), new(bytes.Buffer), []string{"."}, opts)
		if err == nil || !errors.Is(err, expectedErr) {
			t.Fatalf("expected header error %v, got %v", expectedErr, err)
		}
	})

	t.Run("FormatEntry error", func(t *testing.T) {
		expectedErr := errors.New("format entry failure")
		opts := ListOptions{
			Formatter: &failingFormatter{formatEntryErr: expectedErr},
		}

		err := svc.List(context.Background(), new(bytes.Buffer), []string{"."}, opts)
		if err == nil || !errors.Is(err, expectedErr) {
			t.Fatalf("expected entry error %v, got %v", expectedErr, err)
		}
	})

	t.Run("Flush error", func(t *testing.T) {
		expectedErr := errors.New("flush failure")
		opts := ListOptions{
			Formatter: &failingFormatter{flushErr: expectedErr},
		}

		err := svc.List(context.Background(), new(bytes.Buffer), []string{"."}, opts)
		if err == nil || !errors.Is(err, expectedErr) {
			t.Fatalf("expected flush error %v, got %v", expectedErr, err)
		}
	})
}
