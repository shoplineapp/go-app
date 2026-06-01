//go:build pulsar
// +build pulsar

// Producer spans (send) align with Semantic Conventions v1.41.0.
// Single-message API produces "Send" spans without "Create" spans.
// "Send" span context is used as message creation context.

package pulsar

import (
	"context"
	"errors"
	"fmt"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
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

	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "send "+p.topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			messagingSystemPulsar,
			attribute.String("messaging.operation.name", "send"),
			attribute.String("messaging.operation.type", "send"),
			attribute.String("messaging.destination.name", p.topic),
		),
	)
	defer span.End()

	p.injectTraceContext(ctx, msg)

	id, err := p.Producer.Send(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
	}
	if id != nil {
		attrs := []attribute.KeyValue{
			attribute.String("messaging.message.id", fmt.Sprintf("%v", id)),
		}
		if idx := id.PartitionIdx(); idx >= 0 {
			attrs = append(attrs, attribute.String("messaging.destination.partition.id", fmt.Sprintf("%d", idx)))
		}
		span.SetAttributes(attrs...)
	}
	return id, err
}

func (p *instrumentedProducer) SendAsync(ctx context.Context, msg *ap.ProducerMessage, callback func(ap.MessageID, *ap.ProducerMessage, error)) {
	if msg == nil {
		callback(nil, nil, errors.New("message is nil"))
		return
	}

	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "send "+p.topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			messagingSystemPulsar,
			attribute.String("messaging.operation.name", "send"),
			attribute.String("messaging.operation.type", "send"),
			attribute.String("messaging.destination.name", p.topic),
		),
	)

	p.injectTraceContext(ctx, msg)

	p.Producer.SendAsync(ctx, msg, func(id ap.MessageID, pm *ap.ProducerMessage, err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
		}
		if id != nil {
			attrs := []attribute.KeyValue{
				attribute.String("messaging.message.id", fmt.Sprintf("%v", id)),
			}
			if idx := id.PartitionIdx(); idx >= 0 {
				attrs = append(attrs, attribute.String("messaging.destination.partition.id", fmt.Sprintf("%d", idx)))
			}
			span.SetAttributes(attrs...)
		}
		span.End()
		callback(id, pm, err)
	})
}

func (p *instrumentedProducer) injectTraceContext(ctx context.Context, msg *ap.ProducerMessage) {
	properties := make(map[string]string, len(msg.Properties))
	for k, v := range msg.Properties {
		properties[k] = v
	}
	msg.Properties = properties
	propagation.TraceContext{}.Inject(ctx, PulsarMessageCarrier(msg.Properties))
}

func wrapProducer(producer ap.Producer, topic string) ap.Producer {
	return &instrumentedProducer{Producer: producer, topic: topic}
}
