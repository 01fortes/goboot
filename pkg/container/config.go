package container

type ConfigLoader interface {
	Load(builder *ContextBuilder)
}
