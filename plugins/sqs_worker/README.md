# Sqs Worker

SQS consumer with gracefully shutdown and generalized handlings, build tags `MUST` be added

## Usage

See the following sinppet

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  "github.com/shoplineapp/go-app/plugins/sqs_worker"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
	  worker *sqs_worker.AwsSqsWorker,
  ) {
	  _, err := worker.Register(receiveService)
  })
}
```

## OpenTelemetry

When built with `-tags "sqs sqs_worker otel"`, the worker starts a
`SpanKindConsumer` "process" span for each incoming message. The span is
parented to the upstream producer's trace context extracted from the
message's `MessageAttributes` (W3C `traceparent`), so traces span naturally
from the producer through to message processing. The subsequent
`DeleteMessage` call is emitted as a child settle span.
