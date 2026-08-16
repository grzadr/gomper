package app

import (
	"bytes"
	"context"
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
