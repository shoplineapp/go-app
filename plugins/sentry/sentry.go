//go:build sentry
// +build sentry

package sentry

import (
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
		SendDefaultPII: a.env.GetEnv("SENTRY_SEND_DEFAULT_PII") == "true",
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
