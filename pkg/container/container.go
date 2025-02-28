package container

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"
)

// ApplicationContext is the interface used by components to access container resources
type ApplicationContext interface {
	// GetComponent returns a component by name
	GetComponent(name string) (Component, error)
	// GetComponentByType returns a component by type (using the provided pointer)
	GetComponentByType(target interface{}) error
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

// accessTrackingContext wraps a container to track component access during initialization
type accessTrackingContext struct {
	container     *container
	componentName string
	accessedDeps  map[string]bool
	logger        *slog.Logger
}

func (a *accessTrackingContext) GetComponent(name string) (Component, error) {
	// Don't allow a component to access itself during dependency discovery
	if name == a.componentName {
		return nil, CircularDependencyError([]string{name, name})
	}

	// Track that this component was accessed
	a.accessedDeps[name] = true

	// Check if the component exists
	a.logger.Debug("Component dependency detected",
		"component", a.componentName,
		"depends_on", name)

	comp, err := a.container.GetComponent(name)
	if err != nil {
		// During discovery phase, missing components aren't fatal
		// They will be checked again during actual initialization
		return nil, nil
	}

	return comp, nil
}

func (a *accessTrackingContext) GetComponentByType(target interface{}) error {
	// Get target type
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	// Get the element type
	elemType := targetType.Elem()
	targetValue := reflect.ValueOf(target).Elem()

	// Search for a component of the matching type
	a.container.componentsMu.RLock()
	defer a.container.componentsMu.RUnlock()

	// First try direct match with exact type name
	for name, comp := range a.container.components {
		compType := reflect.TypeOf(comp)

		// Try exact type match first
		if compType == elemType || compType == reflect.PtrTo(elemType) {
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
	for name, comp := range a.container.components {
		compType := reflect.TypeOf(comp)

		// Check if the component type is assignable to the target type
		if compType.AssignableTo(elemType) {
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

	return fmt.Errorf("no component found matching type %v", elemType)
}

func (a *accessTrackingContext) GetVariable(name string) string {
	return a.container.GetVariable(name)
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

// container is the central dependency container implementation
type container struct {
	config           *Config
	components       map[string]Component
	variables        map[string]string
	initialized      map[string]bool
	startOrder       []string
	dependencies     map[string]map[string]bool // component name -> dependencies
	metrics          map[string]*ComponentMetrics
	starters         []Starter
	variablesLoaders []VariableLoader
	factories        []Factory
	logger           *slog.Logger

	// Locks for concurrent access
	componentsMu sync.RWMutex
	variablesMu  sync.RWMutex
	metricsMu    sync.RWMutex
	startupTime  time.Time
}

// ComponentMetrics stores metrics for a component
type ComponentMetrics struct {
	Name            string
	InitDuration    time.Duration
	StartDuration   time.Duration
	StopDuration    time.Duration
	DependencyCount int
}

// RegisterComponent adds a component to the container
func (c *container) RegisterComponent(component Component) error {
	if component == nil {
		return fmt.Errorf("cannot register nil component")
	}

	c.componentsMu.Lock()
	defer c.componentsMu.Unlock()

	name := component.Name()
	if name == "" {
		return fmt.Errorf("component name cannot be empty")
	}

	if _, exists := c.components[name]; exists {
		return ComponentAlreadyRegisteredError(name)
	}

	c.logger.Info("Registering component", "name", name)
	c.components[name] = component
	return nil
}

// RegisterVariable adds a variable to the container
func (c *container) RegisterVariable(name string, value string) {
	c.variablesMu.Lock()
	defer c.variablesMu.Unlock()

	c.logger.Debug("Registering variable", "name", name)
	c.variables[name] = value
}

// RegisterVariableLoader adds a variable loader
func (c *container) RegisterVariableLoader(loader VariableLoader) {
	c.variablesLoaders = append(c.variablesLoaders, loader)
}

// RegisterFactory adds a component factory
func (c *container) RegisterFactory(factory Factory) {
	c.factories = append(c.factories, factory)
}

// RegisterStarter adds a starter to the container
func (c *container) RegisterStarter(starter Starter) {
	c.starters = append(c.starters, starter)
}

// HasComponent checks if a component exists
func (c *container) HasComponent(name string) bool {
	c.componentsMu.RLock()
	defer c.componentsMu.RUnlock()

	_, exists := c.components[name]
	return exists
}

// GetComponentNames returns all registered component names
func (c *container) GetComponentNames() []string {
	c.componentsMu.RLock()
	defer c.componentsMu.RUnlock()

	names := make([]string, 0, len(c.components))
	for name := range c.components {
		names = append(names, name)
	}
	return names
}

// GetComponent returns a component by name
func (c *container) GetComponent(name string) (Component, error) {
	c.componentsMu.RLock()
	defer c.componentsMu.RUnlock()

	comp, exists := c.components[name]
	if !exists {
		return nil, ComponentNotFoundError(name)
	}
	return comp, nil
}

// GetComponentByType finds a component matching the type of the provided pointer and sets the pointer
func (c *container) GetComponentByType(target interface{}) error {
	// Get target type
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	// Get the element type
	elemType := targetType.Elem()

	// Lock for reading components
	c.componentsMu.RLock()
	defer c.componentsMu.RUnlock()

	// Search for a component of the matching type
	for _, comp := range c.components {
		compType := reflect.TypeOf(comp)
		if compType.AssignableTo(elemType) {
			// Found a match, set the pointer
			reflect.ValueOf(target).Elem().Set(reflect.ValueOf(comp))
			return nil
		}
	}

	return fmt.Errorf("no component found matching type %v", elemType)
}

// GetVariable returns a variable by name
func (c *container) GetVariable(name string) string {
	c.variablesMu.RLock()
	defer c.variablesMu.RUnlock()

	return c.variables[name]
}

// GetMetrics returns metrics for all components
func (c *container) GetMetrics() map[string]*ComponentMetrics {
	if !c.config.EnableMetrics {
		return nil
	}

	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()

	// Create a copy to avoid races
	result := make(map[string]*ComponentMetrics, len(c.metrics))
	for k, v := range c.metrics {
		copy := *v
		result[k] = &copy
	}

	return result
}

// discoverDependencies runs a component's Init method in tracking mode to discover dependencies
func (c *container) discoverDependencies(name string) (map[string]bool, error) {
	comp, err := c.GetComponent(name)
	if err != nil {
		return nil, err
	}

	// Create a tracking context to discover dependencies
	tracker := &accessTrackingContext{
		container:     c,
		componentName: name,
		accessedDeps:  make(map[string]bool),
		logger:        c.logger,
	}

	// Run the Init method with tracking
	// This won't actually initialize the component fully, just track dependencies
	start := time.Now()
	c.logger.Debug("Discovering dependencies", "component", name)
	comp.Init(tracker)

	// Record metrics
	if c.config.EnableMetrics {
		c.metricsMu.Lock()
		if _, exists := c.metrics[name]; !exists {
			c.metrics[name] = &ComponentMetrics{
				Name: name,
			}
		}
		c.metrics[name].DependencyCount = len(tracker.accessedDeps)
		c.metricsMu.Unlock()
	}

	c.logger.Debug("Dependencies discovered",
		"component", name,
		"dependencies", len(tracker.accessedDeps),
		"time_ms", time.Since(start).Milliseconds())

	// Return the discovered dependencies
	return tracker.accessedDeps, nil
}

// detectCycle checks if adding an edge from source to target would create a cycle
func (c *container) detectCycle(source, target string, visited map[string]bool, path []string) (bool, []string) {
	if source == target {
		return true, append(path, target)
	}

	if visited[target] {
		return false, nil
	}

	visited[target] = true
	path = append(path, target)

	for dep := range c.dependencies[target] {
		if hasCycle, cyclePath := c.detectCycle(source, dep, visited, path); hasCycle {
			return true, cyclePath
		}
	}

	return false, nil
}

// buildDependencyGraph builds the dependency graph and detects cycles
func (c *container) buildDependencyGraph() error {
	// Discover dependencies for all components
	c.logger.Info("Discovering component dependencies")
	for name := range c.components {
		deps, err := c.discoverDependencies(name)
		if err != nil {
			return err
		}

		// Store discovered dependencies
		c.dependencies[name] = deps

		// Check for cycles after adding each component
		for dep := range deps {
			// Skip self-dependencies
			if dep == name {
				continue
			}

			// Check if adding this dependency would create a cycle
			hasCycle, cycle := c.detectCycle(name, dep, make(map[string]bool), []string{name})
			if hasCycle {
				return CircularDependencyError(cycle)
			}
		}
	}

	return nil
}

// validateDependencies validates that all dependencies exist
func (c *container) validateDependencies() error {
	for _, deps := range c.dependencies {
		for dep := range deps {
			if !c.HasComponent(dep) {
				return ComponentNotFoundError(dep)
			}
		}
	}
	return nil
}

// initComponent initializes a component and its dependencies
func (c *container) initComponent(name string, visited map[string]bool, path []string) error {
	c.componentsMu.Lock()
	if c.initialized[name] {
		c.componentsMu.Unlock()
		return nil
	}

	if visited[name] {
		c.componentsMu.Unlock()
		cycle := append(path, name)
		return CircularDependencyError(cycle)
	}

	comp, exists := c.components[name]
	if !exists {
		c.componentsMu.Unlock()
		return ComponentNotFoundError(name)
	}

	deps, hasDeps := c.dependencies[name]
	c.componentsMu.Unlock()

	// Mark as being visited (for cycle detection)
	visited[name] = true
	path = append(path, name)

	// Initialize dependencies first
	if hasDeps {
		for depName := range deps {
			if depName != name { // Skip self-dependencies
				if err := c.initComponent(depName, visited, path); err != nil {
					return err
				}
			}
		}
	}

	// Initialize the component for real this time
	c.logger.Debug("Initializing component", "name", name)
	start := time.Now()
	comp.Init(c)
	duration := time.Since(start)

	// Record metrics
	if c.config.EnableMetrics {
		c.metricsMu.Lock()
		if _, exists := c.metrics[name]; !exists {
			c.metrics[name] = &ComponentMetrics{
				Name: name,
			}
		}
		c.metrics[name].InitDuration = duration
		c.metricsMu.Unlock()
	}

	c.logger.Debug("Component initialized",
		"name", name,
		"time_ms", duration.Milliseconds())

	c.componentsMu.Lock()
	c.initialized[name] = true
	c.startOrder = append(c.startOrder, name)
	c.componentsMu.Unlock()

	// Remove from visited after initialization
	delete(visited, name)

	return nil
}

// initializeAllComponents initializes all components in dependency order
func (c *container) initializeAllComponents() error {
	// Initialize components in dependency order
	c.logger.Info("Initializing components")
	for name := range c.components {
		if !c.initialized[name] {
			if err := c.initComponent(name, make(map[string]bool), []string{}); err != nil {
				return err
			}
		}
	}
	return nil
}

// startAllComponents starts all lifecycle components in initialization order
func (c *container) startAllComponents(ctx context.Context) error {
	// Start components in dependency order
	c.logger.Info("Starting components")

	for _, name := range c.startOrder {
		component, err := c.GetComponent(name)
		if err != nil {
			return err
		}

		if lifecycle, ok := component.(LifecycleComponent); ok {
			c.logger.Debug("Starting component", "name", name)

			start := time.Now()
			lifecycle.Start(ctx)
			duration := time.Since(start)

			if c.config.EnableMetrics {
				c.metricsMu.Lock()
				if _, exists := c.metrics[name]; !exists {
					c.metrics[name] = &ComponentMetrics{
						Name: name,
					}
				}
				c.metrics[name].StartDuration = duration
				c.metricsMu.Unlock()
			}

			c.logger.Info("Component started",
				"name", name,
				"time_ms", duration.Milliseconds())
		}
	}

	return nil
}

// stopAllComponents stops all lifecycle components in reverse initialization order
func (c *container) stopAllComponents(ctx context.Context) {
	c.logger.Info("Stopping components")

	// Stop in reverse order
	for i := len(c.startOrder) - 1; i >= 0; i-- {
		name := c.startOrder[i]

		component, err := c.GetComponent(name)
		if err != nil {
			c.logger.Error("Error getting component during shutdown",
				"name", name,
				"error", err)
			continue
		}

		if lifecycle, ok := component.(LifecycleComponent); ok {
			c.logger.Debug("Stopping component", "name", name)

			start := time.Now()
			lifecycle.Stop(ctx)
			duration := time.Since(start)

			if c.config.EnableMetrics {
				c.metricsMu.Lock()
				if _, exists := c.metrics[name]; !exists {
					c.metrics[name] = &ComponentMetrics{
						Name: name,
					}
				}
				c.metrics[name].StopDuration = duration
				c.metricsMu.Unlock()
			}

			c.logger.Info("Component stopped",
				"name", name,
				"time_ms", duration.Milliseconds())
		}
	}
}

// runStarters runs all registered starters
func (c *container) runStarters() error {
	c.logger.Info("Running starters", "count", len(c.starters))

	for _, starter := range c.starters {
		c.logger.Debug("Running starter", "name", starter.Name())
		if conditionalStarter, ok := starter.(ConditionalStarter); ok {
			if !conditionalStarter.ShouldStart(c) {
				c.logger.Debug("Skipping conditional starter", "name", starter.Name())
				continue
			}
		}

		if err := starter.Start(c); err != nil {
			return fmt.Errorf("starter %s failed: %w", starter.Name(), err)
		}
	}

	return nil
}

// New creates a new container with the given configuration
func New(ctx context.Context, cfg *Config, block func(ContextBuilder)) (ApplicationContext, func(), error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	logger := cfg.Logger

	logger.Info("Creating container")
	startTime := time.Now()

	res := &container{
		config:           cfg,
		variablesLoaders: cfg.DefaultVariableLoaders,
		starters:         cfg.DefaultStarters,
		factories:        []Factory{},
		components:       map[string]Component{},
		variables:        map[string]string{},
		initialized:      map[string]bool{},
		startOrder:       []string{},
		dependencies:     map[string]map[string]bool{},
		metrics:          map[string]*ComponentMetrics{},
		logger:           logger,
		startupTime:      startTime,
	}

	// Register components and variables
	block(res)

	// Run factories to register components
	logger.Info("Running component factories", "count", len(res.factories))
	for _, factory := range res.factories {
		if err := factory.Create(res); err != nil {
			return nil, nil, fmt.Errorf("factory failed: %w", err)
		}
	}

	// Load variables from loaders
	logger.Info("Loading variables", "loaders", len(res.variablesLoaders))
	for _, loader := range res.variablesLoaders {
		if err := loader.Load(res); err != nil {
			return nil, nil, fmt.Errorf("variable loader failed: %w", err)
		}
	}

	// Run starters - these can register more components
	if err := res.runStarters(); err != nil {
		return nil, nil, err
	}

	// Build dependency graph and validate
	if err := res.buildDependencyGraph(); err != nil {
		return nil, nil, err
	}

	// Validate dependencies
	if err := res.validateDependencies(); err != nil {
		return nil, nil, err
	}

	// Initialize all components
	if err := res.initializeAllComponents(); err != nil {
		return nil, nil, err
	}

	// Start all components
	if err := res.startAllComponents(ctx); err != nil {
		// If starting fails, try to stop what we've started
		res.stopAllComponents(ctx)
		return nil, nil, err
	}

	logger.Info("Container started",
		"components", len(res.components),
		"startup_ms", time.Since(startTime).Milliseconds())

	// Return context and shutdown function
	return res, func() {
		res.stopAllComponents(ctx)
	}, nil
}

// Start initializes the container and starts all components
func Start(ctx context.Context, block func(ContextBuilder)) (ApplicationContext, func()) {
	app, shutdown, err := New(ctx, DefaultConfig(), block)
	if err != nil {
		panic(err)
	}
	return app, shutdown
}
