// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//
// Modified by SHOPLINE: added processor options and the SlPrefix option.

package baggagecopy

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Filter returns true if the baggage member should be added to a span.
type Filter func(member baggage.Member) bool

// Option configures a SpanProcessor.
type Option func(*config)

type config struct {
	filter       Filter
	attributeKey func(string) string
}

// WithFilter returns an Option that copies baggage members accepted by filter.
// A nil filter allows all baggage members.
func WithFilter(filter Filter) Option {
	return func(cfg *config) {
		cfg.filter = filter
	}
}

// AllowAllMembers allows all baggage members to be added to a span without
// changing their keys.
var AllowAllMembers Option = func(cfg *config) {
	cfg.filter = func(baggage.Member) bool { return true }
	cfg.attributeKey = func(key string) string { return key }
}

// SlPrefix copies baggage members whose keys start with "sl-". It removes the
// prefix and replaces the remaining hyphens with dots in the span attribute
// key. For example, "sl-user-id" becomes "user.id".
var SlPrefix Option = func(cfg *config) {
	cfg.filter = func(member baggage.Member) bool {
		return strings.HasPrefix(member.Key(), "sl-") && len(member.Key()) > len("sl-")
	}
	cfg.attributeKey = func(key string) string {
		return strings.ReplaceAll(strings.TrimPrefix(key, "sl-"), "-", ".")
	}
}

// SpanProcessor is a [trace.SpanProcessor] implementation that adds baggage
// members onto a span as attributes.
type SpanProcessor struct {
	filter       Filter
	attributeKey func(string) string
}

var _ trace.SpanProcessor = (*SpanProcessor)(nil)

// NewSpanProcessor returns a new SpanProcessor.
//
// The processor duplicates onto a span the baggage found in the parent
// context at the moment the span is started. If no option is provided, all
// baggage members are copied without changing their keys.
func NewSpanProcessor(options ...Option) *SpanProcessor {
	cfg := config{
		filter:       func(baggage.Member) bool { return true },
		attributeKey: func(key string) string { return key },
	}
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	if cfg.filter == nil {
		cfg.filter = func(baggage.Member) bool { return true }
	}

	return &SpanProcessor{
		filter:       cfg.filter,
		attributeKey: cfg.attributeKey,
	}
}

// OnStart is called when a span is started and adds span attributes for
// baggage contents.
func (processor SpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	filter := processor.filter
	if filter == nil {
		filter = func(baggage.Member) bool { return true }
	}
	attributeKey := processor.attributeKey
	if attributeKey == nil {
		attributeKey = func(key string) string { return key }
	}

	for _, member := range baggage.FromContext(ctx).Members() {
		if filter(member) {
			span.SetAttributes(attribute.String(attributeKey(member.Key()), member.Value()))
		}
	}
}

// OnEnd is called when a span is finished and is a no-op for this processor.
func (SpanProcessor) OnEnd(trace.ReadOnlySpan) {}

// Shutdown is called when the SDK shuts down and is a no-op for this processor.
func (SpanProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush exports all ended spans to the configured Exporter that have not
// yet been exported and is a no-op for this processor.
func (SpanProcessor) ForceFlush(context.Context) error { return nil }
