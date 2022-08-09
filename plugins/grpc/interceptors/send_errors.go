//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrgrpc"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewSendErrorsInterceptor)
}

type SendErrorsInterceptor struct{}

func (i SendErrorsInterceptor) Handler(newrelicApp *newrelic.Application) grpc.UnaryServerInterceptor {
	if newrelicApp == nil {
		return nil
	}

	defaultNewrelicInterceptor := nrgrpc.UnaryServerInterceptor(
		newrelicApp,
		nrgrpc.WithStatusHandler(codes.Unknown, nrgrpc.DefaultInterceptorStatusHandler), // do not trigger duplicated error for grpc status UNKNOWN (), similar to ignore_status_codes for http
	)

	customNewrelicInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		resp, err = handler(ctx, req)

		txn := newrelic.FromContext(ctx)

		if err != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
		}

		return resp, err
	}

	return grpc_middleware.ChainUnaryServer(
		defaultNewrelicInterceptor,
		customNewrelicInterceptor,
	)
}

func NewSendErrorsInterceptor() *SendErrorsInterceptor {
	return &SendErrorsInterceptor{}
}
