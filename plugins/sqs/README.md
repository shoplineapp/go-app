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

