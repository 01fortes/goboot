# Dependency Injection

GoBoot provides a simple but powerful dependency injection system that allows components to depend on each other.

## Basic Dependency Injection

The most basic form of dependency injection in GoBoot is manual injection:

```go
type ServiceComponent struct {
    container.ComponentBase
    Repository *RepositoryComponent
}

func main() {
    app := boot.New(func(builder container.ContextBuilder) {
        // Create components
        repo := &RepositoryComponent{}
        service := &ServiceComponent{
            Repository: repo,
        }
        
        // Register components
        builder.RegisterComponent(repo)
        builder.RegisterComponent(service)
    })
    
    app.Run()
}
```

## Container-Based Injection

You can also use the container to look up dependencies:

```go
type ServiceComponent struct {
    container.ComponentBase
    repository *RepositoryComponent
}

func (s *ServiceComponent) Init() error {
    // Get the repository component from the container
    repo, err := s.Container.GetComponent("RepositoryComponent")
    if err != nil {
        return err
    }
    
    // Cast to the correct type
    s.repository = repo.(*RepositoryComponent)
    return nil
}
```

## Automatic Injection

GoBoot can automatically inject dependencies during component initialization:

```go
type ServiceComponent struct {
    container.ComponentBase
    
    // Use the Inject tag to automatically inject dependencies
    Repository *RepositoryComponent `inject:""`
}

// No need to manually look up the dependency in Init()
```

## Interface-Based Injection

You can inject components based on interfaces they implement:

```go
type Repository interface {
    FindById(id string) (interface{}, error)
}

type ServiceComponent struct {
    container.ComponentBase
    
    // Inject any component that implements the Repository interface
    Repo Repository `inject:""`
}
```

## Dependency Resolution Order

The container resolves dependencies in the following order:

1. Exact type matches by name
2. Exact type matches by type
3. Interface matches by name
4. Interface matches by type

## Circular Dependencies

GoBoot detects circular dependencies during initialization and returns an error. To resolve circular dependencies:

1. Use setter injection instead of constructor injection
2. Extract the circular dependency into a separate component
3. Use events to communicate between components instead of direct dependencies

## Best Practices

1. **Prefer constructor injection**: Inject dependencies when creating components
2. **Use interfaces**: Depend on interfaces rather than concrete implementations
3. **Single responsibility**: Keep components focused on a single responsibility
4. **Minimize dependencies**: Keep the number of dependencies manageable
5. **Avoid circular dependencies**: Restructure your components to avoid circular dependencies