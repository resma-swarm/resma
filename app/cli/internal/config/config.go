package config

import "github.com/spf13/viper"

// Config holds the CLI configuration values.
type Config struct {
	APIURL       string
	Token        string
	APIKey       string
	OutputFormat string
	NoColor      bool
	Timeout      int
	Debug        bool
	ConfigFile   string
}

// Load reads configuration from file and environment, returning a Config with defaults applied.
func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	cfg := &Config{
		APIURL:       v.GetString("api-url"),
		Token:        v.GetString("token"),
		APIKey:       v.GetString("api-key"),
		OutputFormat: v.GetString("output"),
		NoColor:      v.GetBool("no-color"),
		Timeout:      v.GetInt("timeout"),
		Debug:        v.GetBool("debug"),
		ConfigFile:   v.GetString("config-file"),
	}

	return cfg, nil
}
