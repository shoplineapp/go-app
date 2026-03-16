//go:build !otel
// +build !otel

package pulsar

import (
	"context"

	"github.com/shoplineapp/go-app/common"
)

func extractMessageContext(ctx context.Context, properties map[string]string) context.Context {
	var traceID string
	if properties != nil {
		traceID = properties["trace_id"]
	}
	return common.NewContextWithTraceID(ctx, traceID)
}

func startProcessSpan(ctx context.Context, _, _, _ string) (context.Context, func(error)) {
	return ctx, func(error) {}
}

func createSettleSpan(_ context.Context, _, _, _ string, _ error) {
}
