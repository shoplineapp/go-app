//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"fmt"

	"github.com/shoplineapp/go-app/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var messagingSystemPulsar = attribute.String("messaging.system", "pulsar")

func extractMessageContext(ctx context.Context, properties map[string]string) context.Context {
	if properties == nil {
		return common.NewContextWithTraceID(ctx, "")
	}

	ctx = propagation.TraceContext{}.Extract(ctx, PulsarMessageCarrier(properties))

	spanCtx := trace.SpanContextFromContext(ctx)
	var traceID string
	if spanCtx.IsValid() {
		traceID = spanCtx.TraceID().String()
	} else if id, ok := properties["trace_id"]; ok {
		traceID = id
	}

	return common.NewContextWithTraceID(ctx, traceID)
}

func startProcessSpan(ctx context.Context, topic, subscriptionName, msgID string) (context.Context, func(error)) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, fmt.Sprintf("process %s", topic),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			messagingSystemPulsar,
			attribute.String("messaging.operation.name", "process"),
			attribute.String("messaging.operation.type", "process"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.destination.subscription.name", subscriptionName),
			attribute.String("messaging.message.id", msgID),
		),
	)

	endSpan := func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
		}
		span.End()
	}

	return ctx, endSpan
}

func createSettleSpan(ctx context.Context, topic, subscriptionName, operation string, err error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, fmt.Sprintf("%s %s", operation, topic),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			messagingSystemPulsar,
			attribute.String("messaging.operation.name", operation),
			attribute.String("messaging.operation.type", "settle"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.destination.subscription.name", subscriptionName),
		),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}
