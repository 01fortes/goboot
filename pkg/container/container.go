package container

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"
)

// Container is the central dependency container implementation
// First letter is capitalized to make it accessible to other files in the package
type container struct {
	config      *Config
	logger      *slog.Logger
	startupTime time.Time

	// Core subsystems
	componentRegistry  ComponentRegistry
	variableRegistry   VariableRegistry
	metricsCollector   MetricsCollector
	dependencyResolver DependencyResolver
	componentInit      ComponentInitializer
	lifecycleManager   ComponentLifecycleManager

	// Factory and starter support
	starters         []Starter
	variablesLoaders []VariableLoader
	factories        []Factory
}

// RegisterComponent adds a component to the container
func (c *container) RegisterComponent(component Component) error {
	return c.componentRegistry.Register(component)
}

// RegisterVariable adds a variable to the container
func (c *container) RegisterVariable(name string, value interface{}) {
	c.variableRegistry.Register(name, value)
}

// AddVariableLoader adds a variable loader
func (c *container) AddVariableLoader(loader VariableLoader) {
	c.variablesLoaders = append(c.variablesLoaders, loader)
}

// RegisterFactory adds a component factory
func (c *container) RegisterFactory(factory Factory) {
	c.factories = append(c.factories, factory)
}

// RegisterStarter adds a starter to the container
func (c *container) RegisterStarter(s interface{}) {
	// Support both our internal Starter and the core.Starter
	var starter Starter

	switch st := s.(type) {
	case Starter:
		starter = st
	default:
		c.logger.Warn("Unknown starter type, ignoring", "type", reflect.TypeOf(s))
		return
	}

	c.starters = append(c.starters, starter)
}

// HasComponent checks if a component exists
func (c *container) HasComponent(name string) bool {
	return c.componentRegistry.Has(name)
}

// GetComponentNames returns all registered component names
func (c *container) GetComponentNames() []string {
	return c.componentRegistry.GetNames()
}

// GetComponentByName returns a component by name
func (c *container) GetComponentByName(name string) (Component, error) {
	return c.componentRegistry.Get(name)
}

// GetComponent finds a component matching the type of the provided pointer and sets the pointer
func (c *container) GetComponent(target interface{}) error {
	// Get target type
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return ErrorWithCode("TARGET_NOT_POINTER", "target must be a pointer")
	}

	// Get the element type
	elemType := targetType.Elem()
	targetValue := reflect.ValueOf(target).Elem()

	// First try exact type match
	components := c.componentRegistry.GetAll()
	for _, comp := range components {
		compType := reflect.TypeOf(comp)
		if compType == elemType || compType == reflect.PtrTo(elemType) {
			// Found a match, set the pointer
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
	for _, comp := range components {
		compType := reflect.TypeOf(comp)
		if compType.AssignableTo(elemType) {
			// Found a match, set the pointer
			targetValue.Set(reflect.ValueOf(comp))
			return nil
		}
	}

	return ErrorWithCode("COMPONENT_TYPE_NOT_FOUND", "no component found matching type %v", elemType)
}

// GetVariable returns a variable by name
func (c *container) GetVariable(name string) string {
	return c.variableRegistry.GetString(name)
}

// GetVariableRaw returns the raw variable value (not converted to string)
func (c *container) GetVariableRaw(name string) interface{} {
	return c.variableRegistry.Get(name)
}

// GetMetrics returns metrics for all components
func (c *container) GetMetrics() map[string]*ComponentMetrics {
	return c.metricsCollector.GetMetrics()
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

	// Initialize the container components
	compRegistry := newComponentRegistry(logger)
	varRegistry := newVariableRegistry(logger)
	metricsCollector := newMetricsCollector(cfg.EnableMetrics)

	res := &container{
		config:            cfg,
		logger:            logger,
		startupTime:       startTime,
		componentRegistry: compRegistry,
		variableRegistry:  varRegistry,
		metricsCollector:  metricsCollector,
		variablesLoaders:  cfg.DefaultVariableLoaders,
		starters:          cfg.DefaultStarters,
		factories:         []Factory{},
	}

	// Register components and variables
	block(res)

	// Set up dependency resolver and initializer
	res.dependencyResolver = newDependencyResolver(res, compRegistry, metricsCollector, logger)

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
	if err := res.dependencyResolver.DiscoverDependencies(); err != nil {
		return nil, nil, err
	}

	// Validate dependencies
	if err := res.dependencyResolver.ValidateDependencies(); err != nil {
		return nil, nil, err
	}

	// Set up component initializer
	res.componentInit = newComponentInitializer(res, compRegistry, res.dependencyResolver, metricsCollector, logger)

	// Initialize all components
	if err := res.componentInit.InitializeAll(); err != nil {
		return nil, nil, err
	}

	// Set up lifecycle manager with initialization order
	res.lifecycleManager = newLifecycleManager(compRegistry, res.componentInit.GetInitOrder(), metricsCollector, logger)

	// Start all components
	if err := res.lifecycleManager.StartAll(ctx); err != nil {
		// If starting fails, try to stop what we've started
		res.lifecycleManager.StopAll(ctx)
		return nil, nil, err
	}

	logger.Info("Container started",
		"components", len(compRegistry.GetAll()),
		"startup_ms", time.Since(startTime).Milliseconds())

	// Return context and shutdown function
	return res, func() {
		res.lifecycleManager.StopAll(ctx)
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
