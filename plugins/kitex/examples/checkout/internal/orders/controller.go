package orders

import (
	"context"

	pb "checkout_service/kitex_gen"

	"github.com/shoplineapp/go-app/plugins/logger"
)

type OrdersController struct {
	logger *logger.Logger
}

func NewOrdersController(logger *logger.Logger) *OrdersController {
	c := &OrdersController{
		logger: logger,
	}
	return c
}

func (c OrdersController) Healthcheck(ctx context.Context, req *pb.Request) (resp *pb.Response, err error) {
	return &pb.Response{
		Message: "OK",
	}, nil
}
