package app

import (
	"bytes"
	"fmt"
	"path/filepath"
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

// Formatter defines the contract for formatting scanned file entries.
type Formatter interface {
	Format(entries []scanner.Entry) (string, error)
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

// Format formats the entries in standard or long format.
func (f *StandardFormatter) Format(entries []scanner.Entry) (string, error) {
	var sb strings.Builder
	for _, entry := range entries {
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
			fmt.Fprintf(&sb, "FILE  %10d B  %s  %s\n", size, mode, displayPath)
		} else {
			fmt.Fprintln(&sb, displayPath)
		}
	}
	return sb.String(), nil
}

// RequiresMetrics indicates that StandardFormatter does not require line/token metrics.
func (f *StandardFormatter) RequiresMetrics() bool {
	return false
}

// DetailedFormatter formats file entries as an aligned table with file, extension, size, lines, and tokens.
type DetailedFormatter struct{}

// NewDetailedFormatter creates a DetailedFormatter instance.
func NewDetailedFormatter() *DetailedFormatter {
	return &DetailedFormatter{}
}

// Hook for tabwriter flushing to allow unit test error simulation.
var tabwriterFlushHook = func(w *tabwriter.Writer) error {
	return w.Flush()
}

// Format formats the entries into an aligned table using text/tabwriter.
func (f *DetailedFormatter) Format(entries []scanner.Entry) (string, error) {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(w, "FILE\tEXTENSION\tSIZE\tLINES\tTOKENS")
	for _, entry := range entries {
		displayPath := entry.RelPath
		if displayPath == "." || displayPath == "" {
			displayPath = entry.Path
		}
		ext := entry.Extension
		if ext == "" && entry.Path != "" {
			ext = filepath.Ext(entry.Path)
		}
		size := entry.Size
		if size == 0 && entry.Info != nil {
			size = entry.Info.Size()
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\n",
			displayPath,
			ext,
			size,
			entry.Lines,
			entry.Tokens,
		)
	}
	if err := tabwriterFlushHook(w); err != nil {
		return "", fmt.Errorf("failed to flush tabwriter: %w", err)
	}
	return buf.String(), nil
}

// RequiresMetrics indicates that DetailedFormatter requires line and token calculation.
func (f *DetailedFormatter) RequiresMetrics() bool {
	return true
}

// NewListFormatter returns the appropriate Formatter for the given format string and long flag.
func NewListFormatter(format string, long bool) (Formatter, error) {
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
