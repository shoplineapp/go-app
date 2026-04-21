//go:build sqs && otel
// +build sqs,otel

package sqs

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// --- helpers ---

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return exporter
}

func attrMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[string(a.Key)] = a.Value.Emit()
	}
	return m
}

// fakeSQSServer serves canned AWS SQS query-protocol responses. It routes
// based on the "Action" form field and captures the last request body for
// inspection.
type fakeSQSServer struct {
	*httptest.Server
	lastBody string
}

func newFakeSQSServer(t *testing.T) *fakeSQSServer {
	f := &fakeSQSServer{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.lastBody = string(body)
		form, _ := url.ParseQuery(f.lastBody)
		action := form.Get("Action")

		md5Of := func(s string) string {
			sum := md5.Sum([]byte(s))
			return hex.EncodeToString(sum[:])
		}

		w.Header().Set("Content-Type", "text/xml")
		w.Header().Set("x-amzn-RequestId", "test-request-id")
		switch action {
		case "SendMessage":
			bodyMD5 := md5Of(form.Get("MessageBody"))
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<SendMessageResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/">
  <SendMessageResult>
    <MessageId>test-message-id</MessageId>
    <MD5OfMessageBody>` + bodyMD5 + `</MD5OfMessageBody>
  </SendMessageResult>
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</SendMessageResponse>`))
		case "SendMessageBatch":
			var entries strings.Builder
			for i := 1; ; i++ {
				prefix := fmt.Sprintf("SendMessageBatchRequestEntry.%d.", i)
				id := form.Get(prefix + "Id")
				if id == "" {
					break
				}
				body := form.Get(prefix + "MessageBody")
				fmt.Fprintf(&entries, `<SendMessageBatchResultEntry><Id>%s</Id><MessageId>mid-%d</MessageId><MD5OfMessageBody>%s</MD5OfMessageBody></SendMessageBatchResultEntry>`, id, i, md5Of(body))
			}
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<SendMessageBatchResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/">
  <SendMessageBatchResult>` + entries.String() + `</SendMessageBatchResult>
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</SendMessageBatchResponse>`))
		case "ReceiveMessage":
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<ReceiveMessageResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/">
  <ReceiveMessageResult></ReceiveMessageResult>
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</ReceiveMessageResponse>`))
		case "DeleteMessage":
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<DeleteMessageResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/">
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</DeleteMessageResponse>`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(f.Server.Close)
	return f
}

func newTestSQSClient(t *testing.T, endpoint string) *aws_sqs.SQS {
	sess, err := aws_session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials("akid", "secret", ""),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
		MaxRetries:       aws.Int(0),
	})
	require.NoError(t, err)
	client := aws_sqs.New(sess)
	instrumentSQSClient(client)
	return client
}

const testQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue"

// --- tests ---

func TestClassifyOperation(t *testing.T) {
	cases := []struct {
		op       string
		expected opClassification
		ok       bool
	}{
		{"SendMessage", opClassification{trace.SpanKindProducer, "send", "send"}, true},
		{"SendMessageBatch", opClassification{trace.SpanKindProducer, "send", "send_batch"}, true},
		{"ReceiveMessage", opClassification{trace.SpanKindConsumer, "receive", "receive"}, true},
		{"DeleteMessage", opClassification{trace.SpanKindClient, "settle", "delete"}, true},
		{"DeleteMessageBatch", opClassification{trace.SpanKindClient, "settle", "delete_batch"}, true},
		{"ChangeMessageVisibility", opClassification{trace.SpanKindClient, "settle", "change_visibility"}, true},
		{"ChangeMessageVisibilityBatch", opClassification{trace.SpanKindClient, "settle", "change_visibility_batch"}, true},
		{"CreateQueue", opClassification{}, false},
	}
	for _, c := range cases {
		got, ok := classifyOperation(c.op)
		assert.Equal(t, c.ok, ok, c.op)
		if c.ok {
			assert.Equal(t, c.expected, got, c.op)
		}
	}
}

func TestQueueNameFromURL(t *testing.T) {
	assert.Equal(t, "MyQueue", queueNameFromURL(testQueueURL))
	assert.Equal(t, "", queueNameFromURL(""))
	assert.Equal(t, "OnlyName", queueNameFromURL("https://sqs.us-east-1.amazonaws.com/OnlyName"))
}

func TestServerAddressFromURL(t *testing.T) {
	assert.Equal(t, "sqs.us-east-1.amazonaws.com", serverAddressFromURL(testQueueURL))
	assert.Equal(t, "", serverAddressFromURL(""))
}

func TestInstrument_SendMessage_CreatesProducerSpan(t *testing.T) {
	exporter := setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.SendMessage(&aws_sqs.SendMessageInput{
		QueueUrl:    aws.String(testQueueURL),
		MessageBody: aws.String("hello"),
	})
	require.NoError(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	s := spans[0]
	assert.Equal(t, "send MyQueue", s.Name)
	assert.Equal(t, trace.SpanKindProducer, s.SpanKind)

	attrs := attrMap(s.Attributes)
	assert.Equal(t, "aws_sqs", attrs["messaging.system"])
	assert.Equal(t, "send", attrs["messaging.operation.name"])
	assert.Equal(t, "send", attrs["messaging.operation.type"])
	assert.Equal(t, "MyQueue", attrs["messaging.destination.name"])
	assert.Equal(t, testQueueURL, attrs["aws.sqs.queue.url"])
	assert.Equal(t, "sqs.us-east-1.amazonaws.com", attrs["server.address"])
	assert.Equal(t, "test-message-id", attrs["messaging.message.id"])
	assert.Equal(t, "test-request-id", attrs["aws.request_id"])
}

func TestInstrument_SendMessage_InjectsTraceparent(t *testing.T) {
	_ = setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.SendMessage(&aws_sqs.SendMessageInput{
		QueueUrl:    aws.String(testQueueURL),
		MessageBody: aws.String("hello"),
	})
	require.NoError(t, err)

	// The serialized query body must contain a MessageAttribute whose Name
	// is "traceparent".
	assert.Contains(t, fake.lastBody, "MessageAttribute.1.Name=traceparent")
	assert.Contains(t, fake.lastBody, "MessageAttribute.1.Value.DataType=String")
}

func TestInstrument_SendMessage_PreservesExistingAttributes(t *testing.T) {
	_ = setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.SendMessage(&aws_sqs.SendMessageInput{
		QueueUrl:    aws.String(testQueueURL),
		MessageBody: aws.String("hello"),
		MessageAttributes: map[string]*aws_sqs.MessageAttributeValue{
			"custom": {DataType: aws.String("String"), StringValue: aws.String("value")},
		},
	})
	require.NoError(t, err)

	assert.Contains(t, fake.lastBody, "Name=custom")
	assert.Contains(t, fake.lastBody, "Name=traceparent")
}

func TestInstrument_SendMessageBatch_InjectsPerEntry(t *testing.T) {
	_ = setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.SendMessageBatch(&aws_sqs.SendMessageBatchInput{
		QueueUrl: aws.String(testQueueURL),
		Entries: []*aws_sqs.SendMessageBatchRequestEntry{
			{Id: aws.String("e1"), MessageBody: aws.String("a")},
			{Id: aws.String("e2"), MessageBody: aws.String("b")},
		},
	})
	require.NoError(t, err)

	assert.Contains(t, fake.lastBody, "SendMessageBatchRequestEntry.1.MessageAttribute.1.Name=traceparent")
	assert.Contains(t, fake.lastBody, "SendMessageBatchRequestEntry.2.MessageAttribute.1.Name=traceparent")
}

func TestInstrument_SendMessageBatch_SpanAttributes(t *testing.T) {
	exporter := setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.SendMessageBatch(&aws_sqs.SendMessageBatchInput{
		QueueUrl: aws.String(testQueueURL),
		Entries: []*aws_sqs.SendMessageBatchRequestEntry{
			{Id: aws.String("e1"), MessageBody: aws.String("a")},
			{Id: aws.String("e2"), MessageBody: aws.String("b")},
		},
	})
	require.NoError(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "send_batch MyQueue", s.Name)
	assert.Equal(t, trace.SpanKindProducer, s.SpanKind)

	attrs := attrMap(s.Attributes)
	assert.Equal(t, "send_batch", attrs["messaging.operation.name"])
	assert.Equal(t, "2", attrs["messaging.batch.message_count"])
}

func TestInstrument_ReceiveMessage_ConsumerKind(t *testing.T) {
	exporter := setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.ReceiveMessage(&aws_sqs.ReceiveMessageInput{
		QueueUrl: aws.String(testQueueURL),
	})
	require.NoError(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "receive MyQueue", s.Name)
	assert.Equal(t, trace.SpanKindConsumer, s.SpanKind)
	attrs := attrMap(s.Attributes)
	assert.Equal(t, "receive", attrs["messaging.operation.type"])
	assert.Equal(t, "receive", attrs["messaging.operation.name"])
}

func TestInstrument_DeleteMessage_SettleKind(t *testing.T) {
	exporter := setupTestTracer(t)
	fake := newFakeSQSServer(t)
	client := newTestSQSClient(t, fake.URL)

	_, err := client.DeleteMessage(&aws_sqs.DeleteMessageInput{
		QueueUrl:      aws.String(testQueueURL),
		ReceiptHandle: aws.String("rh"),
	})
	require.NoError(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	s := spans[0]
	assert.Equal(t, "delete MyQueue", s.Name)
	assert.Equal(t, trace.SpanKindClient, s.SpanKind)
	attrs := attrMap(s.Attributes)
	assert.Equal(t, "settle", attrs["messaging.operation.type"])
	assert.Equal(t, "delete", attrs["messaging.operation.name"])
}

func TestInstrument_RecordsErrorOnFailure(t *testing.T) {
	exporter := setupTestTracer(t)
	// Point at a server that always 400s for SendMessage.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><ErrorResponse><Error><Type>Sender</Type><Code>InvalidParameterValue</Code><Message>boom</Message></Error><RequestId>err-req</RequestId></ErrorResponse>`))
	}))
	t.Cleanup(srv.Close)

	client := newTestSQSClient(t, srv.URL)
	_, err := client.SendMessage(&aws_sqs.SendMessageInput{
		QueueUrl:    aws.String(testQueueURL),
		MessageBody: aws.String("hello"),
	})
	require.Error(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	attrs := attrMap(spans[0].Attributes)
	assert.Contains(t, attrs, "error.type")
}

func TestInstrument_SkipsUnclassifiedOperation(t *testing.T) {
	exporter := setupTestTracer(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<GetQueueUrlResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/">
  <GetQueueUrlResult><QueueUrl>` + testQueueURL + `</QueueUrl></GetQueueUrlResult>
  <ResponseMetadata><RequestId>r</RequestId></ResponseMetadata>
</GetQueueUrlResponse>`))
	}))
	t.Cleanup(srv.Close)

	client := newTestSQSClient(t, srv.URL)
	_, err := client.GetQueueUrl(&aws_sqs.GetQueueUrlInput{QueueName: aws.String("MyQueue")})
	require.NoError(t, err)

	assert.Empty(t, exporter.GetSpans(), "non-messaging operations should not be instrumented")
}
