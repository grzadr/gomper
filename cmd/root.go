package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/grzadr/gomper/internal/app"
	"github.com/grzadr/gomper/internal/config"
	"github.com/grzadr/gomper/internal/setup"
)

// NewRootCommand creates an isolated root command without global package variables.
func NewRootCommand(appInst *setup.App) *cobra.Command {
	if appInst == nil {
		appInst = setup.NewApp(setup.ParseLogLevel("info"))
	}

	v := viper.New()
	cfg := &config.Config{
		LogLevel: "info",
		Format:   app.FormatMarkdown,
	}
	service := app.NewService(appInst)

	rootCmd := &cobra.Command{
		Use:           "gomper",
		Short:         "Dump directory structure into Markdown or XML files",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(cmd, v, cfg); err != nil {
				return err
			}
			appInst.SetLogLevel(setup.ParseLogLevel(cfg.LogLevel))
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfg.ConfigFile, "config", "", "path to custom configuration file")
	rootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "logging level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringSliceVarP(&cfg.Ignore, "ignore", "i", nil, "regex patterns to ignore files or directories")
	rootCmd.PersistentFlags().StringSliceVarP(&cfg.IgnoreDir, "ignore-dir", "D", nil, "directory patterns to ignore following gitignore convention (e.g. 'bin', 'coverage')")
	rootCmd.PersistentFlags().BoolVarP(&cfg.IgnoreDotfiles, "ignore-dotfiles", "d", false, "ignore files and directories starting with '.'")
	rootCmd.PersistentFlags().StringSliceVarP(&cfg.Name, "name", "n", nil, "regex patterns to match file names")
	rootCmd.PersistentFlags().StringSliceVarP(&cfg.Profile, "profile", "p", nil, "ignore profiles (e.g., 'go', 'node', 'python')")

	_ = v.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = v.BindPFlag("ignore", rootCmd.PersistentFlags().Lookup("ignore"))
	_ = v.BindPFlag("ignore_dir", rootCmd.PersistentFlags().Lookup("ignore-dir"))
	_ = v.BindPFlag("ignore_dotfiles", rootCmd.PersistentFlags().Lookup("ignore-dotfiles"))
	_ = v.BindPFlag("name", rootCmd.PersistentFlags().Lookup("name"))
	_ = v.BindPFlag("name_filter", rootCmd.PersistentFlags().Lookup("name"))
	_ = v.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))


	rootCmd.AddCommand(NewListCommand(cfg, service))
	rootCmd.AddCommand(NewDumpCommand(cfg, service))
	rootCmd.AddCommand(NewProfilesCommand())
	rootCmd.AddCommand(NewFormatsCommand())

	return rootCmd
}

func initConfig(cmd *cobra.Command, v *viper.Viper, cfg *config.Config) error {
	_ = v.BindPFlags(cmd.Flags())
	_ = v.BindPFlags(cmd.PersistentFlags())

	if cfg.ConfigFile != "" {
		v.SetConfigFile(cfg.ConfigFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
		}
		v.AddConfigPath(".")
		v.SetConfigName("gomper")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("GOMPER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	v.SetDefault("log_level", "info")
	v.SetDefault("format", string(app.FormatMarkdown))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unable to decode configuration: %w", err)
	}

	return nil
}
