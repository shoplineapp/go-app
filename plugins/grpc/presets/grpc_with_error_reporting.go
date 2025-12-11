//go:build grpc && newrelic && otel && sentry
// +build grpc,newrelic,sentry,otel

package presets

import (
	"context"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	grpc_plugin "github.com/shoplineapp/go-app/plugins/grpc"
	"github.com/shoplineapp/go-app/plugins/grpc/interceptors"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewDefaultGrpcServerWithErrorReporting)
}

type DefaultGrpcServerWithErrorReporting struct {
	grpc_plugin.GrpcServer
}

func NewDefaultGrpcServerWithErrorReporting(
	lc fx.Lifecycle,
	logger *logger.Logger,
	env *env.Env,
	grpcServer *grpc_plugin.GrpcServer,
	deadline *interceptors.DeadlineInterceptor,
	trace_id *interceptors.TraceIdInterceptor,
	locale *interceptors.LocaleInterceptor,
	requestLog *interceptors.RequestLogInterceptor,
	recovery *interceptors.RecoveryInterceptor,
	sentry *interceptors.SentryInterceptor,
	newrelic *interceptors.NewrelicInterceptor,
	otlp *interceptors.OtelInterceptor,
) *DefaultGrpcServerWithErrorReporting {
	s := *grpcServer
	plugin := &DefaultGrpcServerWithErrorReporting{
		GrpcServer: s,
	}

	handles := []grpc.UnaryServerInterceptor{
		trace_id.Handler(),
		locale.Handler(),
		requestLog.Handler(),
		newrelic.Handler(),
		sentry.Handler(),
		deadline.Handler(),
		recovery.Handler(),
		otlp.Handler(),
	}

	grpc_plugin.SetGlobalServerOptions(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	plugin.Configure(
		grpc.ChainUnaryInterceptor(
			handles...,
		),
	)
	healthgrpc.RegisterHealthServer(plugin.Server(), health.NewServer())
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
