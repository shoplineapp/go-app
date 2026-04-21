//go:build sqs && !otel
// +build sqs,!otel

package sqs

import aws_sqs "github.com/aws/aws-sdk-go/service/sqs"

// instrumentSQSClient is a no-op when the otel build tag is absent, so that
// topic.go can call it unconditionally.
func instrumentSQSClient(_ *aws_sqs.SQS) {}
