//go:build kitex && newrelic
// +build kitex,newrelic

package kitex

import (
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	kitex_plugin "github.com/shoplineapp/go-app/plugins/kitex"
	"github.com/shoplineapp/go-app/plugins/kitex/middlewares"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.uber.org/fx"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewDefaultKitexServerWithNewrelic)
}

type DefaultKitexServerWithNewrelic struct {
	*kitex_plugin.KitexServer
}

func NewDefaultKitexServerWithNewrelic(
	lc fx.Lifecycle,
	logger *logger.Logger,
	env *env.Env,
	kitexServer *kitex_plugin.KitexServer,
	traceIDMiddleware *middlewares.KitexTraceIDMiddleware,
	requestLogMiddleware *middlewares.KitexRequestLogMiddleware,
	newrelicMiddleware *middlewares.KitexNewrelicMiddleware,
	deadlineMiddleware *middlewares.KitexDeadlineMiddleware,
) *DefaultKitexServerWithNewrelic {
	plugin := &DefaultKitexServerWithNewrelic{
		KitexServer: kitexServer,
	}
	plugin.KitexServer.SetMiddlewares([]endpoint.Middleware{
		traceIDMiddleware.Handler,
		requestLogMiddleware.Handler,
		newrelicMiddleware.Handler,
		deadlineMiddleware.Handler,
	})
	return plugin
}
