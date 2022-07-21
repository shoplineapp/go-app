package postgres

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

type PgLogger struct{}

func (d PgLogger) BeforeQuery(c context.Context, q *bun.QueryEvent) context.Context {
	return c
}

func (d PgLogger) AfterQuery(c context.Context, q *bun.QueryEvent) {
	fmt.Println(q.Query)
}
