//go:build sqs && sqs_worker && otel
// +build sqs,sqs_worker,otel

package sqs_worker

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return exporter
}

func attrMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[string(a.Key)] = a.Value.Emit()
	}
	return m
}

func stringAttr(s string) *aws_sqs.MessageAttributeValue {
	return &aws_sqs.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(s),
	}
}

func TestExtractMessageContext_WithTraceparent(t *testing.T) {
	_ = setupTestTracer(t)
	attrs := map[string]*aws_sqs.MessageAttributeValue{
		"traceparent": stringAttr("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"),
	}
	ctx := extractMessageContext(context.Background(), attrs)

	spanCtx := trace.SpanContextFromContext(ctx)
	assert.True(t, spanCtx.IsValid())
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spanCtx.TraceID().String())
}

func TestExtractMessageContext_WithoutTraceparent(t *testing.T) {
	_ = setupTestTracer(t)
	attrs := map[string]*aws_sqs.MessageAttributeValue{
		"unrelated": stringAttr("value"),
	}
	ctx := extractMessageContext(context.Background(), attrs)
	spanCtx := trace.SpanContextFromContext(ctx)
	assert.False(t, spanCtx.IsValid())
}

func TestExtractMessageContext_NilAttrs(t *testing.T) {
	_ = setupTestTracer(t)
	ctx := extractMessageContext(context.Background(), nil)
	assert.NotNil(t, ctx)
}

func TestExtractMessageContext_BackwardCompatTraceID(t *testing.T) {
	_ = setupTestTracer(t)
	attrs := map[string]*aws_sqs.MessageAttributeValue{
		"traceparent": stringAttr("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"),
	}
	ctx := extractMessageContext(context.Background(), attrs)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", ctx.Value("trace_id"))
}

func TestExtractMessageContext_FallbackToTraceIDAttribute(t *testing.T) {
	_ = setupTestTracer(t)
	attrs := map[string]*aws_sqs.MessageAttributeValue{
		"trace_id": stringAttr("legacy-trace-id"),
	}
	ctx := extractMessageContext(context.Background(), attrs)
	assert.Equal(t, "legacy-trace-id", ctx.Value("trace_id"))
}

func TestStartProcessSpan_CreatesConsumerSpan(t *testing.T) {
	exporter := setupTestTracer(t)

	_, end := startProcessSpan(context.Background(), "my-queue", "msg-123")
	end(nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "process my-queue", s.Name)
	assert.Equal(t, trace.SpanKindConsumer, s.SpanKind)

	attrs := attrMap(s.Attributes)
	assert.Equal(t, "aws_sqs", attrs["messaging.system"])
	assert.Equal(t, "process", attrs["messaging.operation.name"])
	assert.Equal(t, "process", attrs["messaging.operation.type"])
	assert.Equal(t, "my-queue", attrs["messaging.destination.name"])
	assert.Equal(t, "msg-123", attrs["messaging.message.id"])
}

func TestStartProcessSpan_RecordsError(t *testing.T) {
	exporter := setupTestTracer(t)

	_, end := startProcessSpan(context.Background(), "my-queue", "msg-123")
	end(errors.New("processing failed"))

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
	assert.Equal(t, "processing failed", spans[0].Status.Description)
	assert.Contains(t, attrMap(spans[0].Attributes), "error.type")
}

func TestStartProcessSpan_ContinuesProducerTrace(t *testing.T) {
	exporter := setupTestTracer(t)

	attrs := map[string]*aws_sqs.MessageAttributeValue{
		"traceparent": stringAttr("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"),
	}
	ctx := extractMessageContext(context.Background(), attrs)
	_, end := startProcessSpan(ctx, "my-queue", "msg-123")
	end(nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spans[0].SpanContext.TraceID().String())
}

func TestOtelProcessHook_InstalledAsDefault(t *testing.T) {
	exporter := setupTestTracer(t)

	ctx, end := processHook(context.Background(), "my-queue", "msg-1", nil)
	_ = ctx
	end(nil)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1, "otel init() should override processHook")
	assert.Equal(t, "process my-queue", spans[0].Name)
}
