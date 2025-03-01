package main

import (
	"context"
	"github.com/01fortes/goboot/pkg/boot"
	"github.com/01fortes/goboot/pkg/container"
	"log/slog"
	"time"
)

func mains() {
	// Create and start the application
	app := boot.New(func(cnt container.ContextBuilder) {
		// Register variables
		cnt.RegisterVariable("some.test", "lloololo")

		// Order doesn't matter and dependencies are auto-discovered!
		cnt.RegisterComponent(&RunnableComponent{}) // Will access TestComponent2
		cnt.RegisterComponent(&TestComponent2{})    // Will access TestComponent
		cnt.RegisterComponent(&TestComponent{})     // No dependencies
	})

	// Run the application
	app.Run()
}

// TestComponent has no dependencies
type TestComponent struct {
	str string
}

func (t *TestComponent) Init(ctx container.ApplicationContext) error {
	// Dependencies are auto-discovered by what we access
	t.str = ctx.GetVariable("some.test")
	slog.Info("TestComponent initialized", "str", t.str)
	return nil
}

func (t *TestComponent) Name() string {
	return "test"
}

// TestComponent2 depends on TestComponent
type TestComponent2 struct {
	t *TestComponent
}

func (t *TestComponent2) Init(ctx container.ApplicationContext) error {
	// Use type-based dependency injection - cleaner API without mentioning "type"
	var testComponent *TestComponent
	if err := ctx.GetComponent(&testComponent); err != nil {
		slog.Error("Failed to get test component", "error", err)
		return err
	}

	// Set our field to the discovered component
	t.t = testComponent
	slog.Info("TestComponent2 initialized", "test_str", t.t.str)
	return nil
}

func (t *TestComponent2) Name() string {
	return "test2"
}

// RunnableComponent is a lifecycle component that depends on TestComponent2
type RunnableComponent struct {
	t    *TestComponent2
	done chan struct{}
}

func (t *RunnableComponent) Init(ctx container.ApplicationContext) error {
	// This will auto-register a dependency on "test2" using cleaner type-based API
	var test2 *TestComponent2
	if err := ctx.GetComponent(&test2); err != nil {
		slog.Error("Failed to get test2 component", "error", err)
		return err
	}

	// Set our field directly (no cast needed)
	t.t = test2

	// Always initialize the channel
	t.done = make(chan struct{})
	slog.Info("RunnableComponent initialized")
	return nil
}

func (t *RunnableComponent) Name() string {
	return "runnable"
}

func (t *RunnableComponent) Start(ctx context.Context) {
	slog.Info("RunnableComponent started")

	// Start a goroutine that logs periodically
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Context cancelled, stopping RunnableComponent")
				return
			case <-t.done:
				slog.Info("Done signal received, stopping RunnableComponent")
				return
			case <-ticker.C:
				// Check for nil pointers before accessing
				if t.t != nil && t.t.t != nil {
					slog.Info("RunnableComponent running", "test_name", t.t.t.Name())
				} else {
					slog.Info("RunnableComponent running")
				}
			}
		}
	}()
}

func (t *RunnableComponent) Stop(ctx context.Context) {
	slog.Info("RunnableComponent stopping")
	close(t.done)

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)
	slog.Info("RunnableComponent stopped")
}
