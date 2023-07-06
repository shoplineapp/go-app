//go:build grpc
// +build grpc

package interceptors

import (
	"context"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"path"
	"strings"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtlpInterceptor)
}

type OtlpInterceptor struct {
	agent *opentelemetry.OtlpAgent
}

func (i OtlpInterceptor) Handler() grpc.UnaryServerInterceptor {
	customNewrelicInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		tracer := opentelemetry.GetTracer()
		service := path.Dir(info.FullMethod)[1:]
		if tracer == nil || service == "grpc.health.v1.Health" {
			return handler(ctx, req)
		}

		ctxTraceID, _ := ctx.Value("trace_id").(string)
		// open telemetry trace id can not include -
		ctxTraceID = strings.ReplaceAll(ctxTraceID, "-", "")

		traceID, _ := trace.TraceIDFromHex(ctxTraceID)
		if traceID.IsValid() {
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: traceID,
			})

			ctx = trace.ContextWithSpanContext(ctx, spanContext)
		}

		newCtx, txn := tracer.Start(ctx, info.FullMethod)

		defer txn.End()

		var attrs []attribute.KeyValue

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
