package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewPostgresStore)
}

type PostgresStore struct {
	DB *bun.DB
}

func (s *PostgresStore) Connect(username string, password string, hosts string, databaseName string) {
	config, err := pgx.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", username, password, hosts, databaseName))
	if err != nil {
		panic(err)
	}
	config.PreferSimpleProtocol = true

	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())

	db.AddQueryHook(&PgLogger{})

	s.DB = db
}

func NewPostgresStore() *PostgresStore {
	return &PostgresStore{}
}
