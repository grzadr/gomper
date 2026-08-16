package app

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/grzadr/gomper/internal/scanner"
)

// ListFormat defines supported output formatting modes for the list command.
type ListFormat string

const (
	ListFormatStandard ListFormat = "standard"
	ListFormatDetailed ListFormat = "detailed"
)

// ListFormatter defines the streaming contract for formatting scanned file entries.
type ListFormatter interface {
	WriteHeader(w io.Writer) error
	FormatEntry(w io.Writer, entry scanner.Entry) error
	Flush(w io.Writer) error
	RequiresMetrics() bool
}

// StandardFormatter formats file entries as a simple path list or detailed attributes if Long is true.
type StandardFormatter struct {
	Long bool
}

// NewStandardFormatter creates a StandardFormatter instance.
func NewStandardFormatter(long bool) *StandardFormatter {
	return &StandardFormatter{Long: long}
}

// WriteHeader writes header if needed (no-op for StandardFormatter).
func (f *StandardFormatter) WriteHeader(w io.Writer) error {
	return nil
}

// FormatEntry writes a single entry to w.
func (f *StandardFormatter) FormatEntry(w io.Writer, entry scanner.Entry) error {
	displayPath := entry.RelPath
	if displayPath == "." || displayPath == "" {
		displayPath = entry.Path
	}

	if f != nil && f.Long {
		var mode string
		size := entry.Size
		if entry.Info != nil {
			mode = entry.Info.Mode().String()
			if size == 0 {
				size = entry.Info.Size()
			}
		}
		_, err := fmt.Fprintf(w, "FILE  %10d B  %s  %s\n", size, mode, displayPath)
		return err
	}

	_, err := fmt.Fprintln(w, displayPath)
	return err
}

// Flush flushes buffered output if any (no-op for StandardFormatter).
func (f *StandardFormatter) Flush(w io.Writer) error {
	return nil
}

// RequiresMetrics indicates that StandardFormatter does not require line/token metrics.
func (f *StandardFormatter) RequiresMetrics() bool {
	return false
}

// DetailedFormatter formats file entries as an aligned table with file, extension, language, size, lines, and tokens.
type DetailedFormatter struct {
	tw *tabwriter.Writer
}

// NewDetailedFormatter creates a DetailedFormatter instance.
func NewDetailedFormatter() *DetailedFormatter {
	return &DetailedFormatter{}
}

// Hook for tabwriter flushing to allow unit test error simulation.
var tabwriterFlushHook = func(w *tabwriter.Writer) error {
	return w.Flush()
}

// WriteHeader initializes tabwriter and writes table header columns.
func (f *DetailedFormatter) WriteHeader(w io.Writer) error {
	f.tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, err := fmt.Fprintln(f.tw, "FILE\tEXTENSION\tLANGUAGE\tSIZE\tLINES\tTOKENS")
	return err
}

// FormatEntry writes a single row to the tabwriter.
func (f *DetailedFormatter) FormatEntry(w io.Writer, entry scanner.Entry) error {
	if f.tw == nil {
		if err := f.WriteHeader(w); err != nil {
			return err
		}
	}

	displayPath := entry.RelPath
	if displayPath == "." || displayPath == "" {
		displayPath = entry.Path
	}

	ext := entry.Extension
	if ext == "" {
		ext = "-"
	}

	lang := entry.Language
	if lang == "" {
		lang = "-"
	}

	size := entry.Size
	if size == 0 && entry.Info != nil {
		size = entry.Info.Size()
	}

	_, err := fmt.Fprintf(f.tw, "%s\t%s\t%s\t%d\t%d\t%d\n",
		displayPath,
		ext,
		lang,
		size,
		entry.Lines,
		entry.Tokens,
	)
	return err
}

// Flush flushes the underlying tabwriter.Writer.
func (f *DetailedFormatter) Flush(w io.Writer) error {
	if f.tw == nil {
		if err := f.WriteHeader(w); err != nil {
			return err
		}
	}
	if err := tabwriterFlushHook(f.tw); err != nil {
		return fmt.Errorf("failed to flush tabwriter: %w", err)
	}
	return nil
}

// RequiresMetrics indicates that DetailedFormatter requires line and token calculation.
func (f *DetailedFormatter) RequiresMetrics() bool {
	return true
}

// NewListFormatter returns the appropriate ListFormatter for the given format string and long flag.
func NewListFormatter(format string, long bool) (ListFormatter, error) {
	trimmed := strings.ToLower(strings.TrimSpace(format))
	switch trimmed {
	case "", string(ListFormatStandard):
		return NewStandardFormatter(long), nil
	case string(ListFormatDetailed):
		return NewDetailedFormatter(), nil
	default:
		return nil, fmt.Errorf("unsupported list format %q: must be 'standard' or 'detailed'", format)
	}
}
