//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"errors"
	"sync"
	"testing"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// fakeProducer is a minimal ap.Producer stand-in. We implement Send,
// SendAsync, and Topic explicitly; the embedded ap.Producer interface
// is otherwise left nil because tests never call any other method.
type fakeProducer struct {
	ap.Producer
	topic       string
	mu          sync.Mutex
	sendCount   int
	asyncCount  int
	lastMsg     *ap.ProducerMessage
	id          ap.MessageID
	sendErr     error
}

func (f *fakeProducer) Topic() string { return f.topic }

func (f *fakeProducer) Send(ctx context.Context, msg *ap.ProducerMessage) (ap.MessageID, error) {
	f.mu.Lock()
	f.sendCount++
	f.lastMsg = msg
	f.mu.Unlock()
	return f.id, f.sendErr
}

func (f *fakeProducer) SendAsync(ctx context.Context, msg *ap.ProducerMessage, callback func(ap.MessageID, *ap.ProducerMessage, error)) {
	f.mu.Lock()
	f.asyncCount++
	f.lastMsg = msg
	id, err := f.id, f.sendErr
	f.mu.Unlock()
	callback(id, msg, err)
}

func TestInstrumentedProducer_Send_AttributesAndPropagates(t *testing.T) {
	sr := installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders", id: fakeMessageID{id: "m-1", partition: 3}}
	ip := wrapProducer(fp).(*instrumentedProducer)
	id, err := ip.Send(context.Background(), &ap.ProducerMessage{Payload: []byte("x")})
	require.NoError(t, err)
	assert.Equal(t, "m-1", id.(fakeMessageID).id)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "send shop.orders", s.Name())
	assert.Equal(t, trace.SpanKindProducer, s.SpanKind())
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "messaging.system", "pulsar")
	mustAttr(t, attrs, "messaging.operation.name", "send")
	mustAttr(t, attrs, "messaging.operation.type", "send")
	mustAttr(t, attrs, "messaging.destination.name", "shop.orders")
	// ap.MessageID has no String() method; %v renders the struct fields.
	mustAttr(t, attrs, "messaging.message.id", "{m-1 3 0 0 0 []}")
	mustAttr(t, attrs, "messaging.destination.partition.id", "3")
	// W3C traceparent must have been injected.
	assert.NotEmpty(t, fp.lastMsg.Properties["traceparent"], "W3C traceparent must be injected")
	assert.Equal(t, codes.Unset, s.Status().Code, "no error: status Unset")
}

func TestInstrumentedProducer_Send_NilMessage(t *testing.T) {
	installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders"}
	ip := wrapProducer(fp).(*instrumentedProducer)
	_, err := ip.Send(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, 0, fp.sendCount, "underlying producer should not be called for nil message")
}

func TestInstrumentedProducer_Send_RecordsError(t *testing.T) {
	sr := installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders", sendErr: errors.New("broker offline")}
	ip := wrapProducer(fp).(*instrumentedProducer)
	_, err := ip.Send(context.Background(), &ap.ProducerMessage{})
	assert.Error(t, err)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, codes.Error, s.Status().Code)
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "error.type", "*errors.errorString")
}

func TestInstrumentedProducer_SendAsync_RecordsError(t *testing.T) {
	sr := installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders", sendErr: errors.New("nope")}
	ip := wrapProducer(fp).(*instrumentedProducer)
	var gotErr error
	ip.SendAsync(context.Background(), &ap.ProducerMessage{}, func(_ ap.MessageID, _ *ap.ProducerMessage, err error) {
		gotErr = err
	})
	assert.Error(t, gotErr)
	spans := sr.Ended()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, codes.Error, s.Status().Code)
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "error.type", "*errors.errorString")
}

func TestInstrumentedProducer_SendAsync_NilMessage(t *testing.T) {
	installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders"}
	ip := wrapProducer(fp).(*instrumentedProducer)
	var gotErr error
	ip.SendAsync(context.Background(), nil, func(_ ap.MessageID, _ *ap.ProducerMessage, err error) {
		gotErr = err
	})
	assert.Error(t, gotErr)
	assert.Equal(t, 0, fp.asyncCount, "underlying producer should not be called for nil message")
}

func TestInjectTraceContext_NoDefensiveCopy(t *testing.T) {
	installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders"}
	ip := wrapProducer(fp).(*instrumentedProducer)

	// Start a real span so the W3C propagator has something to inject.
	_, span := otel.Tracer("test").Start(context.Background(), "caller")
	defer span.End()
	ctx := trace.ContextWithSpan(context.Background(), span)

	src := map[string]string{"foo": "bar"}
	msg := &ap.ProducerMessage{Properties: src}
	ip.injectTraceContext(ctx, msg)

	// The carrier aliases the caller's map — see PulsarMessageCarrier
	// in otel.go. No defensive copy: traceparent is written in place
	// and post-injection caller mutations are visible in msg.Properties.
	assert.NotEmpty(t, msg.Properties["traceparent"], "traceparent must be injected into Properties")

	src["foo"] = "MUTATED"
	assert.Equal(t, "MUTATED", msg.Properties["foo"], "no defensive copy: caller mutations are visible in msg.Properties")
}

func TestInjectTraceContext_NilProperties(t *testing.T) {
	installTestTracer(t)
	fp := &fakeProducer{topic: "shop.orders"}
	ip := wrapProducer(fp).(*instrumentedProducer)

	_, span := otel.Tracer("test").Start(context.Background(), "caller")
	defer span.End()
	ctx := trace.ContextWithSpan(context.Background(), span)

	msg := &ap.ProducerMessage{}
	require.Nil(t, msg.Properties)
	ip.injectTraceContext(ctx, msg)
	require.NotNil(t, msg.Properties)
	assert.NotEmpty(t, msg.Properties["traceparent"])
}
