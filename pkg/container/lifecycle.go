package container

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ComponentLifecycleManager handles component lifecycle (start/stop)
type ComponentLifecycleManager interface {
	StartAll(ctx context.Context) error
	StopAll(ctx context.Context)
}

// defaultLifecycleManager implements ComponentLifecycleManager
type defaultLifecycleManager struct {
	registry  ComponentRegistry
	initOrder []string
	metrics   MetricsCollector
	logger    *slog.Logger
}

func newLifecycleManager(registry ComponentRegistry, initOrder []string, metrics MetricsCollector, logger *slog.Logger) *defaultLifecycleManager {
	return &defaultLifecycleManager{
		registry:  registry,
		initOrder: initOrder,
		metrics:   metrics,
		logger:    logger,
	}
}

func (m *defaultLifecycleManager) StartAll(ctx context.Context) error {
	// Start components in dependency order
	m.logger.Info("Starting components")

	// Use a WaitGroup to track all component startups
	var wg sync.WaitGroup
	// Channel to collect any errors from goroutines
	errChan := make(chan error, len(m.initOrder))

	for _, name := range m.initOrder {
		component, err := m.registry.Get(name)
		if err != nil {
			return err
		}

		// Start lifecycle components
		if lifecycle, ok := component.(LifecycleComponent); ok {
			m.logger.Debug("Starting component", "name", name)

			// Start each component in its own goroutine
			wg.Add(1)
			go func(comp LifecycleComponent, compName string) {
				defer wg.Done()

				start := time.Now()

				// Capture panics in component startup
				defer func() {
					if r := recover(); r != nil {
						errChan <- fmt.Errorf("panic in component %s startup: %v", compName, r)
					}
				}()

				comp.Start(ctx)
				duration := time.Since(start)

				m.metrics.RecordStartDuration(compName, duration)

				m.logger.Info("Component started",
					"name", compName,
					"time_ms", duration.Milliseconds())

				// Start background components in managed goroutines
				if background, ok := comp.(BackgroundComponent); ok {
					m.startBackgroundComponent(ctx, background, compName)
				}

				// Start scheduled components with a managed timer
				if scheduled, ok := comp.(ScheduledComponent); ok {
					m.startScheduledComponent(ctx, scheduled, compName)
				}
			}(lifecycle, name)
		}
	}

	// Use a goroutine to wait for all components to start and close the error channel
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for any errors that occurred during startup
	var startupErr error
	for err := range errChan {
		if err != nil {
			m.logger.Error("Error starting component", "error", err)
			if startupErr == nil {
				startupErr = err
			} else {
				startupErr = fmt.Errorf("%v; %w", startupErr, err)
			}
		}
	}

	return startupErr
}

func (m *defaultLifecycleManager) startBackgroundComponent(ctx context.Context, component BackgroundComponent, name string) {
	m.logger.Debug("Starting background component", "name", name)

	// Launch the component in a goroutine
	go func(bgComponent BackgroundComponent, componentName string) {
		m.logger.Info("Background component running", "name", componentName)

		// Run the component's main logic
		bgComponent.Run(ctx)

		m.logger.Info("Background component completed", "name", componentName)
	}(component, name)
}

func (m *defaultLifecycleManager) startScheduledComponent(ctx context.Context, component ScheduledComponent, name string) {
	m.logger.Debug("Starting scheduled component", "name", name)

	// Get schedule
	schedule := component.GetSchedule()

	// Launch the component's scheduler in a goroutine
	go func(schedComponent ScheduledComponent, componentName string, sched Schedule) {
		// Run immediately if configured
		if sched.RunOnStartup {
			m.logger.Debug("Executing scheduled component on startup", "name", componentName)
			schedComponent.Execute(ctx)
		}

		// Wait for initial delay
		if sched.InitialDelay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sched.InitialDelay):
				// Continue after delay
			}
		}

		// Set up ticker for recurring execution
		ticker := time.NewTicker(sched.Interval)
		defer ticker.Stop()

		m.logger.Info("Scheduled component running",
			"name", componentName,
			"interval", sched.Interval.String())

		// Run the scheduled executions
		for {
			select {
			case <-ctx.Done():
				m.logger.Info("Scheduled component stopping due to context cancellation",
					"name", componentName)
				return
			case <-ticker.C:
				m.logger.Debug("Executing scheduled component", "name", componentName)
				schedComponent.Execute(ctx)
			}
		}
	}(component, name, schedule)
}

func (m *defaultLifecycleManager) StopAll(ctx context.Context) {
	m.logger.Info("Stopping components")

	// Create a batch system to control concurrent shutdowns
	// This allows us to shut down in reverse init order but in batches
	// so that dependent components don't shut down before their dependencies
	batchSize := 5 // Shutdown 5 components at a time

	// Group components by initialization order (in reverse)
	totalComponents := len(m.initOrder)
	batches := (totalComponents + batchSize - 1) / batchSize // Ceiling division

	for batch := 0; batch < batches; batch++ {
		startIdx := totalComponents - (batch * batchSize) - 1
		endIdx := max(totalComponents-((batch+1)*batchSize), 0)

		// Process each batch
		batchWg := sync.WaitGroup{}

		// Start shutdown for components in this batch
		for i := startIdx; i >= endIdx; i-- {
			if i < 0 || i >= totalComponents {
				continue
			}

			name := m.initOrder[i]
			component, err := m.registry.Get(name)
			if err != nil {
				m.logger.Error("Error getting component during shutdown",
					"name", name,
					"error", err)
				continue
			}

			if lifecycle, ok := component.(LifecycleComponent); ok {
				batchWg.Add(1)

				// Stop each component in its own goroutine
				go func(comp LifecycleComponent, compName string) {
					defer batchWg.Done()

					m.logger.Debug("Stopping component", "name", compName)

					// Capture panics in component shutdown
					defer func() {
						if r := recover(); r != nil {
							m.logger.Error("Panic in component shutdown",
								"name", compName,
								"error", r)
						}
					}()

					start := time.Now()
					comp.Stop(ctx)
					duration := time.Since(start)

					m.metrics.RecordStopDuration(compName, duration)

					m.logger.Info("Component stopped",
						"name", compName,
						"time_ms", duration.Milliseconds())
				}(lifecycle, name)
			}
		}

		// Wait for all components in this batch to stop before moving to the next batch
		batchWg.Wait()
	}
}

// Helper function for Go versions before 1.21 which don't have max in the std lib
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
