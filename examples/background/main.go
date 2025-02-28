package main

import (
	"GoBoot/pkg/boot"
	"GoBoot/pkg/container"
	"context"
	"log/slog"
	"os"
	"time"
)

func main() {
	// Set up logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Start the application
	app := boot.New(func(builder container.ContextBuilder) {
		// Register components
		builder.RegisterComponent(&ProcessorComponent{})
		builder.RegisterComponent(&ScheduledTaskComponent{})
	})

	// Run the application (blocks until terminated)
	app.Run()
}

// ProcessorComponent is a background component that runs continuously
type ProcessorComponent struct {
	container.ComponentBase
}

// NewProcessorComponent creates a new processor component
func NewProcessorComponent() *ProcessorComponent {
	return &ProcessorComponent{
		ComponentBase: container.NewComponentBase("processor"),
	}
}

// Init initializes the component
func (c *ProcessorComponent) Init(ctx container.ApplicationContext) {
	// Initialize resources here
	slog.Info("Processor component initialized")
}

// Start is called when the container starts the component
func (c *ProcessorComponent) Start(ctx context.Context) {
	slog.Info("Processor component starting")
}

// Stop is called when the container stops the component
func (c *ProcessorComponent) Stop(ctx context.Context) {
	slog.Info("Processor component stopping")
}

// Run is the main processing function - will be executed in a container-managed goroutine
func (c *ProcessorComponent) Run(ctx context.Context) {
	// This is where the main processing logic goes
	// No need to manage goroutines - the container does it for you

	for {
		select {
		case <-ctx.Done():
			// Clean exit when context is cancelled
			slog.Info("Processor received shutdown signal")
			return
		case <-time.After(1 * time.Second):
			// Simulate some work
			slog.Info("Processor doing work")
		}
	}
}

// ScheduledTaskComponent is a component that runs on a schedule
type ScheduledTaskComponent struct {
	container.ComponentBase
	counter int
}

// NewScheduledTaskComponent creates a new scheduled task component
func NewScheduledTaskComponent() *ScheduledTaskComponent {
	return &ScheduledTaskComponent{
		ComponentBase: container.NewComponentBase("scheduler"),
	}
}

// Init initializes the component
func (c *ScheduledTaskComponent) Init(ctx container.ApplicationContext) {
	// Initialize resources here
	slog.Info("Scheduled task component initialized")
}

// Start is called when the container starts
func (c *ScheduledTaskComponent) Start(ctx context.Context) {
	slog.Info("Scheduled task component starting")
}

// Stop is called when the container stops
func (c *ScheduledTaskComponent) Stop(ctx context.Context) {
	slog.Info("Scheduled task component stopping")
}

// GetSchedule returns the schedule for this component
func (c *ScheduledTaskComponent) GetSchedule() container.Schedule {
	return container.Schedule{
		Interval:     5 * time.Second,
		InitialDelay: 1 * time.Second,
		RunOnStartup: true,
	}
}

// Execute is called on each scheduled execution
func (c *ScheduledTaskComponent) Execute(ctx context.Context) {
	// This is executed according to the schedule
	// No need to manage timers or goroutines - the container does it for you
	c.counter++
	slog.Info("Scheduled task executed", "count", c.counter)
}
