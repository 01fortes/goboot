package starter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/01fortes/goboot/pkg/container"
)

// AutoConfiguration is a marker interface for auto-configuration classes
type AutoConfiguration interface {
	// IsAutoConfiguration marks this as an auto-configuration class
	IsAutoConfiguration() bool
}

// EnableAutoConfiguration enables a specific auto-configuration
type EnableAutoConfiguration struct {
	// Enabled indicates if this auto-configuration is enabled
	Enabled bool
}

// IsAutoConfiguration implements the AutoConfiguration interface
func (e *EnableAutoConfiguration) IsAutoConfiguration() bool {
	return e.Enabled
}

// ConditionalOnProperty defines a condition based on a property value
type ConditionalOnProperty struct {
	// Property name to check
	Property string
	// ExpectedValue that the property should have (empty means any non-empty value)
	ExpectedValue string
	// Missing is true if the property should be missing
	Missing bool
}

// ConditionalOnComponent defines a condition based on component existence
type ConditionalOnComponent struct {
	// Component name or type that should exist
	Component string
}

// ConditionalOnMissingComponent defines a condition based on component absence
type ConditionalOnMissingComponent struct {
	// Component name or type that should NOT exist
	Component string
}

// ConditionalOnClass defines a condition based on class existence (package import)
type ConditionalOnClass struct {
	// Type that should be available at runtime
	Class reflect.Type
}

// Properties defines a configuration properties structure
type Properties struct {
	// Prefix for all properties in this group
	Prefix string
	// Target is the target struct to bind properties to
	Target interface{}
}

// AutoConfigurer is a simplified starter for Spring Boot-like auto-configuration
type AutoConfigurer struct {
	// Name of this auto-configuration
	Name string
	// Properties to bind from configuration
	Properties *Properties
	// ConditionalOnProperty specifies a property condition
	ConditionalOnProperty *ConditionalOnProperty
	// ConditionalOnComponent specifies a component condition
	ConditionalOnComponent *ConditionalOnComponent
	// ConditionalOnMissingComponent specifies a missing component condition
	ConditionalOnMissingComponent *ConditionalOnMissingComponent
	// ConditionalOnClass specifies a class condition
	ConditionalOnClass *ConditionalOnClass
	// ConfigureFunc registers components with the container
	ConfigureFunc func(container.ContextBuilder, interface{}) error
}

// Create creates a new starter from the auto-configurer
func (ac *AutoConfigurer) Create() container.Starter {
	// Build condition function based on all conditionals
	condition := func(ctx container.ApplicationContext) bool {
		// Check property condition
		if ac.ConditionalOnProperty != nil {
			value := ctx.GetVariable(ac.ConditionalOnProperty.Property)

			if ac.ConditionalOnProperty.Missing {
				if value != "" {
					return false
				}
			} else {
				if ac.ConditionalOnProperty.ExpectedValue != "" {
					if value != ac.ConditionalOnProperty.ExpectedValue {
						return false
					}
				} else if value == "" {
					return false
				}
			}
		}

		// Check component condition
		if ac.ConditionalOnComponent != nil {
			if !ctx.HasComponent(ac.ConditionalOnComponent.Component) {
				return false
			}
		}

		// Check missing component condition
		if ac.ConditionalOnMissingComponent != nil {
			if ctx.HasComponent(ac.ConditionalOnMissingComponent.Component) {
				return false
			}
		}

		// Class condition is checked during creation, not at runtime

		return true
	}

	// Create the starter
	return container.NewConditionalStarter(
		ac.Name,
		condition,
		func(builder container.ContextBuilder) error {
			// Bind properties to target if specified
			var config interface{}
			if ac.Properties != nil && ac.Properties.Target != nil {
				config = ac.Properties.Target

				// Get all properties with prefix and bind to target
				if err := bindProperties(builder, ac.Properties.Prefix, config); err != nil {
					return err
				}

				// Log configuration (excluding sensitive values)
				logConfig(ac.Name, config)
			}

			// Call configuration function with bound properties
			if ac.ConfigureFunc != nil {
				return ac.ConfigureFunc(builder, config)
			}

			return nil
		},
	)
}

// bindProperties binds properties with the given prefix to the target struct
func bindProperties(ctx container.ApplicationContext, prefix string, target interface{}) error {
	// Get all properties with prefix
	props := getAllPropertiesWithPrefix(ctx, prefix)
	if len(props) == 0 {
		return nil
	}

	// Create a map for JSON conversion
	propMap := make(map[string]interface{})

	// Convert flat properties to nested map
	for key, value := range props {
		// Remove prefix
		key = strings.TrimPrefix(key, prefix)
		if key == "" {
			continue
		}

		// Convert to nested map based on dot notation
		parts := strings.Split(key, ".")
		current := propMap

		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part, set the value
				current[part] = value
			} else {
				// Create nested map if needed
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]interface{})
				}

				// Move to nested map
				current = current[part].(map[string]interface{})
			}
		}
	}

	// Convert map to JSON
	jsonData, err := json.Marshal(propMap)
	if err != nil {
		return fmt.Errorf("error marshaling properties: %w", err)
	}

	// Unmarshal JSON to target
	return json.Unmarshal(jsonData, target)
}

// getAllPropertiesWithPrefix returns all properties with the given prefix
func getAllPropertiesWithPrefix(ctx container.ApplicationContext, prefix string) map[string]string {
	// This would need to be implemented with GetAllVariables
	// For now, return a placeholder
	return map[string]string{
		prefix + "url":      "jdbc:mysql://localhost:3306/db",
		prefix + "username": "admin",
		prefix + "password": "secret",
	}
}

// logConfig logs the configuration, masking sensitive values
func logConfig(name string, config interface{}) {
	// Convert to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal config", "error", err)
		return
	}

	// Convert to map for masking
	var configMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &configMap); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
		return
	}

	// Mask sensitive values (recursive)
	maskSensitiveValues(configMap)

	// Convert back to JSON
	maskedJson, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal masked config", "error", err)
		return
	}

	slog.Info("Auto-configuration "+name, "config", string(maskedJson))
}

// maskSensitiveValues masks sensitive values in a map recursively
func maskSensitiveValues(m map[string]interface{}) {
	for k, v := range m {
		// Check if this key is sensitive
		if isSensitive(k) {
			m[k] = "******"
			continue
		}

		// Recurse into nested maps
		if nestedMap, ok := v.(map[string]interface{}); ok {
			maskSensitiveValues(nestedMap)
		}
	}
}

// isSensitive returns true if the property name suggests it contains sensitive information
func isSensitive(name string) bool {
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "password") ||
		strings.Contains(lowerName, "secret") ||
		strings.Contains(lowerName, "token") ||
		strings.Contains(lowerName, "key") && !strings.Contains(lowerName, "public")
}

// ExtractTypeName returns the name of a type without the package prefix
func ExtractTypeName(t reflect.Type) string {
	name := t.String()
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

// AutoComponent is a generic component implementation with auto-wiring
type AutoComponent[T any] struct {
	name     string
	instance *T
}

// NewAutoComponent creates a component that will be auto-configured
func NewAutoComponent[T any](name string) *AutoComponent[T] {
	return &AutoComponent[T]{
		name:     name,
		instance: new(T),
	}
}

// Name returns the component name
func (c *AutoComponent[T]) Name() string {
	return c.name
}

// Init initializes the component with auto-wiring
func (c *AutoComponent[T]) Init(ctx interface{}) {
	applicationContext := ctx.(container.ApplicationContext)

	// Use reflection to auto-wire dependencies
	t := reflect.TypeOf(c.instance).Elem()
	v := reflect.ValueOf(c.instance).Elem()

	// Iterate over all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Check for autowire tag
		if _, ok := field.Tag.Lookup("autowire"); ok {
			fieldValue := v.Field(i)

			// Skip if already set
			if !fieldValue.IsZero() {
				continue
			}

			// Create a pointer to the field type
			ptr := reflect.New(field.Type)

			// Try to get component by type
			err := applicationContext.GetComponent(ptr.Interface())
			if err == nil {
				// Set the field
				fieldValue.Set(ptr.Elem())
			} else {
				slog.Warn("Failed to autowire field",
					"component", c.name,
					"field", field.Name,
					"type", field.Type.String(),
					"error", err)
			}
		}
	}
}

// Get returns the component instance
func (c *AutoComponent[T]) Get() *T {
	return c.instance
}
