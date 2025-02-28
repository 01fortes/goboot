package boot

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/01fortes/goboot/pkg/container"
	"github.com/01fortes/goboot/pkg/container/starter"
)

// Application represents a complete application
type Application struct {
	ctx               context.Context
	cancel            context.CancelFunc
	container         container.ApplicationContext
	shutdown          func()
	autoConfigEnabled bool
}

// Run starts the application and blocks until shutdown
func (a *Application) Run() {
	// Wait for termination signal
	<-a.ctx.Done()

	// Perform cleanup
	a.Shutdown()
}

// Shutdown gracefully stops the application
func (a *Application) Shutdown() {
	if a.shutdown != nil {
		a.shutdown()
		a.shutdown = nil
	}

	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

// GetContainer returns the application container
func (a *Application) GetContainer() container.ApplicationContext {
	return a.container
}

// DisableAutoConfiguration disables auto-configuration
func (a *Application) DisableAutoConfiguration() *Application {
	a.autoConfigEnabled = false
	return a
}

// New creates a new application with the given configuration
func New(block func(container.ContextBuilder)) *Application {
	// Create a context that can be cancelled
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Create container setup function with auto-configuration
	setupFunc := func(builder container.ContextBuilder) {
		// Call user setup function
		block(builder)
	}

	// Start the container
	slog.Info("Starting application")
	cont, shutdown := container.Start(ctx, setupFunc)

	return &Application{
		ctx:               ctx,
		cancel:            cancel,
		container:         cont,
		shutdown:          shutdown,
		autoConfigEnabled: true, // Enabled by default
	}
}

// AutoConfigurationRegistrar is the interface for registering auto-configurations
type AutoConfigurationRegistrar interface {
	// RegisterAutoConfiguration registers an auto-configuration
	RegisterAutoConfiguration(starter.AutoConfigurer)
}

// Global auto-configuration registry
var (
	autoConfigRegistry = make([]starter.AutoConfigurer, 0)
)

// RegisterAutoConfiguration registers an auto-configuration globally
func RegisterAutoConfiguration(config starter.AutoConfigurer) {
	autoConfigRegistry = append(autoConfigRegistry, config)
}

// EnableAutoConfiguration creates an auto-configuration for a component
func EnableAutoConfiguration(name string, targetType interface{}, properties *starter.Properties, configFunc func(container.ContextBuilder, interface{}) error) starter.AutoConfigurer {
	return starter.AutoConfigurer{
		Name:       name,
		Properties: properties,
		ConfigureFunc: func(builder container.ContextBuilder, config interface{}) error {
			return configFunc(builder, config)
		},
	}
}
