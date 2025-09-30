# Newrelic

The base framework on Opentelemetry agent, build tags `MUST` be added

## Usage

See the following sinppet to get the actual Opentelemetry Agent instance

```golang
package main

import (
	"github.com/grafana/pyroscope-go"
	"github.com/shoplineapp/go-app/plugins"
)

func main() {
	tracer := opentelemetry_plugin.GetTracer()
	newCtx, txn := tracer.Start(ctx, "my_method")
}
```

