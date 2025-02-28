package container

import (
	"log/slog"
	"time"
)

// ComponentInitializer handles component initialization in dependency order
type ComponentInitializer interface {
	InitializeAll() error
	GetInitOrder() []string
}

// defaultComponentInitializer implements ComponentInitializer
type defaultComponentInitializer struct {
	container    *container
	registry     ComponentRegistry
	dependencies DependencyResolver
	initialized  map[string]bool
	initOrder    []string
	metrics      MetricsCollector
	logger       *slog.Logger
}

func newComponentInitializer(container *container, registry ComponentRegistry, dependencies DependencyResolver, metrics MetricsCollector, logger *slog.Logger) *defaultComponentInitializer {
	return &defaultComponentInitializer{
		container:    container,
		registry:     registry,
		dependencies: dependencies,
		initialized:  make(map[string]bool),
		initOrder:    []string{},
		metrics:      metrics,
		logger:       logger,
	}
}

func (i *defaultComponentInitializer) initComponent(name string, visited map[string]bool, path []string) error {
	if i.initialized[name] {
		return nil
	}

	if visited[name] {
		cycle := append(path, name)
		return CircularDependencyError(cycle)
	}

	comp, err := i.registry.Get(name)
	if err != nil {
		return err
	}

	deps := i.dependencies.GetDependencies(name)

	// Mark as being visited (for cycle detection)
	visited[name] = true
	path = append(path, name)

	// Initialize dependencies first
	if deps != nil {
		for depName := range deps {
			if depName != name { // Skip self-dependencies
				if err := i.initComponent(depName, visited, path); err != nil {
					return err
				}
			}
		}
	}

	// Initialize the component for real this time
	i.logger.Debug("Initializing component", "name", name)
	start := time.Now()
	comp.Init(i.container)
	duration := time.Since(start)

	// Record metrics
	i.metrics.RecordInitDuration(name, duration)

	i.logger.Debug("Component initialized",
		"name", name,
		"time_ms", duration.Milliseconds())

	i.initialized[name] = true
	i.initOrder = append(i.initOrder, name)

	// Remove from visited after initialization
	delete(visited, name)

	return nil
}

func (i *defaultComponentInitializer) InitializeAll() error {
	// Initialize components in dependency order
	i.logger.Info("Initializing components")
	components := i.registry.GetAll()

	for name := range components {
		if !i.initialized[name] {
			if err := i.initComponent(name, make(map[string]bool), []string{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *defaultComponentInitializer) GetInitOrder() []string {
	// Return a copy to avoid external modification
	result := make([]string, len(i.initOrder))
	copy(result, i.initOrder)
	return result
}
