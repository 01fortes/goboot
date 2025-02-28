package container

// Starter defines an interface for components that can create and configure other components
// at container startup. This allows creation of modular "starters" like in Spring Boot.
type Starter interface {
	// Name returns the name of this starter
	Name() string

	// Start is called during container initialization to register components
	// It can use the provided builder to register components and variables
	Start(builder ContextBuilder) error
}

// ConditionalStarter is a starter that can determine whether it should be applied
type ConditionalStarter interface {
	Starter

	// ShouldStart determines whether this starter should be applied
	ShouldStart(builder ApplicationContext) bool
}

// StarterFunc is a simple implementation of Starter using a function
type StarterFunc struct {
	name string
	fn   func(ContextBuilder) error
}

// Name returns the name of the starter
func (s *StarterFunc) Name() string {
	return s.name
}

// Start calls the function to register components
func (s *StarterFunc) Start(builder ContextBuilder) error {
	return s.fn(builder)
}

// NewStarter creates a new starter with the given name and function
func NewStarter(name string, fn func(ContextBuilder) error) Starter {
	return &StarterFunc{
		name: name,
		fn:   fn,
	}
}

// CompositeStarter combines multiple starters into one
type CompositeStarter struct {
	name     string
	starters []Starter
}

// Name returns the name of the starter
func (s *CompositeStarter) Name() string {
	return s.name
}

// Start calls all the starters in sequence
func (s *CompositeStarter) Start(builder ContextBuilder) error {
	for _, starter := range s.starters {
		if err := starter.Start(builder); err != nil {
			return err
		}
	}
	return nil
}

// NewCompositeStarter creates a new composite starter
func NewCompositeStarter(name string, starters ...Starter) Starter {
	return &CompositeStarter{
		name:     name,
		starters: starters,
	}
}

// ConditionalStarterFunc is a simple implementation of ConditionalStarter using functions
type ConditionalStarterFunc struct {
	StarterFunc
	condition func(ApplicationContext) bool
}

// ShouldStart determines whether this starter should be applied
func (s *ConditionalStarterFunc) ShouldStart(ctx ApplicationContext) bool {
	return s.condition(ctx)
}

// NewConditionalStarter creates a new conditional starter
func NewConditionalStarter(name string, condition func(ApplicationContext) bool, fn func(ContextBuilder) error) ConditionalStarter {
	return &ConditionalStarterFunc{
		StarterFunc: StarterFunc{
			name: name,
			fn:   fn,
		},
		condition: condition,
	}
}

// PropertyCondition checks if a property has a specific value
func PropertyCondition(property, expectedValue string) func(ApplicationContext) bool {
	return func(ctx ApplicationContext) bool {
		return ctx.GetVariable(property) == expectedValue
	}
}

// PropertyExistsCondition checks if a property exists
func PropertyExistsCondition(property string) func(ApplicationContext) bool {
	return func(ctx ApplicationContext) bool {
		return ctx.GetVariable(property) != ""
	}
}

// ComponentExistsCondition checks if a component exists
func ComponentExistsCondition(name string) func(ApplicationContext) bool {
	return func(ctx ApplicationContext) bool {
		return ctx.HasComponent(name)
	}
}
