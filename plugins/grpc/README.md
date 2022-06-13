# gRPC

Provide gRPC server with gracefully shutdown and common interceptors.

## Usage

There is a preset gRPC server you can use direct

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  "github.com/shoplineapp/go-app/plugins/grpc/presets"
  "github.com/shoplineapp/go-app/plugins/logger"
)

func main() {
  app := go_app.NewApplication()

  // Use DefaultGrpcServerWithNewrelic with presets
  app.Run(func(
    logger *logger.Logger,
    grpc *presets.DefaultGrpcServerWithNewrelic,
  ) {
    logger.Info("Hello world")
  })
}
```

Or you prefer to do that manually with your own implementation

```golang
package main

import (
  "context"
  go_app "github.com/shoplineapp/go-app"
  grpc_plugin "github.com/shoplineapp/go-app/plugins/grpc"
  "github.com/shoplineapp/go-app/plugins"
  "github.com/shoplineapp/go-app/plugins/env"
  "github.com/shoplineapp/go-app/plugins/grpc/healthcheck"
  "github.com/shoplineapp/go-app/plugins/grpc/interceptors"
  "github.com/shoplineapp/go-app/plugins/logger"
  "google.golang.org/grpc"
  "go.uber.org/fx"
)

func main() {
  app := go_app.NewApplication()

  // Inject dependencies we need
  app.Run(func(
    lc fx.Lifecycle,
    logger *logger.Logger,
    env *env.Env,
    grpcServer *grpc_plugin.GrpcServer,
    deadline *interceptors.DeadlineInterceptor,
    requestLog *interceptors.RequestLogInterceptor,
    recovery *interceptors.RecoveryInterceptor,
    healthcheckServer *healthcheck.HealthCheckServer,
  ) {
    // Setup gRPC server
    s := *grpcServer
    plugin := &DefaultGrpcServerWithNewrelic{
      GrpcServer: s,
    }
    // Configure with interceptors
    plugin.Configure(
      grpc.ChainUnaryInterceptor(
        requestLog.Handler(),
        deadline.Handler(),
        recovery.Handler(),
      ),
    )
    // Register servers
    healthcheck.RegisterHealthServer(plugin.Server(), healthcheckServer)

    // Use Uber fx lifecycle and trigger gracefully shutdown
    lc.Append(fx.Hook{
      OnStart: func(ctx context.Context) error {
        plugin.Serve()
        return nil
      },
      OnStop: func(ctx context.Context) error {
        plugin.Shutdown()
        return nil
      },
    })
  })
}
```

---

## Environment variable

Supporting environment variable configurations

| Key | Type | Description |
| --------- | --- | ---- |
| `PORT` | string | Control the port that gRPC server listen to, default: `3000` |


