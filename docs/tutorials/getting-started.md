# Getting Started with GoBoot

This tutorial will guide you through creating your first GoBoot application.

## Installation

First, add GoBoot to your project:

```bash
go get -u github.com/01fortes/goboot
```

## Creating a Simple Application

Let's create a simple "Hello World" application with GoBoot:

```go
package main

import (
    "fmt"
    "log"

    "github.com/01fortes/goboot/pkg/boot"
    "github.com/01fortes/goboot/pkg/container"
)

// HelloComponent is a simple component that prints a greeting
type HelloComponent struct {
    container.ComponentBase
    Name string
}

// Init initializes the component
func (c *HelloComponent) Init() error {
    fmt.Printf("Hello, %s!\n", c.Name)
    return nil
}

func main() {
    // Create the application
    app := boot.New(func(builder container.ContextBuilder) {
        // Register a variable
        builder.RegisterVariable("greeting.name", "World")
        
        // Register our component
        builder.RegisterComponent(&HelloComponent{
            Name: builder.GetVariable("greeting.name"),
        })
    })
    
    // Run the application
    log.Println("Starting application...")
    app.Run()
}
```

## Adding a Background Component

Let's enhance our application with a component that runs in the background:

```go
// TickerComponent is a background component that ticks every second
type TickerComponent struct {
    container.ComponentBase
    Interval string
}

// Start starts the background process
func (c *TickerComponent) Start(ctx context.Context) error {
    // Parse interval
    duration, err := time.ParseDuration(c.Interval)
    if err != nil {
        return err
    }
    
    // Create ticker
    ticker := time.NewTicker(duration)
    
    // Run in background
    go func() {
        for {
            select {
            case <-ticker.C:
                fmt.Println("Tick!")
            case <-ctx.Done():
                ticker.Stop()
                return
            }
        }
    }()
    
    return nil
}

// Now register this component in main()
app := boot.New(func(builder container.ContextBuilder) {
    // Register variables
    builder.RegisterVariable("greeting.name", "World")
    builder.RegisterVariable("ticker.interval", "1s")
    
    // Register components
    builder.RegisterComponent(&HelloComponent{
        Name: builder.GetVariable("greeting.name"),
    })
    
    builder.RegisterComponent(&TickerComponent{
        Interval: builder.GetVariable("ticker.interval"),
    })
})
```

## Next Steps

Now that you've created your first GoBoot application, check out these topics to learn more:

- [Component Lifecycle](../components/lifecycle.md)
- [Dependency Injection](../components/dependency-injection.md)
- [Creating Starters](../starters/creating-starters.md)
- [Configuration Management](../reference/configuration.md)