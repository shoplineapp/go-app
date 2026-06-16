package common

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"

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

// newTraceID returns a fresh 16-byte OTel TraceID sourced from math/rand,
// matching the OTel SDK's own randomIDGenerator pattern (see
// go.opentelemetry.io/otel/sdk/trace/id_generator.go). The rand source is
// seeded once at package init time from crypto/rand. Mutex-guarded since
// math/rand.Rand is not goroutine-safe.
var (
	traceIDRandMu sync.Mutex
	traceIDRand   = func() *rand.Rand {
		var seed int64
		_ = binary.Read(crand.Reader, binary.LittleEndian, &seed)
		return rand.New(rand.NewSource(seed))
	}()
)

func newTraceID() trace.TraceID {
	tid := trace.TraceID{}
	traceIDRandMu.Lock()
	_, _ = traceIDRand.Read(tid[:])
	traceIDRandMu.Unlock()
	return tid
}
