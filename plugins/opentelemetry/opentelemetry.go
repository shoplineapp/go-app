//go:build otel
// +build otel

package opentelemetry

import (
	"context"
	"fmt"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewOtelAgent)
}

type OpentelemetryParams struct {
	fx.In

	Lifecycle fx.Lifecycle `optional:"true"`
}

var (
	traceExporter  *otlptrace.Exporter
	metricExporter *otlpmetrichttp.Exporter
)

type OtelAgent struct{}

type OtelConfig struct {
	AppName string
}

func Configure(config OtelConfig) error {
	ctx := context.Background()
	client := otlptracehttp.NewClient()
	traceExporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(newResource(config.AppName)),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// TODO not support metric collector for now
	//metricExporter, err := otlpmetrichttp.New(ctx)
	//if err != nil {
	//	return fmt.Errorf("creating OTLP metric exporter: %w", err)
	//}
	//p.metricExporter = metricExporter
	//read := metric.NewPeriodicReader(metricExporter, metric.WithInterval(1*time.Second))
	//provider := metric.NewMeterProvider(metric.WithResource(newResource(config.AppName)), metric.WithReader(read))
	//
	//otel.SetMeterProvider(provider)
	//
	//err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	//if err != nil {
	//	return fmt.Errorf("runtime start fail: %w", err)
	//}

	logrus.Debugf("opentelemetry agent started")

	return nil
}

func newResource(appName string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(appName),
	)
}

func (p *OtelAgent) Shutdown() {
	if traceExporter != nil {
		_ = traceExporter.Shutdown(context.Background())
	}

	if metricExporter != nil {
		_ = metricExporter.Shutdown(context.Background())
	}
}

func GetTracer() trace.Tracer {
	return otel.Tracer("")
}

func NewOtelAgent(params OpentelemetryParams) *OtelAgent {
	agent := &OtelAgent{}
	if params.Lifecycle != nil {
		params.Lifecycle.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				agent.Shutdown()
				return nil
			},
		})
	}
	return agent
}
