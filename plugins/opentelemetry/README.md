# Newrelic

The base framework on Opentelemetry agent, build tags `MUST` be added

## Usage

See the following sinppet to get the actual Opentelemetry Agent instance

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  "github.com/shoplineapp/go-app/plugins/grpc/presets"
  newrelic_plguin "github.com/shoplineapp/go-app/plugins/newrelic"
)

func main() {
	otel.Tracer("grpc_request")
	newCtx, txn := tracer.Start(ctx, info.FullMethod)
}
```

