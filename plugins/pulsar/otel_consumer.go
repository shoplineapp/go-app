//go:build pulsar
// +build pulsar

// Consumer spans ("process", "ack"/"nack") align with the OTel
// messaging semantic conventions. See
// https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/.
//
// The "Process" span uses the extracted message creation context as
// its parent (single-message scenario, per spec recommendation).

package pulsar

import (
	"context"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"github.com/shoplineapp/go-app/common"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

// extractMessageContext pulls the upstream message creation context
// out of a Pulsar message's properties.
//
// We use the globally registered TextMapPropagator (not a hardcoded
// W3C instance) so that user-registered composite propagators — e.g.
// W3C + B3 + baggage — work without per-call configuration. When the
// propagator yields a valid SpanContext, the trace ID is also
// re-stored under the legacy "trace_id" context value (via
// common.NewContextWithTraceID) so application code that reads it
// directly — the same way the producer side stores it via
// common.GetTraceID — keeps working. The W3C SpanContext stays on
// the returned context unchanged, so the process span started in
// onMessageReceive is still parented on the producer's send span.
func extractMessageContext(ctx context.Context, properties map[string]string) context.Context {
	if len(properties) == 0 {
		return ctx
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, PulsarMessageCarrier(properties))
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		ctx = common.NewContextWithTraceID(ctx, spanCtx.TraceID().String())
	}
	return ctx
}

// startProcessSpan opens a CONSUMER-kind "process" span parented on
// the upstream message creation context (extracted above). The
// returned endSpan closure finalizes the span; pass the handler's
// error to record a failure. Pass nil on success.
func startProcessSpan(ctx context.Context, topic, subscriptionName string, msgID ap.MessageID) (context.Context, func(error)) {
	attrs := messagingAttributes(spanOpProcess, semconv.MessagingOperationTypeProcess, topic)
	attrs = append(attrs,
		semconv.MessagingDestinationSubscriptionName(subscriptionName),
		semconv.MessagingConsumerGroupName(subscriptionName),
	)
	attrs = append(attrs, messagingMessageIDAttrs(msgID)...)

	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName(spanOpProcess, topic),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)

	endSpan := func(err error) {
		recordSpanError(span, err)
		span.End()
	}
	return ctx, endSpan
}

// createSettleSpan opens a CLIENT-kind "ack"/"nack" span. Per the
// spec, settle is a CLIENT-kind operation parented on the process
// span (which is in ctx). The span is ended before returning; callers
// do not need a defer. The operation name (e.g. "ack") is reported
// via messaging.operation.name; the type is always "settle".
func createSettleSpan(ctx context.Context, topic, subscriptionName, operation string, msgID ap.MessageID, err error) {
	attrs := messagingAttributes(operation, semconv.MessagingOperationTypeSettle, topic)
	attrs = append(attrs,
		semconv.MessagingDestinationSubscriptionName(subscriptionName),
		semconv.MessagingConsumerGroupName(subscriptionName),
	)
	attrs = append(attrs, messagingMessageIDAttrs(msgID)...)

	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, spanName(operation, topic),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	recordSpanError(span, err)
	span.End()
}
