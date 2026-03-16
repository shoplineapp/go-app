//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestExtractMessageContext_WithTraceparent(t *testing.T) {
	_, _ = setupTestTracer(t)

	props := map[string]string{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}
	ctx := extractMessageContext(context.Background(), props)

	spanCtx := trace.SpanContextFromContext(ctx)
	assert.True(t, spanCtx.IsValid())
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spanCtx.TraceID().String())
}

func TestExtractMessageContext_WithoutTraceparent(t *testing.T) {
	_, _ = setupTestTracer(t)

	props := map[string]string{
		"some_key": "some_value",
	}
	ctx := extractMessageContext(context.Background(), props)

	spanCtx := trace.SpanContextFromContext(ctx)
	assert.False(t, spanCtx.IsValid())
}

func TestExtractMessageContext_NilProperties(t *testing.T) {
	_, _ = setupTestTracer(t)

	ctx := extractMessageContext(context.Background(), nil)
	assert.NotNil(t, ctx)
}

func TestExtractMessageContext_BackwardCompat_TraceID(t *testing.T) {
	_, _ = setupTestTracer(t)

	props := map[string]string{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}
	ctx := extractMessageContext(context.Background(), props)

	traceID := ctx.Value("trace_id")
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", traceID)
}

func TestExtractMessageContext_FallbackToTraceIDProperty(t *testing.T) {
	_, _ = setupTestTracer(t)

	props := map[string]string{
		"trace_id": "my-legacy-trace-id",
	}
	ctx := extractMessageContext(context.Background(), props)

	traceID := ctx.Value("trace_id")
	assert.Equal(t, "my-legacy-trace-id", traceID)
}

func TestStartProcessSpan_CreatesConsumerSpan(t *testing.T) {
	_, exporter := setupTestTracer(t)

	ctx := context.Background()
	ctx, endSpan := startProcessSpan(ctx, "my-topic", "my-sub", "msg-123")
	_ = ctx
	endSpan(nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "process my-topic", span.Name)
	assert.Equal(t, trace.SpanKindConsumer, span.SpanKind)

	attrs := attrMap(span.Attributes)
	assert.Equal(t, "pulsar", attrs["messaging.system"])
	assert.Equal(t, "process", attrs["messaging.operation.name"])
	assert.Equal(t, "process", attrs["messaging.operation.type"])
	assert.Equal(t, "my-topic", attrs["messaging.destination.name"])
	assert.Equal(t, "my-sub", attrs["messaging.destination.subscription.name"])
	assert.Equal(t, "msg-123", attrs["messaging.message.id"])
}

func TestStartProcessSpan_RecordsError(t *testing.T) {
	_, exporter := setupTestTracer(t)

	ctx := context.Background()
	_, endSpan := startProcessSpan(ctx, "my-topic", "my-sub", "msg-123")
	endSpan(errors.New("processing failed"))

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Equal(t, "processing failed", span.Status.Description)
}

func TestStartProcessSpan_ContinuesProducerTrace(t *testing.T) {
	_, exporter := setupTestTracer(t)

	props := map[string]string{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}
	ctx := extractMessageContext(context.Background(), props)
	ctx, endSpan := startProcessSpan(ctx, "my-topic", "my-sub", "msg-123")
	_ = ctx
	endSpan(nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spans[0].SpanContext.TraceID().String())
}

func TestCreateSettleSpan_Ack(t *testing.T) {
	_, exporter := setupTestTracer(t)

	ctx := context.Background()
	createSettleSpan(ctx, "my-topic", "my-sub", "ack", nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "ack my-topic", span.Name)
	assert.Equal(t, trace.SpanKindClient, span.SpanKind)

	attrs := attrMap(span.Attributes)
	assert.Equal(t, "pulsar", attrs["messaging.system"])
	assert.Equal(t, "ack", attrs["messaging.operation.name"])
	assert.Equal(t, "settle", attrs["messaging.operation.type"])
	assert.Equal(t, "my-topic", attrs["messaging.destination.name"])
	assert.Equal(t, "my-sub", attrs["messaging.destination.subscription.name"])
}

func TestCreateSettleSpan_Nack(t *testing.T) {
	_, exporter := setupTestTracer(t)

	ctx := context.Background()
	createSettleSpan(ctx, "my-topic", "my-sub", "nack", nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, "nack my-topic", spans[0].Name)
}
