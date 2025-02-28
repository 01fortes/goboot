package container

// ApplicationContext is the interface used by components to access container resources
type ApplicationContext interface {
	// GetComponent returns a component by type using a pointer to a variable of the desired type
	// Example: var logger *LoggerComponent; ctx.GetComponent(&logger)
	GetComponent(target interface{}) error
	// GetComponentByName returns a component by name (generally discouraged - use GetComponent instead)
	GetComponentByName(name string) (Component, error)
	// GetVariable returns a variable by name
	GetVariable(name string) string
	// HasComponent checks if a component exists
	HasComponent(name string) bool
	// GetComponentNames returns all registered component names
	GetComponentNames() []string
	// GetMetrics returns metrics for all components
	GetMetrics() map[string]*ComponentMetrics
}

// ContextBuilder is used during container initialization
type ContextBuilder interface {
	ApplicationContext
	// RegisterComponent adds a component to the container
	RegisterComponent(component Component) error
	// RegisterVariable adds a variable to the container
	RegisterVariable(name string, value string)
	// RegisterVariableLoader adds a variable loader
	RegisterVariableLoader(loader VariableLoader)
	// RegisterFactory adds a component factory
	RegisterFactory(factory Factory)
	// RegisterStarter adds a starter to the container
	RegisterStarter(starter Starter)
}
