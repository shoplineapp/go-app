//go:build sqs && !otel

package sqs

import aws_sqs "github.com/aws/aws-sdk-go/service/sqs"

func InstrumentSQSClient(c *aws_sqs.SQS) {}

func instrument(c *aws_sqs.SQS) {}
