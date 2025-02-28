# GoBoot Documentation

GoBoot is a lightweight dependency injection and application bootstrapping framework for Go, inspired by Spring Boot.

## Table of Contents

1. [Getting Started](./tutorials/getting-started.md)
2. [Core Concepts](./reference/core-concepts.md)
3. [Component System](./components/index.md)
4. [Starter Modules](./starters/index.md)
5. [Examples](./examples/index.md)

## Quick Start

```go
package main

import (
    "github.com/01fortes/goboot/pkg/boot"
    "github.com/01fortes/goboot/pkg/container"
)

func main() {
    app := boot.New(func(builder container.ContextBuilder) {
        // Register components and configuration
        builder.RegisterVariable("app.name", "MyApp")
        builder.RegisterComponent(&MyComponent{})
    })
    
    // Run the application
    app.Run()
}
```

## Features

- Component-based architecture
- Dependency injection
- Configuration management
- Lifecycle management (initialization, startup, shutdown)
- Modular "starters" for easy integration of common components
- Parallel component startup and shutdown