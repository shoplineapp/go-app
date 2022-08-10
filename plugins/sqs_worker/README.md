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

