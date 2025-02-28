package container

type Factory interface {
	Create(*ContextBuilder)
}
