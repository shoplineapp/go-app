//go:build kitex
// +build kitex

package kitex

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/server"
	kitex_server "github.com/cloudwego/kitex/server"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/kitex/middlewares"
	"github.com/shoplineapp/go-app/plugins/logger"
	"go.uber.org/fx"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexServer)
}

type KitexServer struct {
	logger *logger.Logger
	env    *env.Env

	wg          *sync.WaitGroup
	server      server.Server
	middlewares []endpoint.Middleware
	kitexExit   chan error
}

func (s *KitexServer) Configure(initializer func(opts ...kitex_server.Option) kitex_server.Server) {
	s.logger.Info("Kitex server started")

	// Default server options
	s.kitexExit = make(chan error, 1)

	// Default port
	var port string = s.env.GetEnv("KITEX_SERVER_PORT")
	if len(port) == 0 {
		port = "3000"
	}
	addr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%s", port))
	options := []kitex_server.Option{
		kitex_server.WithServiceAddr(addr),
		kitex_server.WithExitSignal(func() <-chan error {
			return s.kitexExit
		}),
	}

	if s.middlewares != nil {
		for _, middleware := range s.middlewares {
			options = append(options, kitex_server.WithMiddleware(middleware))
		}
	}

	s.server = initializer(options...)
	kitex_server.RegisterShutdownHook(func() {
		s.logger.Info("GRPC server gracefully shutting down...")
	})
}

func (s *KitexServer) SetMiddlewares(middlewares []endpoint.Middleware) {
	s.middlewares = middlewares
}

func (s *KitexServer) RegisterGracefullyShutdown(lc fx.Lifecycle) {
	s.wg = &sync.WaitGroup{}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			s.wg.Add(1)
			go s.server.Run()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Manually trigger exit signal to kitex and let it handle the graceful shutdown
			s.kitexExit <- nil

			go func() {
				for {
					time.Sleep(100 * time.Millisecond)

					// Check if kitex cleaned up the server internally
					// https://github.com/cloudwego/kitex/blob/e93d7b77e5d4239d054b81e3e0dee9290a4eccd3/server/server.go#L228
					svr := reflect.ValueOf(s.server).Elem().FieldByName("svr")
					if svr.IsNil() {
						s.wg.Done()
						break
					}
				}
			}()

			s.wg.Wait()
			s.logger.Info("Bye")
			return nil
		},
	})
}

func NewKitexServer(
	lc fx.Lifecycle,
	logger *logger.Logger,
	env *env.Env,
	traceIDMiddleware *middlewares.KitexTraceIDMiddleware,
	requestLogMiddleware *middlewares.KitexRequestLogMiddleware,
	deadlineMiddleware *middlewares.KitexDeadlineMiddleware,
) *KitexServer {
	plugin := &KitexServer{
		logger: logger,
		env:    env,
		middlewares: []endpoint.Middleware{
			traceIDMiddleware.Handler,
			requestLogMiddleware.Handler,
			deadlineMiddleware.Handler,
		},
	}
	plugin.RegisterGracefullyShutdown(lc)
	return plugin
}
