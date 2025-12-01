//go:build sentry && !otel
// +build sentry,!otel

package sentry

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// addTraceContext is a no-op when OpenTelemetry is not enabled.
func (a *SentryAgent) addTraceContext(ctx context.Context, hub *sentry.Hub) {
	// No-op: OpenTelemetry is not enabled
}
