//go:build otel
// +build otel

package opentelemetry

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/shoplineapp/go-app/plugins"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtelAgent)
}

type OtelAgent struct{}

type OtelConfig struct {
	AppName string
}

// Configure sets up the global OTel tracer provider and propagator.
//
// The HTTP exporter endpoint is read from OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
// first, then OTEL_EXPORTER_OTLP_ENDPOINT, and finally falls back to
// http://localhost:4318. The OTel Go SDK's otlptracehttp.NewClient() does
// honour those environment variables automatically when no explicit
// WithEndpointURL option is passed, but we pass WithEndpointURL explicitly
// so the configured endpoint is unambiguous and works across SDK versions.
//
// An error handler is installed that logs every OTel SDK failure to
// stderr. Without an error handler the BatchSpanProcessor silently
// retries failed exports forever with no visible output, which makes
// production wiring issues (wrong endpoint, DNS, TLS, etc.) very hard
// to diagnose.
func Configure(config OtelConfig) error {
	ctx := context.Background()

	otel.SetErrorHandler(otelErrHandler(func(err error) {
		log.Printf("[otel] SDK error: %v", err)
	}))

	endpoint := resolveEndpoint()
	log.Printf("[otel] exporter endpoint resolved to %q", endpoint)

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithInsecure(),
	)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(newResource(config.AppName)),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// resolveEndpoint returns the configured OTLP/HTTP traces endpoint
// following OTel environment-variable precedence.
func resolveEndpoint() string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"); v != "" {
		return normaliseEndpoint(v)
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		return normaliseEndpoint(v)
	}
	return "http://localhost:4318"
}

// normaliseEndpoint ensures the endpoint is a full http(s):// URL that
// WithEndpointURL will accept. WithEndpointURL takes a full URL; if the
// env var contains only host:port (which is allowed for
// OTEL_EXPORTER_OTLP_ENDPOINT), we wrap it in http://.
func normaliseEndpoint(v string) string {
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	return "http://" + v
}

func newResource(appName string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(appName),
	)
}

func GetTracer() trace.Tracer {
	return otel.Tracer("")
}

// otelErrHandler implements otel.ErrorHandler so we can route SDK
// errors (failed exports, dropped spans, etc.) into the standard log
// package instead of dropping them on the floor.
type otelErrHandler func(err error)

func (h otelErrHandler) Handle(err error) { h(err) }

func NewOtelAgent() *OtelAgent {
	return &OtelAgent{}
}
