//go:build grpc && otel
// +build grpc,otel

package interceptors

import (
	"context"
	"path"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtelInterceptor)
}

type OtelInterceptor struct {
	agent *opentelemetry.OtelAgent
}

func (i OtelInterceptor) Handler() grpc.UnaryServerInterceptor {
	customNewrelicInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		tracer := opentelemetry.GetTracer()
		service := path.Dir(info.FullMethod)[1:]
		if tracer == nil || service == "grpc.health.v1.Health" {
			return handler(ctx, req)
		}

		traceId := strings.ReplaceAll(ctx.Value("trace_id").(string), "-", "")
		spanId := strings.ReplaceAll(ctx.Value("span_id").(string), "-", "")
		traceID, _ := trace.TraceIDFromHex(traceId)
		spanID, _ := trace.SpanIDFromHex(spanId)
		if traceID.IsValid() && spanID.IsValid() {
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  spanID,
			})
			ctx = trace.ContextWithSpanContext(ctx, spanContext)
		} else if traceID.IsValid() {
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: traceID,
			})
			ctx = trace.ContextWithSpanContext(ctx, spanContext)
		}

		newCtx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

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
			span.RecordError(err)
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			for key, value := range md {
				attrs = append(attrs, attribute.KeyValue{
					Key:   attribute.Key(key),
					Value: attribute.StringSliceValue(value),
				})
			}
		}

		span.SetAttributes(attrs...)

		return resp, err
	}

	return grpc_middleware.ChainUnaryServer(
		customNewrelicInterceptor,
	)
}

func NewOtelInterceptor(agent *opentelemetry.OtelAgent) *OtelInterceptor {
	return &OtelInterceptor{
		agent: agent,
	}
}
