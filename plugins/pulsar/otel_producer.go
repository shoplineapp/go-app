//go:build pulsar
// +build pulsar

// Producer spans ("send") align with the OTel messaging semantic
// conventions. The single-message Send API produces a "Send" span
// without a separate "Create" span; the Send span context is used as
// the message creation context. See
// https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/.

package pulsar

import (
	"context"
	"errors"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

type instrumentedProducer struct {
	ap.Producer
	topic string
}

func (p *instrumentedProducer) Send(ctx context.Context, msg *ap.ProducerMessage) (ap.MessageID, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}

	ctx, span := p.startSendSpan(ctx)
	defer span.End()

	p.injectTraceContext(ctx, msg)

	id, err := p.Producer.Send(ctx, msg)
	recordSpanError(span, err)
	if id != nil {
		span.SetAttributes(messagingMessageIDAttrs(id)...)
	}
	return id, err
}

func (p *instrumentedProducer) SendAsync(ctx context.Context, msg *ap.ProducerMessage, callback func(ap.MessageID, *ap.ProducerMessage, error)) {
	if msg == nil {
		callback(nil, nil, errors.New("message is nil"))
		return
	}

	ctx, span := p.startSendSpan(ctx)

	p.injectTraceContext(ctx, msg)

	p.Producer.SendAsync(ctx, msg, func(id ap.MessageID, pm *ap.ProducerMessage, err error) {
		recordSpanError(span, err)
		if id != nil {
			span.SetAttributes(messagingMessageIDAttrs(id)...)
		}
		span.End()
		callback(id, pm, err)
	})
}

// startSendSpan is shared by Send and SendAsync. It opens a
// PRODUCER-kind "send" span with the spec-required attributes.
func (p *instrumentedProducer) startSendSpan(ctx context.Context) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName(spanOpSend, p.topic),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(messagingAttributes(spanOpSend, semconv.MessagingOperationTypeKey.String(spanOpSend), p.topic)...),
	)
}

// injectTraceContext writes the current span context into the message
// Properties using whatever TextMapPropagator is registered globally
// (so user-registered composite propagators work without
// per-call configuration).
func (p *instrumentedProducer) injectTraceContext(ctx context.Context, msg *ap.ProducerMessage) {
	if msg.Properties == nil {
		msg.Properties = make(map[string]string)
	}
	otel.GetTextMapPropagator().Inject(ctx, PulsarMessageCarrier(msg.Properties))
}

func wrapProducer(producer ap.Producer) ap.Producer {
	return &instrumentedProducer{Producer: producer, topic: producer.Topic()}
}
