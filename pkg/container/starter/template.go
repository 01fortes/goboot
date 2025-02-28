package starter

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"GoBoot/pkg/container"
)

// StarterTemplate provides a template for easily creating starters
type StarterTemplate struct {
	// Name of the starter
	Name string
	// PropertyPrefix for configuration properties
	PropertyPrefix string
	// RequiredProperties that must be defined for the starter to run
	RequiredProperties []string
	// Condition determines if this starter should run
	Condition func(container.ApplicationContext) bool
	// ComponentsFunc registers components with the container
	ComponentsFunc func(container.ContextBuilder, map[string]string) error
}

// Create creates a new starter from the template
func (t *StarterTemplate) Create() container.Starter {
	condition := t.Condition
	if condition == nil && len(t.RequiredProperties) > 0 {
		// Default condition checks required properties
		condition = func(ctx container.ApplicationContext) bool {
			for _, prop := range t.RequiredProperties {
				if ctx.GetVariable(prop) == "" {
					return false
				}
			}
			return true
		}
	}

	// Create starter function
	return container.NewConditionalStarter(
		t.Name,
		condition,
		func(builder container.ContextBuilder) error {
			// Collect configuration properties
			config := make(map[string]string)

			// Get all variables starting with prefix
			for _, name := range getVariablesWithPrefix(builder, t.PropertyPrefix) {
				// Remove prefix
				key := strings.TrimPrefix(name, t.PropertyPrefix)
				value := builder.GetVariable(name)
				config[key] = value
			}

			// Verify required properties
			for _, prop := range t.RequiredProperties {
				if _, ok := config[prop]; !ok {
					propName := prop
					if t.PropertyPrefix != "" {
						propName = t.PropertyPrefix + prop
					}
					return fmt.Errorf("required property %s not found", propName)
				}
			}

			// Log configuration (excluding sensitive values)
			logConfig := make(map[string]string)
			for k, v := range config {
				if isSensitive(k) {
					logConfig[k] = "******"
				} else {
					logConfig[k] = v
				}
			}
			slog.Info("Starting "+t.Name, "config", fmt.Sprintf("%v", logConfig))

			// Register components
			if t.ComponentsFunc != nil {
				return t.ComponentsFunc(builder, config)
			}

			return nil
		},
	)
}

// getVariablesWithPrefix returns all variables starting with the given prefix
func getVariablesWithPrefix(ctx container.ApplicationContext, prefix string) []string {
	// In a real implementation, this would use GetAllVariables
	// For now, we'll use a placeholder
	return []string{prefix + "url", prefix + "username", prefix + "password"}
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

// CreateTypedComponent creates a component with the given name and init function
func CreateTypedComponent[T any](name string, initFn func(container.ApplicationContext, *T)) *TypedComponent[T] {
	return &TypedComponent[T]{
		name:     name,
		initFn:   initFn,
		instance: new(T),
	}
}

// TypedComponent is a generic component implementation
type TypedComponent[T any] struct {
	name     string
	initFn   func(container.ApplicationContext, *T)
	instance *T
}

// Name returns the component name
func (c *TypedComponent[T]) Name() string {
	return c.name
}

// Init initializes the component
func (c *TypedComponent[T]) Init(ctx container.ApplicationContext) {
	if c.initFn != nil {
		c.initFn(ctx, c.instance)
	}
}

// Component returns the underlying component instance
func (c *TypedComponent[T]) Component() *T {
	return c.instance
}
