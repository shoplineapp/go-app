package grpc

import (
	orders "checkout_service/internal/orders"
	kitex_gen "checkout_service/kitex_gen/checkoutservice"

	kitex_server "github.com/cloudwego/kitex/server"

	go_app "github.com/shoplineapp/go-app"
	kitex_presets "github.com/shoplineapp/go-app/plugins/kitex/presets"
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
			kitex *kitex_presets.DefaultKitexServerWithNewrelic,
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
