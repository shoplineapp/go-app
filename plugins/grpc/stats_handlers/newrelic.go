//go:build grpc && newrelic
// +build grpc,newrelic

package stats_handlers

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shoplineapp/go-app/plugins"
	newrelic_plugin "github.com/shoplineapp/go-app/plugins/newrelic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewGrpcNewrelicStatsHandler)
}

type NewrelicStatsHandler struct {
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

func (i *NewrelicStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	traceId := uuid.New().String()

	if i.nr.App() == nil {
		ctx = context.WithValue(ctx, "trace_id", traceId)
		return ctx
	}

	txn := i.nr.App().StartTransaction(info.FullMethodName)

	txn.SetWebRequest(newRequest(ctx, info.FullMethodName))
	txn.AddAttribute("TraceId", traceId)
	ctx = context.WithValue(newrelic.NewContext(ctx, txn), "trace_id", traceId)

	return ctx
}

func (i *NewrelicStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	if i.nr.App() == nil {
		return
	}

	switch s := s.(type) {
	case *stats.End:
		txn := newrelic.FromContext(ctx)
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			txn.AddAttribute("metadata", md)
		}

		if err := s.Error; err != nil {
			if st, ok := status.FromError(s.Error); ok {
				txn.AddAttribute("grpcStatusCode", st.Code())
			}
			txn.NoticeError(err)
		} else {
			txn.AddAttribute("grpcStatusCode", codes.OK)
		}

		txn.End()
	}
}

func (i *NewrelicStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	// no-op
	return ctx
}

func (i *NewrelicStatsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
	// no-op
}

func NewGrpcNewrelicStatsHandler(nr *newrelic_plugin.NewrelicAgent) *NewrelicStatsHandler {
	return &NewrelicStatsHandler{
		nr: nr,
	}
}
