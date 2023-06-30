//go:build otlp
// +build otlp

package opentelemetry

import (
	"context"
	"fmt"
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
	plugins.Registry = append(plugins.Registry, NewOtlpAgent)
}

type OtlpAgent struct{}

type OtlpConfig struct {
	AppName string
}

func Configure(config OtlpConfig) error {
	ctx := context.Background()
	client := otlptracehttp.NewClient()
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
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return nil
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

func NewOtlpAgent() *OtlpAgent {
	return &OtlpAgent{}
}
