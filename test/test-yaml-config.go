package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/01fortes/goboot/pkg/boot"
	"github.com/01fortes/goboot/pkg/container"
)

// Database configuration structure
type DBConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Pool     struct {
		Min         int `yaml:"min"`
		Max         int `yaml:"max"`
		IdleTimeout int `yaml:"idle-timeout"`
	} `yaml:"pool"`
}

// ConfigService demonstrates using the VariableHelper
type ConfigService struct {
	vars *container.VariableHelper
}

func (c *ConfigService) Name() string {
	return "configService"
}

func (c *ConfigService) Init(ctx container.ApplicationContext) error {
	// Initialize the variable helper
	c.vars = container.NewVariableHelper(ctx)
	return nil
}

func (c *ConfigService) GetServerPort() int {
	return c.vars.GetInt("server.port", 8080)
}

func (c *ConfigService) GetLogLevel() string {
	return c.vars.GetString("logging.level", "info")
}

func (c *ConfigService) IsFeatureEnabled(feature string) bool {
	return c.vars.GetBool(fmt.Sprintf("feature-flags.%s", feature), false)
}

func (c *ConfigService) GetDatabaseConfig() (*DBConfig, error) {
	var config DBConfig
	err := c.vars.GetStruct("database", &config)
	return &config, err
}

// WebServer demonstrates using value injection
type WebServer struct {
	Port        int    `inject:"variable:server.port"`
	Host        string `inject:"variable:server.host"`
	LogLevel    string `inject:"variable:logging.level"`
	LogFormat   string `inject:"variable:logging.format"`
	NewUIActive bool   `inject:"variable:feature-flags.new-ui"`
}

func (s *WebServer) Name() string {
	return "webServer"
}

func (s *WebServer) Init(ctx container.ApplicationContext) error {
	slog.Info("WebServer configuration",
		"port", s.Port,
		"host", s.Host,
		"logLevel", s.LogLevel,
		"logFormat", s.LogFormat,
		"newUIActive", s.NewUIActive,
	)
	return nil
}

func main() {
	// Set up logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Print active profiles from env
	profilesEnv := os.Getenv("GO_BOOT_ACTIVE_PROFILES")
	if profilesEnv != "" {
		fmt.Printf("Active profiles from environment: %s\n\n", profilesEnv)
	} else {
		fmt.Println("No active profiles set in environment. Using default configuration only.\n")
		fmt.Println("To set profiles, use: export GO_BOOT_ACTIVE_PROFILES=dev,local\n")
	}

	// Create and start the application
	app := boot.New(func(builder container.ContextBuilder) {
		// Add profile YAML configuration
		builder.AddVariableLoader(container.ProfileYamlLoader{
			ConfigPath: "test/config",
			// Optional: explicitly set profiles
			// Profiles: []string{"dev"},
		})

		// Register components
		builder.RegisterComponent(&ConfigService{})
		builder.RegisterComponent(&WebServer{})
	})

	// Get the application context
	ctx := app.GetContainer()

	// Get the config service
	var configSvc *ConfigService
	if err := ctx.GetComponent(&configSvc); err != nil {
		slog.Error("Failed to get config service", "error", err)
		return
	}

	// Print some values using the ConfigService
	fmt.Println("===== Using ConfigService =====")
	fmt.Printf("Server Port: %d\n", configSvc.GetServerPort())
	fmt.Printf("Log Level: %s\n", configSvc.GetLogLevel())
	fmt.Printf("New UI Enabled: %v\n", configSvc.IsFeatureEnabled("new-ui"))
	fmt.Printf("Metrics Enabled: %v\n", configSvc.IsFeatureEnabled("metrics"))

	// Get and print database config
	dbConfig, err := configSvc.GetDatabaseConfig()
	if err != nil {
		slog.Error("Failed to get database config", "error", err)
	} else {
		fmt.Println("\n===== Database Configuration =====")
		fmt.Printf("URL: %s\n", dbConfig.URL)
		fmt.Printf("Username: %s\n", dbConfig.Username)
		fmt.Printf("Password: %s\n", dbConfig.Password)
		fmt.Printf("Pool Min: %d\n", dbConfig.Pool.Min)
		fmt.Printf("Pool Max: %d\n", dbConfig.Pool.Max)
		fmt.Printf("Pool Idle Timeout: %d seconds\n", dbConfig.Pool.IdleTimeout)
	}

	// Block for a moment to see output
	fmt.Println("\nPress Ctrl+C to exit...")
	app.Run()
}
