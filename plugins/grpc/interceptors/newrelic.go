//go:build grpc
// +build grpc

package interceptors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shoplineapp/go-app/plugins"
	newrelic_plugin "github.com/shoplineapp/go-app/plugins/newrelic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewNewrelicInterceptor)
}

type NewrelicInterceptor struct {
	nr *newrelic_plugin.NewrelicAgent
}

var (
	mappedHeaders = []struct{ from, to string }{
		{from: newrelic.DistributedTraceNewRelicHeader},
		{from: "user-agent"},
		{from: "x-request-start", to: "x-request-start"},
		{from: "x-queue-start", to: "x-queue-start"},
		{from: "grpcgateway-x-request-start", to: "x-request-start"},
		{from: "grpcgateway-x-queue-start", to: "x-queue-start"},
	}
)

func newRequest(ctx context.Context, fullMethodName string) newrelic.WebRequest {
	h := http.Header{}
	h.Add("content-type", "application/grpc")
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, m := range mappedHeaders {
			from := m.from
			to := m.to
			if to == "" {
				to = from
			}
			if v := md.Get(from); len(v) > 0 {
				h.Add(to, v[0])
			}
		}
	}

	return newrelic.WebRequest{
		Header: h,
		URL:    &url.URL{Path: fullMethodName},
	}
}

func (i NewrelicInterceptor) Handler() grpc.UnaryServerInterceptor {
	customNewrelicInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		txn := i.nr.App().StartTransaction(info.FullMethod)

		traceId, _ := ctx.Value("trace_id").(string)

		txn.SetWebRequest(newRequest(ctx, info.FullMethod))
		txn.AddAttribute("TraceId", traceId)

		ctx = context.WithValue(newrelic.NewContext(ctx, txn), "trace_id", traceId)

		resp, err = handler(ctx, req)

		if err != nil {
			st, _ := status.FromError(err)
			txn.AddAttribute("GrpcStatusMessage", st.Message())
			txn.AddAttribute("GrpcStatusCode", st.Code().String())
			txn.NoticeError(nrpkgerrors.Wrap(err))
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			mmd, _ := json.Marshal(md)
			txn.AddAttribute("metadata", string(mmd))
		}

		txn.End()

		return resp, err
	}

	return grpc_middleware.ChainUnaryServer(
		customNewrelicInterceptor,
	)
}

func NewNewrelicInterceptor(nr *newrelic_plugin.NewrelicAgent) *NewrelicInterceptor {
	return &NewrelicInterceptor{
		nr: nr,
	}
}
