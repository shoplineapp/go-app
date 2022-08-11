//go:build grpc
// +build grpc

package interceptors

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shoplineapp/go-app/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewSendErrorsInterceptor)
}

type NewrelicInterceptor struct {
}

func (i NewrelicInterceptor) Handler() grpc.UnaryServerInterceptor {
	customNewrelicInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		resp, err = handler(ctx, req)

		txn := newrelic.FromContext(ctx)
		if err != nil {
			st, _ := status.FromError(err)
			txn.AddAttribute("GrpcStatusMessage", st.Message())
			txn.AddAttribute("GrpcStatusCode", st.Code().String())
			txn.NoticeError(nrpkgerrors.Wrap(err))
		}

		return resp, err
	}

	return grpc_middleware.ChainUnaryServer(
		customNewrelicInterceptor,
	)
}

func NewSendErrorsInterceptor() *NewrelicInterceptor {
	return &NewrelicInterceptor{}
}
