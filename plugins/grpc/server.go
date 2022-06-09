package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/grpc/healthcheck"
	"github.com/shoplineapp/go-app/plugins/grpc/interceptors"
	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewDefaultGrpcServer, NewGrpcServer)
}

type DefaultGrpcServer struct {
	GrpcServer
}

type GrpcServer struct {
	server *grpc.Server
	logger *logger.Logger
	env    *env.Env
}

func (g GrpcServer) Server() *grpc.Server {
	return g.server
}

func (g GrpcServer) Serve() {
	var port string = g.env.GetEnv("GRPC_SERVER_PORT")
	if len(port) == 0 {
		port = "3000"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		g.logger.WithFields(logrus.Fields{"port": port, "error": err}).Error("Unable to listen to port")
	}

	go func() {
		g.logger.Info(fmt.Sprintf("GRPC server is up and running on 0.0.0.0:%s", port))
		err = g.server.Serve(lis)
		if err != nil {
			g.logger.Fatalf("failed to serve: %v", err)
		}
	}()
}

func (g *GrpcServer) Shutdown() {
	g.logger.Info("GRPC server gracefully shutting down...")
	g.server.GracefulStop()
	g.logger.Info("Bye.")
}

func (g *GrpcServer) Configure(opt ...grpc.ServerOption) {
	grpc := grpc.NewServer()
	reflection.Register(grpc)
	g.server = grpc
}

func NewDefaultGrpcServer(
	lc fx.Lifecycle,
	logger *logger.Logger,
	env *env.Env,
	deadline *interceptors.DeadlineInterceptor,
	requestLog *interceptors.RequestLogInterceptor,
	recovery *interceptors.RecoveryInterceptor,
	healthcheckServer *healthcheck.HealthCheckServer,
) *DefaultGrpcServer {
	plugin := &DefaultGrpcServer{
		GrpcServer: GrpcServer{
			logger: logger,
			env:    env,
		},
	}
	plugin.Configure(
		grpc.ChainUnaryInterceptor(
			requestLog.Handler(),
			deadline.Handler(),
			recovery.Handler(),
		),
	)
	healthcheck.RegisterHealthServer(plugin.Server(), healthcheckServer)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			plugin.Serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			plugin.Shutdown()
			return nil
		},
	})

	return plugin
}

func NewGrpcServer(logger *logger.Logger, env *env.Env) *GrpcServer {
	plugin := &GrpcServer{
		logger: logger,
		env:    env,
	}
	return plugin
}
