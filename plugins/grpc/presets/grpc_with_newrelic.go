//go:build grpc && newrelic
// +build grpc,newrelic

package presets

import (
	"context"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	grpc_plugin "github.com/shoplineapp/go-app/plugins/grpc"
	"github.com/shoplineapp/go-app/plugins/grpc/healthcheck"
	"github.com/shoplineapp/go-app/plugins/grpc/interceptors"
	"github.com/shoplineapp/go-app/plugins/grpc/stats_handlers"
	"github.com/shoplineapp/go-app/plugins/logger"
	"google.golang.org/grpc"

	"go.uber.org/fx"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewDefaultGrpcServerWithNewrelic)
}

type DefaultGrpcServerWithNewrelic struct {
	grpc_plugin.GrpcServer
}

func NewDefaultGrpcServerWithNewrelic(
	lc fx.Lifecycle,
	logger *logger.Logger,
	env *env.Env,
	grpcServer *grpc_plugin.GrpcServer,
	newrelicStat *stats_handlers.NewrelicStatsHandler,
	deadline *interceptors.DeadlineInterceptor,
	requestLog *interceptors.RequestLogInterceptor,
	recovery *interceptors.RecoveryInterceptor,
	senderr *interceptors.NewrelicInterceptor,
	healthcheckServer *healthcheck.HealthCheckServer,
) *DefaultGrpcServerWithNewrelic {
	s := *grpcServer
	plugin := &DefaultGrpcServerWithNewrelic{
		GrpcServer: s,
	}
	plugin.Configure(
		grpc.StatsHandler(newrelicStat),
		grpc.ChainUnaryInterceptor(
			requestLog.Handler(),
			senderr.Handler(),
			deadline.Handler(),
			recovery.Handler(),
		),
	)
	healthcheck.RegisterHealthServer(plugin.Server(), healthcheckServer)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			plugin.Serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			plugin.Shutdown()
			return nil
		},
	})

	return plugin
}
