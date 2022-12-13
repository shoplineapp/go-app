# Kitex

Provide Kitex server with common middlewares.

## Usage

There is a preset kitex server you can use directly
Check `examples/checkout` for workable demo

```golang
package grpc

import (
	orders "checkout_service/internal/orders"
	kitex_gen "checkout_service/kitex_gen/checkoutservice"

	kitex_server "github.com/cloudwego/kitex/server"

	go_app "github.com/shoplineapp/go-app"
	"github.com/shoplineapp/go-app/plugins/kitex"
	"github.com/shoplineapp/go-app/plugins/logger"
)

type GrpcHandler struct {
	orders.OrdersController
}

type GrpcModule struct {
	go_app.AppModuleInterface
	grpcHandler *GrpcHandler
}

func (m *GrpcModule) Controllers() []interface{} {
	return []interface{}{
		// Register module controller constructors
	}
}

func (m *GrpcModule) Provide() []interface{} {
	return []interface{}{
		func(
			kitex *kitex.KitexServer,
			logger *logger.Logger,
			ordersController *orders.OrdersController,
		) *GrpcModule {
			m.grpcHandler = &GrpcHandler{
				OrdersController: *ordersController,
			}

			// Register kitex server with options
			kitex.Configure(func(opts ...kitex_server.Option) kitex_server.Server {
				return kitex_gen.NewServer(m.grpcHandler, opts...)
			})
			return m
		},
	}
}
```

---

## Environment variable

Supporting environment variable configurations

| Key | Type | Description |
| --------- | --- | ---- |
| `KITEX_SERVER_PORT` | string | Control the port that kitex server listen to, default: `3000` |
| `KITEX_HANDLER_DEFAULT_TIMEOUT` | string | Default server-side timeout in seconds, default: `30` |
