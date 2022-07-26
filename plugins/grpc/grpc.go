//go:build grpc
// +build grpc

package grpc

import (
	"fmt"
	"net"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"

	"github.com/shoplineapp/go-app/plugins/logger"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcServer)
}

type GrpcServer struct {
	server   *grpc.Server
	logger   *logger.Logger
	env      *env.Env
	listener *net.Listener
}

func (g GrpcServer) Server() *grpc.Server {
	return g.server
}

func (g GrpcServer) Serve() {
	if g.listener == nil {
		var port string = g.env.GetEnv("GRPC_SERVER_PORT")

		if len(port) == 0 {
			port = "3000"
		}
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			g.logger.WithFields(logrus.Fields{"port": port, "error": err}).Error("Unable to listen to port")
		}
		g.listener = &lis
	}

	go func() {
		lis := *g.listener
		g.logger.Info(fmt.Sprintf("GRPC server is up and running on %s", lis.Addr().String()))
		err := g.server.Serve(lis)
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
	grpc := grpc.NewServer(opt...)
	reflection.Register(grpc)
	g.server = grpc
}

func NewGrpcServer(logger *logger.Logger, env *env.Env) *GrpcServer {
	plugin := &GrpcServer{
		logger: logger,
		env:    env,
	}
	return plugin
}

func (g *GrpcServer) SetListener(lis net.Listener) {
	g.listener = &lis
}
