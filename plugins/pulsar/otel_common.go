//go:build pulsar
// +build pulsar

package pulsar

import (
	"errors"
	"fmt"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// Span-name operation tokens. The OTel messaging spec requires
// `{messaging.operation.name} {destination}` for the span name, so
// these are concatenated with a space by spanName() at span-creation
// time. https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/
const (
	spanOpSend    = "send"
	spanOpProcess = "process"
	spanOpAck     = "ack"
	spanOpNack    = "nack"
)

// messagingSystemPulsar: a single shared KeyValue reused across all
// pulsar spans to avoid allocating a new attribute per span.
var messagingSystemPulsar = semconv.MessagingSystemPulsar

// spanName assembles "{operation} {destination}" per the OTel
// messaging spec's span-name convention.
func spanName(op, destination string) string {
	return op + " " + destination
}

// messagingAttributes returns the attributes common to all pulsar
// spans: system, operation name, operation type, and destination name.
// Callers append the operation-specific attributes (message id,
// partition, subscription, consumer group, ...).
func messagingAttributes(opName string, opType attribute.KeyValue, destination string) []attribute.KeyValue {
	return []attribute.KeyValue{
		messagingSystemPulsar,
		semconv.MessagingOperationName(opName),
		opType,
		semconv.MessagingDestinationName(destination),
	}
}

// messagingMessageIDAttrs builds the message.id attribute and, when
// the message lives on a partitioned topic, the partition id. Returns
// nil when id is nil so callers can splat-append without a guard.
func messagingMessageIDAttrs(id ap.MessageID) []attribute.KeyValue {
	if id == nil {
		return nil
	}
	// ap.MessageID has no String() method; fall back to fmt's default
	// formatting, which renders the concrete type's fields.
	out := []attribute.KeyValue{semconv.MessagingMessageID(fmt.Sprintf("%v", id))}
	if idx := id.PartitionIdx(); idx >= 0 {
		out = append(out, semconv.MessagingDestinationPartitionID(fmt.Sprintf("%d", idx)))
	}
	return out
}

// recordSpanError centralizes the OTel error-recording pattern used
// across all pulsar spans: on a non-nil err, RecordError +
// SetStatus(Error) + the error.type attribute. nil is a no-op.
//
// error.type SHOULD be the canonical type name and SHOULD have low
// cardinality. The semconv spec MAYs unwrapping; we walk the wrap
// chain via errorTypeName so a `fmt.Errorf("ctx: %w", inner)` is
// classified as the inner type, not as "*fmt.wrapError".
func recordSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(semconv.ErrorTypeKey.String(errorTypeName(err)))
}

// errorTypeName walks err's wrap chain (via the Unwrap() error
// method) and returns the deepest concrete type's name. Implemented
// with errors.Unwrap — passing a `*error` target to errors.As trips
// the stdmethods vet check, and `errors.Unwrap` is the standard
// chain-walking primitive for single-error wrappers (e.g. the
// `fmt.Errorf("ctx: %w", inner)` chain we actually see in practice).
func errorTypeName(err error) string {
	for err != nil {
		next := errors.Unwrap(err)
		if next == nil {
			return fmt.Sprintf("%T", err)
		}
		err = next
	}
	return ""
}
