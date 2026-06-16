package common

import (
	"context"
	"encoding/binary"
	"math/rand/v2"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func NewContextWithTraceID(ctx context.Context, traceId string) context.Context {
	if traceId == "" {
		traceId = uuid.New().String()
	}
	return context.WithValue(ctx, "trace_id", traceId)
}

// GetTraceID returns the trace_id associated with ctx, looking in three places
// in order:
//
//  1. The legacy ctx["trace_id"] string key (seeded by the gRPC/Kitex bridge
//     interceptors from the OTel SpanContext, or by NewContextWithTraceID).
//  2. The active OTel SpanContext on ctx (covers code paths that wrap a span
//     without going through the bridge — e.g. background goroutines that
//     inherit a parent span, or direct tracer.Start callers).
//  3. A freshly generated 16-byte W3C TraceID, so callers that invoke
//     GetTraceID on a context.Background() / unset context still receive a
//     valid W3C-format ID for log/Sentry/Pulsar correlation rather than "".
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value("trace_id").(string); ok && v != "" {
		return v
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return newTraceID().String()
}

// newTraceID returns a fresh non-zero 16-byte OTel TraceID, matching the
// OTel SDK's own randomIDGenerator.NewIDs pattern (see
// go.opentelemetry.io/otel/sdk/trace/id_generator.go). math/rand/v2's
// package-level functions are goroutine-safe, so no mutex is required.
func newTraceID() trace.TraceID {
	var tid trace.TraceID
	for {
		binary.NativeEndian.PutUint64(tid[:8], rand.Uint64())
		binary.NativeEndian.PutUint64(tid[8:], rand.Uint64())
		if tid.IsValid() {
			return tid
		}
	}
}
