//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"testing"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// fakeProducer is a minimal ap.Producer stand-in. We implement only
// the methods used by newInstrumentedProducer / instrumentedProducer
// (Topic, Send, SendAsync); the embedded ap.Producer interface is
// left nil because no other method is called from these tests.
type fakeProducer struct {
	ap.Producer
	topic string
	id    ap.MessageID
	// lastMsg captures the most recent message the wrapper passed
	// to the underlying producer — used to assert on property
	// merging without standing up a real broker.
	lastMsg *ap.ProducerMessage
}

func (f *fakeProducer) Topic() string { return f.topic }

func (f *fakeProducer) Send(_ context.Context, msg *ap.ProducerMessage) (ap.MessageID, error) {
	f.lastMsg = msg
	return f.id, nil
}

func (f *fakeProducer) SendAsync(_ context.Context, msg *ap.ProducerMessage, callback func(ap.MessageID, *ap.ProducerMessage, error)) {
	f.lastMsg = msg
	callback(f.id, msg, nil)
}

// TestInstrumentedProducer_Send_EmitsProducerSpan is the producer-side
// happy path. newInstrumentedProducer wraps a fake ap.Producer, and
// Send must open a PRODUCER-kind "send <topic>" span with the
// spec-required messaging.* attributes.
func TestInstrumentedProducer_Send_EmitsProducerSpan(t *testing.T) {
	sr := installTestTracer(t)
	fp := &fakeProducer{
		topic: "shop.orders",
		id:    fakeMessageID{id: "m-1", partition: -1},
	}
	ip := newInstrumentedProducer(fp)

	_, err := ip.Send(context.Background(), &ap.ProducerMessage{Payload: []byte("x")})
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1, "Send must emit exactly one span")
	s := spans[0]
	assert.Equal(t, "send shop.orders", s.Name())
	assert.Equal(t, trace.SpanKindProducer, s.SpanKind())
	attrs := attrsToMap(s.Attributes())
	mustAttr(t, attrs, "messaging.system", "pulsar")
	mustAttr(t, attrs, "messaging.operation.name", "send")
	mustAttr(t, attrs, "messaging.destination.name", "shop.orders")
}

// TestInstrumentedProducer_Send_MergesCallerPropertiesWithTraceparent
// covers the producer's property flow. The framework's TapTraceProperties
// keys (name, host, ip_address) plus any caller-supplied keys must
// ride on the wire together with the W3C traceparent that the OTel
// wrapper injects on top — the final message Properties are the
// union, not a replacement.
func TestInstrumentedProducer_Send_MergesCallerPropertiesWithTraceparent(t *testing.T) {
	sr := installTestTracer(t)
	fp := &fakeProducer{
		topic: "shop.orders",
		id:    fakeMessageID{id: "m-1", partition: -1},
	}
	ip := newInstrumentedProducer(fp)

	// Pre-populated framework-tap keys (what TapTraceProperties would
	// have produced in production).
	callerProps := map[string]string{
		"name":       "fake-producer_producer",
		"host":       "fake-host",
		"ip_address": "10.0.0.1",
	}
	_, err := ip.Send(context.Background(), &ap.ProducerMessage{
		Payload:    []byte("x"),
		Properties: callerProps,
	})
	require.NoError(t, err)

	// The underlying producer received a message whose Properties
	// is the union of the caller's keys and the OTel-injected W3C
	// traceparent — no key is replaced.
	require.NotNil(t, fp.lastMsg, "underlying producer should have received the message")
	got := fp.lastMsg.Properties
	assert.Equal(t, "fake-producer_producer", got["name"], "caller-supplied name must be preserved")
	assert.Equal(t, "fake-host", got["host"], "caller-supplied host must be preserved")
	assert.Equal(t, "10.0.0.1", got["ip_address"], "caller-supplied ip_address must be preserved")
	assert.NotEmpty(t, got["traceparent"], "W3C traceparent must be injected alongside caller properties")

	// Sanity: the send span was still recorded.
	require.Len(t, sr.Ended(), 1)
}
