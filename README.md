# Go App

A domain-driven microframework with extendable plugin architecture

## Overview

It provides the essential features and stucture for building a scaleable service. It comes with some common usage like gRPC server (replaceable with the modular design), you can still swap some plugins Fully extendable, easy-to-use plugin architecture with dependency injection. 

## Getting Started

Minimal setup to create an service with an User module.

```golang
package main

import (
  "my_api/internal/user"

  go_app "github.com/shoplineapp/go-app"
  defaults "github.com/shoplineapp/go-app/plugins/grpc/defaults"
)

func main() {
  // Create a new application
  app := go_app.NewApplication()

  // Register module of your business logic
  app.AddModule(&user.UserModule{})

  // Given the init function with required dependencies
  // Noted that this will be the entrypoint of your dependency tree
  app.Run(func(
    // Inject module/plugin you need here
    userModule *user.UserModule,
    grpc *defaults.DefaultGrpcServerWithNewrelic,
  ) {
  })
}
```

Start the application

```sh
$ go run cmd/api.go
INFO[0000] PROVIDE plugin *env.Env                      
INFO[0000] PROVIDE plugin *logger.Logger                
INFO[0000] PROVIDE plugin *grpc.GrpcServer              
INFO[0000] PROVIDE plugin *healthcheck.HealthCheckServer 
INFO[0000] PROVIDE plugin *interceptors.RecoveryInterceptor 
INFO[0000] PROVIDE plugin *interceptors.RequestLogInterceptor 
INFO[0000] PROVIDE plugin *interceptors.DeadlineInterceptor 
INFO[0000] PROVIDE plugin *newrelic.NewrelicAgent       
INFO[0000] PROVIDE plugin *stats_handlers.NewrelicStatsHandler 
INFO[0000] PROVIDE plugin *defaults.DefaultGrpcServerWithNewrelic 
INFO[0000] PROVIDE plugin *controllers.UsersController  
INFO[0000] PROVIDE plugin *user.UserModule              
INFO[0000] PROVIDE plugin fx.Lifecycle                  
INFO[0000] PROVIDE plugin fx.Shutdowner                 
INFO[0000] PROVIDE plugin fx.DotGraph                   
INFO[0000] = User module init                           
INFO[0000] Application RUNNING                          
INFO[0000] GRPC server is up and running on 0.0.0.0:3000 
^CWARN[0001] Received INTERRUPT                           
INFO[0001] GRPC server gracefully shutting down...      
INFO[0001] Bye.
```

---

Example code of a module

```golang
package user

import (
  "my_api/internal/user/controllers"
  "my_api/protos"
  "github.com/shoplineapp/go-app/plugins/grpc/defaults"
  "github.com/shoplineapp/go-app/plugins/logger"
  go_app "github.com/shoplineapp/go-app"
)

type UserModule struct {
  go_app.AppModuleInterface
  controller *controllers.UsersController
}

func (m *UserModule) Controllers() []interface{} {
  return []interface{}{
    // Register module controller constructors
    controllers.NewUsersController,
  }
}

func (m *UserModule) Provide() []interface{} {
  return []interface{}{
    // Register module with dependencies
    // Requires all the constructor of structs that you need dependency injection
    func(
      controller *controllers.UsersController,
      grpc *defaults.DefaultGrpcServerWithNewrelic,
      logger *logger.Logger,
    ) *UserModule {
      // Register gRPC server with controller
      protos.RegisterUsersServer(grpc.Server(), controller)
      return m
    },
  }
}
```

Example code of module controller

```golang
package controllers

import (
  "context"
  "my_api/protos"
  "github.com/shoplineapp/go-app/plugins/logger"
)

type UsersController struct {
  test.UnimplementedUsersServer
  logger *logger.Logger
}

// Constructor of controller with dependencies needed
func NewUsersController(logger *logger.Logger) *UsersController {
  c := &UsersController{
    logger: logger,
  }
  return c
}

// gRPC handler
func (c UsersController) Userinfo(context.Context, *protos.UserinfoRequest) (*protos.UserinfoEmptyResponse, error) {
  c.logger.Info("Hello there")
  return &protos.UserinfoEmptyResponse{}, nil
}
```

See the [examples](https://github.com/shoplineapp/go-app/tree/master/examples) for detailed information on usage.

---

## Plugins

Available plugins are under [plugins](https://github.com/shoplineapp/go-app/tree/master/plugins) folder.

Some examples:

- gRPC Server (with build tag `grpc`)
- Common gRPC interceptors like request log, server-side timeout, recover
- Logrus logger
- Environment variable with .env file and default values
- Newrelic integration (with build tag `newrelic`)

Plugins are autoloaded and optionally controlled by build tags.