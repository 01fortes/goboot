package main

import (
	"GoBoot/pkg/container"
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	container.Start(ctx, func(cnt container.ContextBuilder) {
		cnt.RegisterComponent(TestComponent{})
		cnt.RegisterComponent(RunnableComponent{})
		cnt.RegisterComponent(TestComponent2{})
		cnt.RegisterVariable("some.test", "lloololo")
	})

}

type TestComponent struct {
	str string
}

func (t TestComponent) Init(applicationContext container.ApplicationContext) {
	t.str = applicationContext.GetVariable("some.test")
}

func (t TestComponent) Name() string {
	return "test"
}

type TestComponent2 struct {
	t TestComponent
}

func (t TestComponent2) Init(applicationContext container.ApplicationContext) {
	t.t = applicationContext.GetComponent("test").(TestComponent)
}
func (t TestComponent2) Name() string {
	return "test2"
}

type RunnableComponent struct {
	t TestComponent2
}

func (t RunnableComponent) Init(applicationContext container.ApplicationContext) {
	t.t = applicationContext.GetComponent("test2").(TestComponent2)
}

func (t RunnableComponent) Name() string {
	return "runnable"
}

func (t RunnableComponent) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			{
				time.Sleep(1 * time.Second)
				slog.Info(t.t.t.Name())
			}
		}

	}
}

func (t RunnableComponent) Stop(ctx context.Context) {

}
