//go:build grpc && otel
// +build grpc,otel

package interceptors

import (
	"context"

	"github.com/shoplineapp/go-app/common"
	"github.com/shoplineapp/go-app/plugins"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewTraceIdInterceptor)
}

type TraceIdInterceptor struct {
}

// Handler bridges the OTel SpanContext (populated by otelgrpc.NewServerHandler
// which is installed as a gRPC StatsHandler on the server) into the legacy
// ctx["trace_id"] string key that downstream consumers (e.g. common.GetTraceID,
// plugins/pulsar/producer.go) still rely on.
func (i TraceIdInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			ctx = common.NewContextWithTraceID(ctx, sc.TraceID().String())
		}
		return handler(ctx, req)
	}
}

func NewTraceIdInterceptor() *TraceIdInterceptor {
	return &TraceIdInterceptor{}
}
