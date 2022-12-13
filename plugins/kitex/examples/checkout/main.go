package main

import (
	"checkout_service/internal/grpc"
	"checkout_service/internal/orders"

	go_app "github.com/shoplineapp/go-app"
)

func main() {
	app := go_app.NewApplication()

	app.AddModule(&grpc.GrpcModule{})
	app.AddModule(&orders.OrderModule{})
	app.Run(func(
		orderModule *grpc.GrpcModule,
	) {
	})
}
