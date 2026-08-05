//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"errors"
	"io"
	"testing"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// fakeApConsumer records Ack/Nack calls made by the manager. The
// embedded ap.Consumer is left nil — only Ack and Nack are invoked
// from the code path under test, and both are re-implemented below.
type fakeApConsumer struct {
	ap.Consumer
	ackCount  int
	nackCount int
}

func (f *fakeApConsumer) Ack(_ ap.Message) error {
	f.ackCount++
	return nil
}

func (f *fakeApConsumer) Nack(_ ap.Message) {
	f.nackCount++
}

// fakeApMessage is a minimal ap.Message stand-in. We implement only
// the methods read by onMessageReceive (Topic, Properties, Payload,
// ID); the embedded ap.Message interface is left nil because no
// other method is called from these tests.
type fakeApMessage struct {
	ap.Message
	topic      string
	properties map[string]string
	payload    []byte
	id         ap.MessageID
}

func (f *fakeApMessage) Topic() string                { return f.topic }
func (f *fakeApMessage) Properties() map[string]string { return f.properties }
func (f *fakeApMessage) Payload() []byte              { return f.payload }
func (f *fakeApMessage) ID() ap.MessageID             { return f.id }

// fakeHandler implements PulsarConsumerInterface. Receive returns
// the configured err so the same test can drive both the ack
// (err == nil) and nack (err != nil) paths in onMessageReceive.
type fakeHandler struct {
	err error
}

func (f *fakeHandler) Label() string                       { return "fake-consumer" }
func (f *fakeHandler) Topic() interface{}                  { return "fake-topic" }
func (f *fakeHandler) ConsumerOptions() *ap.ConsumerOptions { return &ap.ConsumerOptions{SubscriptionName: "fake_consumer_subscription"} }
func (f *fakeHandler) Receive(_ context.Context, _ ap.ConsumerMessage) error { return f.err }

// silentLogger returns a *logger.Logger that discards all output.
// The nack path in onMessageReceive calls cm.logger.WithFields(...).Error(...)
// — we need a non-nil logger there to avoid a nil-pointer panic, but
// don't want the test output cluttered.
func silentLogger() *logger.Logger {
	return &logger.Logger{Logger: logrus.Logger{Out: io.Discard, Level: logrus.PanicLevel}}
}

// assertMessagingAttrs checks the OTel messaging-semconv attributes
// that the consumer code sets on every process/settle span: system,
// operation name, operation type, destination, subscription,
// consumer group, and message id. The partition id is asserted
// absent because the test fixture's fakeMessageID has
// partition == -1, so the conditional attribute must not be
// emitted.
func assertMessagingAttrs(t *testing.T, span sdktrace.ReadOnlySpan, opName, opType, topic, subscription string) {
	t.Helper()
	attrs := attrsToMap(span.Attributes())
	mustAttr(t, attrs, "messaging.system", "pulsar")
	mustAttr(t, attrs, "messaging.operation.name", opName)
	mustAttr(t, attrs, "messaging.operation.type", opType)
	mustAttr(t, attrs, "messaging.destination.name", topic)
	mustAttr(t, attrs, "messaging.destination.subscription.name", subscription)
	mustAttr(t, attrs, "messaging.consumer.group.name", subscription)
	mustAttrPresent(t, attrs, "messaging.message.id")
	mustAttrAbsent(t, attrs, "messaging.destination.partition.id")
}

// TestPulsarConsumer_OnMessageReceive_EmitsProcessAndSettleSpans is
// the consumer-side end-to-end check. It drives onMessageReceive
// with a fake PulsarConsumer + fake ap.ConsumerMessage and asserts:
//   - exactly two spans are emitted (one process + one settle),
//   - the process span is CONSUMER-kind, parented on the upstream
//     W3C traceparent, and carries the spec-required
//     messaging.* attributes,
//   - the settle span is CLIENT-kind, named "ack" or "nack", and
//     carries the same messaging.* attributes with
//     operation.type = "settle",
//   - the nack case records error.type + Error status on BOTH
//     spans; the ack case records neither,
//   - the underlying ap.Consumer receives the corresponding Ack or
//     Nack call.
func TestPulsarConsumer_OnMessageReceive_EmitsProcessAndSettleSpans(t *testing.T) {
	// Upstream W3C trace injected via the producer's
	// injectTraceContext — the consumer's process span must be
	// parented on this trace, not on a fresh root.
	const upstreamTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	const upstreamSpanID = "00f067aa0ba902b7"

	cases := []struct {
		name       string
		handlerErr error
		wantSettle string // "ack" or "nack"
		wantAck    int
		wantNack   int
	}{
		{"handler success -> ack", nil, "ack", 1, 0},
		{"handler error -> nack", errors.New("simulated failure"), "nack", 0, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := installTestTracer(t)

			apConsumer := &fakeApConsumer{}
			msg := ap.ConsumerMessage{
				Consumer: apConsumer,
				Message: &fakeApMessage{
					topic: "persistent://public/default/fake-otel-1",
					properties: map[string]string{
						"traceparent": "00-" + upstreamTraceID + "-" + upstreamSpanID + "-01",
					},
					payload: []byte("payload"),
					id:      fakeMessageID{id: "m-1", partition: -1},
				},
			}
			consumer := &PulsarConsumer{
				Consumer: apConsumer,
				Handler:  &fakeHandler{err: tc.handlerErr},
				options:  &ap.ConsumerOptions{SubscriptionName: "fake_consumer_subscription"},
			}
			cm := &PulsarConsumerManager{
				logger:    silentLogger(),
				consumers: map[string]*PulsarConsumer{"fake-consumer": consumer},
			}

			cm.onMessageReceive(consumer, msg)

			spans := sr.Ended()
			require.Len(t, spans, 2, "expected 1 process span + 1 settle span")

			// Identify spans by kind rather than by recorder order.
			// End order is [settle, process] (createSettleSpan ends its span
			// synchronously; the process span is ended by the deferred
			// endSpan when onMessageReceive returns), so position alone
			// is not a reliable identifier.
			var process, settle sdktrace.ReadOnlySpan
			for _, s := range spans {
				switch s.SpanKind() {
				case trace.SpanKindConsumer:
					process = s
				case trace.SpanKindClient:
					settle = s
				}
			}
			require.NotNil(t, process, "process span (CONSUMER kind) must be recorded")
			require.NotNil(t, settle, "settle span (CLIENT kind) must be recorded")

			// Process span: CONSUMER kind, parented on the upstream W3C trace.
			assert.Equal(t, "process persistent://public/default/fake-otel-1", process.Name())
			assert.Equal(t, upstreamTraceID, process.SpanContext().TraceID().String(),
				"process span must inherit the upstream W3C trace ID")
			parent := process.Parent()
			require.True(t, parent.IsValid(), "process span must have a parent (the upstream W3C span)")
			assert.Equal(t, upstreamSpanID, parent.SpanID().String(),
				"process span must be parented on the upstream W3C span ID")

			// Settle span: CLIENT kind, name matches the expected operation.
			assert.Equal(t, tc.wantSettle+" persistent://public/default/fake-otel-1", settle.Name())

			// Spec-required OTel messaging attributes on both spans.
			const (
				topic        = "persistent://public/default/fake-otel-1"
				subscription = "fake_consumer_subscription"
			)
			assertMessagingAttrs(t, process, "process", "process", topic, subscription)
			assertMessagingAttrs(t, settle, tc.wantSettle, "settle", topic, subscription)

			// Error attributes + status: on the nack path the handler
			// error flows into BOTH the process span (via the deferred
			// endSpan) AND the settle span (via createSettleSpan's
			// recordSpanError call). On the ack path neither span
			// should carry error info.
			processAttrs := attrsToMap(process.Attributes())
			settleAttrs := attrsToMap(settle.Attributes())
			if tc.wantSettle == "nack" {
				mustAttrPresent(t, processAttrs, "error.type")
				mustAttrPresent(t, settleAttrs, "error.type")
				assert.Equal(t, codes.Error, process.Status().Code,
					"process span must have Error status when the handler fails")
				assert.Equal(t, codes.Error, settle.Status().Code,
					"nack span must have Error status when the handler fails")
				assert.NotEmpty(t, process.Events(),
					"process span must record an exception event when the handler fails")
				assert.NotEmpty(t, settle.Events(),
					"nack span must record an exception event when the handler fails")
			} else {
				mustAttrAbsent(t, processAttrs, "error.type")
				mustAttrAbsent(t, settleAttrs, "error.type")
				assert.Equal(t, codes.Unset, process.Status().Code,
					"process span must have Unset status on success")
				assert.Equal(t, codes.Unset, settle.Status().Code,
					"ack span must have Unset status on success")
			}

			// Underlying ap.Consumer received the right call.
			assert.Equal(t, tc.wantAck, apConsumer.ackCount)
			assert.Equal(t, tc.wantNack, apConsumer.nackCount)
		})
	}
}
