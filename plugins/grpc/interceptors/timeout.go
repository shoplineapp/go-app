package interceptors

import (
	"context"
	"strconv"
	"time"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcDeadlineInterceptor)
}

type DeadlineInterceptor struct {
	env *env.Env
}

func (h DeadlineInterceptor) Handler() grpc.UnaryServerInterceptor {
	timeout, pErr := strconv.ParseInt(h.env.GetEnv("GRPC_HANDLER_DEFAULT_TIMEOUT"), 10, 64)
	if pErr != nil {
		timeout = 30
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		innerCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		resCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			res, err := handler(innerCtx, req)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- res
		}()

		select {
		case res := <-resCh:
			return res, nil
		case err := <-errCh:
			return nil, err
		case <-innerCtx.Done():
			return nil, status.Errorf(codes.DeadlineExceeded, "Deadline exceeded or Client cancelled, abandoning")
		}
	}
}

func NewGrpcDeadlineInterceptor(env *env.Env) *DeadlineInterceptor {
	return &DeadlineInterceptor{env: env}
}
