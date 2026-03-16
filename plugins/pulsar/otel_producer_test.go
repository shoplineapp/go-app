//go:build otel
// +build otel

package pulsar

import (
	"context"
	"errors"
	"testing"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// --- test helpers (shared with consumer tests) ---

func setupTestTracer(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return tp, exporter
}

func attrMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[string(a.Key)] = a.Value.Emit()
	}
	return m
}

// --- mock producer ---

type mockProducer struct {
	sentMsg *ap.ProducerMessage
	sentCtx context.Context
	sendErr error
	sendID  ap.MessageID
}

func (m *mockProducer) Send(ctx context.Context, msg *ap.ProducerMessage) (ap.MessageID, error) {
	m.sentCtx = ctx
	m.sentMsg = msg
	return m.sendID, m.sendErr
}

func (m *mockProducer) SendAsync(_ context.Context, _ *ap.ProducerMessage, _ func(ap.MessageID, *ap.ProducerMessage, error)) {
}
func (m *mockProducer) LastSequenceID() int64 { return 0 }
func (m *mockProducer) Flush() error          { return nil }
func (m *mockProducer) Close()                {}
func (m *mockProducer) Topic() string         { return "test-topic" }
func (m *mockProducer) Name() string          { return "mock-producer" }

// --- tests ---

func TestInstrumentedProducer_Send_CreatesSpan(t *testing.T) {
	_, exporter := setupTestTracer(t)
	mock := &mockProducer{}
	producer := wrapProducer(mock, "persistent://tenant/ns/my-topic")

	msg := &ap.ProducerMessage{Payload: []byte("hello")}
	_, _ = producer.Send(context.Background(), msg)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)

	s := spans[0]
	assert.Equal(t, "send persistent://tenant/ns/my-topic", s.Name)
	assert.Equal(t, trace.SpanKindProducer, s.SpanKind)

	attrs := attrMap(s.Attributes)
	assert.Equal(t, "pulsar", attrs["messaging.system"])
	assert.Equal(t, "send", attrs["messaging.operation.name"])
	assert.Equal(t, "send", attrs["messaging.operation.type"])
	assert.Equal(t, "persistent://tenant/ns/my-topic", attrs["messaging.destination.name"])
}

func TestInstrumentedProducer_Send_InjectsTraceparent(t *testing.T) {
	_, _ = setupTestTracer(t)
	mock := &mockProducer{}
	producer := wrapProducer(mock, "test-topic")

	msg := &ap.ProducerMessage{Payload: []byte("hello")}
	_, _ = producer.Send(context.Background(), msg)

	assert.Contains(t, mock.sentMsg.Properties, "traceparent")
	assert.NotEmpty(t, mock.sentMsg.Properties["traceparent"])
}

func TestInstrumentedProducer_Send_PreservesExistingProperties(t *testing.T) {
	_, _ = setupTestTracer(t)
	mock := &mockProducer{}
	producer := wrapProducer(mock, "test-topic")

	msg := &ap.ProducerMessage{
		Payload:    []byte("hello"),
		Properties: map[string]string{"custom-key": "custom-value"},
	}
	_, _ = producer.Send(context.Background(), msg)

	assert.Equal(t, "custom-value", mock.sentMsg.Properties["custom-key"])
	assert.Contains(t, mock.sentMsg.Properties, "traceparent")
}

func TestInstrumentedProducer_Send_RecordsError(t *testing.T) {
	_, exporter := setupTestTracer(t)
	mock := &mockProducer{sendErr: errors.New("send failed")}
	producer := wrapProducer(mock, "test-topic")

	msg := &ap.ProducerMessage{Payload: []byte("hello")}
	_, err := producer.Send(context.Background(), msg)

	assert.Error(t, err)
	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)

	s := spans[0]
	attrs := attrMap(s.Attributes)
	assert.Contains(t, attrs, "error.type")
}
