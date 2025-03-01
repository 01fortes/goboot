# Using YAML Profile Configuration

GoBoot supports Spring Boot style profile-based configuration using YAML files. This allows you to:
- Define default configuration in `application.yml`
- Override specific values for different environments using profile-specific files like `application-dev.yml`, `application-prod.yml`, etc.
- Control which profiles are active via the `GO_BOOT_ACTIVE_PROFILES` environment variable

## Basic Usage

1. Create your configuration files:

```
config/
  application.yml           # Default configuration
  application-dev.yml       # Development overrides
  application-prod.yml      # Production overrides
```

2. Use the `ProfileYamlLoader` in your application setup:

```go
func main() {
    app := boot.New(func(builder container.ContextBuilder) {
        // Set up variable loading
        builder.AddVariableLoader(container.ProfileYamlLoader{
            ConfigPath: "config", // Directory containing your config files
        })
        
        // Register your components
        builder.RegisterComponent("myService", &MyService{})
    })
    
    app.Run()
}
```

3. Set active profiles:

```bash
# Set active profiles via environment variable
export GO_BOOT_ACTIVE_PROFILES=dev,local

# Or specify them directly in code
builder.AddVariableLoader(container.ProfileYamlLoader{
    ConfigPath: "config",
    Profiles: []string{"dev", "local"},
})
```

## Example Configuration Files

**application.yml** (default configuration)
```yaml
server:
  port: 8080
  
database:
  url: jdbc:postgresql://localhost:5432/myapp
  username: postgres
  password: secret
  
logging:
  level: info
```

**application-dev.yml** (development overrides)
```yaml
database:
  username: dev-user
  password: dev-password
  
logging:
  level: debug
```

**application-prod.yml** (production overrides)
```yaml
server:
  port: 80
  
database:
  url: jdbc:postgresql://db.production:5432/myapp
  username: prod-user
  password: ${DB_PASSWORD} # Use environment variable
  
logging:
  level: warn
```

## Accessing Configuration Values

### Using Struct Injection

You can inject configuration values directly into your components:

```go
type MyService struct {
    ServerPort int    `inject:"variable:server.port"`
    DbUrl      string `inject:"variable:database.url"`
    LogLevel   string `inject:"variable:logging.level"`
}

func (s *MyService) Name() string {
    return "myService"
}
```

### Using the VariableHelper

For more flexibility, GoBoot provides a `VariableHelper` that makes it easy to access typed configuration values:

```go
type ConfigService struct {
    ctx container.ApplicationContext
    vars *container.VariableHelper
}

func NewConfigService(ctx container.ApplicationContext) *ConfigService {
    return &ConfigService{
        ctx: ctx,
        vars: container.NewVariableHelper(ctx),
    }
}

func (s *ConfigService) Name() string {
    return "configService"
}

func (s *ConfigService) GetServerPort() int {
    // Get an int with a default value of 8080
    return s.vars.GetInt("server.port", 8080)
}

func (s *ConfigService) IsFeatureEnabled(feature string) bool {
    // Get a boolean with a default value of false
    return s.vars.GetBool(fmt.Sprintf("features.%s.enabled", feature), false)
}

func (s *ConfigService) GetDatabaseConfig() (*DatabaseConfig, error) {
    // Map a configuration section to a struct
    var config DatabaseConfig
    err := s.vars.GetStruct("database", &config)
    return &config, err
}

type DatabaseConfig struct {
    URL      string `yaml:"url"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
    Pool     struct {
        MaxSize      int  `yaml:"maxSize"`
        MinSize      int  `yaml:"minSize"`
        IdleTimeout  int  `yaml:"idleTimeout"`
        EnableMetrics bool `yaml:"enableMetrics"`
    } `yaml:"pool"`
}
```

## Complete Example

Here's a complete example showing how to set up and use profile-based configuration:

```go
package main

import (
	"fmt"
	
	"github.com/01fortes/goboot/pkg/boot"
	"github.com/01fortes/goboot/pkg/container"
)

func main() {
	// Create and run the application
	app := boot.New(func(builder container.ContextBuilder) {
		// Load configuration with profiles
		builder.AddVariableLoader(container.ProfileYamlLoader{
			ConfigPath: "config",
			// Optional: specify profiles directly instead of using environment variable
			// Profiles: []string{"dev", "local"},
		})
		
		// Register configuration service
		builder.RegisterComponent(&ConfigService{})
		
		// Register application services that depend on configuration
		builder.RegisterComponent(&UserService{})
		builder.RegisterComponent(&WebServer{})
	})
	
	// Get the application context
	ctx := app.GetContainer()
	
	// Get the config service to use configuration
	var configSvc *ConfigService
	_ = ctx.GetComponent(&configSvc)
	
	// Print some configuration values
	fmt.Printf("Server running on port: %d\n", configSvc.GetServerPort())
	fmt.Printf("Log level: %s\n", configSvc.GetLogLevel())
	
	// Run the application (blocks until shutdown)
	app.Run()
}

// ConfigService provides typed access to configuration
type ConfigService struct {
	ctx  container.ApplicationContext
	vars *container.VariableHelper
}

func (s *ConfigService) Name() string {
	return "configService"
}

// Init is called by the container after dependency injection
func (s *ConfigService) Init(ctx container.ApplicationContext) error {
	s.ctx = ctx
	s.vars = container.NewVariableHelper(ctx)
	return nil
}

func (s *ConfigService) GetServerPort() int {
	return s.vars.GetInt("server.port", 8080)
}

func (s *ConfigService) GetLogLevel() string {
	return s.vars.GetString("logging.level", "info")
}

func (s *ConfigService) GetDatabaseConfig() (*DatabaseConfig, error) {
	var config DatabaseConfig
	err := s.vars.GetStruct("database", &config)
	return &config, err
}

type DatabaseConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Other components using the configuration
type UserService struct {
	Config *ConfigService `inject:"configService"`
}

func (s *UserService) Name() string {
	return "userService"
}

type WebServer struct {
	ServerPort int    `inject:"variable:server.port"`
	LogLevel   string `inject:"variable:logging.level"`
}

func (s *WebServer) Name() string {
	return "webServer"
}
```

## Priority Order

Configuration values are loaded and merged in this order:
1. Default values from `application.yml`
2. Profile values from `application-{profile}.yml` files (in the order profiles are specified)
3. Environment variables (if using `EnvVariableLoader` as well)

Later sources override earlier ones, so profile-specific values take precedence over defaults.

## Environment Variable Integration

For a complete solution, you can add environment variable support on top of YAML files:

```go
// Add profile YAML loader first
builder.AddVariableLoader(container.ProfileYamlLoader{
    ConfigPath: "config",
})

// Add environment variable loader second (overrides YAML values)
builder.AddVariableLoader(container.EnvVariableLoader{
    Prefix: "APP_", // Only load environment variables with this prefix
})
```

This way, you can override any configuration value using environment variables, following the convention:
- Convert dots to underscores: `server.port` -> `APP_SERVER_PORT`
- Convert to uppercase: `APP_SERVER_PORT`