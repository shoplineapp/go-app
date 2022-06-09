package interceptors

import (
	"context"

	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcErrorRecoveryInterceptor)
}

type RecoveryInterceptor struct{}

func (i RecoveryInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		panicked := true

		defer func() {
			if r := recover(); r != nil || panicked {
				err = status.Errorf(codes.Internal, "%v", r)
			}
		}()

		resp, err := handler(ctx, req)
		panicked = false
		return resp, err
	}
}

func NewGrpcErrorRecoveryInterceptor() *RecoveryInterceptor {
	return &RecoveryInterceptor{}
}
