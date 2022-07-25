# Postgres

Add potgres store with [bun](https://github.com/uptrace/bun) ORM

## Usage

Add potgres into the dependency of application and run `Connect` to establish connection to potgres

```golang
package main

import (
	go_app "github.com/shoplineapp/go-app"
	postgres_presets "github.com/shoplineapp/go-app/plugins/postgres/presets"
	"github.com/shoplineapp/go-app/plugins/env"
)

func main() {
	app := go_app.NewApplication()
	app.Run(func(
		env *env.Env,
		pgStore *postgres_presets.DefaultPostgresStore,
	) {
		pgStore.Connect(
			env.GetEnv("POSTGRES_PROTOCOL"),
			env.GetEnv("POSTGRES_USERNAME"),
			env.GetEnv("POSTGRES_PASSWORD"),
			env.GetEnv("POSTGRES_HOST"),
			env.GetEnv("POSTGRES_DB"),
			env.GetEnv("POSTGRES_PARAMS"),
			env.GetEnv("POSTGRES_ENABLE_LOGGING"),
		)
	})
}
```


Or in simple use case (e.g. in test case)

```golang
package main

import (
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/shoplineapp/go-app/plugins/postgres"
)

func main() {
	env := env.NewEnv()
	logger := logger.NewLogger(env)

	pgStore := postgres.NewPostgresStore(logger)
	pgStore.Connect(
		env.GetEnv("POSTGRES_PROTOCOL"),
		env.GetEnv("POSTGRES_USERNAME"),
		env.GetEnv("POSTGRES_PASSWORD"),
		env.GetEnv("POSTGRES_HOST"),
		env.GetEnv("POSTGRES_DB"),
		env.GetEnv("POSTGRES_PARAMS"),
		env.GetEnv("POSTGRES_ENABLE_LOGGING"),
	)
	defer pgStore.Disconnect()
}
```
