# Core Concepts

GoBoot is built around a few core concepts that make it easy to build modular, flexible applications.

## Components

Components are the basic building blocks of a GoBoot application. A component is any struct that implements the `Component` interface:

```go
type Component interface {
    // Name returns the component name
    Name() string
    
    // Init initializes the component
    Init() error
}
```

GoBoot supports several specialized component types:

1. **Base Components**: These just need to be initialized once.
2. **Lifecycle Components**: These have a start and stop phase.
3. **Background Components**: These run in the background (like services).
4. **Scheduled Components**: These run on a schedule (like cron jobs).

## Container

The container is the central registry for all components and variables in your application. It provides:

- Component registration
- Component lifecycle management
- Dependency injection
- Configuration variable storage

The container is created when the application starts and manages all components throughout the application lifecycle.

## Application Context

The application context is the runtime representation of your application. It provides:

- Access to registered components
- Access to configuration variables
- The ability to query what components and variables exist

## Starters

Starters are a way to modularize your application configuration. A starter is responsible for:

- Registering components
- Setting up configuration
- Checking conditions to determine if it should run

Starters help you create reusable modules that can be shared across projects.

## Variables

Variables are named values stored in the container. They can be used for:

- Configuration settings
- Environmental values
- Sharing data between components

## Application Lifecycle

The GoBoot application lifecycle follows these steps:

1. **Context Creation**: A context is created with signal handling.
2. **Container Initialization**: The container is initialized.
3. **Component Registration**: Components are registered (via direct registration or starters).
4. **Component Initialization**: All components are initialized.
5. **Component Startup**: Background and lifecycle components are started.
6. **Running**: The application runs until a shutdown signal is received.
7. **Shutdown**: Components are stopped in reverse order of their startup.

## Dependency Injection

GoBoot provides dependency injection by:

1. Allowing components to access other components via the container
2. Supporting field injection using struct tags
3. Automatically resolving dependencies when components are initialized

This allows you to create loosely coupled components that work together.