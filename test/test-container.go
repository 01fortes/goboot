package main

import (
	"GoBoot/pkg/boot"
	"GoBoot/pkg/container"
	"context"
	"log/slog"
	"time"
)

func main() {
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

func (t *TestComponent) Init(applicationContext container.ApplicationContext) {
	// Dependencies are auto-discovered by what we access
	t.str = applicationContext.GetVariable("some.test")
	slog.Info("TestComponent initialized", "str", t.str)
}

func (t *TestComponent) Name() string {
	return "test"
}

// TestComponent2 depends on TestComponent
type TestComponent2 struct {
	t *TestComponent
}

func (t *TestComponent2) Init(applicationContext container.ApplicationContext) {
	// Use type-based dependency injection
	// This will detect the TestComponent and set t.t with proper type
	var testComponent *TestComponent
	err := applicationContext.GetComponentByType(&testComponent)
	if err != nil {
		slog.Error("Failed to get test component by type", "error", err)
		return
	}

	// Set our field to the discovered component
	t.t = testComponent
	slog.Info("TestComponent2 initialized", "test_str", t.t.str)
}

func (t *TestComponent2) Name() string {
	return "test2"
}

// RunnableComponent is a lifecycle component that depends on TestComponent2
type RunnableComponent struct {
	t    *TestComponent2
	done chan struct{}
}

func (t *RunnableComponent) Init(applicationContext container.ApplicationContext) {
	// This will auto-register a dependency on "test2"
	test2, err := applicationContext.GetComponent("test2")
	if err != nil {
		panic(err)
	}

	// Check if we can safely cast
	if t2, ok := test2.(*TestComponent2); ok {
		t.t = t2
	}

	// Always initialize the channel
	t.done = make(chan struct{})
	slog.Info("RunnableComponent initialized")
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
