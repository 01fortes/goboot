package container

import (
	"fmt"
	"log/slog"
	"sync"
)

// ComponentRegistry manages component registration and retrieval
type ComponentRegistry interface {
	Register(component Component) error
	Get(name string) (Component, error)
	Has(name string) bool
	GetAll() map[string]Component
	GetNames() []string
}

// defaultComponentRegistry implements ComponentRegistry
type defaultComponentRegistry struct {
	components map[string]Component
	mu         sync.RWMutex
	logger     *slog.Logger
}

func newComponentRegistry(logger *slog.Logger) *defaultComponentRegistry {
	return &defaultComponentRegistry{
		components: make(map[string]Component),
		logger:     logger,
	}
}

func (r *defaultComponentRegistry) Register(component Component) error {
	if component == nil {
		return fmt.Errorf("cannot register nil component")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := component.Name()
	if name == "" {
		return fmt.Errorf("component name cannot be empty")
	}

	if _, exists := r.components[name]; exists {
		return ComponentAlreadyRegisteredError(name)
	}

	r.logger.Info("Registering component", "name", name)
	r.components[name] = component
	return nil
}

func (r *defaultComponentRegistry) Get(name string) (Component, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comp, exists := r.components[name]
	if !exists {
		return nil, ComponentNotFoundError(name)
	}
	return comp, nil
}

func (r *defaultComponentRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.components[name]
	return exists
}

func (r *defaultComponentRegistry) GetAll() map[string]Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]Component, len(r.components))
	for k, v := range r.components {
		result[k] = v
	}
	return result
}

func (r *defaultComponentRegistry) GetNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	return names
}

// VariableRegistry manages variable registration and retrieval
type VariableRegistry interface {
	Register(name string, value interface{})
	Get(name string) interface{}
	GetString(name string) string
}

// defaultVariableRegistry implements VariableRegistry
type defaultVariableRegistry struct {
	variables map[string]interface{}
	mu        sync.RWMutex
	logger    *slog.Logger
}

func newVariableRegistry(logger *slog.Logger) *defaultVariableRegistry {
	return &defaultVariableRegistry{
		variables: make(map[string]interface{}),
		logger:    logger,
	}
}

func (r *defaultVariableRegistry) Register(name string, value interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Debug("Registering variable", "name", name, "type", fmt.Sprintf("%T", value))
	r.variables[name] = value
}

func (r *defaultVariableRegistry) Get(name string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.variables[name]
}

func (r *defaultVariableRegistry) GetString(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value := r.variables[name]
	if value == nil {
		return ""
	}

	// Convert value to string if possible
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
