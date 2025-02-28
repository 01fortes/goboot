# GoBoot

GoBoot is a lightweight dependency injection and application bootstrapping framework for Go, inspired by Spring Boot. It provides a simple way to build modular applications with clean dependency management.

## Features

- Component-based architecture
- Dependency injection
- Configuration management
- Lifecycle management (initialization, startup, shutdown)
- Modular "starters" for easy integration of common components
- Parallel component startup and shutdown

## Installation

```bash
go get github.com/01fortes/goboot
```

## Usage

```go
package main

import (
    "github.com/01fortes/goboot/pkg/boot"
    "github.com/01fortes/goboot/pkg/api/component"
)

func main() {
    app := boot.New(func(builder ContextBuilder) {
        // Register components and configuration
        builder.RegisterVariable("app.name", "MyApp")
        builder.RegisterComponent(&MyComponent{})
        
        // Use starters
        builder.RegisterStarter(myStarter)
    })
    
    // Run the application
    app.Run()
}
```

## Creating Starters

Starters are a powerful feature of GoBoot that allow you to create reusable modules. Each starter can be published as a separate Go module that depends only on the GoBoot API.

### Basic Starter

```go
package mystarter

import (
    "github.com/01fortes/goboot/pkg/api/context"
    "github.com/01fortes/goboot/pkg/api/starter"
)

func MyStarter() starter.Starter {
    return starter.NewStarter(
        "MyStarter",
        func(builder context.ContextBuilder) error {
            // Register components
            builder.RegisterComponent(&MyComponent{})
            return nil
        },
    )
}
```

### Conditional Starter

```go
package mystarter

import (
    "github.com/01fortes/goboot/pkg/api/context"
    "github.com/01fortes/goboot/pkg/api/starter"
)

func MyConditionalStarter() starter.ConditionalStarter {
    return starter.NewConditionalStarter(
        "MyConditionalStarter",
        // Only run if "my.feature.enabled" is set to "true"
        starter.PropertyCondition("my.feature.enabled", "true"),
        func(builder context.ContextBuilder) error {
            // Register components
            builder.RegisterComponent(&MyComponent{})
            return nil
        },
    )
}
```

### Using the Template

```go
package mystarter

import (
    "github.com/01fortes/goboot/pkg/api/context"
    "github.com/01fortes/goboot/pkg/api/starter"
)

func PostgresStarter() starter.Starter {
    template := &starter.StarterTemplate{
        Name:               "PostgresStarter",
        PropertyPrefix:     "postgres.",
        RequiredProperties: []string{"url", "username", "password"},
        ComponentsFunc: func(builder context.ContextBuilder, config map[string]string) error {
            // Create and register components
            datasource := &PostgresDataSource{
                URL:      config["url"],
                Username: config["username"],
                Password: config["password"],
            }
            
            return builder.RegisterComponent(datasource)
        },
    }
    
    return template.Create()
}
```

## Module Structure

The GoBoot framework is designed with a clean separation between API and implementation:

- `pkg/api/` - Public API that starters depend on
  - `context/` - Core context interfaces
  - `component/` - Component interfaces
  - `starter/` - Starter interfaces and utilities
  - `config/` - Configuration interfaces
  - `errors/` - Error types

- `pkg/boot/` - Main application bootstrapping
- `pkg/container/` - Implementation of the dependency injection container

## Creating Starter Modules

To create a new starter module:

1. Create a new Go module
2. Add a dependency on `github.com/01fortes/goboot`
3. Import only from the `pkg/api/` packages
4. Implement the `starter.Starter` or `starter.ConditionalStarter` interface
5. Publish your module

Example module structure:
```
github.com/yourname/goboot-postgres-starter/
├── go.mod
├── starter.go
├── datasource.go
└── README.md
```

## License

MIT