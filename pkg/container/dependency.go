package container

import (
	"log/slog"
	"reflect"
	"time"
)

// DependencyResolver handles component dependency resolution
type DependencyResolver interface {
	DiscoverDependencies() error
	ValidateDependencies() error
	GetDependencies(componentName string) map[string]bool
}

// accessTrackingContext wraps a container to track component access during initialization
type accessTrackingContext struct {
	container     ApplicationContext
	componentName string
	accessedDeps  map[string]bool
	logger        *slog.Logger
	compRegistry  ComponentRegistry
}

func newAccessTrackingContext(container ApplicationContext, componentName string, logger *slog.Logger, registry ComponentRegistry) *accessTrackingContext {
	return &accessTrackingContext{
		container:     container,
		componentName: componentName,
		accessedDeps:  make(map[string]bool),
		logger:        logger,
		compRegistry:  registry,
	}
}

func (a *accessTrackingContext) GetComponent(target interface{}) error {
	// Get target type
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return ErrorWithCode("TARGET_NOT_POINTER", "target must be a pointer")
	}

	// Get the element type
	elemType := targetType.Elem()
	targetValue := reflect.ValueOf(target).Elem()

	// First try direct match with exact type name
	components := a.compRegistry.GetAll()
	for name, comp := range components {
		compType := reflect.TypeOf(comp)

		// Try exact type match first
		if compType == elemType || compType == reflect.PtrTo(elemType) {
			// Don't allow a component to access itself during dependency discovery
			if name == a.componentName {
				return CircularDependencyError([]string{name, name})
			}

			// Track dependency
			a.accessedDeps[name] = true
			a.logger.Debug("Component dependency detected via exact type match",
				"component", a.componentName,
				"depends_on", name,
				"type", elemType.String())

			// Always set the value for both discovery and initialization phases
			if targetValue.Kind() == reflect.Ptr {
				// For pointer targets like **TestComponent
				targetValue.Set(reflect.ValueOf(comp))
			} else if targetValue.Kind() == reflect.Struct && compType.Kind() == reflect.Ptr {
				// For struct targets and pointer components like TestComponent and *TestComponent
				targetValue.Set(reflect.ValueOf(comp).Elem())
			} else {
				// For other cases, try direct assignment
				targetValue.Set(reflect.ValueOf(comp))
			}
			return nil
		}
	}

	// Then try assignable types for interface support
	for name, comp := range components {
		compType := reflect.TypeOf(comp)

		// Check if the component type is assignable to the target type
		if compType.AssignableTo(elemType) {
			// Don't allow a component to access itself during dependency discovery
			if name == a.componentName {
				return CircularDependencyError([]string{name, name})
			}

			// Track dependency for this discovered match
			a.accessedDeps[name] = true
			a.logger.Debug("Component dependency detected via assignable type",
				"component", a.componentName,
				"depends_on", name,
				"type", elemType.String(),
				"comp_type", compType.String())

			// Always set the value for both discovery and initialization phases
			targetValue.Set(reflect.ValueOf(comp))
			return nil
		}
	}

	return ErrorWithCode("COMPONENT_TYPE_NOT_FOUND", "no component found matching type %v", elemType)
}

func (a *accessTrackingContext) GetComponentByName(name string) (Component, error) {
	// Don't allow a component to access itself during dependency discovery
	if name == a.componentName {
		return nil, CircularDependencyError([]string{name, name})
	}

	// Track that this component was accessed
	a.accessedDeps[name] = true

	// Check if the component exists
	a.logger.Debug("Component dependency detected by name",
		"component", a.componentName,
		"depends_on", name)

	comp, err := a.container.GetComponentByName(name)
	if err != nil {
		// During discovery phase, missing components aren't fatal
		// They will be checked again during actual initialization
		return nil, nil
	}

	return comp, nil
}

func (a *accessTrackingContext) GetVariable(name string) string {
	return a.container.GetVariable(name)
}

func (a *accessTrackingContext) GetVariableRaw(name string) interface{} {
	return a.container.GetVariableRaw(name)
}

func (a *accessTrackingContext) HasComponent(name string) bool {
	// Track component checking as well
	exists := a.container.HasComponent(name)
	if exists {
		a.accessedDeps[name] = true
	}
	return exists
}

func (a *accessTrackingContext) GetComponentNames() []string {
	return a.container.GetComponentNames()
}

func (a *accessTrackingContext) GetMetrics() map[string]*ComponentMetrics {
	return nil
}

// defaultDependencyResolver implements DependencyResolver
type defaultDependencyResolver struct {
	container    *container
	registry     ComponentRegistry
	dependencies map[string]map[string]bool
	metrics      MetricsCollector
	logger       *slog.Logger
}

func newDependencyResolver(container *container, registry ComponentRegistry, metrics MetricsCollector, logger *slog.Logger) *defaultDependencyResolver {
	return &defaultDependencyResolver{
		container:    container,
		registry:     registry,
		dependencies: make(map[string]map[string]bool),
		metrics:      metrics,
		logger:       logger,
	}
}

func (r *defaultDependencyResolver) discoverComponentDependencies(name string) (map[string]bool, error) {
	comp, err := r.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// Create a tracking context to discover dependencies
	tracker := newAccessTrackingContext(r.container, name, r.logger, r.registry)

	// Run the Init method with tracking
	// This won't actually initialize the component fully, just track dependencies
	start := time.Now()
	r.logger.Debug("Discovering dependencies", "component", name)
	_ = comp.Init(tracker) // Ignore errors during dependency discovery phase

	// Record metrics
	r.metrics.RecordDependencyCount(name, len(tracker.accessedDeps))

	r.logger.Debug("Dependencies discovered",
		"component", name,
		"dependencies", len(tracker.accessedDeps),
		"time_ms", time.Since(start).Milliseconds())

	// Return the discovered dependencies
	return tracker.accessedDeps, nil
}

func (r *defaultDependencyResolver) detectCycle(source, target string, visited map[string]bool, path []string) (bool, []string) {
	if source == target {
		return true, append(path, target)
	}

	if visited[target] {
		return false, nil
	}

	visited[target] = true
	path = append(path, target)

	for dep := range r.dependencies[target] {
		if hasCycle, cyclePath := r.detectCycle(source, dep, visited, path); hasCycle {
			return true, cyclePath
		}
	}

	return false, nil
}

func (r *defaultDependencyResolver) DiscoverDependencies() error {
	// Discover dependencies for all components
	r.logger.Info("Discovering component dependencies")
	components := r.registry.GetAll()

	for name := range components {
		deps, err := r.discoverComponentDependencies(name)
		if err != nil {
			return err
		}

		// Store discovered dependencies
		r.dependencies[name] = deps

		// Check for cycles after adding each component
		for dep := range deps {
			// Skip self-dependencies
			if dep == name {
				continue
			}

			// Check if adding this dependency would create a cycle
			hasCycle, cycle := r.detectCycle(name, dep, make(map[string]bool), []string{name})
			if hasCycle {
				return CircularDependencyError(cycle)
			}
		}
	}

	return nil
}

func (r *defaultDependencyResolver) ValidateDependencies() error {
	for _, deps := range r.dependencies {
		for dep := range deps {
			if !r.registry.Has(dep) {
				return ComponentNotFoundError(dep)
			}
		}
	}
	return nil
}

func (r *defaultDependencyResolver) GetDependencies(componentName string) map[string]bool {
	deps, exists := r.dependencies[componentName]
	if !exists {
		return nil
	}

	// Return a copy to avoid external modification
	result := make(map[string]bool, len(deps))
	for k, v := range deps {
		result[k] = v
	}
	return result
}
