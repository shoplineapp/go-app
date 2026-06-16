package common

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func NewContextWithTraceID(ctx context.Context, traceId string) context.Context {
	if traceId == "" {
		traceId = uuid.New().String()
	}
	return context.WithValue(ctx, "trace_id", traceId)
}

func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value("trace_id").(string); ok && v != "" {
		return v
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}
