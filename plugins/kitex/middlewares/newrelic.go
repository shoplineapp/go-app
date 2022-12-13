package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/shoplineapp/go-app/plugins"
	newrelic_plugin "github.com/shoplineapp/go-app/plugins/newrelic"
	"google.golang.org/grpc/metadata"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexNewrelicMiddleware)
}

type KitexNewrelicMiddleware struct {
	nr *newrelic_plugin.NewrelicAgent
}

var (
	mappedHeaders = []struct{ from, to string }{
		{from: newrelic.DistributedTraceNewRelicHeader},
		{from: "user-agent"},
		{from: "x-request-start", to: "x-request-start"},
		{from: "x-queue-start", to: "x-queue-start"},
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

func (m KitexNewrelicMiddleware) Handler(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request, response interface{}) error {
		ri := rpcinfo.GetRPCInfo(ctx)
		txnName := ri.To().Method()
		txn := m.nr.App().StartTransaction(txnName)

		traceId, _ := ctx.Value("trace_id").(string)
		txn.SetWebRequest(newRequest(ctx, txnName))
		txn.AddAttribute("TraceId", traceId)

		ctx = context.WithValue(newrelic.NewContext(ctx, txn), "trace_id", traceId)

		err := next(ctx, request, response)

		if err != nil {
			// FIXME: Newrelic notice error with kitex

			// st, _ := status.FromError(err)
			// txn.AddAttribute("GrpcStatusMessage", st.Message())
			// txn.AddAttribute("GrpcStatusCode", st.Code().String())
			// var ae *app_grpc.ApplicationError
			// if errors.As(err, &ae) {
			// 	if !ae.Expected() {
			// 		nrErr, _ := nrpkgerrors.Wrap(err).(newrelic.Error)
			// 		nrErr.Attributes["trace_id"] = ae.TraceID()
			// 		nrErr.Attributes["details"] = fmt.Sprintf("%#v", ae.Details()) // newrelic doesn't allow sending []interface{} in attributes
			// 		txn.NoticeError(nrErr)
			// 	}
			// } else {
			// report any error if it is not caught as ApplicationError from upper stream
			txn.NoticeError(nrpkgerrors.Wrap(err))
			// }
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			mmd, _ := json.Marshal(md)
			txn.AddAttribute("metadata", string(mmd))
		}

		txn.End()
		return err
	}
}

func NewKitexNewrelicMiddleware(nr *newrelic_plugin.NewrelicAgent) *KitexNewrelicMiddleware {
	return &KitexNewrelicMiddleware{
		nr: nr,
	}
}
