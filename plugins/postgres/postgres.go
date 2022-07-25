//go:build postgres
// +build postgres

package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewPostgresStore)
}

type PostgresStore struct {
	db     *bun.DB
	logger *logger.Logger
}

func generateConnectURL(protocol string, username string, password string, hosts string, databaseName string, params string) string {
	var paramsString string
	var credentials string

	if protocol == "" {
		protocol = "postgres"
	}

	if params != "" {
		paramsString = fmt.Sprintf("?%s", params)
	} else {
		paramsString = ""
	}

	if username != "" || password != "" {
		credentials = fmt.Sprintf("%s:%s@", username, password)
	} else {
		credentials = ""
	}

	return fmt.Sprintf("%s://%s%s/%s%s", protocol, credentials, hosts, databaseName, paramsString)
}

func (s *PostgresStore) Connect(protocol string, username string, password string, hosts string, databaseName string, params string, enableLogging string) {
	url := generateConnectURL(protocol, username, password, hosts, databaseName, params)
	fmt.Printf("url: %v\n", url)
	config, err := pgx.ParseConfig(url)
	if err != nil {
		panic(err)
	}
	config.PreferSimpleProtocol = true

	sqldb := stdlib.OpenDB(*config)
	db := bun.NewDB(sqldb, pgdialect.New())

	if enableLogging == "true" {
		db.AddQueryHook(&PgLogger{
			logger: s.logger,
		})
	}

	s.db = db
}

func (s *PostgresStore) Disconnect() {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("errors: %v", err)
		}
		if s.db != nil {
			s.db = nil
		}
	}()
	s.db.Close()
	s.db = nil
}

func (s *PostgresStore) DB() *bun.DB {
	return s.db
}

func NewPostgresStore(logger *logger.Logger) *PostgresStore {
	return &PostgresStore{
		db:     nil,
		logger: logger,
	}
}
