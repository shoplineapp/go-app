package middlewares

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexDeadlineMiddleware)
}

type KitexDeadlineMiddleware struct {
	env *env.Env
}

func (m KitexDeadlineMiddleware) Handler(next endpoint.Endpoint) endpoint.Endpoint {
	timeout, pErr := strconv.ParseInt(m.env.GetEnv("KITEX_HANDLER_DEFAULT_TIMEOUT"), 10, 64)
	if pErr != nil {
		timeout = 30
	}
	return func(ctx context.Context, request, response interface{}) error {
		innerCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		resCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			err := next(innerCtx, request, response)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- 1
		}()

		select {
		case _ = <-resCh:
			return nil
		case err := <-errCh:
			return err
		case <-innerCtx.Done():
			return status.Errorf(codes.DeadlineExceeded, "Deadline exceeded or Client cancelled, abandoning")
		}

		return nil
	}
}

func NewKitexDeadlineMiddleware(env *env.Env) *KitexDeadlineMiddleware {
	return &KitexDeadlineMiddleware{env: env}
}
