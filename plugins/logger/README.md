# Logger

Provide a formatted Logrus logger with your presets.

## Usage

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  "github.com/shoplineapp/go-app/plugins/grpc/presets"
  "github.com/shoplineapp/go-app/plugins/logger"
)

func main() {
  app := go_app.NewApplication()
  app.Run(func(
    logger *logger.Logger,
    grpc *presets.DefaultGrpcServerWithNewrelic,
  ) {
    logger.Info("Hello world")
  })
}
```

---

## Environment variable

Supporting environment variable configurations

| Key | Type | Description |
| --------- | --- | ---- |
| `LOG_LEVEL` | string | Control the log level of logger, possible values: `info`, `debug`, `trace` |
| `LOG_TO_CLOUDWATCH` | boolean | Use JSON formatter on logs |
| `ENVIRONMENT` | string | When environment is `production, logs are forced to JSON format |

