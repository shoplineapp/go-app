//go:build pulsar
// +build pulsar

package pulsar

import (
	"context"
	"fmt"

	ap "github.com/apache/pulsar-client-go/pulsar"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/shoplineapp/go-app/plugins/pulsar"

type instrumentedProducer struct {
	ap.Producer
	topic string
}

func (p *instrumentedProducer) Send(ctx context.Context, msg *ap.ProducerMessage) (ap.MessageID, error) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "send "+p.topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "pulsar"),
			attribute.String("messaging.operation.name", "send"),
			attribute.String("messaging.operation.type", "send"),
			attribute.String("messaging.destination.name", p.topic),
		),
	)
	defer span.End()

	if msg.Properties == nil {
		msg.Properties = make(map[string]string)
	}
	propagation.TraceContext{}.Inject(ctx, PulsarMessageCarrier(msg.Properties))

	id, err := p.Producer.Send(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
	}
	if id != nil {
		span.SetAttributes(attribute.String("messaging.message.id", fmt.Sprintf("%v", id)))
	}
	return id, err
}

func wrapProducer(producer ap.Producer, topic string) ap.Producer {
	return &instrumentedProducer{Producer: producer, topic: topic}
}
