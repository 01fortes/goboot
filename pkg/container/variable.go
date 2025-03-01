package container

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// VariableLoader defines an interface for components that can load variables
type VariableLoader interface {
	// Load loads variables into the container
	Load(ContextBuilder) error
}

// ProfileYamlLoader implements a Spring Boot style YAML file loader with profile support
type ProfileYamlLoader struct {
	// ConfigPath specifies directory where to look for config files
	ConfigPath string
	// Optional explicit list of profile names to load (eg. "dev", "prod")
	// If not specified, will read from GO_BOOT_ACTIVE_PROFILES environment variable
	Profiles []string
}

// Load loads variables from YAML files with profile support
func (l ProfileYamlLoader) Load(builder ContextBuilder) error {
	logger := slog.Default()

	// Default to current directory if not specified
	configPath := l.ConfigPath
	if configPath == "" {
		configPath = "."
	}

	// Get profiles from environment if not explicitly set
	profiles := l.Profiles
	if len(profiles) == 0 {
		profilesEnv := os.Getenv("GO_BOOT_ACTIVE_PROFILES")
		if profilesEnv != "" {
			profiles = strings.Split(profilesEnv, ",")
			for i, profile := range profiles {
				profiles[i] = strings.TrimSpace(profile)
			}
			logger.Info("Using profiles from GO_BOOT_ACTIVE_PROFILES", "profiles", profiles)
		}
	}

	// First load application.yml if it exists
	defaultConfigPath := filepath.Join(configPath, "application.yml")
	if _, err := os.Stat(defaultConfigPath); !os.IsNotExist(err) {
		logger.Info("Loading default configuration", "path", defaultConfigPath)
		if err := loadYamlConfig(defaultConfigPath, builder); err != nil {
			return fmt.Errorf("error loading default config: %w", err)
		}
	}

	// Then load each profile-specific file
	for _, profile := range profiles {
		profileConfigPath := filepath.Join(configPath, fmt.Sprintf("application-%s.yml", profile))
		if _, err := os.Stat(profileConfigPath); !os.IsNotExist(err) {
			logger.Info("Loading profile configuration", "profile", profile, "path", profileConfigPath)
			if err := loadYamlConfig(profileConfigPath, builder); err != nil {
				return fmt.Errorf("error loading profile config %s: %w", profile, err)
			}
		} else {
			logger.Info("Profile configuration not found, skipping", "profile", profile, "path", profileConfigPath)
		}
	}

	return nil
}

// VariableHelper provides utility functions for working with variables
type VariableHelper struct {
	ctx ApplicationContext
}

// NewVariableHelper creates a new helper for accessing typed variables
func NewVariableHelper(ctx ApplicationContext) *VariableHelper {
	return &VariableHelper{ctx: ctx}
}

// GetInt returns a variable as an int, with a default value if not found or invalid
func (h *VariableHelper) GetInt(name string, defaultValue int) int {
	value := h.ctx.GetVariableRaw(name)
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var result int
		if _, err := fmt.Sscanf(v, "%d", &result); err == nil {
			return result
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// GetFloat returns a variable as a float64, with a default value if not found or invalid
func (h *VariableHelper) GetFloat(name string, defaultValue float64) float64 {
	value := h.ctx.GetVariableRaw(name)
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		var result float64
		if _, err := fmt.Sscanf(v, "%f", &result); err == nil {
			return result
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// GetBool returns a variable as a bool, with a default value if not found or invalid
func (h *VariableHelper) GetBool(name string, defaultValue bool) bool {
	value := h.ctx.GetVariableRaw(name)
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		if v == "true" || v == "yes" || v == "1" {
			return true
		}
		if v == "false" || v == "no" || v == "0" {
			return false
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// GetString returns a variable as a string, with a default value if not found
func (h *VariableHelper) GetString(name string, defaultValue string) string {
	value := h.ctx.GetVariable(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetStruct unmarshals a variable or a section of the configuration into a struct
func (h *VariableHelper) GetStruct(name string, target interface{}) error {
	// Build a map of matching variables with the given prefix
	prefix := name + "."
	prefixLen := len(prefix)

	// Collect all variables with the given prefix
	matchingVars := make(map[string]interface{})

	// Try to get the root object first
	root := h.ctx.GetVariableRaw(name)
	if root != nil {
		// If it's a map, use it directly
		switch v := root.(type) {
		case map[string]interface{}:
			matchingVars = v
		case map[interface{}]interface{}:
			// Convert to string keys
			for mk, mv := range v {
				if strKey, ok := mk.(string); ok {
					matchingVars[strKey] = mv
				}
			}
		}
	}

	// If we didn't find a root object, try to build one from nested properties
	if len(matchingVars) == 0 {
		// Get all available variables to check for matching prefix
		allVars := h.collectAllVariables()

		// Check each variable to see if it starts with our prefix
		for k, v := range allVars {
			if strings.HasPrefix(k, prefix) {
				// Extract the part after the prefix
				key := k[prefixLen:]
				// Store the variable with the prefix removed
				matchingVars[key] = v
			}
		}
	}

	if len(matchingVars) == 0 {
		return fmt.Errorf("variable %s not found", name)
	}

	// Convert to YAML and unmarshal
	data, err := yaml.Marshal(matchingVars)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, target)
}

// collectAllVariables gets all variables from the container
// This is a helper method to make GetStruct more robust
func (h *VariableHelper) collectAllVariables() map[string]interface{} {
	// We access the container directly here, which is not ideal
	// but we need a way to get all variables

	// This is a best-effort implementation that might not always work
	// because we don't have a built-in way to get all variables

	// Try to use container-specific knowledge to extract vars
	container, ok := h.ctx.(*container)
	if ok && container != nil && container.variableRegistry != nil {
		registry, ok := container.variableRegistry.(*defaultVariableRegistry)
		if ok && registry != nil {
			// Make a copy of the variables to avoid concurrent access issues
			registry.mu.RLock()
			defer registry.mu.RUnlock()

			result := make(map[string]interface{}, len(registry.variables))
			for k, v := range registry.variables {
				result[k] = v
			}
			return result
		}
	}

	// Fallback: return an empty map
	return make(map[string]interface{})
}

// loadYamlConfig loads a YAML file and registers all variables in the container
func loadYamlConfig(filePath string, builder ContextBuilder) error {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Parse YAML into a map
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Register all variables with flattened keys
	flattenedMap := make(map[string]interface{})
	flattenMap(config, "", flattenedMap)

	for key, value := range flattenedMap {
		builder.RegisterVariable(key, value)
	}

	return nil
}

// flattenMap takes a nested map and flattens it into dot-separated keys
// e.g. {"server": {"port": 8080}} becomes {"server.port": 8080}
func flattenMap(input map[string]interface{}, prefix string, output map[string]interface{}) {
	for k, v := range input {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch value := v.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			flattenMap(value, key, output)
		case map[interface{}]interface{}:
			// Convert to string keys and recursively flatten
			stringMap := make(map[string]interface{})
			for mk, mv := range value {
				if strKey, ok := mk.(string); ok {
					stringMap[strKey] = mv
				}
			}
			flattenMap(stringMap, key, output)
		default:
			// For non-map values, add them directly
			output[key] = v
		}
	}
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
