package common

import (
	"context"

	"github.com/google/uuid"
)

func GetTraceID(ctx context.Context) string {
	traceId := ctx.Value("trace_id")
	if traceId == nil {
		return uuid.New().String()
	}
	return traceId.(string)
}
