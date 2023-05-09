# Newrelic

The base framework on Newrelic agent, build tags `MUST` be added

## Usage

See the following sinppet to get the actual Newrelic Agent instance

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  "github.com/shoplineapp/go-app/plugins/grpc/presets"
  newrelic_plguin "github.com/shoplineapp/go-app/plugins/newrelic"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
    newrelic *newrelic_plguin.NewrelicAgent
    grpc *presets.DefaultGrpcServerWithNewrelic,
  ) {
    newrelic.App().StartTracactions(...)
  })
}
```

