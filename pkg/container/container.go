package container

import "context"

type ApplicationContext interface {
	GetComponent(name string) Component
	GetVariable(name string) string
}

type ContextBuilder interface {
	ApplicationContext
	RegisterComponent(component Component)
	RegisterVariable(name string, value string)
}

type container struct {
	components map[string]Component
	variables  map[string]string

	variablesLoaders []VariableLoader
	configLoaders    []ConfigLoader
	factories        []Factory
}

func (c *container) RegisterComponent(component Component) {
	c.components[component.Name()] = component
}

func (c *container) RegisterVariable(name string, value string) {
	c.variables[name] = value
}

func (c *container) GetComponent(name string) Component {
	return c.components[name]
}

func (c *container) GetVariable(name string) string {
	return c.variables[name]
}

func Start(ctx context.Context, block func(ContextBuilder)) ApplicationContext {
	res := &container{
		variablesLoaders: []VariableLoader{
			SimpleYamlLoader{},
		},
		configLoaders: []ConfigLoader{},
		factories:     []Factory{},
		components:    map[string]Component{},
		variables:     map[string]string{},
	}

	block(res)

	for _, component := range res.components {
		component.Init(res)
	}

	for _, component := range res.components {
		if lifecycle, ok := component.(LifecycleComponent); ok {
			lifecycle.Start(ctx)
		}
	}

	return res
}
