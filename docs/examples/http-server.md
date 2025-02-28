# HTTP Server Example

This example shows how to build an HTTP server with GoBoot.

## Basic HTTP Server

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "strconv"

    "github.com/01fortes/goboot/pkg/boot"
    "github.com/01fortes/goboot/pkg/container"
)

// HTTPServerComponent is a background component that runs an HTTP server
type HTTPServerComponent struct {
    container.ComponentBase
    Port    int
    server  *http.Server
    handler http.Handler
}

// Init initializes the HTTP server
func (c *HTTPServerComponent) Init() error {
    // Create a new server
    c.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", c.Port),
        Handler: c.handler,
    }
    
    return nil
}

// Start starts the HTTP server
func (c *HTTPServerComponent) Start(ctx context.Context) error {
    // Start the server in a goroutine
    go func() {
        log.Printf("Starting HTTP server on port %d", c.Port)
        if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("HTTP server error: %v", err)
        }
    }()
    
    return nil
}

// Stop stops the HTTP server
func (c *HTTPServerComponent) Stop(ctx context.Context) error {
    log.Println("Shutting down HTTP server")
    return c.server.Shutdown(ctx)
}

// Name returns the component name
func (c *HTTPServerComponent) Name() string {
    return "HTTPServer"
}

// HandlerComponent is a component that provides an HTTP handler
type HandlerComponent struct {
    container.ComponentBase
}

// GetHandler returns the HTTP handler
func (c *HandlerComponent) GetHandler() http.Handler {
    mux := http.NewServeMux()
    
    // Register routes
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, World!")
    })
    
    mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Users API")
    })
    
    return mux
}

// Name returns the component name
func (c *HandlerComponent) Name() string {
    return "HandlerComponent"
}

func main() {
    app := boot.New(func(builder container.ContextBuilder) {
        // Register variables
        builder.RegisterVariable("http.port", "8080")
        
        // Create handler component
        handler := &HandlerComponent{}
        
        // Register the handler component
        builder.RegisterComponent(handler)
        
        // Get port from configuration
        portStr := builder.GetVariable("http.port")
        port, err := strconv.Atoi(portStr)
        if err != nil {
            port = 8080 // Default port
        }
        
        // Create and register the HTTP server component
        server := &HTTPServerComponent{
            Port:    port,
            handler: handler.GetHandler(),
        }
        
        builder.RegisterComponent(server)
    })
    
    // Run the application
    log.Println("Starting application...")
    app.Run()
}
```

## HTTP Server Starter

You can also create a starter for the HTTP server:

```go
package httpstarter

import (
    "github.com/01fortes/goboot/pkg/container"
)

// HTTPServerStarter creates a new HTTP server starter
func HTTPServerStarter() container.Starter {
    return container.NewStarter(
        "HTTPServerStarter",
        func(builder container.ContextBuilder) error {
            // Register default variables
            builder.RegisterVariable("http.port", "8080")
            
            // Create handler component
            handler := &HandlerComponent{}
            
            // Register the handler component
            builder.RegisterComponent(handler)
            
            // Get port from configuration
            portStr := builder.GetVariable("http.port")
            port, err := strconv.Atoi(portStr)
            if err != nil {
                port = 8080 // Default port
            }
            
            // Create and register the HTTP server component
            server := &HTTPServerComponent{
                Port:    port,
                handler: handler.GetHandler(),
            }
            
            return builder.RegisterComponent(server)
        },
    )
}
```

Now you can use this starter in your application:

```go
package main

import (
    "log"

    "github.com/01fortes/goboot/pkg/boot"
    "github.com/yourusername/httpstarter"
)

func main() {
    app := boot.New(func(builder container.ContextBuilder) {
        // Override default port
        builder.RegisterVariable("http.port", "9090")
        
        // Register the HTTP server starter
        builder.RegisterStarter(httpstarter.HTTPServerStarter())
    })
    
    // Run the application
    log.Println("Starting application...")
    app.Run()
}
```

## Using a Conditional HTTP Server

You can make the HTTP server conditional:

```go
func ConditionalHTTPServerStarter() container.ConditionalStarter {
    return container.NewConditionalStarter(
        "ConditionalHTTPServerStarter",
        // Only start if http.enabled is true
        container.PropertyCondition("http.enabled", "true"),
        func(builder container.ContextBuilder) error {
            // Register the HTTP server components
            // ...
            return nil
        },
    )
}
```

Now the HTTP server will only start if `http.enabled` is set to `true`.