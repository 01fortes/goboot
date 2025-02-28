package container

import "context"

type Component interface {
	Init(ApplicationContext)
	Name() string
}

type LifecycleComponent interface {
	Component
	Start(context.Context)
	Stop(context.Context)
}
