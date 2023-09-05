//go:build otel
// +build otel

package opentelemetry

import (
	"context"
	"fmt"
	"github.com/shoplineapp/go-app/plugins"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"time"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtelAgent)
}

type OtelAgent struct{}

type OtelConfig struct {
	AppName string
}

func Configure(config OtelConfig) error {
	ctx := context.Background()
	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	exp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return fmt.Errorf("creating OTLP metric exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(newResource(config.AppName)),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	read := metric.NewPeriodicReader(exp, metric.WithInterval(1*time.Second))
	provider := metric.NewMeterProvider(metric.WithResource(newResource(config.AppName)), metric.WithReader(read))

	otel.SetMeterProvider(provider)

	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		return fmt.Errorf("runtime start fail: %w", err)
	}

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

func NewOtelAgent() *OtelAgent {
	return &OtelAgent{}
}
