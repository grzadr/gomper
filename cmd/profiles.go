package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/grzadr/gomper/internal/scanner"
)

// Hook for listing profiles, allowing error injection in unit tests.
var listProfilesFunc = scanner.ListProfiles

// NewProfilesCommand creates the profiles subcommand to list available embedded ignore profiles.
func NewProfilesCommand() *cobra.Command {
	return new(cobra.Command{
		Use:   "profiles",
		Short: "List available embedded gitignore profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := listProfilesFunc()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, "Available ignore profiles:")
			for _, prof := range profiles {
				_, _ = fmt.Fprintf(out, "  - %s\n", prof)
			}
			return nil
		},
	})
}
