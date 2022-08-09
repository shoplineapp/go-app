//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
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
				switch r.(type) {
				case error:
					err = r.(error)
					err = errors.WithStack(err)
				default:
					err = errors.Errorf("%+v", r)
				}
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
