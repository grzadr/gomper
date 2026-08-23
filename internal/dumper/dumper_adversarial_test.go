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

type DomainModule struct {
	Namespace   string
	FileName    string
	Code        string
	TokenBudget int
	Tags        []string
}

func TestAdversarial_DumperGenericDomain(t *testing.T) {
	extractor := func(m DomainModule) scanner.Entry {
		return scanner.Entry{
			Path:    fmt.Sprintf("%s/%s", m.Namespace, m.FileName),
			RelPath: fmt.Sprintf("%s/%s", m.Namespace, m.FileName),
			IsDir:   false,
			Info:    dummyFileInfo{name: m.FileName, size: int64(len(m.Code))},
			Content: io.NopCloser(strings.NewReader(m.Code)),
		}
	}

	d := dumper.NewDumper(nil, extractor)

	t.Run("Custom domain type with XML entity characters and markdown code blocks", func(t *testing.T) {
		modules := []DomainModule{
			{
				Namespace:   "core",
				FileName:    "logic.go",
				Code:        "package core\n\n// Check if x < 10 && y > 20 || str == \"<foo>\"\nfunc Check() bool {\n\treturn true\n}\n",
				TokenBudget: 50,
				Tags:        []string{"core", "parser"},
			},
			{
				Namespace:   "docs",
				FileName:    "guide.md",
				Code:        "# Guide\n\n```xml\n<config key=\"value\">&amp;</config>\n```\n",
				TokenBudget: 30,
				Tags:        []string{"docs"},
			},
		}

		seq := func(yield func(DomainModule, error) bool) {
			for _, m := range modules {
				if !yield(m, nil) {
					return
				}
			}
		}

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

	t.Run("Large sequence (1,000 custom domain items)", func(t *testing.T) {
		const total = 1000
		seq := func(yield func(DomainModule, error) bool) {
			for i := range total {
				mod := DomainModule{
					Namespace: "pkg",
					FileName:  fmt.Sprintf("file_%04d.go", i),
					Code:      fmt.Sprintf("package pkg\n\n// Index %d\nfunc F%d() {}\n", i, i),
				}
				if !yield(mod, nil) {
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
		seqWithErrs := func(yield func(DomainModule, error) bool) {
			_ = yield(DomainModule{Namespace: "ok", FileName: "ok1.go", Code: "package ok1\n"}, nil)
			_ = yield(DomainModule{Namespace: "bad", FileName: "bad.go"}, injectedErr)
			_ = yield(DomainModule{Namespace: "ok", FileName: "ok2.go", Code: "package ok2\n"}, nil)
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
