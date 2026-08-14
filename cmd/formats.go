package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/grzadr/gomper/internal/scanner"
)

// Hook for listing formats, allowing error injection in unit tests.
var listFormatsFunc = scanner.ListFormats

// NewFormatsCommand creates the formats subcommand to list supported file formats and extensions.
func NewFormatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "formats",
		Short: "List supported file formats and extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			exts, specials, err := listFormatsFunc()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
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
	}
}
