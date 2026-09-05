package dumper

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"os"
	"strings"
	"testing"

	"github.com/grzadr/gomper/internal/scanner"
)

func entriesToSeq(entries []scanner.Entry) iter.Seq2[scanner.Entry, error] {
	return func(yield func(scanner.Entry, error) bool) {
		for _, e := range entries {
			if !yield(e, nil) {
				return
			}
		}
	}
}

type failWriter struct {
	failOnWrite bool
}

func (f *failWriter) Write(p []byte) (n int, err error) {
	if f.failOnWrite {
		return 0, errors.New("simulated target write error")
	}
	return len(p), nil
}

type failAfterFirstWriter struct {
	writes int
}

func (f *failAfterFirstWriter) Write(p []byte) (n int, err error) {
	f.writes++
	if f.writes > 1 {
		return 0, errors.New("simulated copy write error")
	}
	return len(p), nil
}

func TestDumper_InternalHooks(t *testing.T) {
	t.Run("Returns error when createTempHook fails in XML and Markdown", func(t *testing.T) {
		origHook := createTempHook
		createTempHook = func(dir, pattern string) (*os.File, error) {
			return nil, errors.New("simulated createTemp error")
		}
		defer func() { createTempHook = origHook }()

		d := NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "test.go", RelPath: "test.go", IsDir: false, Content: io.NopCloser(strings.NewReader("content"))},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", &xmlBuf)
		if err == nil || !strings.Contains(err.Error(), "failed to create staging file") {
			t.Errorf("expected staging file error from GenerateXML, got: %v", err)
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entriesToSeq(entries), "", &mdBuf)
		if err == nil || !strings.Contains(err.Error(), "failed to create staging file") {
			t.Errorf("expected staging file error from GenerateMarkdown, got: %v", err)
		}
	})

	t.Run("Returns error when formatContentHook fails in XML and Markdown", func(t *testing.T) {
		origHook := formatContentHook
		formatContentHook = func(r io.Reader, w io.Writer, escapeXML bool) error {
			return errors.New("simulated format error")
		}
		defer func() { formatContentHook = origHook }()

		d := NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "test.go", RelPath: "test.go", IsDir: false, Content: io.NopCloser(strings.NewReader("content"))},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", &xmlBuf)
		if err == nil || !strings.Contains(err.Error(), "simulated format error") {
			t.Errorf("expected format error from GenerateXML, got: %v", err)
		}

		var mdBuf bytes.Buffer
		err = d.GenerateMarkdown(context.Background(), entriesToSeq(entries), "", &mdBuf)
		if err == nil || !strings.Contains(err.Error(), "simulated format error") {
			t.Errorf("expected format error from GenerateMarkdown, got: %v", err)
		}
	})

	t.Run("Returns error when seekHook fails", func(t *testing.T) {
		origHook := seekHook
		seekHook = func(f *os.File, offset int64, whence int) (int64, error) {
			return 0, errors.New("simulated seek error")
		}
		defer func() { seekHook = origHook }()

		d := NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "test.go", RelPath: "test.go", IsDir: false, Content: io.NopCloser(strings.NewReader("content"))},
		}

		var xmlBuf bytes.Buffer
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", &xmlBuf)
		if err == nil || !strings.Contains(err.Error(), "failed to seek staging file") {
			t.Errorf("expected seek error from GenerateXML, got: %v", err)
		}
	})

	t.Run("Returns error when targetWriter fails during XML dump", func(t *testing.T) {
		d := NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "test.go", RelPath: "test.go", IsDir: false, Content: io.NopCloser(strings.NewReader("content"))},
		}

		failW := new(failWriter{failOnWrite: true})
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", failW)
		if err == nil {
			t.Errorf("expected write error when targetWriter fails, got nil")
		}
	})

	t.Run("Returns error when io.Copy fails copying staging file", func(t *testing.T) {
		d := NewXMLDumper(nil)
		entries := []scanner.Entry{
			{Path: "test.go", RelPath: "test.go", IsDir: false, Content: io.NopCloser(strings.NewReader("content"))},
		}

		failAfterHeaderW := new(failAfterFirstWriter{})
		err := d.GenerateXML(context.Background(), entriesToSeq(entries), "", failAfterHeaderW)
		if err == nil || !strings.Contains(err.Error(), "failed to copy staging file content") {
			t.Errorf("expected copy error from GenerateXML, got: %v", err)
		}
	})
}
