package config

import "github.com/spf13/viper"

const (
	// DefaultAPIURL is the default RESMA API endpoint.
	DefaultAPIURL = "http://localhost:8080"
	// DefaultOutputFormat is the default output format for CLI commands.
	DefaultOutputFormat = "table"
	// DefaultTimeout is the default request timeout in seconds.
	DefaultTimeout = 30
)

// setDefaults applies default configuration values to the viper instance.
func setDefaults(v *viper.Viper) {
	v.SetDefault("api-url", DefaultAPIURL)
	v.SetDefault("output", DefaultOutputFormat)
	v.SetDefault("timeout", DefaultTimeout)
	v.SetDefault("no-color", false)
	v.SetDefault("debug", false)
}
