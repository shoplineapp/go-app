# ENV

Load environment variables from `.env` file with default values.

## Usage

Inject to anywhere you need env, variables are loaded on startup

```golang
package main

import (
	go_app "github.com/shoplineapp/go-app"
	"github.com/shoplineapp/go-app/plugins/grpc/presets"
  "github.com/shoplineapp/go-app/plugins/env"
)

func main() {
	app := go_app.NewApplication()
	app.Run(func(
		env *env.Env,
		grpc *presets.DefaultGrpcServerWithNewrelic,
	) {
    // If given environment variable is not found, default value will be returned
    env.SetDefaultEnv(map[string]string{
      "PORT": "3000",
    })

    // Get environment variable with key
    fmt.Printf("We are in %s environment, listening to port %s\n", env.GetEnv("ENVIRONMENT"), env.GetEnv("PORT"))
	})
}
```

---

## Remarks

If `ENVIRONMENT` is `test`, the plugin will look for `.env.test` instead of `.env`

