//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	"github.com/google/uuid"
	"github.com/shoplineapp/go-app/plugins"
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
		traceId := uuid.New().String()
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("x-trace-id"); len(v) > 0 {
				traceId = v[0]
			} else if v := md.Get("trace_id"); len(v) > 0 {
				traceId = v[0]
			}
		}

		ctx = context.WithValue(ctx, "trace_id", traceId)

		grpc.SetHeader(ctx, metadata.Pairs("x-trace-id", traceId))

		resp, err = handler(ctx, req)

		return resp, err
	}
}

func NewTraceIdInterceptor() *TraceIdInterceptor {
	return &TraceIdInterceptor{}
}
