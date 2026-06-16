//go:build kitex && otel
// +build kitex,otel

package middlewares

import (
	"context"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/shoplineapp/go-app/common"
	"github.com/shoplineapp/go-app/plugins"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexTraceIDMiddleware)
}

type KitexTraceIDMiddleware struct {
}

// Handler bridges the OTel SpanContext (populated by
// kitex-contrib/obs-opentelemetry's tracing.NewServerSuite) into the legacy
// ctx["trace_id"] string key that downstream consumers (e.g. common.GetTraceID,
// plugins/pulsar/producer.go) still rely on.
func (m KitexTraceIDMiddleware) Handler(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request, response interface{}) error {
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			ctx = common.NewContextWithTraceID(ctx, sc.TraceID().String())
		}
		return next(ctx, request, response)
	}
}

func NewKitexTraceIDMiddleware() *KitexTraceIDMiddleware {
	return &KitexTraceIDMiddleware{}
}
