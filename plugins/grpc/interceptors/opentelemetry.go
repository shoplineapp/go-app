//go:build grpc && otel
// +build grpc,otel

package interceptors

import (
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/opentelemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"
	"google.golang.org/grpc"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtelInterceptor)
}

type OtelInterceptor struct {
	agent *opentelemetry.OtelAgent
}

func (i OtelInterceptor) Handler() grpc.UnaryServerInterceptor {
	return grpc_middleware.ChainUnaryServer(
		otelgrpc.UnaryServerInterceptor(otelgrpc.WithInterceptorFilter(
			filters.Not(
				filters.HealthCheck(),
			),
		)),
	)
}

func NewOtelInterceptor(agent *opentelemetry.OtelAgent) *OtelInterceptor {
	return &OtelInterceptor{
		agent: agent,
	}
}
