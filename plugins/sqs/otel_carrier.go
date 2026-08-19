//go:build sqs && otel
// +build sqs,otel

package sqs

import (
	"github.com/aws/aws-sdk-go/aws"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	"go.opentelemetry.io/otel/propagation"
)

// SQSMessageCarrier adapts an SQS MessageAttributeValue map to the
// propagation.TextMapCarrier interface so W3C trace context can be injected
// and extracted from SQS messages.
type SQSMessageCarrier map[string]*aws_sqs.MessageAttributeValue

var _ propagation.TextMapCarrier = SQSMessageCarrier{}

func (c SQSMessageCarrier) Get(key string) string {
	if v, ok := c[key]; ok && v != nil && v.StringValue != nil {
		return *v.StringValue
	}
	return ""
}

func (c SQSMessageCarrier) Set(key, val string) {
	c[key] = &aws_sqs.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(val),
	}
}

func (c SQSMessageCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
