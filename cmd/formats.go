package cmd

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io"

	"github.com/grzadr/gomper/internal/scanner"
	"github.com/spf13/cobra"
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

	cmd := new(cobra.Command{
		Use:   "formats",
		Short: "List supported file formats and extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			exts, specials, err := listFormatsFunc()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			if jsonOutput {
				formatsData := FormatsOutput{
					SupportedFormats: exts,
					SpecialFilenames: specials,
				}

				if err := json.MarshalWrite(out, formatsData, jsontext.WithIndent("  "), json.Deterministic(true)); err != nil {
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

	return cmd
}
