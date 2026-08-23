package dumper_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/dumper"
	"github.com/grzadr/gomper/internal/scanner"
)

func BenchmarkGenericDumper_GenerateXML(b *testing.B) {
	const numEntries = 100
	entries := make([]scanner.Entry, numEntries)
	for i := range numEntries {
		content := fmt.Sprintf("package pkg\n\nfunc Func%d() int { return %d }\n", i, i)
		entries[i] = scanner.Entry{
			Path:    fmt.Sprintf("pkg/file_%03d.go", i),
			RelPath: fmt.Sprintf("pkg/file_%03d.go", i),
			IsDir:   false,
			Info:    dummyFileInfo{name: fmt.Sprintf("file_%03d.go", i), size: int64(len(content))},
			Content: io.NopCloser(strings.NewReader(content)),
		}
	}

	d := dumper.NewXMLDumper(nil)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		seq := func(yield func(scanner.Entry, error) bool) {
			for _, e := range entries {
				// Recreate reader since it is consumed
				content := fmt.Sprintf("package pkg\n\nfunc Func() int { return 0 }\n")
				e.Content = io.NopCloser(strings.NewReader(content))
				if !yield(e, nil) {
					return
				}
			}
		}

		if err := d.GenerateXML(ctx, seq, "Benchmark", io.Discard); err != nil {
			b.Fatalf("GenerateXML failed: %v", err)
		}
	}
}
