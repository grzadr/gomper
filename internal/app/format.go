package app

import (
	"fmt"
	"strings"
)

// OutputFormat defines supported target formats for dumping directory structure.
type OutputFormat string

const (
	FormatMarkdown OutputFormat = "markdown"
	FormatXML      OutputFormat = "xml"
)

func (f OutputFormat) String() string {
	return string(f)
}

func (f OutputFormat) IsValid() bool {
	switch strings.ToLower(string(f)) {
	case string(FormatMarkdown), string(FormatXML):
		return true
	default:
		return false
	}
}

// ParseOutputFormat converts a string to an OutputFormat or returns an error.
func ParseOutputFormat(s string) (OutputFormat, error) {
	fmtVal := OutputFormat(strings.ToLower(strings.TrimSpace(s)))
	if fmtVal.IsValid() {
		return fmtVal, nil
	}
	return "", fmt.Errorf("invalid format %q: must be 'markdown' or 'xml'", s)
}

// Set satisfies the pflag.Value interface for CLI flag binding.
func (f *OutputFormat) Set(s string) error {
	val, err := ParseOutputFormat(s)
	if err != nil {
		return err
	}
	*f = val
	return nil
}

// Type satisfies the pflag.Value interface for CLI flag help output.
func (f OutputFormat) Type() string {
	return "format"
}

// UnmarshalText satisfies encoding.TextUnmarshaler for Viper/mapstructure decoding.
func (f *OutputFormat) UnmarshalText(text []byte) error {
	return f.Set(string(text))
}
