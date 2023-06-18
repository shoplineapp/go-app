//go:build grpc
// +build grpc

package interceptors

import (
	"context"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/opentelemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtlpInterceptor)
}

type OtlpInterceptor struct {
	agent *opentelemetry.OtlpAgent
}

func (i OtlpInterceptor) Handler() grpc.UnaryServerInterceptor {
	customNewrelicInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		tracer := otel.Tracer("grpc_request")
		if tracer == nil {
			return handler(ctx, req)
		}

		newCtx, txn := tracer.Start(ctx, info.FullMethod)
		defer txn.End()

		traceId, _ := ctx.Value("trace_id").(string)

		var attrs []attribute.KeyValue

		traceIdAttr := attribute.KeyValue{
			Key:   "TraceId",
			Value: attribute.StringValue(traceId),
		}
		attrs = append(attrs, traceIdAttr)

		resp, err = handler(newCtx, req)

		if err != nil {
			st, _ := status.FromError(err)
			attrs = append(attrs, attribute.KeyValue{
				Key:   "GrpcStatusMessage",
				Value: attribute.StringValue(st.Message()),
			})
			attrs = append(attrs, attribute.KeyValue{
				Key:   "GrpcStatusCode",
				Value: attribute.StringValue(st.Code().String()),
			})
			txn.RecordError(err)
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			for key, value := range md {
				attrs = append(attrs, attribute.KeyValue{
					Key:   attribute.Key(key),
					Value: attribute.StringSliceValue(value),
				})
			}
		}

		txn.SetAttributes(attrs...)

		return resp, err
	}

	return grpc_middleware.ChainUnaryServer(
		customNewrelicInterceptor,
	)
}

func NewOtlpInterceptor(agent *opentelemetry.OtlpAgent) *OtlpInterceptor {
	return &OtlpInterceptor{
		agent: agent,
	}
}
