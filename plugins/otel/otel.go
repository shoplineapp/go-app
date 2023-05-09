//go:build otel

package otel

import (
	"context"
	"fmt"
	"github.com/shoplineapp/go-app/plugins"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtelAgent)
}

type OtelAgent struct{}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("otlptrace-example"),
		semconv.ServiceVersion("0.0.1"),
	)
}

func Configure() (func(context.Context) error, error) {
	ctx := context.Background()
	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown, nil
}

func NewOtelAgent() *OtelAgent {
	return &OtelAgent{}
}
