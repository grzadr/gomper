package config

import "github.com/grzadr/gomper/internal/app"

// Config encapsulates application configuration populated from flags, env, and config files.
type Config struct {
	ConfigFile     string           `mapstructure:"-"`
	LogLevel       string           `mapstructure:"log_level"`
	Format         app.OutputFormat `mapstructure:"format"`
	Output         string           `mapstructure:"output"`
	Ignore         []string         `mapstructure:"ignore"`
	IgnoreDir      []string         `mapstructure:"ignore_dir"`
	IgnoreDirs     []string         `mapstructure:"ignore_dirs"` // Alias for ignore_dir in YAML config
	IgnoreDotfiles bool             `mapstructure:"ignore_dotfiles"`
	Profile        []string         `mapstructure:"profile"`
	Profiles       []string         `mapstructure:"profiles"` // Alias for profiles in YAML config
	Paths          []string         `mapstructure:"paths"`
	Instructions   string           `mapstructure:"instructions"`
}

// GetEffectiveProfiles returns combined profile list from 'profile' or 'profiles' config keys.
func (c *Config) GetEffectiveProfiles() []string {
	var res []string
	res = append(res, c.Profile...)
	res = append(res, c.Profiles...)
	return res
}

// GetEffectiveIgnoreDirs returns combined ignore directory pattern list from 'ignore_dir' or 'ignore_dirs' config keys.
func (c *Config) GetEffectiveIgnoreDirs() []string {
	var res []string
	res = append(res, c.IgnoreDir...)
	res = append(res, c.IgnoreDirs...)
	return res
}

