//go:build sqs && otel
// +build sqs,otel

package sqs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws/request"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/shoplineapp/go-app/plugins/sqs"

// opClassification describes how a given SQS API operation should be mapped
// onto OTel messaging semantic conventions.
type opClassification struct {
	spanKind trace.SpanKind
	opType   string // messaging.operation.type
	opName   string // messaging.operation.name
}

// classifyOperation returns the classification for a given SQS API operation
// name. The second return value is false for operations that are not
// messaging-relevant and should be skipped.
func classifyOperation(name string) (opClassification, bool) {
	switch name {
	case "SendMessage":
		return opClassification{trace.SpanKindProducer, "send", "send"}, true
	case "SendMessageBatch":
		return opClassification{trace.SpanKindProducer, "send", "send_batch"}, true
	case "ReceiveMessage":
		return opClassification{trace.SpanKindConsumer, "receive", "receive"}, true
	case "DeleteMessage":
		return opClassification{trace.SpanKindClient, "settle", "delete"}, true
	case "DeleteMessageBatch":
		return opClassification{trace.SpanKindClient, "settle", "delete_batch"}, true
	case "ChangeMessageVisibility":
		return opClassification{trace.SpanKindClient, "settle", "change_visibility"}, true
	case "ChangeMessageVisibilityBatch":
		return opClassification{trace.SpanKindClient, "settle", "change_visibility_batch"}, true
	}
	return opClassification{}, false
}

// instrumentSQSClient attaches OpenTelemetry instrumentation to a v1
// aws-sdk-go SQS client. It:
//   - starts a span in the Validate phase (before params are serialized)
//   - injects W3C trace context into outgoing message attributes for send
//     operations (also Validate phase, after span start)
//   - ends the span in the Complete phase, recording any error and setting
//     AWS/messaging semconv attributes.
func instrumentSQSClient(c *aws_sqs.SQS) {
	if c == nil {
		return
	}
	c.Handlers.Validate.PushFrontNamed(request.NamedHandler{
		Name: "otel.sqs.StartSpan",
		Fn:   startSpan,
	})
	c.Handlers.Validate.PushBackNamed(request.NamedHandler{
		Name: "otel.sqs.InjectTraceContext",
		Fn:   injectTraceContext,
	})
	c.Handlers.Complete.PushBackNamed(request.NamedHandler{
		Name: "otel.sqs.EndSpan",
		Fn:   endSpan,
	})
}

func startSpan(r *request.Request) {
	class, ok := classifyOperation(r.Operation.Name)
	if !ok {
		return
	}

	queueURL := extractQueueURL(r.Params)
	queueName := queueNameFromURL(queueURL)

	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "aws_sqs"),
		attribute.String("messaging.operation.name", class.opName),
		attribute.String("messaging.operation.type", class.opType),
	}
	if queueURL != "" {
		attrs = append(attrs, attribute.String("aws.sqs.queue.url", queueURL))
	}
	if queueName != "" {
		attrs = append(attrs, attribute.String("messaging.destination.name", queueName))
	}
	if addr := serverAddressFromURL(queueURL); addr != "" {
		attrs = append(attrs, attribute.String("server.address", addr))
	}
	if batch, ok := batchCount(r.Params); ok {
		attrs = append(attrs, attribute.Int("messaging.batch.message_count", batch))
	}

	spanName := class.opName
	if queueName != "" {
		spanName = class.opName + " " + queueName
	}

	tracer := otel.Tracer(tracerName)
	ctx, _ := tracer.Start(r.Context(), spanName,
		trace.WithSpanKind(class.spanKind),
		trace.WithAttributes(attrs...),
	)
	r.SetContext(ctx)
}

func injectTraceContext(r *request.Request) {
	propagator := otel.GetTextMapPropagator()
	switch p := r.Params.(type) {
	case *aws_sqs.SendMessageInput:
		if p == nil {
			return
		}
		if p.MessageAttributes == nil {
			p.MessageAttributes = map[string]*aws_sqs.MessageAttributeValue{}
		}
		propagator.Inject(r.Context(), SQSMessageCarrier(p.MessageAttributes))
	case *aws_sqs.SendMessageBatchInput:
		if p == nil {
			return
		}
		for i := range p.Entries {
			if p.Entries[i] == nil {
				continue
			}
			if p.Entries[i].MessageAttributes == nil {
				p.Entries[i].MessageAttributes = map[string]*aws_sqs.MessageAttributeValue{}
			}
			propagator.Inject(r.Context(), SQSMessageCarrier(p.Entries[i].MessageAttributes))
		}
	}
}

func endSpan(r *request.Request) {
	span := trace.SpanFromContext(r.Context())
	if !span.IsRecording() {
		return
	}

	if r.RequestID != "" {
		span.SetAttributes(attribute.String("aws.request_id", r.RequestID))
	}

	switch out := r.Data.(type) {
	case *aws_sqs.SendMessageOutput:
		if out != nil && out.MessageId != nil {
			span.SetAttributes(attribute.String("messaging.message.id", *out.MessageId))
		}
	}

	if r.Error != nil {
		span.RecordError(r.Error)
		span.SetStatus(codes.Error, r.Error.Error())
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", r.Error)))
	}

	span.End()
}

// extractQueueURL returns the QueueUrl from a known SQS input type, or "".
func extractQueueURL(params interface{}) string {
	switch p := params.(type) {
	case *aws_sqs.SendMessageInput:
		return derefString(p.QueueUrl)
	case *aws_sqs.SendMessageBatchInput:
		return derefString(p.QueueUrl)
	case *aws_sqs.ReceiveMessageInput:
		return derefString(p.QueueUrl)
	case *aws_sqs.DeleteMessageInput:
		return derefString(p.QueueUrl)
	case *aws_sqs.DeleteMessageBatchInput:
		return derefString(p.QueueUrl)
	case *aws_sqs.ChangeMessageVisibilityInput:
		return derefString(p.QueueUrl)
	case *aws_sqs.ChangeMessageVisibilityBatchInput:
		return derefString(p.QueueUrl)
	}
	return ""
}

// batchCount returns the number of entries in a batch request, if applicable.
func batchCount(params interface{}) (int, bool) {
	switch p := params.(type) {
	case *aws_sqs.SendMessageBatchInput:
		return len(p.Entries), true
	case *aws_sqs.DeleteMessageBatchInput:
		return len(p.Entries), true
	case *aws_sqs.ChangeMessageVisibilityBatchInput:
		return len(p.Entries), true
	}
	return 0, false
}

// queueNameFromURL extracts the queue name (last path segment) from a
// standard SQS queue URL such as
// https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue.
func queueNameFromURL(queueURL string) string {
	if queueURL == "" {
		return ""
	}
	u, err := url.Parse(queueURL)
	if err != nil {
		return ""
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// serverAddressFromURL returns the host portion of an SQS queue URL.
func serverAddressFromURL(queueURL string) string {
	if queueURL == "" {
		return ""
	}
	u, err := url.Parse(queueURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
