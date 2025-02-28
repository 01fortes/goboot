package container

type VariableLoader interface {
	Load(ContextBuilder)
}

type SimpleYamlLoader struct {
}

func (l SimpleYamlLoader) Load(builder ContextBuilder) {}
