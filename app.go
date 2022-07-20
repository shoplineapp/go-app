package app

import (
	"github.com/shoplineapp/go-app/plugins"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type Application struct {
	name    string
	fx      *fx.App
	plugins []interface{}
}
type AppOption struct{}

func NewApplication() *Application {
	return &Application{
		plugins: plugins.Registry,
	}
}

func (a *Application) SetPlugins(plugins ...interface{}) {
	a.plugins = plugins
}

func (a *Application) AddModule(module AppModuleInterface) {
	controllers := module.Controllers()
	provides := module.Provide()

	if len(controllers) > 0 {
		a.plugins = append(a.plugins, module.Controllers()...)
	}

	if len(provides) > 0 {
		a.plugins = append(a.plugins, module.Provide()...)
	}
}

func (a *Application) Provide(constructors ...interface{}) {
	a.plugins = append(a.plugins, constructors...)
}

func (app *Application) Run(funcs ...interface{}) {
	app.fx = fx.New(
		fx.WithLogger(func() fxevent.Logger { return AppLogger{} }),
		fx.Provide(
			app.plugins...,
		),
		fx.Invoke(funcs...),
	)
	app.fx.Run()
}

func (app *Application) Invoke(funcs ...interface{}) {
	fx.New(
		fx.WithLogger(func() fxevent.Logger { return AppLogger{} }),
		fx.Provide(
			app.plugins...,
		),
		fx.Invoke(funcs...),
	)
}
