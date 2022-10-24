package middlewares

import (
	"context"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/google/uuid"
	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexTraceIDMiddleware)
}

type KitexTraceIDMiddleware struct {
}

func (m KitexTraceIDMiddleware) Handler(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request, response interface{}) error {
		traceId := uuid.New().String()
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("x-trace-id"); len(v) > 0 {
				traceId = v[0]
			}
		}

		ctx = context.WithValue(ctx, "trace_id", traceId)
		grpc.SetHeader(ctx, metadata.Pairs("x-trace-id", traceId))
		err := next(ctx, request, response)
		return err
	}
}

func NewKitexTraceIDMiddleware() *KitexTraceIDMiddleware {
	return &KitexTraceIDMiddleware{}
}
