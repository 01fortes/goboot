package boot

import (
	"GoBoot/pkg/container"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// Application represents a complete application
type Application struct {
	ctx       context.Context
	cancel    context.CancelFunc
	container container.ApplicationContext
	shutdown  func()
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

// New creates a new application with the given configuration
func New(block func(container.ContextBuilder)) *Application {
	// Create a context that can be cancelled
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Start the container
	slog.Info("Starting application")
	cont, shutdown := container.Start(ctx, block)

	return &Application{
		ctx:       ctx,
		cancel:    cancel,
		container: cont,
		shutdown:  shutdown,
	}
}
