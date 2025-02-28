package container

import "context"

// Component represents a container-managed component
type Component interface {
	// Init initializes the component with container context
	// The container will track which components are accessed during Init
	// to determine dependencies automatically
	Init(ApplicationContext)
	// Name returns the unique identifier for this component
	Name() string
}

// LifecycleComponent extends Component with lifecycle methods
type LifecycleComponent interface {
	Component
	// Start is called when the container starts
	Start(context.Context)
	// Stop is called when the container shuts down
	Stop(context.Context)
}

// ConfigurableComponent can be configured after creation
type ConfigurableComponent interface {
	Component
	// Configure configures the component with the provided options
	Configure(options map[string]interface{}) error
}

// OrderedComponent allows explicit control of initialization order
type OrderedComponent interface {
	Component
	// GetOrder returns the initialization order (lower values are initialized first)
	GetOrder() int
}

// ConditionalComponent can decide whether it should be initialized
type ConditionalComponent interface {
	Component
	// ShouldInitialize determines whether this component should be initialized
	ShouldInitialize(ApplicationContext) bool
}

// ComponentBase provides a basic implementation of Component methods
type ComponentBase struct {
	name string
}

// NewComponentBase creates a new ComponentBase with the given name
func NewComponentBase(name string) ComponentBase {
	return ComponentBase{name: name}
}

// Name returns the component name
func (c ComponentBase) Name() string {
	return c.name
}

// NoOpInit is a no-op implementation of Init
func (c ComponentBase) Init(ApplicationContext) {
	// No-op implementation
}

// Ensure that ComponentBase implements Component
var _ Component = (*ComponentBase)(nil)
