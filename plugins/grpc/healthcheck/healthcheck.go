package healthcheck

import (
	context "context"

	"github.com/shoplineapp/go-app/plugins"
	grpc_plugin "github.com/shoplineapp/go-app/plugins/grpc"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcHealthCheckServer)
}

type HealthCheckServer struct {
	UnimplementedHealthServer
}

func (s *HealthCheckServer) Check(ctx context.Context, in *HealthCheckRequest) (*HealthCheckResponse, error) {
	return &HealthCheckResponse{
		Status: HealthCheckResponse_SERVING,
	}, nil
}

func (s *HealthCheckServer) Watch(in *HealthCheckRequest, server Health_WatchServer) error {
	return server.Send(&HealthCheckResponse{
		Status: HealthCheckResponse_SERVING,
	})
}

func (s *HealthCheckServer) Register(grpc *grpc_plugin.GrpcServer) {
	RegisterHealthServer(grpc.Server(), s)
}

func NewGrpcHealthCheckServer(grpc *grpc_plugin.GrpcServer) *HealthCheckServer {
	return &HealthCheckServer{}
}
