//go:build postgres
// +build postgres

package postgres

import (
	"context"

	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/uptrace/bun"
)

type PgLogger struct {
	logger *logger.Logger
}

func (d PgLogger) BeforeQuery(c context.Context, q *bun.QueryEvent) context.Context {
	return c
}

func (d PgLogger) AfterQuery(c context.Context, q *bun.QueryEvent) {
	d.logger.Debug(q.Query)
}
