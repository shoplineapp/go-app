//go:build grpc && newrelic && otel
// +build grpc,newrelic,otel

package presets

import (
	"context"
	"fmt"
	"strconv"

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
	deadline *interceptors.DeadlineInterceptor,
	trace_id *interceptors.TraceIdInterceptor,
	locale *interceptors.LocaleInterceptor,
	requestLog *interceptors.RequestLogInterceptor,
	recovery *interceptors.RecoveryInterceptor,
	newrelic *interceptors.NewrelicInterceptor,
	otlp *interceptors.OtelInterceptor,
) *DefaultGrpcServerWithNewrelic {
	s := *grpcServer
	plugin := &DefaultGrpcServerWithNewrelic{
		GrpcServer: s,
	}

	handles := []grpc.UnaryServerInterceptor{
		trace_id.Handler(),
		locale.Handler(),
		requestLog.Handler(),
		newrelic.Handler(),
		deadline.Handler(),
		recovery.Handler(),
		otlp.Handler(),
	}

	grpc_plugin.SetGlobalServerOptions(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	maxRecvMsgSize, pErr := strconv.ParseInt(env.GetEnv("GRPC_DEFAULT_MSG_SIZE"), 10, 64)
	if pErr != nil {
		maxRecvMsgSize = 4 * 1024 * 1024
	}

	fmt.Println("maxRecvMsgSize:", maxRecvMsgSize)

	plugin.Configure(
		grpc.MaxSendMsgSize(int(maxRecvMsgSize)),
		grpc.MaxRecvMsgSize(int(maxRecvMsgSize)),
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
