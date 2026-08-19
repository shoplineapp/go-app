// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//
// Modified by SHOPLINE: added coverage for processor options and SlPrefix.

package baggagecopy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

type testExporter struct {
	spans []trace.ReadOnlySpan
}

func (exporter *testExporter) ExportSpans(_ context.Context, spans []trace.ReadOnlySpan) error {
	exporter.spans = append(exporter.spans, spans...)
	return nil
}

func (*testExporter) Shutdown(context.Context) error { return nil }

func TestSpanProcessorAllowAllMembers(t *testing.T) {
	attributes := collectAttributes(t, AllowAllMembers,
		member(t, "baggage.test", "baggage value"),
	)

	require.Equal(t, []attribute.KeyValue{
		attribute.String("baggage.test", "baggage value"),
	}, attributes)
}

func TestSpanProcessorWithFilter(t *testing.T) {
	attributes := collectAttributes(t, WithFilter(func(member baggage.Member) bool {
		return member.Key() == "included"
	}),
		member(t, "included", "yes"),
		member(t, "excluded", "no"),
	)

	require.Equal(t, []attribute.KeyValue{
		attribute.String("included", "yes"),
	}, attributes)
}

func TestSpanProcessorSlPrefix(t *testing.T) {
	attributes := collectAttributes(t, SlPrefix,
		member(t, "sl-user-id", "123"),
		member(t, "sl-request-source-name", "storefront"),
		member(t, "other-key", "not copied"),
		member(t, "sl-", "not copied"),
	)

	require.ElementsMatch(t, []attribute.KeyValue{
		attribute.String("user.id", "123"),
		attribute.String("request.source.name", "storefront"),
	}, attributes)
}

func TestSpanProcessorDefaultsToAllowAllMembers(t *testing.T) {
	attributes := collectAttributes(t, nil, member(t, "key", "value"))

	require.Equal(t, []attribute.KeyValue{
		attribute.String("key", "value"),
	}, attributes)
}

func TestZeroSpanProcessorDefaultsToAllowAllMembers(t *testing.T) {
	bag, err := baggage.New(member(t, "key", "value"))
	require.NoError(t, err)
	ctx := baggage.ContextWithBaggage(context.Background(), bag)
	span := &recordingSpan{}

	require.NotPanics(t, func() {
		new(SpanProcessor).OnStart(ctx, span)
	})
	require.Equal(t, []attribute.KeyValue{
		attribute.String("key", "value"),
	}, span.attributes)
}

type recordingSpan struct {
	trace.ReadWriteSpan
	attributes []attribute.KeyValue
}

func (span *recordingSpan) SetAttributes(attributes ...attribute.KeyValue) {
	span.attributes = append(span.attributes, attributes...)
}

func collectAttributes(t *testing.T, option Option, members ...baggage.Member) []attribute.KeyValue {
	t.Helper()

	bag, err := baggage.New(members...)
	require.NoError(t, err)
	ctx := baggage.ContextWithBaggage(context.Background(), bag)

	exporter := &testExporter{}
	provider := trace.NewTracerProvider(
		trace.WithSpanProcessor(NewSpanProcessor(option)),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })

	_, span := provider.Tracer("test").Start(ctx, "test")
	span.End()

	require.Len(t, exporter.spans, 1)
	return exporter.spans[0].Attributes()
}

func member(t *testing.T, key, value string) baggage.Member {
	t.Helper()
	result, err := baggage.NewMemberRaw(key, value)
	require.NoError(t, err)
	return result
}
