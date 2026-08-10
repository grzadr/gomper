package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/grzadr/gomper/internal/app"
	"github.com/grzadr/gomper/internal/config"
)

// NewListCommand creates and initializes the list subcommand.
func NewListCommand(cfg *config.Config, service app.Dumper) *cobra.Command {
	var filesOnly bool
	var longFormat bool
	var localIgnore []string
	var localProfiles []string

	cmd := &cobra.Command{
		Use:   "list [path...]",
		Short: "List files in the specified directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPaths, err := resolvePaths(args, cfg.Paths)
			if err != nil {
				return err
			}

			allIgnore := append([]string{}, cfg.Ignore...)
			allIgnore = append(allIgnore, localIgnore...)

			allProfiles := append([]string{}, cfg.GetEffectiveProfiles()...)
			allProfiles = append(allProfiles, localProfiles...)

			opts := app.ListOptions{
				FilesOnly:      filesOnly,
				LongFormat:     longFormat,
				IgnorePatterns: allIgnore,
				IgnoreDotfiles: cfg.IgnoreDotfiles,
				Profiles:       allProfiles,
			}
			return service.List(cmd.Context(), cmd.OutOrStdout(), targetPaths, opts)
		},
	}

	cmd.Flags().BoolVarP(&filesOnly, "files-only", "f", false, "list regular files only, skipping directories")
	cmd.Flags().BoolVarP(&longFormat, "long", "l", false, "display detailed file attributes (size, permissions)")
	cmd.Flags().StringSliceVarP(&localIgnore, "ignore", "i", nil, "regex patterns to ignore matching files or directories")
	cmd.Flags().StringSliceVarP(&localProfiles, "profile", "p", nil, "ignore profiles (e.g. 'go', 'node', 'python')")

	return cmd
}

func resolvePaths(cliArgs []string, cfgPaths []string) ([]string, error) {
	if len(cliArgs) > 0 {
		return cliArgs, nil
	}
	if len(cfgPaths) > 0 {
		return cfgPaths, nil
	}
	return nil, fmt.Errorf("requires at least 1 path (via positional argument or config file 'paths')")
}
