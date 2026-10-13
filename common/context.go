package common

import (
	"context"

	"github.com/google/uuid"
)

func NewContextWithTraceID(ctx context.Context, traceId string) context.Context {
	if traceId == "" {
		traceId = uuid.New().String()
	}
	return context.WithValue(ctx, "trace_id", traceId)
}

func GetTraceID(ctx context.Context) string {
	traceId := ctx.Value("trace_id")
	if traceId == nil {
		return uuid.New().String()
	}
	return traceId.(string)
}
