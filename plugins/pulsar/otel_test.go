//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"errors"
	"fmt"
	"testing"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// installTestTracer wires an in-memory SpanRecorder + W3C TraceContext
// propagator into the OTel globals for the duration of t, restoring
// the previous globals on cleanup. Returns the recorder so the test
// can assert on emitted spans.
func installTestTracer(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
	return sr
}

// fakeMessageID is a minimal ap.MessageID implementation for tests.
// The interface only requires Serialize, LedgerID, EntryID, BatchIdx,
// PartitionIdx; the rest are unused.
type fakeMessageID struct {
	id         string
	partition  int32
	ledger     int64
	entry      int64
	batchIdx   int32
	serialized []byte
}

func (f fakeMessageID) Serialize() []byte   { return f.serialized }
func (f fakeMessageID) LedgerID() int64     { return f.ledger }
func (f fakeMessageID) EntryID() int64      { return f.entry }
func (f fakeMessageID) BatchIdx() int32     { return f.batchIdx }
func (f fakeMessageID) PartitionIdx() int32 { return f.partition }

// attrsToMap flattens attribute KVs to a map keyed by attribute.Key
// with stringified values. Used to assert on emitted attributes.
func attrsToMap(kvs []attribute.KeyValue) map[attribute.Key]string {
	out := make(map[attribute.Key]string, len(kvs))
	for _, kv := range kvs {
		out[kv.Key] = kv.Value.Emit()
	}
	return out
}

func mustAttr(t *testing.T, m map[attribute.Key]string, key, want string) {
	t.Helper()
	got, ok := m[attribute.Key(key)]
	require.Truef(t, ok, "missing attribute %q (have: %v)", key, m)
	assert.Equal(t, want, got, "attribute %q value", key)
}

func TestSpanName_Format(t *testing.T) {
	assert.Equal(t, "send shop.orders", spanName("send", "shop.orders"))
	assert.Equal(t, "process shop.orders", spanName("process", "shop.orders"))
	assert.Equal(t, "ack shop.orders", spanName("ack", "shop.orders"))
	assert.Equal(t, "nack shop.orders", spanName("nack", "shop.orders"))
}

func TestPulsarMessageCarrier_GetSetKeys(t *testing.T) {
	c := PulsarMessageCarrier{}
	c.Set("traceparent", "00-aaa-bbb-01")
	assert.Equal(t, "00-aaa-bbb-01", c.Get("traceparent"))
	keys := c.Keys()
	assert.Contains(t, keys, "traceparent")
}

func TestExtractMessageContext_W3C(t *testing.T) {
	installTestTracer(t)
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	const spanID = "00f067aa0ba902b7"
	props := map[string]string{
		"traceparent": "00-" + traceID + "-" + spanID + "-01",
	}
	ctx := extractMessageContext(context.Background(), props)
	sc := trace.SpanContextFromContext(ctx)
	require.True(t, sc.IsValid(), "W3C Extract should yield a valid SpanContext")
	assert.Equal(t, traceID, sc.TraceID().String())
	// trace_id context value must mirror the extracted W3C trace ID
	// so app code reading ctx.Value("trace_id") sees the parent's trace.
	got := ctx.Value("trace_id")
	require.NotNil(t, got, "trace_id should be populated from W3C traceparent")
	assert.Equal(t, traceID, got.(string))
}

func TestExtractMessageContext_NilProperties(t *testing.T) {
	installTestTracer(t)
	in := context.WithValue(context.Background(), "caller-key", "caller-value")
	out := extractMessageContext(in, nil)
	// With no properties we must not touch the context — any caller
	// values should pass through unchanged, no SpanContext is forged,
	// and no fake trace_id is generated.
	assert.Equal(t, "caller-value", out.Value("caller-key"))
	sc := trace.SpanContextFromContext(out)
	assert.False(t, sc.IsValid(), "nil properties: no SpanContext")
	assert.Nil(t, out.Value("trace_id"), "nil properties: trace_id must not be forged")
}

func TestStartProcessSpan_AttributesAndKind(t *testing.T) {
	sr := installTestTracer(t)
	msgID := fakeMessageID{id: "msg-1", partition: 7}
	ctx, end := startProcessSpan(context.Background(), "shop.orders", "orders-sub", msgID)
	assert.NotNil(t, ctx)
	end(nil)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "process shop.orders", s.Name())
	assert.Equal(t, trace.SpanKindConsumer, s.SpanKind())
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "messaging.system", "pulsar")
	mustAttr(t, attrs, "messaging.operation.name", "process")
	mustAttr(t, attrs, "messaging.operation.type", "process")
	mustAttr(t, attrs, "messaging.destination.name", "shop.orders")
	mustAttr(t, attrs, "messaging.destination.subscription.name", "orders-sub")
	mustAttr(t, attrs, "messaging.consumer.group.name", "orders-sub")
	// ap.MessageID has no String() method, so we use fmt.Sprintf("%v", id)
	// which renders the struct as {id partition ledger entry batchIdx serialized}.
	mustAttr(t, attrs, "messaging.message.id", "{msg-1 7 0 0 0 []}")
	mustAttr(t, attrs, "messaging.destination.partition.id", "7")
	assert.Equal(t, codes.Unset, s.Status().Code, "no error: status should be Unset")
}

func TestStartProcessSpan_NoPartitionWhenNonPartitioned(t *testing.T) {
	sr := installTestTracer(t)
	msgID := fakeMessageID{id: "msg-1", partition: -1}
	_, end := startProcessSpan(context.Background(), "shop.orders", "orders-sub", msgID)
	end(nil)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	attrs := attrsToMap(spans[0].Attributes())
	_, has := attrs["messaging.destination.partition.id"]
	assert.False(t, has, "non-partitioned topic should omit partition.id")
}

func TestStartProcessSpan_RecordsError(t *testing.T) {
	sr := installTestTracer(t)
	_, end := startProcessSpan(context.Background(), "shop.orders", "orders-sub",
		fakeMessageID{id: "msg-1", partition: -1})
	end(errors.New("boom"))
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, codes.Error, s.Status().Code)
	assert.Contains(t, s.Status().Description, "boom")
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "error.type", "*errors.errorString")
}

func TestCreateSettleSpan_AckSuccess(t *testing.T) {
	sr := installTestTracer(t)
	createSettleSpan(context.Background(), "shop.orders", "orders-sub", spanOpAck,
		fakeMessageID{id: "m", partition: -1}, nil)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "ack shop.orders", s.Name())
	assert.Equal(t, trace.SpanKindClient, s.SpanKind())
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "messaging.operation.name", "ack")
	mustAttr(t, attrs, "messaging.operation.type", "settle")
	assert.Equal(t, codes.Unset, s.Status().Code)
}

func TestCreateSettleSpan_NackRecordsErrorAndUnwraps(t *testing.T) {
	sr := installTestTracer(t)
	inner := errors.New("inner")
	wrapped := fmt.Errorf("ctx: %w", inner)
	createSettleSpan(context.Background(), "shop.orders", "orders-sub", spanOpNack,
		fakeMessageID{id: "m", partition: -1}, wrapped)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, codes.Error, s.Status().Code)
	attrs := attrsToMap(s.Attributes())
	// Unwrap chain: fmt.wrapError -> errors.errorString
	mustAttr(t, attrs, "error.type", "*errors.errorString")
}

func TestRecordSpanError_NilIsNoOp(t *testing.T) {
	sr := installTestTracer(t)
	_, span := otel.Tracer("test").Start(context.Background(), "noop")
	recordSpanError(span, nil)
	span.End()
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
	attrs := attrsToMap(spans[0].Attributes())
	_, has := attrs["error.type"]
	assert.False(t, has, "nil err: no error.type attribute")
}

func TestErrorTypeName_Unwraps(t *testing.T) {
	inner := errors.New("inner")
	wrapped := fmt.Errorf("ctx: %w", inner)
	assert.Equal(t, "*errors.errorString", errorTypeName(wrapped))
}

func TestErrorTypeName_Plain(t *testing.T) {
	assert.Equal(t, "*errors.errorString", errorTypeName(errors.New("plain")))
}

func TestMessagingMessageIDAttrs_NilId(t *testing.T) {
	assert.Nil(t, messagingMessageIDAttrs(nil))
}

func TestMessagingMessageIDAttrs_WithPartition(t *testing.T) {
	got := messagingMessageIDAttrs(fakeMessageID{id: "x", partition: 3})
	attrs := attrsToMap(got)
	mustAttr(t, attrs, "messaging.message.id", "{x 3 0 0 0 []}")
	mustAttr(t, attrs, "messaging.destination.partition.id", "3")
}

// Ensure fakeMessageID satisfies the ap.MessageID interface.
var _ ap.MessageID = fakeMessageID{}
