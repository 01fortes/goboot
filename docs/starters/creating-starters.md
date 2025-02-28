# Creating Starters

This guide walks you through creating your own GoBoot starters.

## Basic Starter

A basic starter is the simplest type of starter. It just registers components and doesn't have any conditions.

```go
package mystarter

import (
    "github.com/01fortes/goboot/pkg/container"
)

// MyStarter creates a new starter
func MyStarter() container.Starter {
    return container.NewStarter(
        "MyStarter",
        func(builder container.ContextBuilder) error {
            // Register components
            builder.RegisterComponent(&MyComponent{})
            
            // Register variables
            builder.RegisterVariable("my.setting", "default-value")
            
            return nil
        },
    )
}
```

## Using Configuration Variables

Starters often need to configure components based on variables:

```go
func DatabaseStarter() container.Starter {
    return container.NewStarter(
        "DatabaseStarter",
        func(builder container.ContextBuilder) error {
            // Get configuration
            url := builder.GetVariable("db.url")
            username := builder.GetVariable("db.username")
            password := builder.GetVariable("db.password")
            
            // Validate configuration
            if url == "" || username == "" || password == "" {
                return errors.New("missing required database configuration")
            }
            
            // Create and register components
            datasource := &DatabaseConnection{
                URL:      url,
                Username: username,
                Password: password,
            }
            
            return builder.RegisterComponent(datasource)
        },
    )
}
```

## Conditional Starters

Conditional starters only run if certain conditions are met:

```go
func OptionalFeatureStarter() container.ConditionalStarter {
    return container.NewConditionalStarter(
        "OptionalFeatureStarter",
        // Only run if "feature.enabled" is set to "true"
        container.PropertyCondition("feature.enabled", "true"),
        func(builder container.ContextBuilder) error {
            // Register components
            builder.RegisterComponent(&FeatureComponent{})
            return nil
        },
    )
}
```

GoBoot provides several built-in conditions:

- `PropertyCondition(property, value)` - Checks if a property has a specific value
- `PropertyExistsCondition(property)` - Checks if a property exists
- `ComponentExistsCondition(name)` - Checks if a component exists

You can also create custom conditions:

```go
// Custom condition function
func customCondition(ctx container.ApplicationContext) bool {
    // Check multiple conditions
    return ctx.GetVariable("feature.enabled") == "true" && 
           ctx.HasComponent("RequiredComponent")
}

// Use in a conditional starter
container.NewConditionalStarter(
    "CustomConditionStarter",
    customCondition,
    func(builder container.ContextBuilder) error {
        // ...
    },
)
```

## Composite Starters

Composite starters combine multiple starters into one:

```go
func PostgresStarter() container.Starter {
    // Create database connection starter
    dbStarter := container.NewStarter(
        "PostgresConnectionStarter",
        func(builder container.ContextBuilder) error {
            // Register datasource component
            return builder.RegisterComponent(&PostgresDataSource{
                URL:      builder.GetVariable("postgres.url"),
                Username: builder.GetVariable("postgres.username"),
                Password: builder.GetVariable("postgres.password"),
            })
        },
    )
    
    // Create transaction manager starter
    txStarter := container.NewStarter(
        "PostgresTransactionStarter",
        func(builder container.ContextBuilder) error {
            // Register transaction manager component
            return builder.RegisterComponent(&PostgresTransactionManager{})
        },
    )
    
    // Create repository starter
    repoStarter := container.NewStarter(
        "PostgresRepositoryStarter",
        func(builder container.ContextBuilder) error {
            // Register repository components
            builder.RegisterComponent(&UserRepository{})
            builder.RegisterComponent(&ProductRepository{})
            return nil
        },
    )
    
    // Combine them into a composite starter
    return container.NewCompositeStarter(
        "PostgresStarter",
        dbStarter,
        txStarter,
        repoStarter,
    )
}
```

## Best Practices

When creating starters, follow these best practices:

1. **Clear Names**: Give your starter a clear, descriptive name
2. **Validation**: Validate all required configuration
3. **Documentation**: Document all required variables
4. **Error Handling**: Return clear error messages if configuration is invalid
5. **Composability**: Make starters composable with other starters
6. **Conditional Activation**: Use conditions to avoid conflicts
7. **Default Values**: Provide sensible defaults where possible