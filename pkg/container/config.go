package container

import "log/slog"

// Config contains configuration options for the container
type Config struct {
	// EnableMetrics enables component metrics
	EnableMetrics bool
	// Logger for container operations (uses slog.Default if nil)
	Logger *slog.Logger
	// DefaultVariableLoaders are loaded by default
	DefaultVariableLoaders []VariableLoader
	// DefaultStarters are loaded by default
	DefaultStarters []Starter
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		EnableMetrics: true,
		Logger:        slog.Default(),
		DefaultVariableLoaders: []VariableLoader{
			&SimpleYamlLoader{},
			&EnvVariableLoader{},
		},
		DefaultStarters: []Starter{},
	}
}
