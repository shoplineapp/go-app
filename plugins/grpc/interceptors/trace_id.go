//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	"github.com/google/uuid"
	"github.com/shoplineapp/go-app/plugins"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewTraceIdInterceptor)
}

type TraceIdInterceptor struct {
}

func (i TraceIdInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		var traceId string
		var spanId string

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("trace_id"); len(v) > 0 {
				traceId = v[0]
			} else if v := md.Get("x-trace-id"); len(v) > 0 {
				traceId = v[0]
			}

			if v := md.Get("span_id"); len(v) > 0 {
				spanId = v[0]
			} else if v := md.Get("x-span-id"); len(v) > 0 {
				spanId = v[0]
			}
		}

		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.IsValid() {
			if traceId == "" {
				traceId = spanContext.TraceID().String()
				spanId = spanContext.SpanID().String()
			} else if traceId == spanContext.TraceID().String() {
				spanId = spanContext.SpanID().String()
			}
		}

		if traceId == "" {
			traceId = uuid.New().String()

		}
		if spanId == "" {
			spanId = uuid.New().String()

		}

		ctx = context.WithValue(ctx, "trace_id", traceId)
		ctx = context.WithValue(ctx, "span_id", spanId)

		grpc.SetHeader(ctx, metadata.Pairs("x-trace-id", traceId))
		grpc.SetHeader(ctx, metadata.Pairs("x-span-id", spanId))

		resp, err = handler(ctx, req)
		return resp, err
	}
}

func NewTraceIdInterceptor() *TraceIdInterceptor {
	return &TraceIdInterceptor{}
}
