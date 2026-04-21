//go:build sqs && sqs_worker && otel
// +build sqs,sqs_worker,otel

package sqs_worker

import (
	"context"
	"fmt"

	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/shoplineapp/go-app/common"
	sqsplugin "github.com/shoplineapp/go-app/plugins/sqs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/shoplineapp/go-app/plugins/sqs_worker"

func init() {
	processHook = otelProcessHook
}

// otelProcessHook extracts the upstream trace context from the message's
// MessageAttributes and starts a SpanKindConsumer "process" span as a child
// of that upstream context. The returned context carries the new span, so
// any subsequent SQS call (e.g. DeleteMessage) made with it will be
// captured as a child settle span.
func otelProcessHook(
	ctx context.Context,
	topic, msgID string,
	attrs map[string]*aws_sqs.MessageAttributeValue,
) (context.Context, func(error)) {
	ctx = extractMessageContext(ctx, attrs)
	return startProcessSpan(ctx, topic, msgID)
}

// extractMessageContext extracts W3C trace context from SQS message
// attributes and seeds the legacy "trace_id" context value for backward
// compatibility with existing handlers.
func extractMessageContext(
	ctx context.Context,
	attrs map[string]*aws_sqs.MessageAttributeValue,
) context.Context {
	if attrs == nil {
		return common.NewContextWithTraceID(ctx, "")
	}

	ctx = otel.GetTextMapPropagator().Extract(ctx, sqsplugin.SQSMessageCarrier(attrs))

	spanCtx := trace.SpanContextFromContext(ctx)
	var traceID string
	if spanCtx.IsValid() {
		traceID = spanCtx.TraceID().String()
	} else if v, ok := attrs["trace_id"]; ok && v != nil && v.StringValue != nil {
		traceID = *v.StringValue
	}

	return common.NewContextWithTraceID(ctx, traceID)
}

// startProcessSpan starts a SpanKindConsumer "process" span for a single
// SQS message and returns the new context plus an end function that should
// be called with any processing error.
func startProcessSpan(ctx context.Context, topic, msgID string) (context.Context, func(error)) {
	tracer := otel.Tracer(tracerName)
	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "aws_sqs"),
		attribute.String("messaging.operation.name", "process"),
		attribute.String("messaging.operation.type", "process"),
		attribute.String("messaging.destination.name", topic),
	}
	if msgID != "" {
		attrs = append(attrs, attribute.String("messaging.message.id", msgID))
	}

	ctx, span := tracer.Start(ctx, "process "+topic,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)

	end := func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
		}
		span.End()
	}
	return ctx, end
}
