# Component Lifecycle

Each component in GoBoot goes through a well-defined lifecycle that ensures proper initialization, startup, and shutdown.

## Basic Component Lifecycle

The basic component lifecycle includes:

1. **Registration**: The component is registered with the container
2. **Initialization**: The `Init()` method is called

```go
type Component interface {
    // Name returns the component name
    Name() string
    
    // Init initializes the component
    Init() error
}
```

## Lifecycle Component

Lifecycle components extend the basic lifecycle with start and stop phases:

```go
type LifecycleComponent interface {
    Component
    
    // Start starts the component
    Start(ctx context.Context) error
    
    // Stop stops the component
    Stop(ctx context.Context) error
}
```

The lifecycle is:

1. **Registration**: The component is registered with the container
2. **Initialization**: The `Init()` method is called
3. **Startup**: The `Start(ctx)` method is called
4. **Running**: The component runs until shutdown
5. **Shutdown**: The `Stop(ctx)` method is called

## Background Component

Background components run in their own goroutine:

```go
type BackgroundComponent interface {
    Component
    
    // Start starts the background process
    Start(ctx context.Context) error
}
```

The lifecycle is:

1. **Registration**: The component is registered with the container
2. **Initialization**: The `Init()` method is called
3. **Startup**: The `Start(ctx)` method is called, which should start a goroutine
4. **Running**: The component runs in the background until shutdown
5. **Shutdown**: When the context is cancelled, the component should clean up

## Scheduled Component

Scheduled components run on a specified schedule:

```go
type ScheduledComponent interface {
    Component
    
    // GetSchedule returns the component's schedule (e.g., "*/5 * * * *")
    GetSchedule() string
    
    // Run executes the scheduled task
    Run(ctx context.Context) error
}
```

The lifecycle is:

1. **Registration**: The component is registered with the container
2. **Initialization**: The `Init()` method is called
3. **Scheduling**: The component's schedule is parsed
4. **Running**: The `Run(ctx)` method is called according to the schedule
5. **Shutdown**: When the context is cancelled, no more runs are scheduled

## Lifecycle Order

Components are initialized, started, and stopped in a specific order:

1. **Initialization**: Components are initialized in registration order
2. **Startup**: Components are started in registration order
3. **Shutdown**: Components are stopped in reverse registration order (last in, first out)

This ensures that dependencies are properly set up before components that need them, and dependencies are not shut down before the components that use them.

## Error Handling

If a component's `Init()` or `Start()` method returns an error, the application startup fails and the error is logged. All components that were already started will be stopped.

During shutdown, errors from `Stop()` methods are logged but do not prevent other components from being stopped.