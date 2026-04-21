# Sqs

Provide a plugin to maintain SQS queue clients and receive/send messages, build tags `MUST` be added

## Usage

See the following sinppet

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
  "github.com/shoplineapp/go-app/plugins/sqs"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
	  sqs sqs.AwsTopicManager,
  ) {
	  topic := sqs.AddTopic("Example", "AWS_ARN")
	  topic.SendMessage(&aws_sqs.SendMessageInput{})
  })
}
```

## OpenTelemetry

When built with `-tags "sqs otel"`, every `*sqs.SQS` client produced by
`AwsTopicManager.AddTopic` is transparently instrumented via the
`aws-sdk-go` v1 request handler chain. Each SQS API call emits a span
following the OTel messaging semantic conventions:

| Operation                                     | Span kind | `messaging.operation.type` | `messaging.operation.name` |
| --------------------------------------------- | --------- | -------------------------- | -------------------------- |
| `SendMessage`                                 | Producer  | `send`                     | `send`                     |
| `SendMessageBatch`                            | Producer  | `send`                     | `send_batch`               |
| `ReceiveMessage`                              | Consumer  | `receive`                  | `receive`                  |
| `DeleteMessage(Batch)`                        | Client    | `settle`                   | `delete[_batch]`           |
| `ChangeMessageVisibility(Batch)`              | Client    | `settle`                   | `change_visibility[_batch]`|

Common attributes include `messaging.system=aws_sqs`,
`messaging.destination.name`, `aws.sqs.queue.url`, `aws.request_id`,
`server.address`, and — for `SendMessage` — `messaging.message.id`.

W3C trace context (`traceparent`) is injected into the outgoing message's
`MessageAttributes` (per entry for `SendMessageBatch`), enabling downstream
consumers to continue the trace.
