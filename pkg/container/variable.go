package container

import (
	"log/slog"
	"os"
	"strings"
)

// VariableLoader defines an interface for components that can load variables
type VariableLoader interface {
	// Load loads variables into the container
	Load(ContextBuilder) error
}

// SimpleYamlLoader implements a basic YAML file variable loader
type SimpleYamlLoader struct {
	// ConfigPath specifies where to look for config files
	ConfigPath string
	// Optional list of profile names to load (eg. "dev", "prod")
	Profiles []string
}

// Load loads variables from YAML files
func (l SimpleYamlLoader) Load(builder ContextBuilder) error {
	logger := slog.Default()

	// Default to application.yml in current directory if not specified
	configPath := l.ConfigPath
	if configPath == "" {
		configPath = "application.yml"
	}

	// Check if main file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Info("Config file not found, skipping", "path", configPath)
		return nil
	}

	// In this simplified implementation, we'll use environment variables
	// A real implementation would parse YAML files
	logger.Info("Loading variables from environment (simulating YAML loading)")
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			builder.RegisterVariable(key, value)
		}
	}

	return nil
}

// EnvVariableLoader loads variables from environment
type EnvVariableLoader struct {
	// Prefix filters environment variables to only those with this prefix
	Prefix string
}

// Load loads variables from environment
func (l EnvVariableLoader) Load(builder ContextBuilder) error {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Apply prefix filter if specified
			if l.Prefix == "" || strings.HasPrefix(key, l.Prefix) {
				// Remove prefix if it exists
				if l.Prefix != "" {
					key = strings.TrimPrefix(key, l.Prefix)
				}

				// Convert to lowercase and replace _ with .
				key = strings.ToLower(key)
				key = strings.ReplaceAll(key, "_", ".")

				builder.RegisterVariable(key, value)
			}
		}
	}

	return nil
}

// PropertiesVariableLoader loads variables from .properties files
type PropertiesVariableLoader struct {
	// Path to the properties file
	Path string
}

// Load loads variables from .properties file
func (l PropertiesVariableLoader) Load(builder ContextBuilder) error {
	// Skip if file doesn't exist
	if _, err := os.Stat(l.Path); os.IsNotExist(err) {
		slog.Info("Properties file not found, skipping", "path", l.Path)
		return nil
	}

	// Read file
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return err
	}

	// Parse properties
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// Skip comments and empty lines
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first equals sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Register variable
			builder.RegisterVariable(key, value)
		}
	}

	return nil
}
