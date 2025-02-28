package container

// Factory is an interface for components that can create and register other components
type Factory interface {
	// Create creates components and registers them with the container
	Create(ContextBuilder) error
}

// ComponentFactory is a simple implementation of Factory
type ComponentFactory struct {
	// Components is a slice of components to register
	Components []Component
}

// Create registers all components with the container
func (f ComponentFactory) Create(builder ContextBuilder) error {
	for _, component := range f.Components {
		if err := builder.RegisterComponent(component); err != nil {
			return err
		}
	}
	return nil
}

// FactoryFunc is a factory implemented as a function
type FactoryFunc struct {
	fn func(ContextBuilder) error
}

// Create calls the function to create components
func (f FactoryFunc) Create(builder ContextBuilder) error {
	return f.fn(builder)
}

// NewFactory creates a new factory with the given function
func NewFactory(fn func(ContextBuilder) error) Factory {
	return &FactoryFunc{fn: fn}
}
