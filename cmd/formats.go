package cmd

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"

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

					resolved, resolveErr := resolveJSONPointer(formatsData, ptr)
					if resolveErr != nil {
						return resolveErr
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

func resolveJSONPointer(data FormatsOutput, ptr jsontext.Pointer) (any, error) {
	var current any = data
	for tok := range ptr.Tokens() {
		switch v := current.(type) {
		case FormatsOutput:
			switch tok {
			case "supported_formats":
				current = v.SupportedFormats
			case "special_filenames":
				current = v.SpecialFilenames
			default:
				return nil, fmt.Errorf("json pointer token %q not found in root", tok)
			}
		case []scanner.FormatEntry:
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("invalid or out-of-range index %q for supported_formats", tok)
			}
			current = v[idx]
		case []scanner.SpecialFileEntry:
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("invalid or out-of-range index %q for special_filenames", tok)
			}
			current = v[idx]
		case scanner.FormatEntry:
			switch tok {
			case "extension":
				current = v.Extension
			case "language":
				current = v.Language
			default:
				return nil, fmt.Errorf("unknown field %q for FormatEntry", tok)
			}
		case scanner.SpecialFileEntry:
			switch tok {
			case "filename":
				current = v.Filename
			case "language":
				current = v.Language
			default:
				return nil, fmt.Errorf("unknown field %q for SpecialFileEntry", tok)
			}
		default:
			return nil, fmt.Errorf("cannot navigate into value of type %T with token %q", current, tok)
		}
	}
	return current, nil
}
