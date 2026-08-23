package cmd

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/grzadr/gomper/internal/scanner"
)

// Hook for listing formats, allowing error injection in unit tests.
var listFormatsFunc = scanner.ListFormats

// FormatsOutput encapsulates supported format and special filename entries for JSON serialization.
type FormatsOutput struct {
	SupportedFormats []scanner.FormatEntry      `json:"supported_formats,omitzero"`
	SpecialFilenames []scanner.SpecialFileEntry `json:"special_filenames,omitzero"`
}

// NewFormatsCommand creates the formats subcommand to list supported file formats and extensions.
func NewFormatsCommand() *cobra.Command {
	var jsonOutput bool
	var query string

	cmd := new(cobra.Command{
		Use:   "formats",
		Short: "List supported file formats and extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			exts, specials, err := listFormatsFunc()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			if jsonOutput || query != "" {
				formatsData := FormatsOutput{
					SupportedFormats: exts,
					SpecialFilenames: specials,
				}

				var dataToEncode any = formatsData

				if query != "" {
					ptr := jsontext.Pointer(query)
					if !ptr.IsValid() {
						return fmt.Errorf("invalid json pointer query %q", query)
					}

					resolved, err := resolveFormatsPointer(formatsData, query)
					if err != nil {
						return err
					}
					dataToEncode = resolved
				}

				if err := json.MarshalWrite(out, dataToEncode, jsontext.WithIndent("  "), json.Deterministic(true)); err != nil {
					return err
				}
				_, _ = io.WriteString(out, "\n")
				return nil
			}

			_, _ = fmt.Fprintln(out, "Supported file formats:")
			for _, ext := range exts {
				_, _ = fmt.Fprintf(out, "  - %s (%s)\n", ext.Extension, ext.Language)
			}
			if len(specials) > 0 {
				_, _ = fmt.Fprintln(out, "\nSpecial filenames:")
				for _, spec := range specials {
					_, _ = fmt.Fprintf(out, "  - %s (%s)\n", spec.Filename, spec.Language)
				}
			}
			return nil
		},
	})

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output format list as JSON")
	cmd.Flags().StringVar(&query, "query", "", "JSON pointer query (RFC 6901) to filter JSON output")

	return cmd
}

// resolveFormatsPointer walks a FormatsOutput value using an RFC 6901 JSON pointer string,
// returning the resolved value or a descriptive error.
func resolveFormatsPointer(data FormatsOutput, ptr string) (any, error) {
	if ptr == "" || ptr == "/" {
		return data, nil
	}

	// RFC 6901: split on "/" after stripping the leading "/"
	tokens := strings.Split(strings.TrimPrefix(ptr, "/"), "/")

	// Unescape RFC 6901 tokens (~1 → /, ~0 → ~)
	unescape := func(s string) string {
		s = strings.ReplaceAll(s, "~1", "/")
		s = strings.ReplaceAll(s, "~0", "~")
		return s
	}

	root := unescape(tokens[0])
	rest := tokens[1:]

	// Dispatch on the root token
	switch root {
	case "supported_formats":
		return resolveSlicePointer("supported_formats", data.SupportedFormats, rest,
			func(e scanner.FormatEntry, field string) (any, error) {
				switch field {
				case "extension":
					return e.Extension, nil
				case "language":
					return e.Language, nil
				default:
					return nil, fmt.Errorf("unknown field %q on FormatEntry", field)
				}
			},
		)
	case "special_filenames":
		return resolveSlicePointer("special_filenames", data.SpecialFilenames, rest,
			func(e scanner.SpecialFileEntry, field string) (any, error) {
				switch field {
				case "filename":
					return e.Filename, nil
				case "language":
					return e.Language, nil
				default:
					return nil, fmt.Errorf("unknown field %q on SpecialFileEntry", field)
				}
			},
		)
	default:
		return nil, fmt.Errorf("key %q not found in root FormatsOutput", root)
	}
}

// resolveSlicePointer handles navigation into a typed slice, then optionally into a struct field.
func resolveSlicePointer[T any](sliceName string, slice []T, tokens []string, fieldResolver func(T, string) (any, error)) (any, error) {
	if len(tokens) == 0 {
		return slice, nil
	}

	idxStr := tokens[0]
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(slice) {
		return nil, fmt.Errorf("invalid or out-of-range index %q for %s (length %d)", idxStr, sliceName, len(slice))
	}

	entry := slice[idx]
	remaining := tokens[1:]

	if len(remaining) == 0 {
		return entry, nil
	}

	field := remaining[0]
	val, err := fieldResolver(entry, field)
	if err != nil {
		return nil, err
	}

	// Scalars cannot be navigated into
	if len(remaining) > 1 {
		return nil, fmt.Errorf("cannot navigate into value of type %T at %q", val, field)
	}

	return val, nil
}
