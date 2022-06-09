package app

import (
	"github.com/shoplineapp/go-app/plugins"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type Application struct {
	name string
	fx   *fx.App
}

type AppOption struct{}

func NewApplication() *Application {
	return &Application{}
}

func (app *Application) Run(funcs ...interface{}) {
	app.fx = fx.New(
		fx.WithLogger(func() fxevent.Logger { return AppLogger{} }),
		fx.Provide(
			plugins.Registry...,
		),
		fx.Invoke(funcs...),
	)
	app.fx.Run()
}
