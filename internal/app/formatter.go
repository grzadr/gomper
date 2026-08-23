package app

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"strings"

	"github.com/grzadr/gomper/internal/scanner"
)

// ListFormat defines supported output formatting modes for the list command.
type ListFormat string

const (
	ListFormatStandard ListFormat = "standard"
	ListFormatDetailed ListFormat = "detailed"
	ListFormatJSON     ListFormat = "json"
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
	return new(StandardFormatter{Long: long})
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

// DetailedFormatter formats file entries in a fixed-width, path-trailing layout with O(1) memory overhead.
type DetailedFormatter struct{}

// NewDetailedFormatter creates a DetailedFormatter instance.
func NewDetailedFormatter() *DetailedFormatter {
	return new(DetailedFormatter{})
}

// WriteHeader writes fixed-width column headers directly to w.
func (f *DetailedFormatter) WriteHeader(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%-12s %7s %8s  %-10s  %-10s  %s\n",
		"SIZE", "LINES", "TOKENS", "EXTENSION", "LANGUAGE", "FILE")
	return err
}

// FormatEntry writes a single formatted row directly to w without intermediate buffering.
func (f *DetailedFormatter) FormatEntry(w io.Writer, entry scanner.Entry) error {
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

	formattedSize := fmt.Sprintf("%d B", size)

	_, err := fmt.Fprintf(w, "%-12s %7d %8d  %-10s  %-10s  %s\n",
		formattedSize,
		entry.Lines,
		entry.Tokens,
		ext,
		lang,
		displayPath,
	)
	return err
}

// Flush flushes any buffered output (no-op for streaming DetailedFormatter).
func (f *DetailedFormatter) Flush(w io.Writer) error {
	return nil
}

// RequiresMetrics indicates that DetailedFormatter requires line and token calculation.
func (f *DetailedFormatter) RequiresMetrics() bool {
	return true
}

// JSONFormatter formats file entries as a streaming JSON array with zero intermediate buffering.
type JSONFormatter struct {
	first bool
	count int
}

// NewJSONFormatter creates a JSONFormatter instance.
func NewJSONFormatter() *JSONFormatter {
	return new(JSONFormatter{first: true})
}

// WriteHeader writes the opening bracket for the JSON array.
func (f *JSONFormatter) WriteHeader(w io.Writer) error {
	f.first = true
	f.count = 0
	_, err := io.WriteString(w, "[\n")
	return err
}

// FormatEntry writes a single entry directly to w formatted as JSON.
func (f *JSONFormatter) FormatEntry(w io.Writer, entry scanner.Entry) error {
	if !f.first {
		if _, err := io.WriteString(w, ",\n"); err != nil {
			return err
		}
	}
	f.first = false
	f.count++

	displayPath := entry.RelPath
	if displayPath == "." || displayPath == "" {
		displayPath = entry.Path
	}
	entry.RelPath = displayPath

	if entry.Size == 0 && entry.Info != nil {
		entry.Size = entry.Info.Size()
	}

	return json.MarshalWrite(w, entry, jsontext.WithIndent("  "), json.Deterministic(true))
}

// Flush closes the JSON array.
func (f *JSONFormatter) Flush(w io.Writer) error {
	if f.count > 0 {
		_, err := io.WriteString(w, "\n]\n")
		return err
	}
	_, err := io.WriteString(w, "]\n")
	return err
}

// RequiresMetrics indicates that JSONFormatter requires line and token metrics.
func (f *JSONFormatter) RequiresMetrics() bool {
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
	case string(ListFormatJSON):
		return NewJSONFormatter(), nil
	default:
		return nil, fmt.Errorf("unsupported list format %q: must be 'standard', 'detailed', or 'json'", format)
	}
}
