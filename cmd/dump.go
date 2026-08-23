package cmd

import (
	"github.com/spf13/cobra"
	"github.com/grzadr/gomper/internal/app"
	"github.com/grzadr/gomper/internal/config"
)

// NewDumpCommand creates and initializes the dump subcommand.
func NewDumpCommand(cfg *config.Config, service app.Dumper) *cobra.Command {
	var localIgnore []string
	var localIgnoreDir []string
	var localNameFilter []string
	var localProfiles []string

	cmd := new(cobra.Command{
		Use:   "dump [path...]",
		Short: "Dump directory structure into a single file",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPaths, err := resolvePaths(args, cfg.Paths)
			if err != nil {
				return err
			}

			allIgnore := append([]string{}, cfg.Ignore...)
			allIgnore = append(allIgnore, localIgnore...)

			allIgnoreDir := append([]string{}, cfg.GetEffectiveIgnoreDirs()...)
			allIgnoreDir = append(allIgnoreDir, localIgnoreDir...)

			allNameFilters := append([]string{}, cfg.GetEffectiveNameFilters()...)
			allNameFilters = append(allNameFilters, localNameFilter...)

			allProfiles := append([]string{}, cfg.GetEffectiveProfiles()...)
			allProfiles = append(allProfiles, localProfiles...)

			opts := app.DumpOptions{
				Format:         cfg.Format,
				OutputPath:     cfg.Output,
				IgnorePatterns: allIgnore,
				IgnoreDirs:     allIgnoreDir,
				IgnoreDotfiles: cfg.IgnoreDotfiles,
				NameFilters:    allNameFilters,
				Profiles:       allProfiles,
				Instructions:   cfg.Instructions,
			}
			return service.Dump(cmd.Context(), cmd.OutOrStdout(), targetPaths, opts)
		},
	})

	cmd.Flags().VarP(&cfg.Format, "format", "f", "output format ('markdown' or 'xml')")
	cmd.Flags().StringVarP(&cfg.Output, "output", "o", "", "output file path (defaults to stdout)")
	cmd.Flags().StringVarP(&cfg.Instructions, "instructions", "u", "", "user instructions to include in dump header")
	cmd.Flags().StringSliceVarP(&localIgnore, "ignore", "i", nil, "regex patterns to ignore matching files or directories")
	cmd.Flags().StringSliceVarP(&localIgnoreDir, "ignore-dir", "D", nil, "directory patterns to ignore following gitignore convention (e.g. 'bin', 'coverage')")
	cmd.Flags().StringSliceVarP(&localNameFilter, "name", "n", nil, "regex patterns to match file names")
	cmd.Flags().StringSliceVarP(&localProfiles, "profile", "p", nil, "ignore profiles (e.g. 'go', 'node', 'python')")

	return cmd

}
