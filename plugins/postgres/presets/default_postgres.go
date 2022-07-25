//go:build postgres
// +build postgres

package presets

import (
	"context"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/shoplineapp/go-app/plugins/postgres"
	"go.uber.org/fx"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewDefaultPostgresStore)
}

type DefaultPostgresStore struct {
	postgres.PostgresStore
}

func NewDefaultPostgresStore(
	lc fx.Lifecycle,
	logger *logger.Logger,
) *DefaultPostgresStore {
	plugin := &DefaultPostgresStore{
		PostgresStore: *postgres.NewPostgresStore(logger),
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			plugin.PostgresStore.Disconnect()
			return nil
		},
	})

	return plugin
}
