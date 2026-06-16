//go:build kitex && !otel
// +build kitex,!otel

package middlewares

import (
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/shoplineapp/go-app/plugins"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewKitexTraceIDMiddleware)
}

type KitexTraceIDMiddleware struct {
}

// Handler is a no-op when OpenTelemetry is not enabled. The OTel bridge lives
// in trace_id_otel.go; without OTel there is no SpanContext to copy into
// ctx["trace_id"], so the legacy key stays unset and common.GetTraceID
// returns "" for Kitex requests.
func (m KitexTraceIDMiddleware) Handler(next endpoint.Endpoint) endpoint.Endpoint {
	return next
}

func NewKitexTraceIDMiddleware() *KitexTraceIDMiddleware {
	return &KitexTraceIDMiddleware{}
}
