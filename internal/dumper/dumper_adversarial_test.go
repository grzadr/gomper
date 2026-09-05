package dumper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/grzadr/gomper/internal/dumper"
	"github.com/grzadr/gomper/internal/scanner"
)

func TestAdversarial_DumperDomain(t *testing.T) {
	d := dumper.NewDumper(nil)

	entryFor := func(namespace, fileName, code string) scanner.Entry {
		path := fmt.Sprintf("%s/%s", namespace, fileName)
		return scanner.Entry{
			Path:    path,
			RelPath: path,
			IsDir:   false,
			Info:    dummyFileInfo{name: fileName, size: int64(len(code))},
			Content: io.NopCloser(strings.NewReader(code)),
		}
	}

	t.Run("Domain entries with XML entity characters and markdown code blocks", func(t *testing.T) {
		entries := []scanner.Entry{
			entryFor("core", "logic.go", "package core\n\n// Check if x < 10 && y > 20 || str == \"<foo>\"\nfunc Check() bool {\n\treturn true\n}\n"),
			entryFor("docs", "guide.md", "# Guide\n\n```xml\n<config key=\"value\">&amp;</config>\n```\n"),
		}

		seq := entriesToSeq(entries)

		// XML generation test
		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), seq, "Review core modules", &xmlBuf)
		if err != nil {
			t.Fatalf("GenerateXML failed: %v", err)
		}
		xmlOutput := xmlBuf.String()

		if !strings.Contains(xmlOutput, `<file path="core/logic.go" language="go"`) {
			t.Errorf("expected file path 'core/logic.go' in XML output")
		}
		if !strings.Contains(xmlOutput, `&lt;foo&gt;`) {
			t.Errorf("expected XML escaped entity in XML output")
		}
		if !strings.Contains(xmlOutput, `<file path="docs/guide.md" language="markdown"`) {
			t.Errorf("expected file path 'docs/guide.md' in XML output")
		}

		// Markdown generation test
		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), seq, "Review core modules", &mdBuf)
		if err != nil {
			t.Fatalf("GenerateMarkdown failed: %v", err)
		}
		mdOutput := mdBuf.String()

		if !strings.Contains(mdOutput, "### File: `core/logic.go`") {
			t.Errorf("expected file section 'core/logic.go' in Markdown output")
		}
		if !strings.Contains(mdOutput, "### File: `docs/guide.md`") {
			t.Errorf("expected file section 'docs/guide.md' in Markdown output")
		}
	})

	t.Run("Large sequence (1,000 domain items)", func(t *testing.T) {
		const total = 1000
		seq := func(yield func(scanner.Entry, error) bool) {
			for i := range total {
				code := fmt.Sprintf("package pkg\n\n// Index %d\nfunc F%d() {}\n", i, i)
				if !yield(entryFor("pkg", fmt.Sprintf("file_%04d.go", i), code), nil) {
					return
				}
			}
		}

		var xmlBuf bytes.Buffer
		start := time.Now()
		err := d.GenerateXML(context.Background(), seq, "Large dump test", &xmlBuf)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected GenerateXML error on large sequence: %v", err)
		}

		if elapsed > 5*time.Second {
			t.Errorf("GenerateXML took too long (%v) for 1000 items", elapsed)
		}

		xmlStr := xmlBuf.String()
		if !strings.Contains(xmlStr, fmt.Sprintf("<total_files>%d</total_files>", total)) {
			t.Errorf("expected %d total files in summary", total)
		}
	})

	t.Run("Sequence with item errors skipped gracefully and logged", func(t *testing.T) {
		injectedErr := errors.New("upstream generator failure")
		seqWithErrs := func(yield func(scanner.Entry, error) bool) {
			_ = yield(entryFor("ok", "ok1.go", "package ok1\n"), nil)
			_ = yield(scanner.Entry{Path: "bad/bad.go", RelPath: "bad/bad.go"}, injectedErr)
			_ = yield(entryFor("ok", "ok2.go", "package ok2\n"), nil)
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), seqWithErrs, "", &xmlBuf)
		if err != nil {
			t.Fatalf("unexpected error when sequence yields error items: %v", err)
		}

		out := xmlBuf.String()
		if !strings.Contains(out, "<total_files>2</total_files>") {
			t.Errorf("expected 2 files dumped (bad item skipped), got:\n%s", out)
		}
	})
}
