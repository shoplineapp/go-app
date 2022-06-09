package healthcheck

import (
	context "context"

	"github.com/shoplineapp/go-app/plugins"
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

func NewGrpcHealthCheckServer() *HealthCheckServer {
	return &HealthCheckServer{}
}
