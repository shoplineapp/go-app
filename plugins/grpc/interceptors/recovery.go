//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shoplineapp/go-app/plugins"
	app_grpc "github.com/shoplineapp/go-app/plugins/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcErrorRecoveryInterceptor)
}

type StackTracer interface {
	error
	StackTrace() errors.StackTrace
}

type RecoveryInterceptor struct{}

func (i RecoveryInterceptor) Handler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		panicked := true

		defer func() {
			if r := recover(); r != nil || panicked {
				switch r.(type) {
				case StackTracer:
					err = r.(error)
				case error:
					err = r.(error)
					err = errors.WithStack(err)
				default:
					err = errors.Errorf("%+v", r)
				}
				// to be reported in newrelic interceptor
				traceID, _ := ctx.Value("trace_id").(string)
				err = app_grpc.NewApplicationError(traceID, err, codes.Internal, false, "panic recovered from RecoveryInterceptor")
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
