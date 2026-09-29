//go:build pulsar
// +build pulsar

// OpenTelemetry messaging instrumentation for Apache Pulsar.
//
// Span structure and attributes follow the OTel messaging semantic
// conventions (https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/),
// using constants from go.opentelemetry.io/otel/semconv/v1.27.0.
//
// Span layout:
//   - Producer: "send <topic>"        (SpanKindProducer) — message creation context
//   - Consumer: "process <topic>"     (SpanKindConsumer) — child of the producer trace
//   - Settle:   "ack <topic>" | "nack <topic>"  (SpanKindClient) — child of process

package pulsar

const tracerName = "github.com/shoplineapp/go-app/plugins/pulsar"

// PulsarMessageCarrier adapts a Pulsar message Properties map to the
// propagation.TextMapCarrier interface. It is a map alias (not a
// struct wrapper) because the underlying pulsar property storage is
// itself a map[string]string; aliasing the type avoids a copy on the
// hot path. Use it in both Inject (producer) and Extract (consumer) —
// the OTel propagator writes/reads the W3C traceparent and tracestate
// keys into the same map.
type PulsarMessageCarrier map[string]string

func (c PulsarMessageCarrier) Get(key string) string {
	return c[key]
}

func (c PulsarMessageCarrier) Set(key string, value string) {
	c[key] = value
}

func (c PulsarMessageCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
