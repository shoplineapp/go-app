package orders

import (
	go_app "github.com/shoplineapp/go-app"
	"github.com/shoplineapp/go-app/plugins/logger"
)

type OrderModule struct {
	go_app.AppModuleInterface
	controller *OrdersController
}

func (m *OrderModule) Controllers() []interface{} {
	return []interface{}{
		// Register module controller constructors
		NewOrdersController,
	}
}

func (m *OrderModule) Provide() []interface{} {
	return []interface{}{
		func(
			controller *OrdersController,
			logger *logger.Logger,
		) *OrderModule {
			m.controller = controller
			return m
		},
	}
}
