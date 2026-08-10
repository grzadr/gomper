package config

import "github.com/grzadr/gomper/internal/app"

// Config encapsulates application configuration populated from flags, env, and config files.
type Config struct {
	ConfigFile     string           `mapstructure:"-"`
	LogLevel       string           `mapstructure:"log_level"`
	Format         app.OutputFormat `mapstructure:"format"`
	Output         string           `mapstructure:"output"`
	Ignore         []string         `mapstructure:"ignore"`
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
