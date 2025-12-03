//go:build sentry
// +build sentry

package sentry

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/getsentry/sentry-go"

	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/shoplineapp/go-app/plugins/logger"
)

var ErrSentryNotInitialized = errors.New("missing environment variable SENTRY_DSN: sentry is not initialized")

func init() {
	plugins.Registry = append(plugins.Registry, NewSentryAgent)
}

type SentryAgent struct {
	env    *env.Env
	logger *logger.Logger
}

// ConfigOption is a function that configures sentry.ClientOptions
type ConfigOption func(*sentry.ClientOptions)

// Configure initializes Sentry using environment variables with optional overrides
func (a *SentryAgent) Configure(opts ...ConfigOption) error {
	dsn := a.env.GetEnv("SENTRY_DSN")
	if dsn == "" {
		a.logger.Warn("SENTRY_DSN not set, Sentry will not be initialized")
		return ErrSentryNotInitialized
	}

	// Default options from environment variables
	options := sentry.ClientOptions{
		Dsn:            dsn,
		Debug:          a.env.GetEnv("SENTRY_DEBUG") == "true",
		Environment:    a.env.GetEnv("ENVIRONMENT"),
		Release:        a.env.GetEnv("RELEASE"),
		SendDefaultPII: true,
		SampleRate:     a.parseSampleRate(),
	}
	for _, opt := range opts {
		opt(&options)
	}

	if err := sentry.Init(options); err != nil {
		a.logger.Error("Sentry initialization failed:", err)
		return fmt.Errorf("sentry initialization failed: %w", err)
	}

	a.logger.Info("Sentry initialized successfully")
	return nil
}

// NewSentryAgent creates a new Sentry agent instance
func NewSentryAgent(env *env.Env, logger *logger.Logger) *SentryAgent {
	return &SentryAgent{
		env:    env,
		logger: logger,
	}
}

func (a *SentryAgent) parseSampleRate() float64 {
	rate := a.env.GetEnv("SENTRY_SAMPLE_RATE")
	if rate == "" {
		return 0.0
	}
	parsed, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return 0.0
	}
	if parsed >= 0.0 && parsed <= 1.0 {
		return parsed
	}
	return 0.0
}

// Hub returns the current Sentry hub for advanced usage.
func (a *SentryAgent) Hub() *sentry.Hub {
	return sentry.CurrentHub()
}

// HubFromContext returns the Sentry hub from context, or clones the current hub if not found.
func (a *SentryAgent) HubFromContext(ctx context.Context) *sentry.Hub {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub
	}
	return sentry.CurrentHub().Clone()
}

// CaptureException captures an exception.
// When built with the otel tag, it automatically adds the trace ID from the OpenTelemetry span context.
func (a *SentryAgent) CaptureException(ctx context.Context, err error) *sentry.EventID {
	hub := a.HubFromContext(ctx)
	a.addTraceContext(ctx, hub)
	return hub.CaptureException(err)
}

// CaptureMessage captures a message.
// When built with the otel tag, it automatically adds the trace ID from the OpenTelemetry span context.
func (a *SentryAgent) CaptureMessage(ctx context.Context, message string) *sentry.EventID {
	hub := a.HubFromContext(ctx)
	a.addTraceContext(ctx, hub)
	return hub.CaptureMessage(message)
}

// RecoverWithContext recovers from a panic and captures it with Sentry.
// When built with the otel tag, it automatically adds the trace ID from the OpenTelemetry span context.
func (a *SentryAgent) RecoverWithContext(ctx context.Context, err any) *sentry.EventID {
	if err == nil {
		err = recover()
	}
	if err == nil {
		return nil
	}
	hub := a.HubFromContext(ctx)
	a.addTraceContext(ctx, hub)
	return hub.RecoverWithContext(ctx, err)
}
