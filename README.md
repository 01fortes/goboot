# GoBoot

GoBoot is a lightweight dependency injection and application bootstrapping framework for Go, inspired by Spring Boot. It provides a simple way to build modular applications with clean dependency management.

## Features

- Component-based architecture
- Dependency injection
- Configuration management with Spring Boot-style profiles
- YAML configuration with profile support (application.yml, application-dev.yml, etc.)
- Environment variable binding and overrides
- Lifecycle management (initialization, startup, shutdown)
- Modular "starters" for easy integration of common components
- Parallel component startup and shutdown

## Installation

```bash
go get github.com/01fortes/goboot
```

## Usage

### Basic Application

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
        
        // Use starters
        builder.RegisterStarter(myStarter)
    })
    
    // Run the application
    app.Run()
}
```

### Using Profile-Based Configuration

```go
package main

import (
    "github.com/01fortes/goboot/pkg/boot"
    "github.com/01fortes/goboot/pkg/container"
)

func main() {
    app := boot.New(func(builder container.ContextBuilder) {
        // Add YAML configuration with profile support
        builder.AddVariableLoader(container.ProfileYamlLoader{
            ConfigPath: "config", // Directory containing config files
            // Optional: explicitly set profiles instead of using GO_BOOT_ACTIVE_PROFILES env var
            // Profiles: []string{"dev", "local"},
        })
        
        // Add environment variable support (with prefix)
        builder.AddVariableLoader(container.EnvVariableLoader{
            Prefix: "APP_", // Only load environment variables with APP_ prefix 
        })
        
        // Register components
        builder.RegisterComponent(&MyService{})
    })
    
    // Run the application
    app.Run()
}

// Component that uses configuration values
type MyService struct {
    // Values can be injected from config files or environment variables
    ServerPort int    `inject:"variable:server.port"`
    AppName    string `inject:"variable:app.name"`
}

func (s *MyService) Name() string {
    return "myService"
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

### Composite Starter

```go
package mystarter

import (
    "github.com/01fortes/goboot/pkg/api/context"
    "github.com/01fortes/goboot/pkg/api/starter"
)

func PostgresStarter() starter.Starter {
    // Create database connection starter
    dbStarter := starter.NewStarter(
        "PostgresConnectionStarter",
        func(builder context.ContextBuilder) error {
            // Get configuration from properties
            url := builder.GetVariable("postgres.url")
            username := builder.GetVariable("postgres.username")
            password := builder.GetVariable("postgres.password")
            
            // Validate required properties
            if url == "" || username == "" || password == "" {
                return errors.New("missing required postgres configuration")
            }
            
            // Create and register datasource
            datasource := &PostgresDataSource{
                URL:      url,
                Username: username,
                Password: password,
            }
            
            return builder.RegisterComponent(datasource)
        },
    )
    
    // Create transaction manager starter
    txStarter := starter.NewStarter(
        "PostgresTransactionStarter",
        func(builder context.ContextBuilder) error {
            return builder.RegisterComponent(&PostgresTransactionManager{})
        },
    )
    
    // Combine them into a composite starter
    return starter.NewCompositeStarter(
        "PostgresStarter",
        dbStarter,
        txStarter,
    )
}
```

## Module Structure

The GoBoot framework is designed with a clean separation between core functionality and implementation:

- `pkg/boot/` - Main application bootstrapping
  - Application lifecycle management
  - Signal handling for graceful shutdown

- `pkg/container/` - Implementation of the dependency injection container
  - Component management
  - Lifecycle management
  - Dependency resolution
  - Variable registry
  - Starter interfaces

## Creating Starter Modules

To create a new starter module:

1. Create a new Go module
2. Add a dependency on `github.com/01fortes/goboot`
3. Implement the `container.Starter` or `container.ConditionalStarter` interface
4. Publish your module

Example module structure:
```
github.com/yourname/goboot-postgres-starter/
├── go.mod
├── starter.go
├── datasource.go
└── README.md
```

A typical starter module would:

1. Provide components that handle a specific functionality (e.g., database access, HTTP server)
2. Have conditional activation based on configuration properties
3. Register these components in the container
4. Properly handle dependencies on other components

## License

MIT