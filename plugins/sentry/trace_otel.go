//go:build sentry && otel
// +build sentry,otel

package sentry

import (
	"context"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
)

// addTraceContext adds OpenTelemetry trace context to the Sentry scope.
func (a *SentryAgent) addTraceContext(ctx context.Context, hub *sentry.Hub) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("otel.trace_id", spanCtx.TraceID().String())
		scope.SetTag("otel.span_id", spanCtx.SpanID().String())
	})
}
