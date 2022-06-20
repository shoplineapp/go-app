# Mongodb

Add mongodb store with [mgm](https://github.com/Kamva/mgm) ORM

## Usage

Add mongodb into the dependency of application and run `Connect` to establish connection to mongodb

```golang
package main

import (
  go_app "github.com/shoplineapp/go-app"
  "github.com/shoplineapp/go-app/plugins/mongodb"
  "github.com/shoplineapp/go-app/plugins/env"
)

type User struct {
  mgm.DefaultModel `bson:",inline"`
  Username         string `bson:"username"`
}

func main() {
  app := go_app.NewApplication()
  app.Run(func(
    env *env.Env,
    mongoStore *mongodb.MongoStore,
  ) {
    // If given environment variable is not found, default value will be returned
    env.SetDefaultEnv(map[string]string{
      "PORT": "3000",
    })

    // Connect mongo with configuration
    mongoStore.Connect(
      env.GetEnv("ATLAS_MONGOID_SESSIONS_DEFAULT_PROTOCOL"),
      env.GetEnv("ATLAS_MONGOID_SESSIONS_DEFAULT_USERNAME"),
      env.GetEnv("ATLAS_MONGOID_SESSIONS_DEFAULT_PASSWORD"),
      env.GetEnv("ATLAS_MONGOID_SESSIONS_DEFAULT_SRV_URI"),
      env.GetEnv("ATLAS_MONGOID_SESSIONS_DEFAULT_DATABASE"),
      env.GetEnv("ATLAS_MONGOID_SESSIONS_DEFAULT_PARAMS"),
    )

    user := &User{}
    mongoStore.Collection("users").SimpleFind(&user, bson.M{
      "username": "test",
    })
    fmt.Println("=== Found user ", user.Username)
  })
}
```

